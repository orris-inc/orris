// Package forward provides HTTP handlers for forward rule management.
package forward

import (
	"github.com/orris-inc/orris/internal/application/forward/services"
	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ForwardHandler handles HTTP requests for forward rules.
type ForwardHandler struct {
	createRuleUC   *usecases.CreateForwardRuleUseCase
	getRuleUC      *usecases.GetForwardRuleUseCase
	updateRuleUC   *usecases.UpdateForwardRuleUseCase
	deleteRuleUC   *usecases.DeleteForwardRuleUseCase
	listRulesUC    *usecases.ListForwardRulesUseCase
	enableRuleUC   *usecases.EnableForwardRuleUseCase
	disableRuleUC  *usecases.DisableForwardRuleUseCase
	resetTrafficUC *usecases.ResetForwardRuleTrafficUseCase
	probeService   *services.ProbeService
	logger         logger.Interface
}

// NewForwardHandler creates a new ForwardHandler.
func NewForwardHandler(
	createRuleUC *usecases.CreateForwardRuleUseCase,
	getRuleUC *usecases.GetForwardRuleUseCase,
	updateRuleUC *usecases.UpdateForwardRuleUseCase,
	deleteRuleUC *usecases.DeleteForwardRuleUseCase,
	listRulesUC *usecases.ListForwardRulesUseCase,
	enableRuleUC *usecases.EnableForwardRuleUseCase,
	disableRuleUC *usecases.DisableForwardRuleUseCase,
	resetTrafficUC *usecases.ResetForwardRuleTrafficUseCase,
	probeService *services.ProbeService,
) *ForwardHandler {
	return &ForwardHandler{
		createRuleUC:   createRuleUC,
		getRuleUC:      getRuleUC,
		updateRuleUC:   updateRuleUC,
		deleteRuleUC:   deleteRuleUC,
		listRulesUC:    listRulesUC,
		enableRuleUC:   enableRuleUC,
		disableRuleUC:  disableRuleUC,
		resetTrafficUC: resetTrafficUC,
		probeService:   probeService,
		logger:         logger.NewLogger(),
	}
}
