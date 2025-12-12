// Package services provides application services for the forward domain.
package services

import (
	"context"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	probeTimeout        = 10 * time.Second
	probeSessionTimeout = 30 * time.Second
)

// ProbeService handles probe operations for forward rules.
// It implements agent.MessageHandler interface.
type ProbeService struct {
	repo          forward.Repository
	agentRepo     forward.AgentRepository
	nodeRepo      node.NodeRepository
	statusQuerier ProbeStatusQuerier

	// Hub interface for sending messages
	hub ProbeHub

	// Pending probes: map[taskID]chan *dto.ProbeTaskResult
	pendingProbes   map[string]chan *dto.ProbeTaskResult
	pendingProbesMu sync.RWMutex

	logger logger.Interface
}

// ProbeHub defines the interface for sending messages through the hub.
type ProbeHub interface {
	IsAgentOnline(agentID uint) bool
	SendMessageToAgent(agentID uint, msg *dto.HubMessage) error
}

// ProbeStatusQuerier queries agent status for ws_listen_port.
type ProbeStatusQuerier interface {
	GetStatus(ctx context.Context, agentID uint) (*dto.AgentStatusDTO, error)
}

// NewProbeService creates a new ProbeService.
func NewProbeService(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	statusQuerier ProbeStatusQuerier,
	hub ProbeHub,
	log logger.Interface,
) *ProbeService {
	return &ProbeService{
		repo:          repo,
		agentRepo:     agentRepo,
		nodeRepo:      nodeRepo,
		statusQuerier: statusQuerier,
		hub:           hub,
		pendingProbes: make(map[string]chan *dto.ProbeTaskResult),
		logger:        log,
	}
}

// String implements fmt.Stringer for logging purposes.
func (s *ProbeService) String() string {
	return "ProbeService"
}

// ProbeRuleByShortID probes a single forward rule by short ID and returns the latency results.
// ipVersionOverride allows overriding the rule's IP version for this probe only.
func (s *ProbeService) ProbeRuleByShortID(ctx context.Context, shortID string, ipVersionOverride string) (*dto.RuleProbeResponse, error) {
	// Get the rule
	rule, err := s.repo.GetByShortID(ctx, shortID)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, forward.ErrRuleNotFound
	}
	return s.probeRule(ctx, rule, ipVersionOverride)
}

// ProbeRule probes a single forward rule and returns the latency results.
// ipVersionOverride allows overriding the rule's IP version for this probe only.
// Deprecated: Use ProbeRuleByShortID instead for external API.
func (s *ProbeService) ProbeRule(ctx context.Context, ruleID uint, ipVersionOverride string) (*dto.RuleProbeResponse, error) {
	// Get the rule
	rule, err := s.repo.GetByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, forward.ErrRuleNotFound
	}
	return s.probeRule(ctx, rule, ipVersionOverride)
}

// probeRule is the internal implementation for probing a rule.
func (s *ProbeService) probeRule(ctx context.Context, rule *forward.ForwardRule, ipVersionOverride string) (*dto.RuleProbeResponse, error) {

	// Determine IP version to use (override or rule's default)
	ipVersion := rule.IPVersion()
	if ipVersionOverride != "" {
		ipVersion = vo.IPVersion(ipVersionOverride)
		if !ipVersion.IsValid() {
			return nil, forward.ErrInvalidIPVersion
		}
	}

	ruleType := rule.RuleType().String()
	response := &dto.RuleProbeResponse{
		RuleID:   id.FormatForwardRuleID(rule.ShortID()),
		RuleType: ruleType,
	}

	switch ruleType {
	case "direct":
		return s.probeDirectRule(ctx, rule, ipVersion, response)
	case "entry":
		return s.probeEntryRule(ctx, rule, ipVersion, response)
	case "chain":
		return s.probeChainRule(ctx, rule, ipVersion, response)
	case "direct_chain":
		return s.probeDirectChainRule(ctx, rule, ipVersion, response)
	case "exit":
		// Exit rules are probed through entry rules
		response.Error = "exit rules cannot be probed directly"
		return response, nil
	default:
		response.Error = "unknown rule type"
		return response, nil
	}
}
