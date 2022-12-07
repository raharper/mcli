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
package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	Config            *MachineDaemonConfig
	Router            *gin.Engine
	ClusterController ClusterController
	Server            *http.Server
	wgShutDown        *sync.WaitGroup
	portNumber        int
}

func NewController(config *MachineDaemonConfig) *Controller {
	var controller Controller

	controller.Config = config
	controller.wgShutDown = new(sync.WaitGroup)

	return &controller
}

func (c *Controller) Run(ctx context.Context) error {
	// load existing clusters
	clusterDir := filepath.Join(c.Config.ConfigDirectory, "clusters")
	if PathExists(clusterDir) {
		fmt.Println("Loading saved cluster configs...")
		err := filepath.Walk(clusterDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				clusterConf := filepath.Join(path, "machine.yaml")
				if PathExists(clusterConf) {
					newCluster, err := LoadConfig(clusterConf)
					if err != nil {
						return err
					}
					c.ClusterController.Clusters = append(c.ClusterController.Clusters, newCluster)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	unixSocket := APISocketPath()
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
	defer os.Remove(unixSocket)

	fmt.Println("machined service running on: %s", unixSocket)
	engine := gin.Default()
	c.Router = engine

	//  configure routes
	_ = NewRouteHandler(c)

	// re-implement gin.Engine.RunUnix() so we can set the context ourselves
	listener, err := net.Listen("unix", unixSocket)
	if err != nil {
		panic("Failed to create a unix socket listener")
	}
	defer listener.Close()

	c.Server = &http.Server{Handler: c.Router.Handler()}

	return c.Server.Serve(listener)
}

func (c *Controller) InitClusterController(ctx context.Context) error {
	c.ClusterController = ClusterController{}

	// TODO
	// look for serialized Cluster configuration files in data dir
	// for each one, read them in and add to the Controller
	return nil
}

func (c *Controller) Shutdown(ctx context.Context) error {
	c.wgShutDown.Wait()
	if err := c.Server.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func PathExists(d string) bool {
	_, err := os.Stat(d)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}
