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
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
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
	Name         string     `yaml:"name"`
	Serial       string     `yaml:"serial"`
	Nics         []NicDef   `yaml:"nics"`
	Disks        []QemuDisk `yaml:"disks"`
	Boot         string     `yaml:"boot"`
	Cdrom        string     `yaml:"cdrom"`
	UefiVars     string     `yaml:"uefi-vars"`
	TPM          bool       `yaml:"tpm"`
	TPMVersion   string     `yaml:"tpm-version"`
	KVMExtraOpts []string   `yaml:"extra-opts"`
	SecureBoot   bool       `yaml:"secure-boot"`
	Gui          bool       `yaml:"gui"`
}

type NicDef struct {
	BusAddr   string `yaml:"addr"`
	Device    string `yaml:"device"`
	ID        string `yaml:"id"`
	Mac       string `yaml:"mac"`
	IFName    string
	Network   string
	Ports     []PortRule `yaml:"ports"`
	BootIndex string     `yaml:"bootindex"`
}

// Ports are a list of PortRules
// nics:
//  - id: nic1
//    ports:
//      - "tcp:localhost:22222": "localhost:22"
//      - 1234: 23
//      - 8080: 80

// A PortRule is a single entry map where the key and value represent
// the host and guest mapping respectively. The Host and Guest value

type PortRule struct {
	Protocol string
	Host     Port
	Guest    Port
}

type Port struct {
	Address string
	Port    int
}

func (p *PortRule) UnmarshalYAML(unmarshal func(interface{}) error) error {
	DefaultPortProtocol := "tcp"
	DefaultPortHostAddress := ""
	DefaultPortGuestAddress := ""
	var ruleVal map[string]string
	var err error

	if err = unmarshal(&ruleVal); err != nil {
		return err
	}

	for hostVal, guestVal := range ruleVal {
		hostToks := strings.Split(hostVal, ":")
		if len(hostToks) == 3 {
			p.Protocol = hostToks[0]
			p.Host.Address = hostToks[1]
			p.Host.Port, err = strconv.Atoi(hostToks[2])
			if err != nil {
				return err
			}
		} else if len(hostToks) == 2 {
			p.Protocol = DefaultPortProtocol
			p.Host.Address = hostToks[0]
			p.Host.Port, err = strconv.Atoi(hostToks[1])
			if err != nil {
				return err
			}
		} else {
			p.Protocol = DefaultPortProtocol
			p.Host.Address = DefaultPortHostAddress
			p.Host.Port, err = strconv.Atoi(hostToks[0])
			if err != nil {
				return err
			}
		}
		guestToks := strings.Split(guestVal, ":")
		if len(guestToks) == 2 {
			p.Guest.Address = guestToks[0]
			p.Guest.Port, err = strconv.Atoi(guestToks[1])
			if err != nil {
				return err
			}
		} else {
			p.Guest.Address = DefaultPortGuestAddress
			p.Guest.Port, err = strconv.Atoi(guestToks[0])
			if err != nil {
				return err
			}
		}
		break
	}
	if p.Protocol != "tcp" && p.Protocol != "udp" {
		return fmt.Errorf("Invalid PortRule.Protocol value: %s . Must be 'tcp' or 'udp'", p.Protocol)
	}
	return nil
}

func (p *PortRule) String() string {
	return fmt.Sprintf("%s:%s:%d-%s:%d", p.Protocol,
		p.Host.Address, p.Host.Port, p.Guest.Address, p.Guest.Port)
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

type VM struct {
	Ctx    context.Context
	Cancel context.CancelFunc
	Config VMDef
	cmd    *exec.Cmd
	State  VMState
}

func newVM(ctx context.Context, vmConfig VMDef) (VM, error) {
	ctx, cancelFn := context.WithCancel(ctx)
	return VM{
		Config: vmConfig,
		Ctx:    ctx,
		Cancel: cancelFn,
		State:  VMInit,
	}, nil
}

func (v *VM) Start() error {
	fmt.Printf("Starting VM:%s\n", v.Config.Name)
	v.State = VMStarted
	go func(v *VM) {
		fmt.Printf("VM:%s running until cancelled\n", v.Config.Name)
		for {
			select {
			case <-v.Ctx.Done():
				err := v.Ctx.Err()
				if err != nil {
					fmt.Printf("Error with VM:%s %s\n", v.Config.Name, err)
				}
				fmt.Printf("VM:%s finished\n", v.Config.Name)
				return
			}
		}
	}(v)
	return nil
}

func (v *VM) Stop() error {
	fmt.Printf("Stopping VM:%s\n", v.Config.Name)
	fmt.Printf("Called v.Cancel()")
	v.Cancel()
	fmt.Printf("Set State to VMStopped")
	v.State = VMStopped
	fmt.Printf("returning nil")
	return nil
}
