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
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "machined",
	Short: "A daemon to handle the lifecycle of machine clusters",
	Long: `The daemon runs a RESTful service to interact with and manage
machine clusters.`,
	Run: doServerRun,
}

func PathExists(d string) bool {
	_, err := os.Stat(d)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func doServerRun(cmd *cobra.Command, args []string) {
	unixSocket := api.APISocketPath()
	if len(unixSocket) == 0 {
		panic("Failed to get an API Socket path")
	}
	// mkdir -p on dirname(unixSocet)
	err := os.MkdirAll(filepath.Dir(unixSocket), 0755)
	if err != nil {
		panic(fmt.Sprintf("Failed to create directory path to: %s", unixSocket))
	}
	// FIXME to check if another machined is running/pidfile?, flock?
	if PathExists(unixSocket) {
		os.Remove(unixSocket)
	}
	fmt.Println("machined service running on: %s", unixSocket)
	router := gin.Default()
	router.GET("/clusters", api.GetClusters)
	router.POST("/clusters", api.PostClusters)
	router.RunUnix(unixSocket)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.server.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".server" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".server")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
