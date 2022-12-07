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
	"net/http"

	"github.com/gin-gonic/gin"
)

type RouteHandler struct {
	c *Controller
}

func NewRouteHandler(c *Controller) *RouteHandler {
	routeHandler := &RouteHandler{c: c}
	routeHandler.SetupRoutes()

	return routeHandler
}

func (rh *RouteHandler) SetupRoutes() {
	rh.c.Router.GET("/clusters", rh.GetClusters)
	rh.c.Router.POST("/clusters", rh.PostClusters)
	rh.c.Router.PUT("/clusters/:clustername", rh.UpdateClusters)
	rh.c.Router.DELETE("/clusters/:clustername", rh.DeleteClusters)
}

func (rh *RouteHandler) GetClusters(ctx *gin.Context) {
	ctx.IndentedJSON(http.StatusOK, rh.c.ClusterController.GetClusters())
}

func (rh *RouteHandler) PostClusters(ctx *gin.Context) {
	var newCluster Cluster
	if err := ctx.BindJSON(&newCluster); err != nil {
		return
	}
	conf := rh.c.Config.ConfigDirectory
	if err := rh.c.ClusterController.AddCluster(newCluster, conf); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func (rh *RouteHandler) DeleteClusters(ctx *gin.Context) {
	clusterName := ctx.Param("clustername")
	conf := rh.c.Config.ConfigDirectory
	// TODO refuse if cluster status is running, handle --force param
	err := rh.c.ClusterController.DeleteCluster(clusterName, conf)
	if err != nil {
		fmt.Printf("ERROR: Failed to delete cluster '%s': %s\n", clusterName, err)
	}
}

func (rh *RouteHandler) UpdateClusters(ctx *gin.Context) {
	var newCluster Cluster
	if err := ctx.BindJSON(&newCluster); err != nil {
		return
	}
	conf := rh.c.Config.ConfigDirectory
	if err := rh.c.ClusterController.UpdateCluster(newCluster, conf); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}
