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
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	ClusterStatusStopped  string = "stopped"
	ClusterStatusStarting string = "starting"
	ClusterStatusRunning  string = "running"
	ClusterStatusStopping string = "stopping"
)

type StopChannel chan struct{}

type ClusterController struct {
	Clusters []Cluster
}

type Cluster struct {
	ctx         context.Context
	Type        string        `yaml:"type"`
	Config      ClusterConfig `yaml:"config"`
	Description string        `yaml:"description"`
	Ephemeral   bool          `yaml:"ephemeral"`
	Name        string        `yaml:"name"`
	Status      string
	statusCode  int64
	vmCount     sync.WaitGroup
}

type ClusterConfig struct {
	Machines    []VMDef      `yaml:"machines"`
	Networks    []NetworkDef `yaml:"networks"`
	Connections ConnDef      `yaml:"connections"`
	instances   []*VM
}

type VMNicNetLinks map[string]string
type ConnDef map[string]VMNicNetLinks

func (ctl *ClusterController) GetClusterByName(clusterName string) (*Cluster, error) {
	log.Infof("FindClusterByName: clusters:%v clusterName: %s", ctl.Clusters, clusterName)
	for _, cluster := range ctl.Clusters {
		if cluster.Name == clusterName {
			log.Infof("found it")
			return &cluster, nil
		}
	}
	log.Infof("returning")
	return &Cluster{}, fmt.Errorf("Failed to find cluster with Name: %s", clusterName)
}

func (ctl *ClusterController) GetClusters() []Cluster {
	return ctl.Clusters
}

func (ctl *ClusterController) AddCluster(newCluster Cluster, cfg *MachineDaemonConfig) error {
	if _, err := ctl.GetClusterByName(newCluster.Name); err == nil {
		return fmt.Errorf("Cluster '%s' is already defined", newCluster.Name)
	}
	newCluster.Status = ClusterStatusStopped
	newCluster.ctx = cfg.GetConfigContext()
	if !newCluster.Ephemeral {
		if err := newCluster.SaveConfig(); err != nil {
			return fmt.Errorf("Could not save '%s' cluster to %q: %s", newCluster.Name, newCluster.ConfigFile(), err)
		}
	}
	ctl.Clusters = append(ctl.Clusters, newCluster)
	return nil
}

func (ctl *ClusterController) StopClusters() error {
	for idx, _ := range ctl.Clusters {
		cluster := ctl.Clusters[idx]
		if cluster.IsRunning() {
			if err := cluster.Stop(); err != nil {
				log.Infof("Error while stopping cluster '%s': %s", cluster.Name, err)
			}
		}
	}
	return nil
}

func (ctl *ClusterController) DeleteCluster(clusterName string, cfg *MachineDaemonConfig) error {
	clusters := []Cluster{}
	for idx, _ := range ctl.Clusters {
		cluster := ctl.Clusters[idx]
		if cluster.Name != clusterName {
			clusters = append(clusters, cluster)
		} else {
			err := cluster.Delete()
			if err != nil {
				return fmt.Errorf("Cluster:%s delete failed: %s", cluster.Name, err)
			}
			log.Infof("Deleted cluster: %s", cluster.Name)
		}
	}
	ctl.Clusters = clusters
	return nil
}

func (ctl *ClusterController) UpdateCluster(updateCluster Cluster, cfg *MachineDaemonConfig) error {
	// FIXME: decide if update will modify the in-memory state (I think yes, but
	// maybe only the on-disk format if it's running? but what does subsequent
	// GET return (on-disk or in-memory?)

	for idx, cluster := range ctl.Clusters {
		if cluster.Name == updateCluster.Name {
			updateCluster.ctx = cfg.GetConfigContext()
			ctl.Clusters[idx] = updateCluster
			if !updateCluster.Ephemeral {
				if err := updateCluster.SaveConfig(); err != nil {
					return fmt.Errorf("Could not save '%s' cluster to %q: %s", updateCluster.Name, updateCluster.ConfigFile(), err)
				}
			}
			log.Infof("Updated cluster '%s'", updateCluster.Name)
			break
		}
	}
	return nil
}

func (ctl *ClusterController) StartCluster(clusterName string) error {
	for idx, cluster := range ctl.Clusters {
		if cluster.Name == clusterName {
			err := ctl.Clusters[idx].Start()
			if err != nil {
				return fmt.Errorf("Could not start '%s' cluster: %s", clusterName, err)
			}
			return nil
		}
	}
	return fmt.Errorf("Failed to find cluster '%s', cannot start unknown cluster", clusterName)
}

func (ctl *ClusterController) StopCluster(clusterName string) error {
	for idx, cluster := range ctl.Clusters {
		if cluster.Name == clusterName {
			err := ctl.Clusters[idx].Stop()
			if err != nil {
				return fmt.Errorf("Could not stop '%s' cluster: %s", clusterName, err)
			}
			return nil
		}
	}
	return fmt.Errorf("Failed to find cluster '%s', cannot stop unknown cluster", clusterName)
}

//
// Cluster Functions Below
//
func (cls *Cluster) ConfigDir() string {
	return filepath.Join(cls.ctx.Value(mdcCtxConfDir).(string), "clusters", cls.Name)
}

func (cls *Cluster) DataDir() string {
	return filepath.Join(cls.ctx.Value(mdcCtxDataDir).(string), "clusters", cls.Name)
}

