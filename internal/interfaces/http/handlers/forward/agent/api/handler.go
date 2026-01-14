// Package api provides HTTP handlers for forward agent REST API.
package api

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/adapters"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// Handler handles RESTful agent API requests for forward client
type Handler struct {
	getEnabledRulesUC      usecases.GetEnabledRulesForAgentExecutor
	refreshRuleUC          usecases.RefreshRuleForAgentExecutor
	reportStatusUC         *usecases.ReportAgentStatusUseCase
	reportRuleSyncStatusUC *usecases.ReportRuleSyncStatusUseCase
	repo                   forward.Repository      // kept for other methods
	agentRepo              forward.AgentRepository // kept for other methods
	nodeRepo               node.NodeRepository     // kept for other methods
	statusQuerier          usecases.AgentStatusQuerier
	agentTokenService      *auth.AgentTokenService
	trafficRecorder        adapters.ForwardTrafficRecorder
	logger                 logger.Interface
	ruleConverter          *dto.AgentRuleConverter // kept for SetNodeRepo
}

// NewHandler creates a new Handler instance
func NewHandler(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	reportStatusUC *usecases.ReportAgentStatusUseCase,
	reportRuleSyncStatusUC *usecases.ReportRuleSyncStatusUseCase,
	statusQuerier usecases.AgentStatusQuerier,
	tokenSigningSecret string,
	trafficRecorder adapters.ForwardTrafficRecorder,
	logger logger.Interface,
) *Handler {
	agentTokenService := auth.NewAgentTokenService(tokenSigningSecret)

	// Create AgentRuleConverter
	ruleConverter := dto.NewAgentRuleConverter(
		agentRepo,
		nodeRepo,
		statusQuerier,
		agentTokenService,
		logger,
	)

	// Create UseCases
	getEnabledRulesUC := usecases.NewGetEnabledRulesForAgentUseCase(
		repo,
		agentRepo,
		ruleConverter,
		logger,
	)

	refreshRuleUC := usecases.NewRefreshRuleForAgentUseCase(
		repo,
		ruleConverter,
		logger,
	)

	return &Handler{
		getEnabledRulesUC:      getEnabledRulesUC,
		refreshRuleUC:          refreshRuleUC,
		reportStatusUC:         reportStatusUC,
		reportRuleSyncStatusUC: reportRuleSyncStatusUC,
		repo:                   repo,
		agentRepo:              agentRepo,
		nodeRepo:               nodeRepo,
		statusQuerier:          statusQuerier,
		agentTokenService:      agentTokenService,
		trafficRecorder:        trafficRecorder,
		logger:                 logger,
		ruleConverter:          ruleConverter,
	}
}

// SetNodeRepo sets the node repository for circular dependency handling.
// This allows the handler to be created before nodeRepo is available.
func (h *Handler) SetNodeRepo(nodeRepo node.NodeRepository) {
	h.nodeRepo = nodeRepo
	if h.ruleConverter != nil {
		h.ruleConverter.SetNodeRepo(nodeRepo)
	}
}

// getAuthenticatedAgentID extracts the authenticated forward agent ID from context.
// Returns the agent ID or an error if not found.
func (h *Handler) getAuthenticatedAgentID(c *gin.Context) (uint, error) {
	agentID, exists := c.Get("forward_agent_id")
	if !exists {
		return 0, fmt.Errorf("forward_agent_id not found in context")
	}
	id, ok := agentID.(uint)
	if !ok {
		return 0, fmt.Errorf("invalid forward_agent_id type in context")
	}
	return id, nil
}
