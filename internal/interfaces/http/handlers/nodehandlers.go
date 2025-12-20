package handlers

import (
	nodeHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/node"
)

type NodeHandler = nodeHandlers.NodeHandler
type UserNodeHandler = nodeHandlers.UserNodeHandler
type NodeSubscriptionHandler = nodeHandlers.SubscriptionHandler

var NewNodeHandler = nodeHandlers.NewNodeHandler
var NewUserNodeHandler = nodeHandlers.NewUserNodeHandler
var NewNodeSubscriptionHandler = nodeHandlers.NewSubscriptionHandler
