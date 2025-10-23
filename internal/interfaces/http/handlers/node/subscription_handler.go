package node

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"orris/internal/application/node/usecases"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
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

// GetSubscription handles GET /sub/:token
// @Summary Get subscription
// @Description Get base64 encoded subscription by token
// @Tags subscriptions
// @Accept json
// @Produce text/plain
// @Param token path string true "Subscription token"
// @Success 200 {string} string "Base64 encoded subscription"
// @Failure 400 {object} utils.APIResponse "Invalid token"
// @Failure 404 {object} utils.APIResponse "Subscription not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /sub/{token} [get]
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.ErrorResponseWithError(c, errors.NewValidationError("Subscription token is required"))
		return
	}

	cmd := usecases.GenerateSubscriptionCommand{
		SubscriptionToken: token,
		Format:            "base64",
	}

	result, err := h.generateSubscriptionUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	c.Header("Content-Type", result.ContentType)
	c.Header("Subscription-Userinfo", "upload=0; download=0; total=0; expire=0")
	c.String(http.StatusOK, result.Content)
}

// GetClashSubscription handles GET /sub/:token/clash
// @Summary Get Clash subscription
// @Description Get Clash format subscription by token
// @Tags subscriptions
// @Accept json
// @Produce application/yaml
// @Param token path string true "Subscription token"
// @Success 200 {string} string "Clash YAML configuration"
// @Failure 400 {object} utils.APIResponse "Invalid token"
// @Failure 404 {object} utils.APIResponse "Subscription not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /sub/{token}/clash [get]
func (h *SubscriptionHandler) GetClashSubscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.ErrorResponseWithError(c, errors.NewValidationError("Subscription token is required"))
		return
	}

	cmd := usecases.GenerateSubscriptionCommand{
		SubscriptionToken: token,
		Format:            "clash",
	}

	result, err := h.generateSubscriptionUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	c.Header("Content-Type", result.ContentType)
	c.Header("Content-Disposition", "attachment; filename=clash.yaml")
	c.String(http.StatusOK, result.Content)
}

// GetV2RaySubscription handles GET /sub/:token/v2ray
// @Summary Get V2Ray subscription
// @Description Get V2Ray format subscription by token
// @Tags subscriptions
// @Accept json
// @Produce application/json
// @Param token path string true "Subscription token"
// @Success 200 {string} string "V2Ray JSON configuration"
// @Failure 400 {object} utils.APIResponse "Invalid token"
// @Failure 404 {object} utils.APIResponse "Subscription not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /sub/{token}/v2ray [get]
func (h *SubscriptionHandler) GetV2RaySubscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.ErrorResponseWithError(c, errors.NewValidationError("Subscription token is required"))
		return
	}

	cmd := usecases.GenerateSubscriptionCommand{
		SubscriptionToken: token,
		Format:            "v2ray",
	}

	result, err := h.generateSubscriptionUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	c.Header("Content-Type", result.ContentType)
	c.String(http.StatusOK, result.Content)
}

// GetSIP008Subscription handles GET /sub/:token/sip008
// @Summary Get SIP008 subscription
// @Description Get SIP008 format subscription by token (Shadowsocks)
// @Tags subscriptions
// @Accept json
// @Produce application/json
// @Param token path string true "Subscription token"
// @Success 200 {string} string "SIP008 JSON configuration"
// @Failure 400 {object} utils.APIResponse "Invalid token"
// @Failure 404 {object} utils.APIResponse "Subscription not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /sub/{token}/sip008 [get]
func (h *SubscriptionHandler) GetSIP008Subscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.ErrorResponseWithError(c, errors.NewValidationError("Subscription token is required"))
		return
	}

	cmd := usecases.GenerateSubscriptionCommand{
		SubscriptionToken: token,
		Format:            "sip008",
	}

	result, err := h.generateSubscriptionUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	c.Header("Content-Type", result.ContentType)
	c.String(http.StatusOK, result.Content)
}

// GetSurgeSubscription handles GET /sub/:token/surge
// @Summary Get Surge subscription
// @Description Get Surge format subscription by token
// @Tags subscriptions
// @Accept json
// @Produce text/plain
// @Param token path string true "Subscription token"
// @Success 200 {string} string "Surge configuration"
// @Failure 400 {object} utils.APIResponse "Invalid token"
// @Failure 404 {object} utils.APIResponse "Subscription not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /sub/{token}/surge [get]
func (h *SubscriptionHandler) GetSurgeSubscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.ErrorResponseWithError(c, errors.NewValidationError("Subscription token is required"))
		return
	}

	cmd := usecases.GenerateSubscriptionCommand{
		SubscriptionToken: token,
		Format:            "surge",
	}

	result, err := h.generateSubscriptionUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	c.Header("Content-Type", result.ContentType)
	c.String(http.StatusOK, result.Content)
}
