package handlers

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

// XrayRHandler handles XrayR backend API requests (v2raysocks compatible)
type XrayRHandler struct {
	getNodeConfigUC     GetNodeConfigExecutor
	getNodeUsersUC      GetNodeUsersExecutor
	reportUserTrafficUC ReportUserTrafficExecutor
	reportNodeStatusUC  ReportNodeStatusExecutor
	reportOnlineUsersUC ReportOnlineUsersExecutor
	logger              logger.Interface
}

// NewXrayRHandler creates a new XrayRHandler instance
func NewXrayRHandler(
	getNodeConfigUC GetNodeConfigExecutor,
	getNodeUsersUC GetNodeUsersExecutor,
	reportUserTrafficUC ReportUserTrafficExecutor,
	reportNodeStatusUC ReportNodeStatusExecutor,
	reportOnlineUsersUC ReportOnlineUsersExecutor,
	logger logger.Interface,
) *XrayRHandler {
	return &XrayRHandler{
		getNodeConfigUC:     getNodeConfigUC,
		getNodeUsersUC:      getNodeUsersUC,
		reportUserTrafficUC: reportUserTrafficUC,
		reportNodeStatusUC:  reportNodeStatusUC,
		reportOnlineUsersUC: reportOnlineUsersUC,
		logger:              logger,
	}
}

// HandleXrayRRequest godoc
// @Summary Node backend unified API endpoint (v2raysocks compatible)
// @Description Unified endpoint for node backend operations (XrayR/v2ray compatible). Routes requests based on 'act' parameter.
// @Description
// @Description Supported actions and their request bodies:
// @Description - act=config (GET): No body required - returns node configuration
// @Description - act=user (GET): No body required - returns authorized user list
// @Description - act=submit (POST): Body should be []UserTrafficItem - report user traffic data
// @Description - act=nodestatus (POST): Body should be ReportNodeStatusRequest - report node system status
// @Description - act=onlineusers (POST): Body should be ReportOnlineUsersRequest - report online users
// @Tags node-backend
// @Accept json
// @Produce json
// @Param act query string true "Action type" Enums(config, user, submit, nodestatus, onlineusers)
// @Param node_id query int true "Node ID"
// @Param token query string true "Node authentication token"
// @Param node_type query string false "Node type (shadowsocks, trojan)" Enums(shadowsocks, trojan)
// @Success 200 {object} utils.V2RaySocksResponse "Success response with data"
// @Failure 400 {object} utils.V2RaySocksResponse "Invalid parameters"
// @Failure 401 {object} utils.V2RaySocksResponse "Unauthorized"
// @Failure 404 {object} utils.V2RaySocksResponse "Node not found"
// @Failure 500 {object} utils.V2RaySocksResponse "Internal server error"
// @Router /api/node [get]
// @Router /api/node [post]
// @Security NodeToken
func (h *XrayRHandler) HandleXrayRRequest(c *gin.Context) {
	act := c.Query("act")
	nodeIDStr := c.Query("node_id")
	nodeType := c.Query("node_type")

	h.logger.Infow("XrayR API request received",
		"act", act,
		"node_id", nodeIDStr,
		"node_type", nodeType,
		"method", c.Request.Method,
		"ip", c.ClientIP(),
	)

	// Parse node_id
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
	if err != nil {
		h.logger.Warnw("invalid node_id parameter", "node_id", nodeIDStr, "error", err)
		utils.V2RaySocksError(c, http.StatusBadRequest, "invalid node_id parameter")
		return
	}

	// Dispatch to appropriate handler based on act parameter
	switch act {
	case "config":
		h.handleGetConfig(c, uint(nodeID), nodeType)
	case "user":
		h.handleGetUsers(c, uint(nodeID))
	case "submit":
		h.handleReportTraffic(c, uint(nodeID))
	case "nodestatus":
		h.handleReportStatus(c, uint(nodeID))
	case "onlineusers":
		h.handleReportOnline(c, uint(nodeID))
	default:
		h.logger.Warnw("unknown act parameter", "act", act)
		utils.V2RaySocksError(c, http.StatusBadRequest, "unknown action")
	}
}

