// Package rule provides HTTP handlers for forward rule management.
package rule

import (
	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/services"
	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// Handler handles HTTP requests for forward rules.
type Handler struct {
	createRuleUC   *usecases.CreateForwardRuleUseCase
	getRuleUC      *usecases.GetForwardRuleUseCase
	updateRuleUC   *usecases.UpdateForwardRuleUseCase
	deleteRuleUC   *usecases.DeleteForwardRuleUseCase
	listRulesUC    *usecases.ListForwardRulesUseCase
	enableRuleUC   *usecases.EnableForwardRuleUseCase
	disableRuleUC  *usecases.DisableForwardRuleUseCase
	resetTrafficUC *usecases.ResetForwardRuleTrafficUseCase
	reorderRulesUC *usecases.ReorderForwardRulesUseCase
	probeService   *services.ProbeService
	logger         logger.Interface
}

// NewHandler creates a new Handler.
func NewHandler(
	createRuleUC *usecases.CreateForwardRuleUseCase,
	getRuleUC *usecases.GetForwardRuleUseCase,
	updateRuleUC *usecases.UpdateForwardRuleUseCase,
	deleteRuleUC *usecases.DeleteForwardRuleUseCase,
	listRulesUC *usecases.ListForwardRulesUseCase,
	enableRuleUC *usecases.EnableForwardRuleUseCase,
	disableRuleUC *usecases.DisableForwardRuleUseCase,
	resetTrafficUC *usecases.ResetForwardRuleTrafficUseCase,
	reorderRulesUC *usecases.ReorderForwardRulesUseCase,
	probeService *services.ProbeService,
) *Handler {
	return &Handler{
		createRuleUC:   createRuleUC,
		getRuleUC:      getRuleUC,
		updateRuleUC:   updateRuleUC,
		deleteRuleUC:   deleteRuleUC,
		listRulesUC:    listRulesUC,
		enableRuleUC:   enableRuleUC,
		disableRuleUC:  disableRuleUC,
		resetTrafficUC: resetTrafficUC,
		reorderRulesUC: reorderRulesUC,
		probeService:   probeService,
		logger:         logger.NewLogger(),
	}
}

// CreateForwardRuleRequest represents a request to create a forward rule.
// Required fields by rule type:
// - direct: agent_id, listen_port, (target_address+target_port OR target_node_id)
// - entry: agent_id, exit_agent_id, listen_port, (target_address+target_port OR target_node_id)
// - chain: agent_id, chain_agent_ids, listen_port, (target_address+target_port OR target_node_id)
// - direct_chain: agent_id, chain_agent_ids, chain_port_config, (target_address+target_port OR target_node_id)
type CreateForwardRuleRequest struct {
	AgentID           string            `json:"agent_id" binding:"required" example:"fa_xK9mP2vL3nQ"`
	RuleType          string            `json:"rule_type" binding:"required,oneof=direct entry chain direct_chain" example:"direct"`
	ExitAgentID       string            `json:"exit_agent_id,omitempty" example:"fa_yL8nQ3wM4oR"`
	ChainAgentIDs     []string          `json:"chain_agent_ids,omitempty" example:"[\"fa_aaa\",\"fa_bbb\"]"`
	ChainPortConfig   map[string]uint16 `json:"chain_port_config,omitempty" example:"{\"fa_xK9mP2vL3nQ\":8080,\"fa_yL8nQ3wM4oR\":9090}"`
	TunnelHops        *int              `json:"tunnel_hops,omitempty" binding:"omitempty,gte=0,lte=10" example:"2"`
	TunnelType        string            `json:"tunnel_type,omitempty" binding:"omitempty,oneof=ws tls" example:"ws"`
	Name              string            `json:"name" binding:"required" example:"MySQL-Forward"`
	ListenPort        uint16            `json:"listen_port,omitempty" example:"13306"`
	TargetAddress     string            `json:"target_address,omitempty" example:"192.168.1.100"`
	TargetPort        uint16            `json:"target_port,omitempty" example:"3306"`
	TargetNodeID      string            `json:"target_node_id,omitempty" example:"node_xK9mP2vL3nQ"`
	BindIP            string            `json:"bind_ip,omitempty" example:"192.168.1.1"`
	IPVersion         string            `json:"ip_version,omitempty" binding:"omitempty,oneof=auto ipv4 ipv6" example:"auto"`
	Protocol          string            `json:"protocol" binding:"required,oneof=tcp udp both" example:"tcp"`
	TrafficMultiplier *float64          `json:"traffic_multiplier,omitempty" binding:"omitempty,gte=0,lte=1000000" example:"1.5"`
	SortOrder         *int              `json:"sort_order,omitempty" binding:"omitempty,gte=0" example:"100"`
	Remark            string            `json:"remark,omitempty" example:"Forward to internal MySQL server"`
}

// UpdateForwardRuleRequest represents a request to update a forward rule.
type UpdateForwardRuleRequest struct {
	Name              *string           `json:"name,omitempty" example:"MySQL-Forward-Updated"`
	AgentID           *string           `json:"agent_id,omitempty" example:"fa_xK9mP2vL3nQ"`
	ExitAgentID       *string           `json:"exit_agent_id,omitempty" example:"fa_yL8nQ3wM4oR"`
	ChainAgentIDs     []string          `json:"chain_agent_ids,omitempty" example:"[\"fa_aaa\",\"fa_bbb\"]"`
	ChainPortConfig   map[string]uint16 `json:"chain_port_config,omitempty" example:"{\"fa_xK9mP2vL3nQ\":8080,\"fa_yL8nQ3wM4oR\":9090}"`
	TunnelHops        *int              `json:"tunnel_hops,omitempty" binding:"omitempty,gte=0,lte=10" example:"2"`
	TunnelType        *string           `json:"tunnel_type,omitempty" binding:"omitempty,oneof=ws tls" example:"ws"`
	ListenPort        *uint16           `json:"listen_port,omitempty" example:"13307"`
	TargetAddress     *string           `json:"target_address,omitempty" example:"192.168.1.101"`
	TargetPort        *uint16           `json:"target_port,omitempty" example:"3307"`
	TargetNodeID      *string           `json:"target_node_id,omitempty" example:"node_xK9mP2vL3nQ"`
	BindIP            *string           `json:"bind_ip,omitempty" example:"192.168.1.1"`
	IPVersion         *string           `json:"ip_version,omitempty" binding:"omitempty,oneof=auto ipv4 ipv6" example:"auto"`
	Protocol          *string           `json:"protocol,omitempty" binding:"omitempty,oneof=tcp udp both" example:"tcp"`
	TrafficMultiplier *float64          `json:"traffic_multiplier,omitempty" binding:"omitempty,gte=0,lte=1000000" example:"1.5"`
	SortOrder         *int              `json:"sort_order,omitempty" example:"100"`
	Remark            *string           `json:"remark,omitempty" example:"Updated remark"`
}

// UpdateStatusRequest represents a request to update forward rule status.
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=enabled disabled" example:"enabled"`
}

