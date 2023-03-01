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
	log "github.com/sirupsen/logrus"
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
	rh.c.Router.GET("/machines", rh.GetMachines)
	rh.c.Router.POST("/machines", rh.PostMachine)
	rh.c.Router.PUT("/machines/:machinename", rh.UpdateMachine)
	rh.c.Router.DELETE("/machines/:machinename", rh.DeleteMachine)
	rh.c.Router.POST("/machines/:machinename/start", rh.StartMachine)
	rh.c.Router.POST("/machines/:machinename/stop", rh.StopMachine)
}

func (rh *RouteHandler) GetMachines(ctx *gin.Context) {
	ctx.IndentedJSON(http.StatusOK, rh.c.MachineController.GetMachines())
}

func (rh *RouteHandler) PostMachine(ctx *gin.Context) {
	var newMachine Machine
	if err := ctx.BindJSON(&newMachine); err != nil {
		return
	}
	cfg := rh.c.Config
	if err := rh.c.MachineController.AddMachine(newMachine, cfg); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func (rh *RouteHandler) DeleteMachine(ctx *gin.Context) {
	machineName := ctx.Param("machinename")
	cfg := rh.c.Config
	// TODO refuse if machine status is running, handle --force param
	err := rh.c.MachineController.DeleteMachine(machineName, cfg)
	if err != nil {
		log.Errorf("Failed to delete machine '%s': %s\n", machineName, err)
	}
}

func (rh *RouteHandler) UpdateMachine(ctx *gin.Context) {
	var newMachine Machine
	if err := ctx.ShouldBindJSON(&newMachine); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cfg := rh.c.Config
	if err := rh.c.MachineController.UpdateMachine(newMachine, cfg); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func (rh *RouteHandler) StartMachine(ctx *gin.Context) {
	machineName := ctx.Param("machinename")
	var request struct {
		Status string `json:"status"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if request.Status == "running" {
		if err := rh.c.MachineController.StartMachine(machineName); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
	} else {
		err := fmt.Errorf("Invalid Start request: '%v;", request)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
}

func (rh *RouteHandler) StopMachine(ctx *gin.Context) {
	machineName := ctx.Param("machinename")
	var request struct {
		Status string `json:"status"`
		Force  bool   `json:"force"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if request.Status == "stopped" {
		if err := rh.c.MachineController.StopMachine(machineName, request.Force); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
	} else {
		err := fmt.Errorf("Invalid Stop request: '%v;", request)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
}
