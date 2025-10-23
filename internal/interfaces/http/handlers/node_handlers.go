package handlers

import (
	nodeHandlers "orris/internal/interfaces/http/handlers/node"
)

type NodeHandler = nodeHandlers.NodeHandler
type NodeGroupHandler = nodeHandlers.NodeGroupHandler
type NodeSubscriptionHandler = nodeHandlers.SubscriptionHandler
type NodeReportHandler = nodeHandlers.ReportHandler

var NewNodeHandler = nodeHandlers.NewNodeHandler
var NewNodeGroupHandler = nodeHandlers.NewNodeGroupHandler
var NewNodeSubscriptionHandler = nodeHandlers.NewSubscriptionHandler
var NewNodeReportHandler = nodeHandlers.NewReportHandler