// ProbeRuleRequest represents the request body for probing a rule.
type ProbeRuleRequest struct {
	IPVersion string `json:"ip_version"` // optional: auto, ipv4, ipv6
}

// ReorderForwardRulesRequest represents a request to reorder forward rules.
type ReorderForwardRulesRequest struct {
	RuleOrders []ForwardRuleOrder `json:"rule_orders" binding:"required,min=1,dive"`
}

// ForwardRuleOrder represents a single rule's sort order.
type ForwardRuleOrder struct {
	RuleID    string `json:"rule_id" binding:"required" example:"fr_xK9mP2vL3nQ"`
	SortOrder int    `json:"sort_order" binding:"gte=0" example:"100"`
}

// parseRuleShortID validates a prefixed rule ID and returns the SID (e.g., "fr_xK9mP2vL3nQ").
// Note: Despite the name, this returns the full SID (with prefix) as stored in the database.
func parseRuleShortID(c *gin.Context) (string, error) {
	prefixedID := c.Param("id")
	if prefixedID == "" {
		return "", errors.NewValidationError("forward rule ID is required")
	}

	// Validate the prefix is correct, but return the full prefixed ID
	// because the database stores SIDs with prefix (e.g., "fr_xxx")
	if err := id.ValidatePrefix(prefixedID, id.PrefixForwardRule); err != nil {
		return "", errors.NewValidationError("invalid forward rule ID format, expected fr_xxxxx")
	}

	return prefixedID, nil
}
