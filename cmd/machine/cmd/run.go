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

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:        "run <cluster_name> <cluster config>",
	Args:       cobra.MinimumNArgs(2),
	ArgAliases: []string{"clusterName"},
	Short:      "create and start a new cluster",
	Long:       `create a new cluster from config and start the cluster.`,
	Run:        doRun,
}

// Initialize a new cluster from config file and then start it up
func doRun(cmd *cobra.Command, args []string) {
	clusterName := args[0]
	clusterConfig := args[1]
	editCluster := false

	// FIXME: handle mismatch between name in arg and value in config file
	if err := DoCreateCluster(clusterName, clusterConfig, editCluster); err != nil {
		panic(fmt.Sprintf("Failed to create cluster '%s' from config '%s': %s", clusterName, clusterConfig, err))
	}

	if err := DoStartCluster(clusterName); err != nil {
		panic(fmt.Sprintf("Failed to start cluster '%s' from config '%s': %s", clusterName, clusterConfig, err))
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
}
