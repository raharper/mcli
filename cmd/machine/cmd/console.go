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

	"github.com/spf13/cobra"
)

const SerialConsole = "console"
const VGAConsole = "vga"

// consoleCmd represents the console command
var consoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Connect to machine console (serial or graphical)",
	Long:  `Connect to machine text console (serial) or graphical`,
	Run:   doConsole,
}

func init() {
	rootCmd.AddCommand(consoleCmd)
	consoleCmd.PersistentFlags().StringP(
		"type",
		"t", "",
		`type of connectiotn to establish: 'console' for serial console, 'vga' for SPICE graphical output (default \"console\")"`,
	)
}

func doConsole(cmd *cobra.Command, args []string) {
	consoleType := cmd.Flag("type").Value.String()
	if consoleType == "" {
		consoleType = SerialConsole
	}
	if consoleType != SerialConsole || consoleType != VGAConsole {
		panic(fmt.Sprintf("Invalid console type '%s'", consoleType))
	}
	var machineName string
	if len(args) < 1 {
		panic("Missing required machine name")
	}
	machineName = args[0]
	DoConsoleAttach(machineName, consoleType)
}

func DoConsoleAttach(machineName, consoleType string) error {

	return nil
}
