// Package admin provides HTTP handlers for administrative operations.
package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/admin/dto"
	"github.com/orris-inc/orris/internal/application/admin/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// TrafficStatsHandler handles admin traffic statistics operations
type TrafficStatsHandler struct {
	overviewUseCase            *usecases.GetTrafficOverviewUseCase
	userTrafficUseCase         *usecases.GetUserTrafficStatsUseCase
	subscriptionTrafficUseCase *usecases.GetSubscriptionTrafficStatsUseCase
	nodeTrafficUseCase         *usecases.GetAdminNodeTrafficStatsUseCase
	rankingUseCase             *usecases.GetTrafficRankingUseCase
	trendUseCase               *usecases.GetTrafficTrendUseCase
	logger                     logger.Interface
}

// NewTrafficStatsHandler creates a new admin traffic stats handler
func NewTrafficStatsHandler(
	overviewUC *usecases.GetTrafficOverviewUseCase,
	userTrafficUC *usecases.GetUserTrafficStatsUseCase,
	subscriptionTrafficUC *usecases.GetSubscriptionTrafficStatsUseCase,
	nodeTrafficUC *usecases.GetAdminNodeTrafficStatsUseCase,
	rankingUC *usecases.GetTrafficRankingUseCase,
	trendUC *usecases.GetTrafficTrendUseCase,
	logger logger.Interface,
) *TrafficStatsHandler {
	return &TrafficStatsHandler{
		overviewUseCase:            overviewUC,
		userTrafficUseCase:         userTrafficUC,
		subscriptionTrafficUseCase: subscriptionTrafficUC,
		nodeTrafficUseCase:         nodeTrafficUC,
		rankingUseCase:             rankingUC,
		trendUseCase:               trendUC,
		logger:                     logger,
	}
}

// GetOverview handles GET /admin/traffic-stats/overview
func (h *TrafficStatsHandler) GetOverview(c *gin.Context) {
	var req dto.TrafficStatsQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warnw("invalid request query for traffic overview", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetTrafficOverviewQuery{
		From:         req.From,
		To:           req.To,
		ResourceType: req.ResourceType,
	}

	result, err := h.overviewUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get traffic overview", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Traffic overview retrieved successfully", result)
}

// GetUserStats handles GET /admin/traffic-stats/users
func (h *TrafficStatsHandler) GetUserStats(c *gin.Context) {
	var req dto.TrafficStatsQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warnw("invalid request query for user traffic stats", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetUserTrafficStatsQuery{
		From:         req.From,
		To:           req.To,
		ResourceType: req.ResourceType,
		Page:         req.Page,
		PageSize:     req.PageSize,
	}

	result, err := h.userTrafficUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get user traffic stats", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Items, result.Total, result.Page, result.PageSize)
}

// GetSubscriptionStats handles GET /admin/traffic-stats/subscriptions
func (h *TrafficStatsHandler) GetSubscriptionStats(c *gin.Context) {
	var req dto.TrafficStatsQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warnw("invalid request query for subscription traffic stats", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetSubscriptionTrafficStatsQuery{
		From:         req.From,
		To:           req.To,
		ResourceType: req.ResourceType,
		Page:         req.Page,
		PageSize:     req.PageSize,
	}

	result, err := h.subscriptionTrafficUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get subscription traffic stats", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Items, result.Total, result.Page, result.PageSize)
}

// GetNodeStats handles GET /admin/traffic-stats/nodes
func (h *TrafficStatsHandler) GetNodeStats(c *gin.Context) {
	var req dto.TrafficStatsQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warnw("invalid request query for node traffic stats", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetAdminNodeTrafficStatsQuery{
		From:     req.From,
		To:       req.To,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	result, err := h.nodeTrafficUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get node traffic stats", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Items, result.Total, result.Page, result.PageSize)
}

// GetUserRanking handles GET /admin/traffic-stats/ranking/users
func (h *TrafficStatsHandler) GetUserRanking(c *gin.Context) {
	var req dto.TrafficRankingRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warnw("invalid request query for user traffic ranking", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetTrafficRankingQuery{
		From:         req.From,
		To:           req.To,
		ResourceType: req.ResourceType,
		Limit:        req.Limit,
		RankingType:  "user",
	}

	result, err := h.rankingUseCase.ExecuteUserRanking(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get user traffic ranking", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "User traffic ranking retrieved successfully", result)
}

// GetSubscriptionRanking handles GET /admin/traffic-stats/ranking/subscriptions
func (h *TrafficStatsHandler) GetSubscriptionRanking(c *gin.Context) {
	var req dto.TrafficRankingRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warnw("invalid request query for subscription traffic ranking", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetTrafficRankingQuery{
		From:         req.From,
		To:           req.To,
		ResourceType: req.ResourceType,
		Limit:        req.Limit,
		RankingType:  "subscription",
	}

	result, err := h.rankingUseCase.ExecuteSubscriptionRanking(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get subscription traffic ranking", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription traffic ranking retrieved successfully", result)
}

// GetTrend handles GET /admin/traffic-stats/trend
func (h *TrafficStatsHandler) GetTrend(c *gin.Context) {
	var req dto.TrafficTrendRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warnw("invalid request query for traffic trend", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetTrafficTrendQuery{
		From:         req.From,
		To:           req.To,
		ResourceType: req.ResourceType,
		Granularity:  req.Granularity,
	}

	result, err := h.trendUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get traffic trend", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Traffic trend retrieved successfully", result)
}
