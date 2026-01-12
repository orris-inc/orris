package node

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/shared/id"
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

// ReportSubscriptionUsageExecutor defines the interface for executing ReportSubscriptionUsage use case
type ReportSubscriptionUsageExecutor interface {
	Execute(ctx context.Context, cmd usecases.ReportSubscriptionUsageCommand) (*usecases.ReportSubscriptionUsageResult, error)
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
	reportSubscriptionUsageUC   ReportSubscriptionUsageExecutor
	reportNodeStatusUC          ReportNodeStatusExecutor
	reportOnlineSubscriptionsUC ReportOnlineSubscriptionsExecutor
	logger                      logger.Interface
}

// NewAgentHandler creates a new AgentHandler instance
func NewAgentHandler(
	getNodeConfigUC GetNodeConfigExecutor,
	getNodeSubscriptionsUC GetNodeSubscriptionsExecutor,
	reportSubscriptionUsageUC ReportSubscriptionUsageExecutor,
	reportNodeStatusUC ReportNodeStatusExecutor,
	reportOnlineSubscriptionsUC ReportOnlineSubscriptionsExecutor,
	logger logger.Interface,
) *AgentHandler {
	return &AgentHandler{
		getNodeConfigUC:             getNodeConfigUC,
		getNodeSubscriptionsUC:      getNodeSubscriptionsUC,
		reportSubscriptionUsageUC:   reportSubscriptionUsageUC,
		reportNodeStatusUC:          reportNodeStatusUC,
		reportOnlineSubscriptionsUC: reportOnlineSubscriptionsUC,
		logger:                      logger,
	}
}

// validateNodeAccess validates that the URL node SID matches the authenticated node SID from token.
// Returns the validated internal node ID or an error if validation fails.
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

	// Get authenticated node SID from context
	authenticatedNodeSID, exists := c.Get("node_sid")
	if !exists {
		return 0, fmt.Errorf("node_sid not found in context")
	}

	authNodeSID, ok := authenticatedNodeSID.(string)
	if !ok {
		return 0, fmt.Errorf("invalid node_sid type in context")
	}

	// Parse requested node SID from URL path parameter
	requestedNodeSID := c.Param("nodesid")
	if err := id.ValidatePrefix(requestedNodeSID, id.PrefixNode); err != nil {
		return 0, fmt.Errorf("invalid node SID format: %s", requestedNodeSID)
	}

	// Validate that requested node SID matches authenticated node SID
	if requestedNodeSID != authNodeSID {
		h.logger.Warnw("node access denied: token does not match requested node",
			"requested_node_sid", requestedNodeSID,
			"authenticated_node_sid", authNodeSID,
			"ip", c.ClientIP(),
		)
		return 0, fmt.Errorf("access denied: token does not authorize access to node %s", requestedNodeSID)
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

	// Execute use case
	cmd := usecases.GetNodeConfigCommand{
		NodeID:   nodeID,
		NodeType: nodeType,
	}

	result, err := h.getNodeConfigUC.Execute(ctx, cmd)
	if err != nil {
		// Determine appropriate status code based on error
		statusCode := http.StatusInternalServerError
		message := "failed to retrieve node configuration"

		if err.Error() == "node not found" {
			statusCode = http.StatusNotFound
			message = "node not found"
		}

		utils.ErrorResponse(c, statusCode, message)
		return
	}

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

	// Execute use case
	cmd := usecases.GetNodeSubscriptionsCommand{
		NodeID: nodeID,
	}

	result, err := h.getNodeSubscriptionsUC.Execute(ctx, cmd)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to retrieve subscription list")
		return
	}

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
	var subscriptions []dto.SubscriptionUsageItem
	if err := c.ShouldBindJSON(&subscriptions); err != nil {
		h.logger.Warnw("invalid usage report request body",
			"error", err,
			"node_id", nodeID,
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Execute use case
	cmd := usecases.ReportSubscriptionUsageCommand{
		NodeID:        nodeID,
		Subscriptions: subscriptions,
	}

	result, err := h.reportSubscriptionUsageUC.Execute(ctx, cmd)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to process usage report")
		return
	}

	// Return success response
	utils.SuccessResponse(c, http.StatusOK, "usage reported successfully", map[string]any{
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
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Execute use case
	cmd := usecases.ReportNodeStatusCommand{
		NodeID: nodeID,
		Status: status,
	}

	_, err = h.reportNodeStatusUC.Execute(ctx, cmd)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to process status report")
		return
	}

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
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Execute use case
	cmd := usecases.ReportOnlineSubscriptionsCommand{
		NodeID:        nodeID,
		Subscriptions: req.Subscriptions,
	}

	result, err := h.reportOnlineSubscriptionsUC.Execute(ctx, cmd)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to process online subscriptions report")
		return
	}

	// Return success response
	utils.SuccessResponse(c, http.StatusOK, "online subscriptions updated successfully", map[string]any{
		"online_count": result.OnlineCount,
	})
}
