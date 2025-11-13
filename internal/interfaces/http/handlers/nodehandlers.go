package handlers

import (
	nodeHandlers "orris/internal/interfaces/http/handlers/node"
)

type NodeHandler = nodeHandlers.NodeHandler
type NodeGroupHandler = nodeHandlers.NodeGroupHandler
type NodeSubscriptionHandler = nodeHandlers.SubscriptionHandler

var NewNodeHandler = nodeHandlers.NewNodeHandler
var NewNodeGroupHandler = nodeHandlers.NewNodeGroupHandler
var NewNodeSubscriptionHandler = nodeHandlers.NewSubscriptionHandler
