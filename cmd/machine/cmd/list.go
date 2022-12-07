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

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list all of the defined clusters",
	Long:  `list all of the defined clusters`,
	Run:   doList,
}

func doList(cmd *cobra.Command, args []string) {
	clusters, err := getClusters()
	if err != nil {
		panic(err)
	}
	for _, cluster := range clusters {
		fmt.Println(cluster.Name)
	}
}

func init() {
	rootCmd.AddCommand(listCmd)
}
