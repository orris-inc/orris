package node

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

type GenerateSubscriptionExecutor interface {
	Execute(ctx context.Context, cmd usecases.GenerateSubscriptionCommand) (*usecases.GenerateSubscriptionResult, error)
}

type SubscriptionHandler struct {
	generateSubscriptionUC GenerateSubscriptionExecutor
	logger                 logger.Interface
}

func NewSubscriptionHandler(
	generateSubscriptionUC GenerateSubscriptionExecutor,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		generateSubscriptionUC: generateSubscriptionUC,
		logger:                 logger.NewLogger(),
	}
}

// GetSubscription handles GET /s/:token with auto-format detection from User-Agent
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.ErrorResponseWithError(c, errors.NewValidationError("Subscription token is required"))
		return
	}

	// Auto-detect format from User-Agent header
	userAgent := c.GetHeader("User-Agent")
	format := detectFormatFromUserAgent(userAgent)

	cmd := usecases.GenerateSubscriptionCommand{
		SubscriptionToken: token,
		Format:            format,
		NodeMode:          c.DefaultQuery("mode", usecases.NodeModeAll),
	}

	result, err := h.generateSubscriptionUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	c.Header("Content-Type", result.ContentType)
	c.Header("Subscription-Userinfo", h.formatUserInfo(result.UserInfo))

	// Set filename header based on format
	if format == "clash" {
		c.Header("Content-Disposition", "attachment; filename=clash.yaml")
	}

	c.String(http.StatusOK, result.Content)
}

// formatUserInfo formats SubscriptionUserInfo into the standard Subscription-Userinfo header format.
func (h *SubscriptionHandler) formatUserInfo(userInfo *usecases.SubscriptionUserInfo) string {
	if userInfo == nil {
		return "upload=0; download=0; total=0; expire=0"
	}
	return fmt.Sprintf("upload=%d; download=%d; total=%d; expire=%d",
		userInfo.Upload,
		userInfo.Download,
		userInfo.Total,
		userInfo.Expire,
	)
}

// GetClashSubscription handles GET /s/:token/clash
func (h *SubscriptionHandler) GetClashSubscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.ErrorResponseWithError(c, errors.NewValidationError("Subscription token is required"))
		return
	}

	cmd := usecases.GenerateSubscriptionCommand{
		SubscriptionToken: token,
		Format:            "clash",
		NodeMode:          c.DefaultQuery("mode", usecases.NodeModeAll),
	}

	result, err := h.generateSubscriptionUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	c.Header("Content-Type", result.ContentType)
	c.Header("Subscription-Userinfo", h.formatUserInfo(result.UserInfo))
	c.Header("Content-Disposition", "attachment; filename=clash.yaml")
	c.String(http.StatusOK, result.Content)
}

// GetV2RaySubscription handles GET /s/:token/v2ray
func (h *SubscriptionHandler) GetV2RaySubscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.ErrorResponseWithError(c, errors.NewValidationError("Subscription token is required"))
		return
	}

	cmd := usecases.GenerateSubscriptionCommand{
		SubscriptionToken: token,
		Format:            "v2ray",
		NodeMode:          c.DefaultQuery("mode", usecases.NodeModeAll),
	}

	result, err := h.generateSubscriptionUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	c.Header("Content-Type", result.ContentType)
	c.Header("Subscription-Userinfo", h.formatUserInfo(result.UserInfo))
	c.String(http.StatusOK, result.Content)
}

// GetSIP008Subscription handles GET /s/:token/sip008
func (h *SubscriptionHandler) GetSIP008Subscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.ErrorResponseWithError(c, errors.NewValidationError("Subscription token is required"))
		return
	}

	cmd := usecases.GenerateSubscriptionCommand{
		SubscriptionToken: token,
		Format:            "sip008",
		NodeMode:          c.DefaultQuery("mode", usecases.NodeModeAll),
	}

	result, err := h.generateSubscriptionUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	c.Header("Content-Type", result.ContentType)
	c.Header("Subscription-Userinfo", h.formatUserInfo(result.UserInfo))
	c.String(http.StatusOK, result.Content)
}

// GetSurgeSubscription handles GET /s/:token/surge
func (h *SubscriptionHandler) GetSurgeSubscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.ErrorResponseWithError(c, errors.NewValidationError("Subscription token is required"))
		return
	}

	cmd := usecases.GenerateSubscriptionCommand{
		SubscriptionToken: token,
		Format:            "surge",
		NodeMode:          c.DefaultQuery("mode", usecases.NodeModeAll),
	}

	result, err := h.generateSubscriptionUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	c.Header("Content-Type", result.ContentType)
	c.Header("Subscription-Userinfo", h.formatUserInfo(result.UserInfo))
	c.String(http.StatusOK, result.Content)
}

// detectFormatFromUserAgent detects subscription format from User-Agent header
func detectFormatFromUserAgent(userAgent string) string {
	if userAgent == "" {
		return "base64"
	}

	ua := strings.ToLower(userAgent)

	// Clash clients
	if strings.Contains(ua, "clash") {
		return "clash"
	}

	// Surge clients
	if strings.Contains(ua, "surge") {
		return "surge"
	}

	// Quantumult clients
	if strings.Contains(ua, "quantumult") {
		return "base64"
	}

	// Shadowrocket clients
	if strings.Contains(ua, "shadowrocket") {
		return "base64"
	}

	// V2RayN/V2RayNG clients - use base64 format (supports all protocol URIs)
	// These are general-purpose clients that parse base64-encoded URI lists
	// (vmess://, vless://, trojan://, ss://, etc.)
	if strings.Contains(ua, "v2rayn") || strings.Contains(ua, "v2rayng") {
		return "base64"
	}

	// Default to base64 format for unknown clients
	// Note: "v2ray" format (JSON) is only for Shadowsocks-only clients,
	// accessible via explicit /s/:token/v2ray endpoint
	return "base64"
}
