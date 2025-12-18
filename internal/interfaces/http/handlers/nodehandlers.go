package handlers

import (
	nodeHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/node"
)

type NodeHandler = nodeHandlers.NodeHandler
type NodeSubscriptionHandler = nodeHandlers.SubscriptionHandler

var NewNodeHandler = nodeHandlers.NewNodeHandler
var NewNodeSubscriptionHandler = nodeHandlers.NewSubscriptionHandler
