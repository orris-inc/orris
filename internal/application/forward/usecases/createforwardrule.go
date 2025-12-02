package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/forward"
	vo "orris/internal/domain/forward/value_objects"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// CreateForwardRuleCommand represents the input for creating a forward rule.
type CreateForwardRuleCommand struct {
	AgentID       uint
	NextAgentID   uint // 0=direct forward, >0=chain forward to next agent
	Name          string
	ListenPort    uint16
	TargetAddress string // required when NextAgentID=0
	TargetPort    uint16 // required when NextAgentID=0
	Protocol      string
	Remark        string
}

// CreateForwardRuleResult represents the output of creating a forward rule.
type CreateForwardRuleResult struct {
	ID            uint   `json:"id"`
	AgentID       uint   `json:"agent_id"`
	NextAgentID   uint   `json:"next_agent_id"`
	IsChain       bool   `json:"is_chain"`
	Name          string `json:"name"`
	ListenPort    uint16 `json:"listen_port"`
	TargetAddress string `json:"target_address,omitempty"`
	TargetPort    uint16 `json:"target_port,omitempty"`
	Protocol      string `json:"protocol"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

// CreateForwardRuleUseCase handles forward rule creation.
type CreateForwardRuleUseCase struct {
	repo   forward.Repository
	logger logger.Interface
}

// NewCreateForwardRuleUseCase creates a new CreateForwardRuleUseCase.
func NewCreateForwardRuleUseCase(
	repo forward.Repository,
	logger logger.Interface,
) *CreateForwardRuleUseCase {
	return &CreateForwardRuleUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute creates a new forward rule.
func (uc *CreateForwardRuleUseCase) Execute(ctx context.Context, cmd CreateForwardRuleCommand) (*CreateForwardRuleResult, error) {
	uc.logger.Infow("executing create forward rule use case", "name", cmd.Name, "listen_port", cmd.ListenPort)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid create forward rule command", "error", err)
		return nil, err
	}

	// Check if listen port is already in use
	exists, err := uc.repo.ExistsByListenPort(ctx, cmd.ListenPort)
	if err != nil {
		uc.logger.Errorw("failed to check existing forward rule", "port", cmd.ListenPort, "error", err)
		return nil, fmt.Errorf("failed to check existing rule: %w", err)
	}
	if exists {
		uc.logger.Warnw("listen port already in use", "port", cmd.ListenPort)
		return nil, errors.NewConflictError("listen port is already in use", fmt.Sprintf("%d", cmd.ListenPort))
	}

	// Create domain entity
	protocol := vo.ForwardProtocol(cmd.Protocol)
	rule, err := forward.NewForwardRule(
		cmd.AgentID,
		cmd.NextAgentID,
		cmd.Name,
		cmd.ListenPort,
		cmd.TargetAddress,
		cmd.TargetPort,
		protocol,
		cmd.Remark,
	)
	if err != nil {
		uc.logger.Errorw("failed to create forward rule entity", "error", err)
		return nil, fmt.Errorf("failed to create forward rule: %w", err)
	}

	// Persist
	if err := uc.repo.Create(ctx, rule); err != nil {
		uc.logger.Errorw("failed to persist forward rule", "error", err)
		return nil, fmt.Errorf("failed to save forward rule: %w", err)
	}

	result := &CreateForwardRuleResult{
		ID:            rule.ID(),
		AgentID:       rule.AgentID(),
		NextAgentID:   rule.NextAgentID(),
		IsChain:       rule.IsChainForward(),
		Name:          rule.Name(),
		ListenPort:    rule.ListenPort(),
		TargetAddress: rule.TargetAddress(),
		TargetPort:    rule.TargetPort(),
		Protocol:      rule.Protocol().String(),
		Status:        rule.Status().String(),
		CreatedAt:     rule.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	uc.logger.Infow("forward rule created successfully", "id", result.ID, "name", cmd.Name)
	return result, nil
}

func (uc *CreateForwardRuleUseCase) validateCommand(cmd CreateForwardRuleCommand) error {
	if cmd.AgentID == 0 {
		return errors.NewValidationError("agent_id is required")
	}
	if cmd.Name == "" {
		return errors.NewValidationError("name is required")
	}
	if cmd.ListenPort == 0 {
		return errors.NewValidationError("listen port is required")
	}
	// For direct forward (NextAgentID=0), target is required
	if cmd.NextAgentID == 0 {
		if cmd.TargetAddress == "" {
			return errors.NewValidationError("target_address is required for direct forward")
		}
		if cmd.TargetPort == 0 {
			return errors.NewValidationError("target_port is required for direct forward")
		}
	}
	if cmd.Protocol == "" {
		return errors.NewValidationError("protocol is required")
	}

	protocol := vo.ForwardProtocol(cmd.Protocol)
	if !protocol.IsValid() {
		return errors.NewValidationError(fmt.Sprintf("invalid protocol: %s, must be tcp, udp or both", cmd.Protocol))
	}

	return nil
}
