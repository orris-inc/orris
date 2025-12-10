package node

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// GetNodeConfigExecutor defines the interface for executing GetNodeConfig use case
type GetNodeConfigExecutor interface {
	Execute(ctx context.Context, cmd usecases.GetNodeConfigCommand) (*usecases.GetNodeConfigResult, error)
}

// GetNodeSubscriptionsExecutor defines the interface for executing GetNodeSubscriptions use case
type GetNodeSubscriptionsExecutor interface {
	Execute(ctx context.Context, cmd usecases.GetNodeSubscriptionsCommand) (*usecases.GetNodeSubscriptionsResult, error)
}

// ReportSubscriptionTrafficExecutor defines the interface for executing ReportSubscriptionTraffic use case
type ReportSubscriptionTrafficExecutor interface {
	Execute(ctx context.Context, cmd usecases.ReportSubscriptionTrafficCommand) (*usecases.ReportSubscriptionTrafficResult, error)
}

// ReportNodeStatusExecutor defines the interface for executing ReportNodeStatus use case
type ReportNodeStatusExecutor interface {
	Execute(ctx context.Context, cmd usecases.ReportNodeStatusCommand) (*usecases.ReportNodeStatusResult, error)
}

// ReportOnlineSubscriptionsExecutor defines the interface for executing ReportOnlineSubscriptions use case
type ReportOnlineSubscriptionsExecutor interface {
	Execute(ctx context.Context, cmd usecases.ReportOnlineSubscriptionsCommand) (*usecases.ReportOnlineSubscriptionsResult, error)
}

// AgentHandler handles RESTful agent API requests
type AgentHandler struct {
	getNodeConfigUC             GetNodeConfigExecutor
	getNodeSubscriptionsUC      GetNodeSubscriptionsExecutor
	reportSubscriptionTrafficUC ReportSubscriptionTrafficExecutor
	reportNodeStatusUC          ReportNodeStatusExecutor
	reportOnlineSubscriptionsUC ReportOnlineSubscriptionsExecutor
	logger                      logger.Interface
}

// NewAgentHandler creates a new AgentHandler instance
func NewAgentHandler(
	getNodeConfigUC GetNodeConfigExecutor,
	getNodeSubscriptionsUC GetNodeSubscriptionsExecutor,
	reportSubscriptionTrafficUC ReportSubscriptionTrafficExecutor,
	reportNodeStatusUC ReportNodeStatusExecutor,
	reportOnlineSubscriptionsUC ReportOnlineSubscriptionsExecutor,
	logger logger.Interface,
) *AgentHandler {
	return &AgentHandler{
		getNodeConfigUC:             getNodeConfigUC,
		getNodeSubscriptionsUC:      getNodeSubscriptionsUC,
		reportSubscriptionTrafficUC: reportSubscriptionTrafficUC,
		reportNodeStatusUC:          reportNodeStatusUC,
		reportOnlineSubscriptionsUC: reportOnlineSubscriptionsUC,
		logger:                      logger,
	}
}

// validateNodeAccess validates that the URL node ID matches the authenticated node ID from token.
// Returns the validated node ID or an error if validation fails.
func (h *AgentHandler) validateNodeAccess(c *gin.Context) (uint, error) {
	// Get authenticated node ID from context (set by middleware after token validation)
	authenticatedNodeID, exists := c.Get("node_id")
	if !exists {
		return 0, fmt.Errorf("node_id not found in context")
	}

	authNodeID, ok := authenticatedNodeID.(uint)
	if !ok {
		return 0, fmt.Errorf("invalid node_id type in context")
	}

	// Parse requested node ID from URL path parameter
	nodeIDStr := c.Param("id")
	requestedNodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid node_id parameter: %s", nodeIDStr)
	}

	// Validate that requested node ID matches authenticated node ID
	if uint(requestedNodeID) != authNodeID {
		h.logger.Warnw("node access denied: token does not match requested node",
			"requested_node_id", requestedNodeID,
			"authenticated_node_id", authNodeID,
			"ip", c.ClientIP(),
		)
		return 0, fmt.Errorf("access denied: token does not authorize access to node %d", requestedNodeID)
	}

	return authNodeID, nil
}

func (h *AgentHandler) GetConfig(c *gin.Context) {
	ctx := c.Request.Context()

	// Validate node access: ensure URL node ID matches authenticated token
	nodeID, err := h.validateNodeAccess(c)
	if err != nil {
		h.logger.Warnw("node access validation failed",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusForbidden, "access denied")
		return
	}

	// Get optional node_type query parameter
	nodeType := c.Query("node_type")

	h.logger.Infow("node configuration request received",
		"node_id", nodeID,
		"node_type", nodeType,
		"ip", c.ClientIP(),
	)

	// Execute use case
	cmd := usecases.GetNodeConfigCommand{
		NodeID:   nodeID,
		NodeType: nodeType,
	}

	result, err := h.getNodeConfigUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.Errorw("failed to get node config",
			"error", err,
			"node_id", nodeID,
		)

		// Determine appropriate status code based on error
		statusCode := http.StatusInternalServerError
		message := "failed to retrieve node configuration"

		if err.Error() == "node not found" {
			statusCode = http.StatusNotFound
			message = "node not found"
		} else if err.Error() == "node is not active" {
			statusCode = http.StatusNotFound
			message = "node is not available"
		}

		utils.ErrorResponse(c, statusCode, message)
		return
	}

	h.logger.Infow("node configuration retrieved",
		"node_id", nodeID,
		"ip", c.ClientIP(),
	)

	// Return success response
	utils.SuccessResponse(c, http.StatusOK, "node configuration retrieved successfully", result.Config)
}

