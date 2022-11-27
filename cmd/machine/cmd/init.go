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

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new machine cluster from yaml",
	Long: `Initilize a new machine cluster by specifying a machine cluster
yaml configuring one or more machines, networks and connections.`,
	Run: doInit,
}

func doInit(cmd *cobra.Command, args []string) {
	fileName := cmd.Flag("file").Value.String()
	editFile := cmd.Flag("edit").Value.String()
	fmt.Println("init command:")
	fmt.Println("  file: ", fileName)
	fmt.Println("  edit: ", editFile)

	// check if a cluster of this name is already defined
	// check if edit is set whether we're a terminal or not
	// if file, read contents, else read from stdin
	// launch editor with contents
	// post-edit attempt to marshal contents into Cluster definition, retry on failure
	// If cluster.Persistent is set, then write contents to config dir, else call api.AddCluster()
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.PersistentFlags().StringP("file", "f", "-", "yaml file to import.  If unspecified, use stdin")
	initCmd.PersistentFlags().BoolP("edit", "e", false, "edit the yaml file inline")
}