func (cls *Cluster) StateDir() string {
	return filepath.Join(cls.ctx.Value(mdcCtxStateDir).(string), "clusters", cls.Name)
}

var (
	clsCtx         = "cluster-ctx"
	clsCtxConfDir  = mdcCtx + "-confdir"
	clsCtxDataDir  = mdcCtx + "-datadir"
	clsCtxStateDir = mdcCtx + "-statedir"
)

func (cls *Cluster) Context() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, clsCtxConfDir, cls.ConfigDir())
	ctx = context.WithValue(ctx, clsCtxDataDir, cls.DataDir())
	ctx = context.WithValue(ctx, clsCtxStateDir, cls.StateDir())
	return ctx
}

func (cls *Cluster) ConfigFile() string {
	// FIXME: need to decide on the name of this yaml file
	return filepath.Join(cls.ConfigDir(), "machine.yaml")
}

func (cls *Cluster) SaveConfig() error {
	configFile := cls.ConfigFile()
	clustersDir := filepath.Dir(configFile)
	log.Debugf("clustersDir: %q configFile: %q", clustersDir, configFile)
	if !PathExists(clustersDir) {
		if err := os.MkdirAll(clustersDir, 0755); err != nil {
			return fmt.Errorf("Failed to create clustersDir %q: %s", clustersDir, err)
		}
	}
	contents, err := yaml.Marshal(cls)
	if err != nil {
		return fmt.Errorf("Failed to marshal cluster config: %s", err)
	}
	if err := ioutil.WriteFile(configFile, contents, 0644); err != nil {
		return fmt.Errorf("Failed write cluster config to '%q': %s", configFile, err)
	}
	return nil
}

func LoadConfig(configFile string) (Cluster, error) {
	var newCluster Cluster
	clusterBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return newCluster, fmt.Errorf("Error reading cluster config file '%q': %s", configFile, err)
	}
	if err := yaml.Unmarshal(clusterBytes, &newCluster); err != nil {
		return newCluster, fmt.Errorf("Error unmarshaling cluster config file %q: %s", configFile, err)
	}
	return newCluster, nil
}

func (cls *Cluster) Start() error {

	// check if cluster is running, if so return
	if cls.IsRunning() {
		return fmt.Errorf("Cluster is already running")
	}
	cls.Status = ClusterStatusStarting
	for _, vmdef := range cls.Config.Machines {
		vmCtx := cls.Context()
		vm, err := newVM(vmCtx, cls.Name, vmdef)
		if err != nil {
			cls.Status = ClusterStatusStopped
			return fmt.Errorf("Failed to create new VM '%s.%s': %s", cls.Name, vmdef.Name, err)
		}

		log.Infof("Cluster.StartCluster, VM instances before append: %d", len(cls.Config.instances))
		// append the VM unless newVM failed so we can cleanup the VM on delete
		cls.Config.instances = append(cls.Config.instances, &vm)

		err = vm.Start()
		if err != nil {
			cls.Status = ClusterStatusStopped
			return fmt.Errorf("Failed to start VM '%s.%s': %s", cls.Name, vm.Config.Name, err)
		}

		log.Infof("Cluster.StartCluster, VM instances after  append: %d %v", len(cls.Config.instances), cls.Config.instances)
		cls.vmCount.Add(1)
	}
	cls.Status = ClusterStatusRunning
	return nil
}

func (cls *Cluster) Stop() error {

	log.Infof("Cluster.Stop called on cluster %s, status: %s", cls.Name, cls.Status)
	// check if cluster is stopped, if so return
	if !cls.IsRunning() {
		return fmt.Errorf("Cluster is already stopped")
	}

	log.Infof("Cluster.Stop, VM instances: %d", len(cls.Config.instances))
	for _, vm := range cls.Config.instances {
		log.Infof("Cluster.Stop, VM instance: %s, calling stop", vm.Config.Name)
		err := vm.Stop()
		if err != nil {
			return fmt.Errorf("Failed to stop VM '%s.%s': %s", cls.Name, vm.Config.Name, err)
		}
		cls.vmCount.Done()
	}
	cls.Status = ClusterStatusStopped
	return nil
}

func (cls *Cluster) Delete() error {
	// Stop cluster, if running
	// for each VM:
	//   delete VM (stop and remove state)
	// Remove Cluster Config

	log.Infof("Cluster.Delete called on cluster %s, status: %s", cls.Name, cls.Status)
	if cls.IsRunning() {
		err := cls.Stop()
		if err != nil {
			return fmt.Errorf("Failed to stop cluster '%s': %s", cls.Name, err)
		}
	}

	for _, vm := range cls.Config.instances {
		log.Infof("Cluster.Delete, VM instance: %s, calling delete", vm.Name())
		err := vm.Delete()
		if err != nil {
			return fmt.Errorf("Failed to delete VM '%s.%s': %s", cls.Name, vm.Name(), err)
		}
	}
	cls.Config.instances = []*VM{}

	dirs := []string{cls.ConfigDir(), cls.DataDir(), cls.StateDir()}
	for _, dir := range dirs {
		if PathExists(dir) {
			log.Infof("Removing cluster dir %q", dir)
			err := os.RemoveAll(dir)
			if err != nil {
				return fmt.Errorf("Failed to remove cluster %s dir %q", cls.Name, dir)
			}
		}
	}

	return nil
}

func (cls *Cluster) IsRunning() bool {
	if cls.Status == ClusterStatusRunning || cls.Status == ClusterStatusStarting {
		return true
	}
	return false
}
