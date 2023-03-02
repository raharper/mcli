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
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	MachineStatusStopped  string = "stopped"
	MachineStatusStarting string = "starting"
	MachineStatusRunning  string = "running"
	MachineStatusStopping string = "stopping"
)

type StopChannel chan struct{}

type MachineController struct {
	Machines []Machine
}

type Machine struct {
	ctx         context.Context
	Type        string `yaml:"type"`
	Config      VMDef  `yaml:"config"`
	Description string `yaml:"description"`
	Ephemeral   bool   `yaml:"ephemeral"`
	Name        string `yaml:"name"`
	Status      string
	statusCode  int64
	vmCount     sync.WaitGroup
	instance    *VM
}

func (ctl *MachineController) GetMachineByName(machineName string) (*Machine, error) {
	log.Infof("FindMachineByName: machines:%v machineName: %s", ctl.Machines, machineName)
	for _, machine := range ctl.Machines {
		if machine.Name == machineName {
			log.Infof("found it")
			return &machine, nil
		}
	}
	log.Infof("returning")
	return &Machine{}, fmt.Errorf("Failed to find machine with Name: %s", machineName)
}

func (ctl *MachineController) GetMachines() []Machine {
	return ctl.Machines
}

func (ctl *MachineController) GetMachine(machineName string) (Machine, error) {
	for _, machine := range ctl.Machines {
		if machine.Name == machineName {
			return machine, nil
		}
	}
	return Machine{}, fmt.Errorf("Failed to find machine with Name: %s", machineName)
}

func (ctl *MachineController) AddMachine(newMachine Machine, cfg *MachineDaemonConfig) error {
	if _, err := ctl.GetMachineByName(newMachine.Name); err == nil {
		return fmt.Errorf("Machine '%s' is already defined", newMachine.Name)
	}
	newMachine.Status = MachineStatusStopped
	newMachine.ctx = cfg.GetConfigContext()
	if !newMachine.Ephemeral {
		if err := newMachine.SaveConfig(); err != nil {
			return fmt.Errorf("Could not save '%s' machine to %q: %s", newMachine.Name, newMachine.ConfigFile(), err)
		}
	}
	ctl.Machines = append(ctl.Machines, newMachine)
	return nil
}

func (ctl *MachineController) StopMachines() error {
	for idx, _ := range ctl.Machines {
		machine := ctl.Machines[idx]
		if machine.IsRunning() {
			if err := machine.Stop(false); err != nil {
				log.Infof("Error while stopping machine '%s': %s", machine.Name, err)
			}
		}
	}
	return nil
}

func (ctl *MachineController) DeleteMachine(machineName string, cfg *MachineDaemonConfig) error {
	machines := []Machine{}
	for idx, _ := range ctl.Machines {
		machine := ctl.Machines[idx]
		if machine.Name != machineName {
			machines = append(machines, machine)
		} else {
			err := machine.Delete()
			if err != nil {
				return fmt.Errorf("Machine:%s delete failed: %s", machine.Name, err)
			}
			log.Infof("Deleted machine: %s", machine.Name)
		}
	}
	ctl.Machines = machines
	return nil
}

func (ctl *MachineController) UpdateMachine(updateMachine Machine, cfg *MachineDaemonConfig) error {
	// FIXME: decide if update will modify the in-memory state (I think yes, but
	// maybe only the on-disk format if it's running? but what does subsequent
	// GET return (on-disk or in-memory?)

	for idx, machine := range ctl.Machines {
		if machine.Name == updateMachine.Name {
			updateMachine.ctx = cfg.GetConfigContext()
			ctl.Machines[idx] = updateMachine
			if !updateMachine.Ephemeral {
				if err := updateMachine.SaveConfig(); err != nil {
					return fmt.Errorf("Could not save '%s' machine to %q: %s", updateMachine.Name, updateMachine.ConfigFile(), err)
				}
			}
			log.Infof("Updated machine '%s'", updateMachine.Name)
			break
		}
	}
	return nil
}

func (ctl *MachineController) StartMachine(machineName string) error {
	for idx, machine := range ctl.Machines {
		if machine.Name == machineName {
			err := ctl.Machines[idx].Start()
			if err != nil {
				return fmt.Errorf("Could not start '%s' machine: %s", machineName, err)
			}
			return nil
		}
	}
	return fmt.Errorf("Failed to find machine '%s', cannot start unknown machine", machineName)
}

func (ctl *MachineController) StopMachine(machineName string, force bool) error {
	for idx, machine := range ctl.Machines {
		if machine.Name == machineName {
			err := ctl.Machines[idx].Stop(force)
			if err != nil {
				return fmt.Errorf("Could not stop '%s' machine: %s", machineName, err)
			}
			return nil
		}
	}
	return fmt.Errorf("Failed to find machine '%s', cannot stop unknown machine", machineName)
}

//
// Machine Functions Below
//
func (cls *Machine) ConfigDir() string {
	return filepath.Join(cls.ctx.Value(mdcCtxConfDir).(string), "machines", cls.Name)
}

func (cls *Machine) DataDir() string {
	return filepath.Join(cls.ctx.Value(mdcCtxDataDir).(string), "machines", cls.Name)
}

