package node

import (
	"context"
	"net/http"

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

// GetSubscription handles GET /s/:token
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.ErrorResponseWithError(c, errors.NewValidationError("Subscription token is required"))
		return
	}

	cmd := usecases.GenerateSubscriptionCommand{
		SubscriptionToken: token,
		Format:            "base64",
		NodeMode:          c.DefaultQuery("mode", usecases.NodeModeAll),
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
	c.String(http.StatusOK, result.Content)
}
