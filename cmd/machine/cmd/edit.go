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
	"mcli-v2/pkg/api"
	"os"

	"github.com/lxc/lxd/shared"
	"github.com/lxc/lxd/shared/termios"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v2"
)

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:        "edit <cluster name>",
	Args:       cobra.MinimumNArgs(1),
	ArgAliases: []string{"clusterName"},
	Short:      "edit a cluster's configuration file",
	Long:       `Read the cluster configuration into an editor for modification`,
	Run:        doEdit,
}

// edit requires one to:
// - GET the cluster configuration from REST API
// - render this to a temp file
// - invoke $EDITOR to allow user to make changes
//
// Option 1:
// - (optionally) before posting, run JSON validator on the new file?
// - PATCH/UPDATE the cluster configuration back to API
//   (and symantically what does that mean if the instance is running)
//
// Option 2:
// - write out changes to config file on disk and not modifying in-memory state
//   via PATCH/UPDATE operations.
//
func doEdit(cmd *cobra.Command, args []string) {
	clusterName := args[0]
	clusters, err := getClusters()
	if err != nil {
		panic(err)
	}

	var clusterBytes []byte
	onTerm := termios.IsTerminal(unix.Stdin)
	editCluster := &api.Cluster{}

	for _, cluster := range clusters {
		if cluster.Name == clusterName {
			editCluster = &cluster
			break
		}
	}
	if editCluster.Name == "" {
		panic(fmt.Sprintf("Failed to find cluster '%s'", clusterName))
	}

	clusterBytes, err = yaml.Marshal(editCluster)
	if err != nil {
		panic(fmt.Sprintf("Error marshalling cluster '%s'", clusterName))
	}

	clusterBytes, err = shared.TextEditor("", clusterBytes)
	if err != nil {
		panic("Error calling editor")
	}

	newCluster := api.Cluster{Name: clusterName}
	for {
		if err = yaml.Unmarshal(clusterBytes, &newCluster); err == nil {
			break
		}
		if !onTerm {
			panic(fmt.Sprintf("Error parsing configuration: %s", err))
		}
		fmt.Printf("Error parsing yaml: %v\n", err)
		fmt.Println("Press enter to re-open editor, or ctrl-c to abort")
		_, err := os.Stdin.Read(make([]byte, 1))
		if err != nil {
			panic(fmt.Sprintf("Error reading reply: %s", err))
		}
		clusterBytes, err = shared.TextEditor("", clusterBytes)
		if err != nil {
			panic(fmt.Sprintf("Error calling editor: %s", err))
		}
	}
	// persist config if not ephemeral

	err = putCluster(newCluster)
	if err != nil {
		panic(err.Error())
	}
}

func init() {
	rootCmd.AddCommand(editCmd)
}