func (h *AgentHandler) GetSubscriptions(c *gin.Context) {
	ctx := c.Request.Context()

	// Validate node access: ensure URL node ID matches authenticated token
	nodeID, err := h.validateNodeAccess(c)
	if err != nil {
		h.logger.Warnw("node access validation failed",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusForbidden, "access denied")
		return
	}

	h.logger.Infow("node subscriptions request received",
		"node_id", nodeID,
		"ip", c.ClientIP(),
	)

	// Execute use case
	cmd := usecases.GetNodeSubscriptionsCommand{
		NodeID: nodeID,
	}

	result, err := h.getNodeSubscriptionsUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.Errorw("failed to get node subscriptions",
			"error", err,
			"node_id", nodeID,
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to retrieve subscription list")
		return
	}

	h.logger.Infow("node subscriptions retrieved",
		"node_id", nodeID,
		"subscription_count", len(result.Subscriptions.Subscriptions),
		"ip", c.ClientIP(),
	)

	// Return success response
	utils.SuccessResponse(c, http.StatusOK, "subscription list retrieved successfully", result.Subscriptions.Subscriptions)
}

func (h *AgentHandler) ReportTraffic(c *gin.Context) {
	ctx := c.Request.Context()

	// Validate node access: ensure URL node ID matches authenticated token
	nodeID, err := h.validateNodeAccess(c)
	if err != nil {
		h.logger.Warnw("node access validation failed",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusForbidden, "access denied")
		return
	}

	// Parse request body
	var subscriptions []dto.SubscriptionTrafficItem
	if err := c.ShouldBindJSON(&subscriptions); err != nil {
		h.logger.Warnw("invalid traffic report request body",
			"error", err,
			"node_id", nodeID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	h.logger.Infow("traffic report received",
		"node_id", nodeID,
		"subscription_count", len(subscriptions),
		"ip", c.ClientIP(),
	)

	// Execute use case
	cmd := usecases.ReportSubscriptionTrafficCommand{
		NodeID:        nodeID,
		Subscriptions: subscriptions,
	}

	result, err := h.reportSubscriptionTrafficUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.Errorw("failed to report subscription traffic",
			"error", err,
			"node_id", nodeID,
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to process traffic report")
		return
	}

	h.logger.Infow("traffic reported successfully",
		"node_id", nodeID,
		"subscriptions_updated", result.SubscriptionsUpdated,
		"ip", c.ClientIP(),
	)

	// Return success response
	utils.SuccessResponse(c, http.StatusOK, "traffic reported successfully", map[string]any{
		"subscriptions_updated": result.SubscriptionsUpdated,
	})
}

func (h *AgentHandler) UpdateStatus(c *gin.Context) {
	ctx := c.Request.Context()

	// Validate node access: ensure URL node ID matches authenticated token
	nodeID, err := h.validateNodeAccess(c)
	if err != nil {
		h.logger.Warnw("node access validation failed",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusForbidden, "access denied")
		return
	}

	// Parse request body
	var status dto.ReportNodeStatusRequest
	if err := c.ShouldBindJSON(&status); err != nil {
		h.logger.Warnw("invalid status report request body",
			"error", err,
			"node_id", nodeID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	h.logger.Infow("node status update received",
		"node_id", nodeID,
		"cpu", status.CPU,
		"memory", status.Mem,
		"ip", c.ClientIP(),
	)

	// Execute use case
	cmd := usecases.ReportNodeStatusCommand{
		NodeID: nodeID,
		Status: status,
	}

	_, err = h.reportNodeStatusUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.Errorw("failed to report node status",
			"error", err,
			"node_id", nodeID,
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to process status report")
		return
	}

	h.logger.Infow("node status updated successfully",
		"node_id", nodeID,
		"ip", c.ClientIP(),
	)

	// Return success response
	utils.SuccessResponse(c, http.StatusOK, "status updated successfully", map[string]any{
		"status": "ok",
	})
}

func (h *AgentHandler) UpdateOnlineSubscriptions(c *gin.Context) {
	ctx := c.Request.Context()

	// Validate node access: ensure URL node ID matches authenticated token
	nodeID, err := h.validateNodeAccess(c)
	if err != nil {
		h.logger.Warnw("node access validation failed",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusForbidden, "access denied")
		return
	}

	// Parse request body
	var req dto.ReportOnlineSubscriptionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid online subscriptions report request body",
			"error", err,
			"node_id", nodeID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	h.logger.Infow("online subscriptions update received",
		"node_id", nodeID,
		"subscription_count", len(req.Subscriptions),
		"ip", c.ClientIP(),
	)

	// Execute use case
	cmd := usecases.ReportOnlineSubscriptionsCommand{
		NodeID:        nodeID,
		Subscriptions: req.Subscriptions,
	}

	result, err := h.reportOnlineSubscriptionsUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.Errorw("failed to report online subscriptions",
			"error", err,
			"node_id", nodeID,
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to process online subscriptions report")
		return
	}

	h.logger.Infow("online subscriptions updated successfully",
		"node_id", nodeID,
		"online_count", result.OnlineCount,
		"ip", c.ClientIP(),
	)

	// Return success response
	utils.SuccessResponse(c, http.StatusOK, "online subscriptions updated successfully", map[string]any{
		"online_count": result.OnlineCount,
	})
}
