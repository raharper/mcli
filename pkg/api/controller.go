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
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/apex/log"
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
					newCluster.ctx = c.Config.GetConfigContext()
					fmt.Println("  loaded cluster ", newCluster.Name)
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

	fmt.Printf("machined service running on: %s\n", unixSocket)
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

//
// utility functions below here
//

func PathExists(d string) bool {
	_, err := os.Stat(d)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func WaitForPath(path string, retries, sleepSeconds int) bool {
	var numRetries int
	if retries == 0 {
		numRetries = 1
	} else {
		numRetries = retries
	}
	for i := 0; i < numRetries; i++ {
		if PathExists(path) {
			return true
		}
		time.Sleep(time.Duration(sleepSeconds) * time.Second)
	}
	return PathExists(path)
}

func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("couldn't make dirs: %s", err)
	}
	return nil
}

// CopyFileBits - copy file content from a to b
// differs from CopyFile in:
//  - does not do permissions - new files created with 0644
//  - if src is a symlink, copies content, not link.
//  - does not invoke sh.
func CopyFileBits(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

// Copy one file to a new path, i.e. cp a b
func CopyFileRefSparse(src, dest string) error {
	if err := EnsureDir(filepath.Dir(src)); err != nil {
		return err
	}
	if err := EnsureDir(filepath.Dir(dest)); err != nil {
		return err
	}
	cmdtxt := fmt.Sprintf("cp --force --reflink=auto --sparse=auto %s %s", src, dest)
	return RunCommand("sh", "-c", cmdtxt)
}

func RunCommand(args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s: %s", strings.Join(args, " "), err, string(output))
	}
	return nil
}

func RunCommandWithOutputErrorRc(args ...string) ([]byte, []byte, int) {
	cmd := exec.Command(args[0], args[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), GetCommandErrorRC(err)
}

func GetCommandErrorRCDefault(err error, rcError int) int {
	if err == nil {
		return 0
	}
	exitError, ok := err.(*exec.ExitError)
	if ok {
		if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	log.Debugf("Unavailable return code for %s. returning %d", err, rcError)
	return rcError
}

func GetCommandErrorRC(err error) int {
	return GetCommandErrorRCDefault(err, 127)
}
