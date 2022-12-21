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
	fmt.Printf("FindClusterByName: clusters:%v clusterName: %s\n", ctl.Clusters, clusterName)
	for _, cluster := range ctl.Clusters {
		if cluster.Name == clusterName {
			fmt.Println("found it")
			return &cluster, nil
		}
	}
	fmt.Println("returning")
	return &Cluster{}, fmt.Errorf("Failed to find cluster with Name: %s", clusterName)
}

func (ctl *ClusterController) GetClusters() []Cluster {
	return ctl.Clusters
}

func (ctl *ClusterController) AddCluster(newCluster Cluster, confDir string) error {
	if _, err := ctl.GetClusterByName(newCluster.Name); err == nil {
		return fmt.Errorf("Cluster '%s' is already defined", newCluster.Name)
	}
	newCluster.Status = ClusterStatusStopped
	if !newCluster.Ephemeral {
		cluster := &newCluster
		if err := cluster.SaveConfig(confDir); err != nil {
			return fmt.Errorf("Could not save '%s' cluster to %q: %s", cluster.Name, cluster.ConfigFile(confDir), err)
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
				fmt.Printf("Error while stopping cluster '%s': %s\n", cluster.Name, err)
			}
		}
	}
	return nil
}

func (ctl *ClusterController) DeleteCluster(clusterName string, confDir string) error {
	clusters := []Cluster{}
	for idx, _ := range ctl.Clusters {
		cluster := ctl.Clusters[idx]
		if cluster.Name != clusterName {
			clusters = append(clusters, cluster)
		} else {
			if cluster.IsRunning() {
				err := cluster.Stop()
				if err != nil {
					fmt.Println("Failed stopping cluster, continuing with Delete")
				}
			}
			err := cluster.RemoveConfig(cluster.ConfigFile(confDir))
			if err != nil {
				return fmt.Errorf("Failed to remove cluster config file: %s", err)
			}
			fmt.Println("Removed cluster: ", cluster.Name)
		}
	}
	ctl.Clusters = clusters
	return nil
}

func (ctl *ClusterController) UpdateCluster(updateCluster Cluster, confDir string) error {
	// FIXME: decide if update will modify the in-memory state (I think yes, but
	// maybe only the on-disk format if it's running? but what does subsequent
	// GET return (on-disk or in-memory?)

	for idx, cluster := range ctl.Clusters {
		if cluster.Name == updateCluster.Name {
			ctl.Clusters[idx] = updateCluster
			if !updateCluster.Ephemeral {
				if err := updateCluster.SaveConfig(confDir); err != nil {
					return fmt.Errorf("Could not save '%s' cluster to %q: %s", updateCluster.Name, updateCluster.ConfigFile(confDir), err)
				}
			}
			fmt.Printf("Updated cluster '%s'\n", updateCluster.Name)
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
func (cls *Cluster) ConfigFile(confDir string) string {
	// FIXME: need to decide on the name of this yaml file
	confPath := filepath.Join(confDir, "clusters", cls.Name)
	configFile := filepath.Join(confPath, "machine.yaml")
	return configFile
}

func (cls *Cluster) SaveConfig(confDir string) error {
	configFile := cls.ConfigFile(confDir)
	clustersDir := filepath.Dir(configFile)
	fmt.Printf("clustersDir: %q configFile: %q\n", clustersDir, configFile)
	if !PathExists(clustersDir) {
		if err := os.MkdirAll(clustersDir, 0755); err != nil {
			return fmt.Errorf("Failed to create clustersDir %q: %s", clustersDir, err)
		}
	}
	contents, err := yaml.Marshal(cls)
	if err != nil {
		return fmt.Errorf("Failed to marshal cluster config: %s", err)
	}
	fmt.Println(string(contents))
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

func (cls *Cluster) RemoveConfig(configFile string) error {
	if PathExists(configFile) {
		// remove everything under the cluster dir
		clustersDir := filepath.Dir(configFile)
		fmt.Printf("Removing cluster config dir %q\n", clustersDir)
		return os.RemoveAll(clustersDir)
	}
	return nil
}

func (cls *Cluster) Start() error {

	// check if cluster is running, if so return
	if cls.IsRunning() {
		return fmt.Errorf("Cluster is already running")
	}
	cls.Status = ClusterStatusStarting
	for _, vmdef := range cls.Config.Machines {
		ctx := context.Background()
		vm, err := newVM(ctx, vmdef)
		if err != nil {
			return fmt.Errorf("Failed to create new VM '%s.%s': %s", cls.Name, vmdef.Name, err)
		}
		err = vm.Start()
		if err != nil {
			return fmt.Errorf("Failed to start VM '%s.%s': %s", cls.Name, vm.Config.Name, err)
		}
		fmt.Printf("Cluster.StartCluster, VM instances before append: %d\n", len(cls.Config.instances))
		cls.Config.instances = append(cls.Config.instances, &vm)
		fmt.Printf("Cluster.StartCluster, VM instances after  append: %d %v\n", len(cls.Config.instances), cls.Config.instances)
		cls.vmCount.Add(1)
	}
	cls.Status = ClusterStatusRunning
	return nil
}

func (cls *Cluster) Stop() error {

	fmt.Printf("Cluster.Stop called on cluster %s, status: %s\n", cls.Name, cls.Status)
	// check if cluster is stopped, if so return
	if !cls.IsRunning() {
		return fmt.Errorf("Cluster is already stopped")
	}

	fmt.Printf("Cluster.Stop, VM instances: %d\n", len(cls.Config.instances))
	for _, vm := range cls.Config.instances {
		fmt.Printf("Cluster.Stop, VM instance: %s, calling stop\n", vm.Config.Name)
		err := vm.Stop()
		if err != nil {
			return fmt.Errorf("Failed to stop VM '%s.%s': %s", cls.Name, vm.Config.Name, err)
		}
		cls.vmCount.Done()
	}
	cls.Status = ClusterStatusStopped
	return nil
}

func (cls *Cluster) IsRunning() bool {
	if cls.Status == ClusterStatusRunning || cls.Status == ClusterStatusStarting {
		return true
	}
	return false
}
