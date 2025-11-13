package node

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"orris/internal/application/node/dto"
	"orris/internal/application/node/usecases"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
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

// AgentHandler handles RESTful agent API requests (v2raysocks compatible)
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

// GetConfig godoc
//
//	@Summary		Get node configuration
//	@Description	Retrieve node configuration for agent clients (XrayR/v2ray compatible)
//	@Tags			agent-v1
//	@Accept			json
//	@Produce		json
//	@Param			id			path		int					true	"Node ID"
//	@Param			node_type	query		string				false	"Node type override"	Enums(shadowsocks, trojan)
//	@Success		200			{object}	utils.APIResponse	"Node configuration retrieved successfully"
//	@Failure		400			{object}	utils.APIResponse	"Invalid node ID parameter"
//	@Failure		404			{object}	utils.APIResponse	"Node not found"
//	@Failure		500			{object}	utils.APIResponse	"Internal server error"
//	@Router			/agents/{id}/config [get]
//	@Security		NodeToken
func (h *AgentHandler) GetConfig(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse node ID from path parameter
	nodeIDStr := c.Param("id")
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
	if err != nil {
		h.logger.Warnw("invalid node_id parameter",
			"node_id", nodeIDStr,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid node_id parameter")
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
		NodeID:   uint(nodeID),
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

// GetSubscriptions godoc
//
//	@Summary		Get active subscriptions for node
//	@Description	Retrieve list of active subscriptions authorized to access the node
//	@Tags			agent-v1
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int					true	"Node ID"
//	@Success		200	{object}	utils.APIResponse	"Subscription list retrieved successfully"
//	@Failure		400	{object}	utils.APIResponse	"Invalid node ID parameter"
//	@Failure		500	{object}	utils.APIResponse	"Internal server error"
//	@Router			/agents/{id}/subscriptions [get]
//	@Security		NodeToken
func (h *AgentHandler) GetSubscriptions(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse node ID from path parameter
	nodeIDStr := c.Param("id")
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
	if err != nil {
		h.logger.Warnw("invalid node_id parameter",
			"node_id", nodeIDStr,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid node_id parameter")
		return
	}

	h.logger.Infow("node subscriptions request received",
		"node_id", nodeID,
		"ip", c.ClientIP(),
	)

	// Execute use case
	cmd := usecases.GetNodeSubscriptionsCommand{
		NodeID: uint(nodeID),
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

// ReportTraffic godoc
//
//	@Summary		Report subscription traffic data
//	@Description	Submit subscription traffic statistics for the node
//	@Tags			agent-v1
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int								true	"Node ID"
//	@Param			traffic	body		[]dto.SubscriptionTrafficItem	true	"Subscription traffic data"
//	@Success		200		{object}	utils.APIResponse				"Traffic reported successfully"
//	@Failure		400		{object}	utils.APIResponse				"Invalid request body or node ID"
//	@Failure		500		{object}	utils.APIResponse				"Internal server error"
//	@Router			/agents/{id}/traffic [post]
//	@Security		NodeToken
func (h *AgentHandler) ReportTraffic(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse node ID from path parameter
	nodeIDStr := c.Param("id")
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
	if err != nil {
		h.logger.Warnw("invalid node_id parameter",
			"node_id", nodeIDStr,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid node_id parameter")
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
		NodeID:        uint(nodeID),
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

// UpdateStatus godoc
//
//	@Summary		Update node system status
//	@Description	Report node system metrics (CPU, memory, disk, network, uptime)
//	@Tags			agent-v1
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int							true	"Node ID"
//	@Param			status	body		dto.ReportNodeStatusRequest	true	"Node system status"
//	@Success		200		{object}	utils.APIResponse			"Status updated successfully"
//	@Failure		400		{object}	utils.APIResponse			"Invalid request body or node ID"
//	@Failure		500		{object}	utils.APIResponse			"Internal server error"
//	@Router			/agents/{id}/status [put]
//	@Security		NodeToken
func (h *AgentHandler) UpdateStatus(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse node ID from path parameter
	nodeIDStr := c.Param("id")
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
	if err != nil {
		h.logger.Warnw("invalid node_id parameter",
			"node_id", nodeIDStr,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid node_id parameter")
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
		NodeID: uint(nodeID),
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

// UpdateOnlineSubscriptions godoc
//
//	@Summary		Update online subscriptions list
//	@Description	Report currently connected subscriptions on the node
//	@Tags			agent-v1
//	@Accept			json
//	@Produce		json
//	@Param			id				path		int										true	"Node ID"
//	@Param			subscriptions	body		dto.ReportOnlineSubscriptionsRequest	true	"Online subscriptions list"
//	@Success		200				{object}	utils.APIResponse						"Online subscriptions updated successfully"
//	@Failure		400				{object}	utils.APIResponse						"Invalid request body or node ID"
//	@Failure		500				{object}	utils.APIResponse						"Internal server error"
//	@Router			/agents/{id}/online-subscriptions [put]
//	@Security		NodeToken
func (h *AgentHandler) UpdateOnlineSubscriptions(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse node ID from path parameter
	nodeIDStr := c.Param("id")
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
	if err != nil {
		h.logger.Warnw("invalid node_id parameter",
			"node_id", nodeIDStr,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid node_id parameter")
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
		NodeID:        uint(nodeID),
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