func (cls *Machine) StateDir() string {
	return filepath.Join(cls.ctx.Value(mdcCtxStateDir).(string), "machines", cls.Name)
}

var (
	clsCtx         = "machine-ctx"
	clsCtxConfDir  = mdcCtx + "-confdir"
	clsCtxDataDir  = mdcCtx + "-datadir"
	clsCtxStateDir = mdcCtx + "-statedir"
)

func (cls *Machine) Context() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, clsCtxConfDir, cls.ConfigDir())
	ctx = context.WithValue(ctx, clsCtxDataDir, cls.DataDir())
	ctx = context.WithValue(ctx, clsCtxStateDir, cls.StateDir())
	return ctx
}

func (cls *Machine) ConfigFile() string {
	// FIXME: need to decide on the name of this yaml file
	return filepath.Join(cls.ConfigDir(), "machine.yaml")
}

func (cls *Machine) SaveConfig() error {
	configFile := cls.ConfigFile()
	machinesDir := filepath.Dir(configFile)
	log.Debugf("machinesDir: %q configFile: %q", machinesDir, configFile)
	if !PathExists(machinesDir) {
		if err := os.MkdirAll(machinesDir, 0755); err != nil {
			return fmt.Errorf("Failed to create machinesDir %q: %s", machinesDir, err)
		}
	}
	contents, err := yaml.Marshal(cls)
	if err != nil {
		return fmt.Errorf("Failed to marshal machine config: %s", err)
	}
	if err := ioutil.WriteFile(configFile, contents, 0644); err != nil {
		return fmt.Errorf("Failed write machine config to '%q': %s", configFile, err)
	}
	return nil
}

func LoadConfig(configFile string) (Machine, error) {
	var newMachine Machine
	machineBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return newMachine, fmt.Errorf("Error reading machine config file '%q': %s", configFile, err)
	}
	if err := yaml.Unmarshal(machineBytes, &newMachine); err != nil {
		return newMachine, fmt.Errorf("Error unmarshaling machine config file %q: %s", configFile, err)
	}
	return newMachine, nil
}

func (cls *Machine) Start() error {

	// check if machine is running, if so return
	if cls.IsRunning() {
		return fmt.Errorf("Machine is already running")
	}

	cls.Status = MachineStatusStarting
	vmCtx := cls.Context()
	vm, err := newVM(vmCtx, cls.Name, cls.Config)
	if err != nil {
		cls.Status = MachineStatusStopped
		return fmt.Errorf("Failed to create new VM '%s': %s", cls.Name, err)
	}
	cls.instance = &vm

	err = vm.Start()
	if err != nil {
		forceStop := true
		vm.Stop(forceStop)
		cls.Status = MachineStatusStopped
		return fmt.Errorf("Failed to start VM '%s.%s': %s", cls.Name, vm.Config.Name, err)
	}

	cls.vmCount.Add(1)
	cls.Status = MachineStatusRunning
	return nil
}

func (cls *Machine) Stop(force bool) error {

	log.Infof("Machine.Stop called on machine %s, status: %s, force: %v", cls.Name, cls.Status, force)
	// check if machine is stopped, if so return
	if !cls.IsRunning() {
		return fmt.Errorf("Machine is already stopped")
	}

	if cls.instance != nil {
		log.Infof("Machine.Stop, VM instance: %s, calling stop", cls.Name)
		err := cls.instance.Stop(force)
		if err != nil {
			return fmt.Errorf("Failed to stop VM '%s': %s", cls.Name, err)
		}
		cls.vmCount.Done()
	} else {
		log.Debugf("Machine instanace was nil, marking stop")
	}
	cls.Status = MachineStatusStopped
	return nil
}

func (cls *Machine) Delete() error {
	// Stop machine, if running
	// Delete VM (stop and remove state)
	// Remove Machine Config

	log.Infof("Machine.Delete called on machine %s, status: %s", cls.Name, cls.Status)

	if cls.instance != nil {
		if cls.IsRunning() {
			forceStop := true
			err := cls.Stop(forceStop)
			if err != nil {
				return fmt.Errorf("Failed to stop machine '%s': %s", cls.Name, err)
			}
		}

		log.Infof("Machine.Delete, VM instance: %s, calling delete", cls.Name)
		err := cls.Delete()
		if err != nil {
			return fmt.Errorf("Failed to delete VM '%s': %s", cls.Name, err)
		}
	}
	cls.instance = nil

	dirs := []string{cls.ConfigDir(), cls.DataDir(), cls.StateDir()}
	for _, dir := range dirs {
		if PathExists(dir) {
			log.Infof("Removing machine dir %q", dir)
			err := os.RemoveAll(dir)
			if err != nil {
				return fmt.Errorf("Failed to remove machine %s dir %q", cls.Name, dir)
			}
		}
	}

	return nil
}

func (cls *Machine) IsRunning() bool {
	if cls.instance != nil {
		if cls.Status == MachineStatusRunning || cls.Status == MachineStatusStarting {
			return true
		}
	} else {
		log.Debugf("IsRunning called on Machine with nil instance")
	}
	return false
}
