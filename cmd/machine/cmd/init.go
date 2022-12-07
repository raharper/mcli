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
	Use:   "init <cluster name>",
	Short: "Initialize a new machine cluster from yaml",
	Long: `Initilize a new machine cluster by specifying a machine cluster
yaml configuring one or more machines, networks and connections.`,
	Run: doInit,
}

func doInit(cmd *cobra.Command, args []string) {
	var err error
	fileName := cmd.Flag("file").Value.String()
	// Hi cobra, this is awkward...  why isn't there .Value.Bool()?
	editFile, _ := cmd.Flags().GetBool("edit")
	fmt.Println("init command:")
	fmt.Println("  args: ", args)
	fmt.Println("  file: ", fileName)
	fmt.Println("  edit: ", editFile)

	var clusterName string
	if len(args) > 0 {
		clusterName = args[0]
	} else {
		clusterName = petname.Generate(petNameWords, petNameSep)
	}
	fmt.Println("  clusterName:", clusterName)

	// check if edit is set whether we're a terminal or not
	// if file, read contents, else read from stdin
	// launch editor with contents
	// post-edit attempt to marshal contents into Cluster definition, retry on failure
	// If cluster.Persistent is set, then write contents to config dir, else call api.AddCluster()

	// fmt.Println("Seeing if we need to edit this thing...")
	var clusterBytes []byte
	onTerm := termios.IsTerminal(unix.Stdin)

	if editFile && !onTerm {
		panic("Aborting edit since stdin is not a terminal")
	}

	if fileName == "-" {
		clusterBytes, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic("Error reading definition from stdin")
		}
	} else {
		clusterBytes, err = os.ReadFile(fileName)
		if err != nil {
			panic(fmt.Sprintf("Error reading definition from %s", fileName))
		}
	}

	if editFile {
		clusterBytes, err = shared.TextEditor("", clusterBytes)
		if err != nil {
			panic("Error calling editor")
		}
	}
	fmt.Printf("Got config:\n%s", string(clusterBytes))

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

	err = postCluster(newCluster)
	if err != nil {
		panic(err.Error())
	}

}

func postCluster(newCluster api.Cluster) error {
	postURL := api.GetAPIURL("clusters")
	if len(postURL) == 0 {
		return fmt.Errorf("Failed to get API URL for 'clusters' endpoint")
	}
	resp, err := rootclient.R().EnableTrace().SetBody(newCluster).Post(postURL)
	if err != nil {
		return fmt.Errorf("Failed POST to 'clusters' endpoint: %s", err)
	}
	fmt.Printf("%s %s\n", resp, resp.Status())
	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.PersistentFlags().StringP("file", "f", "", "yaml file to import.  If unspecified, use stdin")
	initCmd.PersistentFlags().BoolP("edit", "e", false, "edit the yaml file inline")
}
