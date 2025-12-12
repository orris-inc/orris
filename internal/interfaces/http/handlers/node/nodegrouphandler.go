package node

import (
	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// NodeGroupHandler handles HTTP requests for node group operations
type NodeGroupHandler struct {
	createNodeGroupUC           usecases.CreateNodeGroupExecutor
	getNodeGroupUC              usecases.GetNodeGroupExecutor
	updateNodeGroupUC           usecases.UpdateNodeGroupExecutor
	deleteNodeGroupUC           usecases.DeleteNodeGroupExecutor
	listNodeGroupsUC            usecases.ListNodeGroupsExecutor
	addNodeToGroupUC            usecases.AddNodeToGroupExecutor
	removeNodeFromGroupUC       usecases.RemoveNodeFromGroupExecutor
	batchAddNodesToGroupUC      usecases.BatchAddNodesToGroupExecutor
	batchRemoveNodesFromGroupUC usecases.BatchRemoveNodesFromGroupExecutor
	listGroupNodesUC            usecases.ListGroupNodesExecutor
	associateGroupWithPlanUC    usecases.AssociateGroupWithPlanExecutor
	disassociateGroupFromPlanUC usecases.DisassociateGroupFromPlanExecutor
	logger                      logger.Interface
}

// NewNodeGroupHandler creates a new NodeGroupHandler instance
func NewNodeGroupHandler(
	createNodeGroupUC usecases.CreateNodeGroupExecutor,
	getNodeGroupUC usecases.GetNodeGroupExecutor,
	updateNodeGroupUC usecases.UpdateNodeGroupExecutor,
	deleteNodeGroupUC usecases.DeleteNodeGroupExecutor,
	listNodeGroupsUC usecases.ListNodeGroupsExecutor,
	addNodeToGroupUC usecases.AddNodeToGroupExecutor,
	removeNodeFromGroupUC usecases.RemoveNodeFromGroupExecutor,
	batchAddNodesToGroupUC usecases.BatchAddNodesToGroupExecutor,
	batchRemoveNodesFromGroupUC usecases.BatchRemoveNodesFromGroupExecutor,
	listGroupNodesUC usecases.ListGroupNodesExecutor,
	associateGroupWithPlanUC usecases.AssociateGroupWithPlanExecutor,
	disassociateGroupFromPlanUC usecases.DisassociateGroupFromPlanExecutor,
) *NodeGroupHandler {
	return &NodeGroupHandler{
		createNodeGroupUC:           createNodeGroupUC,
		getNodeGroupUC:              getNodeGroupUC,
		updateNodeGroupUC:           updateNodeGroupUC,
		deleteNodeGroupUC:           deleteNodeGroupUC,
		listNodeGroupsUC:            listNodeGroupsUC,
		addNodeToGroupUC:            addNodeToGroupUC,
		removeNodeFromGroupUC:       removeNodeFromGroupUC,
		batchAddNodesToGroupUC:      batchAddNodesToGroupUC,
		batchRemoveNodesFromGroupUC: batchRemoveNodesFromGroupUC,
		listGroupNodesUC:            listGroupNodesUC,
		associateGroupWithPlanUC:    associateGroupWithPlanUC,
		disassociateGroupFromPlanUC: disassociateGroupFromPlanUC,
		logger:                      logger.NewLogger(),
	}
}