// handleGetConfig handles GET /api/xrayr?act=config
// Returns node configuration with ETag caching support
func (h *XrayRHandler) handleGetConfig(c *gin.Context, nodeID uint, nodeType string) {
	ctx := c.Request.Context()

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
		utils.V2RaySocksError(c, http.StatusInternalServerError, "failed to retrieve node configuration")
		return
	}

	// Return with ETag caching support
	utils.V2RaySocksSuccessWithETag(c, result.Config)
}

// handleGetUsers handles GET /api/xrayr?act=user
// Returns list of authorized users with ETag caching support
func (h *XrayRHandler) handleGetUsers(c *gin.Context, nodeID uint) {
	ctx := c.Request.Context()

	cmd := usecases.GetNodeUsersCommand{
		NodeID: nodeID,
	}

	result, err := h.getNodeUsersUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.Errorw("failed to get node users",
			"error", err,
			"node_id", nodeID,
		)
		utils.V2RaySocksError(c, http.StatusInternalServerError, "failed to retrieve user list")
		return
	}

	// Return with ETag caching support
	utils.V2RaySocksSuccessWithETag(c, result.Users.Users)
}

// handleReportTraffic handles POST /api/xrayr?act=submit
// Processes user traffic data reporting
func (h *XrayRHandler) handleReportTraffic(c *gin.Context, nodeID uint) {
	ctx := c.Request.Context()

	// Parse request body
	var users []dto.UserTrafficItem
	if err := c.ShouldBindJSON(&users); err != nil {
		h.logger.Warnw("invalid traffic report request body",
			"error", err,
			"node_id", nodeID,
		)
		utils.V2RaySocksError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	cmd := usecases.ReportUserTrafficCommand{
		NodeID: nodeID,
		Users:  users,
	}

	result, err := h.reportUserTrafficUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.Errorw("failed to report user traffic",
			"error", err,
			"node_id", nodeID,
		)
		utils.V2RaySocksError(c, http.StatusInternalServerError, "failed to process traffic report")
		return
	}

	// Return success with ret=1
	utils.V2RaySocksSuccessWithRet(c, map[string]any{
		"users_updated": result.UsersUpdated,
	}, 1)
}

// handleReportStatus handles POST /api/xrayr?act=nodestatus
// Processes node system status reporting
func (h *XrayRHandler) handleReportStatus(c *gin.Context, nodeID uint) {
	ctx := c.Request.Context()

	// Parse request body
	var status dto.ReportNodeStatusRequest
	if err := c.ShouldBindJSON(&status); err != nil {
		h.logger.Warnw("invalid status report request body",
			"error", err,
			"node_id", nodeID,
		)
		utils.V2RaySocksError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	cmd := usecases.ReportNodeStatusCommand{
		NodeID: nodeID,
		Status: status,
	}

	_, err := h.reportNodeStatusUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.Errorw("failed to report node status",
			"error", err,
			"node_id", nodeID,
		)
		utils.V2RaySocksError(c, http.StatusInternalServerError, "failed to process status report")
		return
	}

	// Return success with ret=1
	utils.V2RaySocksSuccessWithRet(c, map[string]any{
		"status": "ok",
	}, 1)
}

// handleReportOnline handles POST /api/xrayr?act=onlineusers
// Processes online users reporting
func (h *XrayRHandler) handleReportOnline(c *gin.Context, nodeID uint) {
	ctx := c.Request.Context()

	// Parse request body
	var req dto.ReportOnlineUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid online users report request body",
			"error", err,
			"node_id", nodeID,
		)
		utils.V2RaySocksError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	cmd := usecases.ReportOnlineUsersCommand{
		NodeID: nodeID,
		Users:  req.Users,
	}

	result, err := h.reportOnlineUsersUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.Errorw("failed to report online users",
			"error", err,
			"node_id", nodeID,
		)
		utils.V2RaySocksError(c, http.StatusInternalServerError, "failed to process online users report")
		return
	}

	// Return success with ret=1
	utils.V2RaySocksSuccessWithRet(c, map[string]any{
		"online_count": result.OnlineCount,
	}, 1)
}
