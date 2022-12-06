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

import "fmt"

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

func (c *ClusterController) AddCluster(newCluster Cluster) error {
	if _, err := c.GetClusterByName(newCluster.Name); err == nil {
		return fmt.Errorf("Cluster '%s' is already defined", newCluster.Name)
	}
	c.Clusters = append(c.Clusters, newCluster)
	return nil
}
