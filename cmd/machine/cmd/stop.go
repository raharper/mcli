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

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:        "stop <cluster_name>",
	Args:       cobra.MinimumNArgs(1),
	ArgAliases: []string{"clusterName"},
	Short:      "stop the specified cluster",
	Long:       `stop the specified cluster if it exists`,
	Run:        doStop,
}

// need to see about stopping single machine under cluster and whole cluster
func doStop(cmd *cobra.Command, args []string) {
	clusterName := args[0]
	var request struct {
		Status string `json:"status"`
	}
	request.Status = "stopped"

	endpoint := fmt.Sprintf("clusters/%s/stop", clusterName)
	stopURL := api.GetAPIURL(endpoint)
	if len(stopURL) == 0 {
		panic(fmt.Sprintf("Failed to get API URL for 'clusters/%s/stop' endpoint", clusterName))
	}
	resp, err := rootclient.R().EnableTrace().SetBody(request).Post(stopURL)
	if err != nil {
		panic(fmt.Sprintf("Failed POST to 'clusters/%s/stop' endpoint: %s", clusterName, err))
	}
	fmt.Printf("%s %s\n", resp, resp.Status())
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
