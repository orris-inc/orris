package node

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"orris/internal/application/node/usecases"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

type ReportNodeDataExecutor interface {
	Execute(ctx context.Context, cmd usecases.ReportNodeDataCommand) (*usecases.ReportNodeDataResult, error)
}

type ValidateNodeTokenExecutor interface {
	Execute(ctx context.Context, cmd usecases.ValidateNodeTokenCommand) (*usecases.ValidateNodeTokenResult, error)
}

type ReportHandler struct {
	reportNodeDataUC    ReportNodeDataExecutor
	validateNodeTokenUC ValidateNodeTokenExecutor
	logger              logger.Interface
}

func NewReportHandler(
	reportNodeDataUC ReportNodeDataExecutor,
	validateNodeTokenUC ValidateNodeTokenExecutor,
) *ReportHandler {
	return &ReportHandler{
		reportNodeDataUC:    reportNodeDataUC,
		validateNodeTokenUC: validateNodeTokenUC,
		logger:              logger.NewLogger(),
	}
}

// ReportNodeData handles POST /nodes/report
// @Summary Report node data
// @Description Report node traffic and status data (requires node token)
// @Tags nodes
// @Accept json
// @Produce json
// @Security NodeToken
// @Param data body ReportNodeDataRequest true "Node report data"
// @Success 200 {object} utils.APIResponse "Data reported successfully"
// @Failure 400 {object} utils.APIResponse "Bad request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /nodes/report [post]
func (h *ReportHandler) ReportNodeData(c *gin.Context) {
	nodeID, exists := c.Get("node_id")
	if !exists {
		h.logger.Warnw("node_id not found in context")
		utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("Node authentication required"))
		return
	}

	var req ReportNodeDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for report node data", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.ReportNodeDataCommand{
		NodeID:      nodeID.(uint),
		Upload:      req.Upload,
		Download:    req.Download,
		OnlineUsers: req.OnlineUsers,
		Status:      req.Status,
		Timestamp:   time.Now(),
	}

	if req.SystemInfo != nil {
		cmd.SystemInfo = &usecases.SystemInfo{
			Load:        req.SystemInfo.Load,
			MemoryUsage: req.SystemInfo.MemoryUsage,
			DiskUsage:   req.SystemInfo.DiskUsage,
		}
	}

	result, err := h.reportNodeDataUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Data reported successfully", result)
}

// Heartbeat handles POST /nodes/report/heartbeat
// @Summary Node heartbeat
// @Description Send heartbeat signal to indicate node is alive (requires node token)
// @Tags nodes
// @Accept json
// @Produce json
// @Security NodeToken
// @Success 200 {object} utils.APIResponse "Heartbeat received"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /nodes/report/heartbeat [post]
func (h *ReportHandler) Heartbeat(c *gin.Context) {
	nodeID, exists := c.Get("node_id")
	if !exists {
		h.logger.Warnw("node_id not found in context")
		utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("Node authentication required"))
		return
	}

	h.logger.Infow("heartbeat received", "node_id", nodeID)
	utils.SuccessResponse(c, http.StatusOK, "Heartbeat received", map[string]interface{}{
		"node_id":   nodeID,
		"timestamp": time.Now().Unix(),
	})
}

// NodeTokenMiddleware validates node API token
func (h *ReportHandler) NodeTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			h.logger.Warnw("missing authorization header")
			utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("Authorization header required"))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			h.logger.Warnw("invalid authorization header format")
			utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("Invalid authorization header format"))
			c.Abort()
			return
		}

		token := parts[1]
		if token == "" {
			h.logger.Warnw("empty token")
			utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("Token is required"))
			c.Abort()
			return
		}

		cmd := usecases.ValidateNodeTokenCommand{
			PlainToken: token,
			IPAddress:  c.ClientIP(),
		}

		result, err := h.validateNodeTokenUC.Execute(c.Request.Context(), cmd)
		if err != nil {
			h.logger.Warnw("token validation failed", "error", err, "ip", c.ClientIP())
			utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("Invalid or expired token"))
			c.Abort()
			return
		}

		c.Set("node_id", result.NodeID)
		c.Set("node_name", result.Name)

		h.logger.Infow("node authenticated successfully",
			"node_id", result.NodeID,
			"node_name", result.Name,
			"ip", c.ClientIP(),
		)

		c.Next()
	}
}

type ReportNodeDataRequest struct {
	Upload      uint64      `json:"upload" binding:"required"`
	Download    uint64      `json:"download" binding:"required"`
	OnlineUsers int         `json:"online_users"`
	Status      string      `json:"status"`
	SystemInfo  *SystemInfo `json:"system_info,omitempty"`
}

type SystemInfo struct {
	Load        float64 `json:"load"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
}
