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

// GetNodeUsersExecutor defines the interface for executing GetNodeUsers use case
type GetNodeUsersExecutor interface {
	Execute(ctx context.Context, cmd usecases.GetNodeUsersCommand) (*usecases.GetNodeUsersResult, error)
}

// ReportUserTrafficExecutor defines the interface for executing ReportUserTraffic use case
type ReportUserTrafficExecutor interface {
	Execute(ctx context.Context, cmd usecases.ReportUserTrafficCommand) (*usecases.ReportUserTrafficResult, error)
}

// ReportNodeStatusExecutor defines the interface for executing ReportNodeStatus use case
type ReportNodeStatusExecutor interface {
	Execute(ctx context.Context, cmd usecases.ReportNodeStatusCommand) (*usecases.ReportNodeStatusResult, error)
}

// ReportOnlineUsersExecutor defines the interface for executing ReportOnlineUsers use case
type ReportOnlineUsersExecutor interface {
	Execute(ctx context.Context, cmd usecases.ReportOnlineUsersCommand) (*usecases.ReportOnlineUsersResult, error)
}

// AgentHandler handles RESTful agent API requests (v2raysocks compatible)
type AgentHandler struct {
	getNodeConfigUC     GetNodeConfigExecutor
	getNodeUsersUC      GetNodeUsersExecutor
	reportUserTrafficUC ReportUserTrafficExecutor
	reportNodeStatusUC  ReportNodeStatusExecutor
	reportOnlineUsersUC ReportOnlineUsersExecutor
	logger              logger.Interface
}

// NewAgentHandler creates a new AgentHandler instance
func NewAgentHandler(
	getNodeConfigUC GetNodeConfigExecutor,
	getNodeUsersUC GetNodeUsersExecutor,
	reportUserTrafficUC ReportUserTrafficExecutor,
	reportNodeStatusUC ReportNodeStatusExecutor,
	reportOnlineUsersUC ReportOnlineUsersExecutor,
	logger logger.Interface,
) *AgentHandler {
	return &AgentHandler{
		getNodeConfigUC:     getNodeConfigUC,
		getNodeUsersUC:      getNodeUsersUC,
		reportUserTrafficUC: reportUserTrafficUC,
		reportNodeStatusUC:  reportNodeStatusUC,
		reportOnlineUsersUC: reportOnlineUsersUC,
		logger:              logger,
	}
}

// GetConfig godoc
// @Summary Get node configuration
// @Description Retrieve node configuration for agent clients (XrayR/v2ray compatible)
// @Tags agent-v1
// @Accept json
// @Produce json
// @Param id path int true "Node ID"
// @Param node_type query string false "Node type override" Enums(shadowsocks, trojan)
// @Success 200 {object} utils.APIResponse "Node configuration retrieved successfully"
// @Failure 400 {object} utils.APIResponse "Invalid node ID parameter"
// @Failure 404 {object} utils.APIResponse "Node not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /agents/{id}/config [get]
// @Security NodeToken
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

// GetUsers godoc
// @Summary Get authorized users list
// @Description Retrieve list of users authorized to access the node
// @Tags agent-v1
// @Accept json
// @Produce json
// @Param id path int true "Node ID"
// @Success 200 {object} utils.APIResponse "User list retrieved successfully"
// @Failure 400 {object} utils.APIResponse "Invalid node ID parameter"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /agents/{id}/users [get]
// @Security NodeToken
func (h *AgentHandler) GetUsers(c *gin.Context) {
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

	h.logger.Infow("node users request received",
		"node_id", nodeID,
		"ip", c.ClientIP(),
	)

	// Execute use case
	cmd := usecases.GetNodeUsersCommand{
		NodeID: uint(nodeID),
	}

	result, err := h.getNodeUsersUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.Errorw("failed to get node users",
			"error", err,
			"node_id", nodeID,
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to retrieve user list")
		return
	}

	h.logger.Infow("node users retrieved",
		"node_id", nodeID,
		"user_count", len(result.Users.Users),
		"ip", c.ClientIP(),
	)

	// Return success response
	utils.SuccessResponse(c, http.StatusOK, "user list retrieved successfully", result.Users.Users)
}

// ReportTraffic godoc
// @Summary Report user traffic data
// @Description Submit user traffic statistics for the node
// @Tags agent-v1
// @Accept json
// @Produce json
// @Param id path int true "Node ID"
// @Param traffic body []dto.UserTrafficItem true "User traffic data"
// @Success 200 {object} utils.APIResponse "Traffic reported successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request body or node ID"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /agents/{id}/traffic [post]
// @Security NodeToken
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
	var users []dto.UserTrafficItem
	if err := c.ShouldBindJSON(&users); err != nil {
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
		"user_count", len(users),
		"ip", c.ClientIP(),
	)

	// Execute use case
	cmd := usecases.ReportUserTrafficCommand{
		NodeID: uint(nodeID),
		Users:  users,
	}

	result, err := h.reportUserTrafficUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.Errorw("failed to report user traffic",
			"error", err,
			"node_id", nodeID,
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to process traffic report")
		return
	}

	h.logger.Infow("traffic reported successfully",
		"node_id", nodeID,
		"users_updated", result.UsersUpdated,
		"ip", c.ClientIP(),
	)

	// Return success response
	utils.SuccessResponse(c, http.StatusOK, "traffic reported successfully", map[string]any{
		"users_updated": result.UsersUpdated,
	})
}

// UpdateStatus godoc
// @Summary Update node system status
// @Description Report node system metrics (CPU, memory, disk, network, uptime)
// @Tags agent-v1
// @Accept json
// @Produce json
// @Param id path int true "Node ID"
// @Param status body dto.ReportNodeStatusRequest true "Node system status"
// @Success 200 {object} utils.APIResponse "Status updated successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request body or node ID"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /agents/{id}/status [put]
// @Security NodeToken
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

// UpdateOnlineUsers godoc
// @Summary Update online users list
// @Description Report currently connected users on the node
// @Tags agent-v1
// @Accept json
// @Produce json
// @Param id path int true "Node ID"
// @Param users body dto.ReportOnlineUsersRequest true "Online users list"
// @Success 200 {object} utils.APIResponse "Online users updated successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request body or node ID"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /agents/{id}/online-users [put]
// @Security NodeToken
func (h *AgentHandler) UpdateOnlineUsers(c *gin.Context) {
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
	var req dto.ReportOnlineUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid online users report request body",
			"error", err,
			"node_id", nodeID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	h.logger.Infow("online users update received",
		"node_id", nodeID,
		"user_count", len(req.Users),
		"ip", c.ClientIP(),
	)

	// Execute use case
	cmd := usecases.ReportOnlineUsersCommand{
		NodeID: uint(nodeID),
		Users:  req.Users,
	}

	result, err := h.reportOnlineUsersUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.Errorw("failed to report online users",
			"error", err,
			"node_id", nodeID,
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to process online users report")
		return
	}

	h.logger.Infow("online users updated successfully",
		"node_id", nodeID,
		"online_count", result.OnlineCount,
		"ip", c.ClientIP(),
	)

	// Return success response
	utils.SuccessResponse(c, http.StatusOK, "online users updated successfully", map[string]any{
		"online_count": result.OnlineCount,
	})
}
