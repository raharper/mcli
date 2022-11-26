package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Machine struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Cluster string `json:"cluster"`
	// VMDef here
}

type Cluster struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Machines []Machine `json:"machines"`
	State    string    `json:"state"`
}

// var m1 = machine{ID: "1", Name: "bob", Cluster: "cluster1"}
// var m2 = machine{ID: "1", Name: "karl", Cluster: "c2"}

var clusters = []Cluster{
	{
		ID:   "1",
		Name: "cluster1",
		Machines: []Machine{
			{
				ID:      "1.1",
				Name:    "bob",
				Cluster: "cluster1",
			},
		},
		State: "running",
	},
}

// TODO:
//  - cluster configuration persistence
//

func GetClusters(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, clusters)
}

func PostClusters(c *gin.Context) {
	var newCluster Cluster

	if err := c.BindJSON(&newCluster); err != nil {
		return
	}

	clusters = append(clusters, newCluster)
}

func GetAPIURL(endpoint string) string {
	return fmt.Sprintf("http://machined/%s", endpoint)
}
