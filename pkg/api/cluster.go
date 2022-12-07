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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type ClusterController struct {
	Clusters []Cluster
}

type Cluster struct {
	Type        string        `yaml:"type"`
	Config      ClusterConfig `yaml:"config"`
	Description string        `yaml:"description"`
	Ephemeral   bool          `yaml:"ephemeral"`
	Name        string        `yaml:"name"`
	Status      string        `yaml:"status"`
	StatusCode  int64         `yaml:"status_code"`
}

type ClusterConfig struct {
	Machines    []VMDef      `yaml:"machines"`
	Networks    []NetworkDef `yaml:"networks"`
	Connections ConnDef      `yaml:"connections"`
}

type VMNicNetLinks map[string]string
type ConnDef map[string]VMNicNetLinks

func (c *ClusterController) GetClusterByName(clusterName string) (*Cluster, error) {
	fmt.Printf("FindClusterByName: clusters:%v clusterName: %s\n", c.Clusters, clusterName)
	for _, cluster := range c.Clusters {
		if cluster.Name == clusterName {
			fmt.Println("found it")
			return &cluster, nil
		}
	}
	fmt.Println("returning")
	return &Cluster{}, fmt.Errorf("Failed to find cluster with Name: %s", clusterName)
}

func (c *ClusterController) GetClusters() []Cluster {
	return c.Clusters
}

func (c *ClusterController) AddCluster(newCluster Cluster, confDir string) error {
	if _, err := c.GetClusterByName(newCluster.Name); err == nil {
		return fmt.Errorf("Cluster '%s' is already defined", newCluster.Name)
	}
	if !newCluster.Ephemeral {
		cluster := &newCluster
		if err := cluster.SaveConfig(confDir); err != nil {
			return fmt.Errorf("Could not save '%s' cluster to %q: %s", cluster.Name, cluster.ConfigFile(confDir), err)
		}
	}
	c.Clusters = append(c.Clusters, newCluster)
	return nil
}

func (c *ClusterController) DeleteCluster(clusterName string, confDir string) error {
	clusters := []Cluster{}
	for _, cluster := range c.Clusters {
		if cluster.Name != clusterName {
			clusters = append(clusters, cluster)
		} else {
			err := cluster.RemoveConfig(cluster.ConfigFile(confDir))
			if err != nil {
				return fmt.Errorf("Failed to remove cluster config file: %s", err)
			}
			fmt.Println("Removed cluster: ", cluster.Name)
		}
	}
	c.Clusters = clusters
	return nil
}

func (c *ClusterController) UpdateCluster(updateCluster Cluster, confDir string) error {
	// FIXME: decide if update will modify the in-memory state (I think yes, but
	// maybe only the on-disk format if it's running? but what does subsequent
	// GET return (on-disk or in-memory?)
	for idx, cluster := range c.Clusters {
		if cluster.Name == updateCluster.Name {
			c.Clusters[idx] = updateCluster
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

func (c *Cluster) ConfigFile(confDir string) string {
	// FIXME: need to decide on the name of this yaml file
	confPath := filepath.Join(confDir, "clusters", c.Name)
	configFile := filepath.Join(confPath, "machine.yaml")
	return configFile
}

func (c *Cluster) SaveConfig(confDir string) error {
	configFile := c.ConfigFile(confDir)
	clustersDir := filepath.Dir(configFile)
	fmt.Printf("clustersDir: %q configFile: %q\n", clustersDir, configFile)
	if !PathExists(clustersDir) {
		if err := os.MkdirAll(clustersDir, 0755); err != nil {
			return fmt.Errorf("Failed to create clustersDir %q: %s", clustersDir, err)
		}
	}
	contents, err := yaml.Marshal(c)
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

func (c *Cluster) RemoveConfig(configFile string) error {
	if PathExists(configFile) {
		// remove everything under the cluster dir
		clustersDir := filepath.Dir(configFile)
		fmt.Printf("Removing cluster config dir %q\n", clustersDir)
		return os.RemoveAll(clustersDir)
	}
	return nil
}
