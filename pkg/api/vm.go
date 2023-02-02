/*

Copyright © 2022 Ryan Harper <rharper@woxford.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package api

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"mcli-v2/pkg/qcli"

	"github.com/apex/log"
)

type VMState int

const (
	VMInit VMState = iota
	VMStarted
	VMStopped
	VMFailed
	VMCleaned
)

type VMDef struct {
	Name       string     `yaml:"name"`
	Cpus       uint32     `yaml:"cpus"`
	Memory     uint32     `yaml:"memory"`
	Serial     string     `yaml:"serial"`
	Nics       []NicDef   `yaml:"nics"`
	Disks      []QemuDisk `yaml:"disks"`
	Boot       string     `yaml:"boot"`
	Cdrom      string     `yaml:"cdrom"`
	UefiVars   string     `yaml:"uefi-vars"`
	TPM        bool       `yaml:"tpm"`
	TPMVersion string     `yaml:"tpm-version"`
	SecureBoot bool       `yaml:"secure-boot"`
	Gui        bool       `yaml:"gui"`
}

var QemuTypeIndex map[string]int

// Allocate the next number per Qemu Type string
// This is use to create unique, increasing index integers used to
// enumerate qemu id= parameters used to bind various objects together
// on the QEMU command line: e.g
//
// -object iothread,id=iothread2
// -drive id=drv1
// -device scsi-hd,drive=drv1,iothread=iothread2
//
func getNextQemuIndex(qtype string) int {
	currentIndex := 0
	ok := false
	if QemuTypeIndex == nil {
		QemuTypeIndex = make(map[string]int)
	}
	if currentIndex, ok = QemuTypeIndex[qtype]; !ok {
		currentIndex = -1
	}
	QemuTypeIndex[qtype] = currentIndex + 1
	return QemuTypeIndex[qtype]
}

func clearAllQemuIndex() {
	for key := range QemuTypeIndex {
		delete(QemuTypeIndex, key)
	}
}

// TODO: Rename fields
type VM struct {
	Ctx    context.Context
	Cancel context.CancelFunc
	Config VMDef
	State  VMState
	RunDir string
	Cmd    *exec.Cmd
	qcli   *qcli.Config
	qmp    *qcli.QMP
	wg     sync.WaitGroup
}

func newVM(ctx context.Context, clusterName string, vmConfig VMDef) (VM, error) {
	ctx, cancelFn := context.WithCancel(ctx)
	runDir := filepath.Join(ctx.Value(clsCtxStateDir).(string), vmConfig.Name)

	log.Infof("newVM: Generating qcli Config rundir=%s", runDir)
	qcfg, err := GenerateQConfig(runDir, vmConfig)
	if err != nil {
		return VM{}, fmt.Errorf("Failed to generate qcli Config from VM definition: %s", err)
	}

	cmdParams, err := qcli.ConfigureParams(qcfg, nil)
	if err != nil {
		return VM{}, fmt.Errorf("Failed to generate new VM command parameters: %s", err)
	}
	log.Infof("newVM: generated qcli config parameters: %s", cmdParams)

	return VM{
		Config: vmConfig,
		Ctx:    ctx,
		Cancel: cancelFn,
		State:  VMInit,
		Cmd:    exec.Command(qcfg.Path, cmdParams...),
		qcli:   qcfg,
		RunDir: runDir,
	}, nil
}

func (v *VM) Name() string {
	return v.Config.Name
}

func (v *VM) runVM() error {
	// add to waitgroup and spawn goroutine to run the command
	errCh := make(chan error, 1)

	v.wg.Add(1)
	go func() {
		var stderr bytes.Buffer
		defer v.wg.Done()

		log.Infof("VM:%s starting QEMU process", v.Name())
		v.Cmd.Stderr = &stderr
		err := v.Cmd.Start()
		if err != nil {
			errCh <- fmt.Errorf("VM:%s failed with: %s", stderr.String())
			return
		}

		log.Infof("VM:%s waiting for QEMU process to exit...", v.Name())
		err = v.Cmd.Wait()
		if err != nil {
			errCh <- fmt.Errorf("VM:%s wait failed with: %s", stderr.String())
			return
		}
		log.Infof("VM:%s QEMU process exited", v.Name())
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		if err != nil {
			log.Errorf("runVM failed: %s", err)
			return err
		}
	}

	return nil
}

func (v *VM) StartQMP() error {
	var wg sync.WaitGroup
	errCh := make(chan error, 1)
	qmpCh := make(chan struct{})

	// FIXME: are there more than one qmp sockets allowed?
	numQMP := len(v.qcli.QMPSockets)
	if numQMP != 1 {
		return fmt.Errorf("StartQMP failed, expected 1 QMP socket, found: %d", numQMP)
	}

	// start qmp goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		// watch for qmp/monitor/serial sockets
		waitOn, err := qcli.GetSocketPaths(v.qcli)
		if err != nil {
			errCh <- fmt.Errorf("StartQMP failed to fetch VM socket paths: %s", err)
			return
		}

		// wait up to for 10 seconds for each.
		for _, sock := range waitOn {
			if !WaitForPath(sock, 10, 1) {
				errCh <- fmt.Errorf("VM:%s socket %s does not exist", v.Name(), sock)
				return
			}
		}

		qmpCfg := qcli.QMPConfig{
			Logger: QMPMachineLogger{},
		}

		qmpSocketFile := v.qcli.QMPSockets[0].Name
		log.Infof("VM:%s connecting to QMP socket %s", v.Name(), qmpSocketFile)
		q, qver, err := qcli.QMPStart(v.Ctx, qmpSocketFile, qmpCfg, qmpCh)
		if err != nil {
			errCh <- fmt.Errorf("Failed to connect to qmp socket: %s", err.Error())
			return
		}
		log.Infof("VM:%s QMP:%v QMPVersion:%v", v.Name(), q, qver)

		// This has to be the first command executed in a QMP session.
		err = q.ExecuteQMPCapabilities(v.Ctx)
		if err != nil {
			errCh <- err
			return
		}

		v.qmp = q
		errCh <- nil
	}()

	// wait until qmp setup is complete (or failed)
	wg.Wait()

	select {
	case err := <-errCh:
		if err != nil {
			log.Errorf("StartQMP failed: %s", err)
			return err
		}
	}

	return nil
}

func (v *VM) BackgroundRun() error {
	// start vm command in background goroutine
	err := v.runVM()
	if err != nil {
		return err
	}

	err = v.StartQMP()
	if err != nil {
		return err
	}

	return nil
}

func (v *VM) Start() error {
	log.Infof("VM:%s starting...", v.Name())
	err := v.BackgroundRun()
	if err != nil {
		log.Errorf("VM:%s failed to start VM:%s %s", v.Name(), err)
		return err
	}
	v.State = VMStarted
	return nil
}

func (v *VM) Stop() error {
	log.Infof("VM:%s stopping...\n", v.Name())

	// FIXME: configurable?
	// Try shutdown via QMP, wait up to 10 seconds before force shutting down
	timeout := time.Second * 10

	// Let's try to shutdown the VM.  If it hasn't shutdown in 10 seconds we'll
	// send a quit message.
	log.Infof("VM:%s trying graceful shutdown via system_powerdown (%s timeout before cancelling)..", v.Name(), timeout.String())
	err := v.qmp.ExecuteSystemPowerdown(v.Ctx)
	if err != nil {
		log.Errorf("VM:%s error:%s", v.Name(), err.Error())
	}

	select {
	case <-v.Ctx.Done():
		log.Infof("VM:%s has exited without cancel", v.Name())
	case <-time.After(timeout):
		log.Warnf("VM:%s timed out, killing via cancel context...", v.Name())
		v.Cancel()
	}
	v.wg.Wait()
	v.State = VMStopped
	return nil
}

func (v *VM) IsRunning() bool {
	if v.State == VMStarted {
		return true
	}
	return false
}

func (v *VM) Delete() error {
	log.Infof("VM:%s deleting self...", v.Name())
	if v.IsRunning() {
		err := v.Stop()
		if err != nil {
			return fmt.Errorf("Failed to delete VM:%s :%s", v.Name(), err)
		}
	}

	if PathExists(v.RunDir) {
		log.Infof("VM:%s removing state dir: %q", v.Name(), v.RunDir)
		return os.RemoveAll(v.RunDir)
	}

	return nil
}
