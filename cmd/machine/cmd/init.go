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
package cmd

import (
	"fmt"
	"io/ioutil"
	"mcli-v2/pkg/api"
	"os"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/lxc/lxd/shared"
	"github.com/lxc/lxd/shared/termios"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v2"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init <machine name>",
	Short: "Initialize a new machine from yaml",
	Long:  `Initilize a new machine by specifying a machine yaml configuring.`,
	Run:   doInit,
}

func doInit(cmd *cobra.Command, args []string) {
	fileName := cmd.Flag("file").Value.String()
	// Hi cobra, this is awkward...  why isn't there .Value.Bool()?
	editFile, _ := cmd.Flags().GetBool("edit")
	var machineName string
	if len(args) > 0 {
		machineName = args[0]
	} else {
		machineName = petname.Generate(petNameWords, petNameSep)
	}
	fmt.Println("  machineName:", machineName)

	if err := DoCreateMachine(machineName, fileName, editFile); err != nil {
		panic("Failed to create a machine")
	}
}

func DoCreateMachine(machineName, fileName string, editFile bool) error {
	var err error
	var machineBytes []byte
	onTerm := termios.IsTerminal(unix.Stdin)
	newMachine := api.Machine{
		Name:      machineName,
		Ephemeral: true,
		Status:    api.MachineStatusStopped,
	}
	fmt.Printf("Creating machine %s ...\n", machineName)

	// check if edit is set whether we're a terminal or not
	// if file, read contents, else read from stdin
	// launch editor with contents
	// post-edit attempt to marshal contents into Machine definition, retry on failure
	// If machine.Persistent is set, then write contents to config dir, else call api.AddMachine()

	if editFile && !onTerm {
		return fmt.Errorf("Aborting edit since stdin is not a terminal")
	}

	if fileName == "-" {
		machineBytes, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("Error reading machine definition from stdin: %s", err)
		}
	} else {
		if len(fileName) > 0 {
			machineBytes, err = os.ReadFile(fileName)
			if err != nil {
				return fmt.Errorf("Error reading definition from %s: %s", fileName, err)
			}
		} else {
			fmt.Println("No file specified, using defaults..\n")
			machineBytes, err = yaml.Marshal(newMachine)
			if err != nil {
				return fmt.Errorf("Failed reading empty machine config: %s", err)
			}
			editFile = true
		}
	}

	if editFile {
		machineBytes, err = shared.TextEditor("", machineBytes)
		if err != nil {
			return fmt.Errorf("Error calling editor: %s", err)
		}
	}
	fmt.Printf("Got config:\n%s", string(machineBytes))

	for {
		if err = yaml.Unmarshal(machineBytes, &newMachine); err == nil {
			break
		}
		if !onTerm {
			return fmt.Errorf("Error parsing configuration: %s", err)
		}
		fmt.Printf("Error parsing yaml: %v\n", err)
		fmt.Println("Press enter to re-open editor, or ctrl-c to abort")
		_, err := os.Stdin.Read(make([]byte, 1))
		if err != nil {
			return fmt.Errorf("Error reading reply: %s", err)
		}
		machineBytes, err = shared.TextEditor("", machineBytes)
		if err != nil {
			fmt.Errorf("Error calling editor: %s", err)
		}
	}
	// persist config if not ephemeral

	err = postMachine(newMachine)
	if err != nil {
		return fmt.Errorf("Error while POST'ing new machine config: %s", err)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.PersistentFlags().StringP("file", "f", "", "yaml file to import.  If unspecified, use stdin")
	initCmd.PersistentFlags().BoolP("edit", "e", false, "edit the yaml file inline")
}
