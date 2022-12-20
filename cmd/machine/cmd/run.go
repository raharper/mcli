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

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:        "run <cluster_name>",
	Args:       cobra.MinimumNArgs(1),
	ArgAliases: []string{"clusterName"},
	Short:      "run the specified cluster",
	Long:       `run the specified cluster if it exists`,
	Run:        doRun,
}

// Ideally:
// 	running a cluster requires POST'ing an update to the cluster state
// 	which toggles from the 'stopped' state to the 'running' state.
// 	Asynchronously the cluster will start, in a separate goroutine spawned by
// 	machined, and depending on client flags (blocking/non-blocking) the server
// 	will return back an new URL for status on the cluster instance
//
// TBD, the affecting the machines in each cluster
//
// Currently we now post a request with {'status': 'running'} to start a cluster

func doRun(cmd *cobra.Command, args []string) {
	clusterName := args[0]
	var request struct {
		Status string `json:"status"`
	}
	request.Status = "running"

	endpoint := fmt.Sprintf("clusters/%s/start", clusterName)
	startURL := api.GetAPIURL(endpoint)
	if len(startURL) == 0 {
		panic(fmt.Sprintf("Failed to get API URL for 'clusters/%s/start' endpoint", clusterName))
	}
	resp, err := rootclient.R().EnableTrace().SetBody(request).Post(startURL)
	if err != nil {
		panic(fmt.Sprintf("Failed POST to 'clusters/%s/start' endpoint: %s", clusterName, err))
	}
	fmt.Printf("%s %s\n", resp, resp.Status())
}

func init() {
	rootCmd.AddCommand(runCmd)
}
