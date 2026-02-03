package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// CreateForwardAgentCommand represents the input for creating a forward agent.
type CreateForwardAgentCommand struct {
	Name             string
	PublicAddress    string
	TunnelAddress    string
	Remark           string
	GroupSID         string   // Resource group SID to associate with (empty means no association)
	AllowedPortRange string   // Port range string (e.g., "80,443,8000-9000"), empty means all ports allowed
	BlockedProtocols []string // Protocols to block (e.g., ["socks5", "http_connect"]), empty means no blocking
	SortOrder        *int     // Custom sort order for UI display (nil: use default 0, non-nil: set explicitly)
}

// CreateForwardAgentResult represents the output of creating a forward agent.
type CreateForwardAgentResult struct {
	ID            string `json:"id"` // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	Name          string `json:"name"`
	PublicAddress string `json:"public_address"`
	TunnelAddress string `json:"tunnel_address,omitempty"`
	Token         string `json:"token"`
	Status        string `json:"status"`
	Remark        string `json:"remark"`
	CreatedAt     string `json:"created_at"`
}

// CreateForwardAgentUseCase handles forward agent creation.
type CreateForwardAgentUseCase struct {
	repo              forward.AgentRepository
	resourceGroupRepo resource.Repository
	tokenGen          AgentTokenGenerator
	logger            logger.Interface
}

// NewCreateForwardAgentUseCase creates a new CreateForwardAgentUseCase.
func NewCreateForwardAgentUseCase(
	repo forward.AgentRepository,
	resourceGroupRepo resource.Repository,
	tokenGen AgentTokenGenerator,
	logger logger.Interface,
) *CreateForwardAgentUseCase {
	return &CreateForwardAgentUseCase{
		repo:              repo,
		resourceGroupRepo: resourceGroupRepo,
		tokenGen:          tokenGen,
		logger:            logger,
	}
}

// Execute creates a new forward agent.
func (uc *CreateForwardAgentUseCase) Execute(ctx context.Context, cmd CreateForwardAgentCommand) (*CreateForwardAgentResult, error) {
	uc.logger.Infow("executing create forward agent use case", "name", cmd.Name)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid create forward agent command", "error", err)
		return nil, err
	}

	// Check if agent name already exists
	exists, err := uc.repo.ExistsByName(ctx, cmd.Name)
	if err != nil {
		uc.logger.Errorw("failed to check existing forward agent", "name", cmd.Name, "error", err)
		return nil, fmt.Errorf("failed to check existing agent: %w", err)
	}
	if exists {
		uc.logger.Warnw("agent name already exists", "name", cmd.Name)
		return nil, errors.NewConflictError("agent name already exists", cmd.Name)
	}

	// Parse allowed port range if provided
	var portRange *vo.PortRange
	if cmd.AllowedPortRange != "" {
		portRange, err = vo.ParsePortRange(cmd.AllowedPortRange)
		if err != nil {
			uc.logger.Errorw("invalid allowed port range", "range", cmd.AllowedPortRange, "error", err)
			return nil, errors.NewValidationError(fmt.Sprintf("invalid allowed port range: %v", err))
		}
	}

	// Validate blocked protocols if provided
	if len(cmd.BlockedProtocols) > 0 {
		invalidProtocols := vo.ValidateBlockedProtocols(cmd.BlockedProtocols)
		if len(invalidProtocols) > 0 {
			uc.logger.Errorw("invalid blocked protocols", "protocols", cmd.BlockedProtocols, "invalid", invalidProtocols)
			return nil, errors.NewValidationError(fmt.Sprintf("invalid blocked protocols: %v, valid protocols are: %v", invalidProtocols, vo.ValidBlockedProtocolNames()))
		}
	}

	// Create domain entity with HMAC-based token generator
	agent, err := forward.NewForwardAgent(cmd.Name, cmd.PublicAddress, cmd.TunnelAddress, cmd.Remark, id.NewForwardAgentID, uc.tokenGen.Generate)
	if err != nil {
		uc.logger.Errorw("failed to create forward agent entity", "error", err)
		return nil, fmt.Errorf("failed to create forward agent: %w", err)
	}

	// Set allowed port range if provided
	if portRange != nil {
		if err := agent.SetAllowedPortRange(portRange); err != nil {
			uc.logger.Errorw("failed to set allowed port range", "error", err)
			return nil, errors.NewValidationError(fmt.Sprintf("invalid allowed port range: %v", err))
		}
	}

	// Set blocked protocols if provided
	if len(cmd.BlockedProtocols) > 0 {
		blockedProtocols := vo.NewBlockedProtocols(cmd.BlockedProtocols)
		agent.SetBlockedProtocols(blockedProtocols)
	}

	// Set sort order if explicitly provided
	if cmd.SortOrder != nil {
		agent.UpdateSortOrder(*cmd.SortOrder)
	}

	// Handle GroupSID (resolve SID to internal ID)
	if cmd.GroupSID != "" {
		group, err := uc.resourceGroupRepo.GetBySID(ctx, cmd.GroupSID)
		if err != nil {
			uc.logger.Errorw("failed to get resource group by SID", "group_sid", cmd.GroupSID, "error", err)
			return nil, errors.NewNotFoundError("resource group", cmd.GroupSID)
		}
		if group == nil {
			return nil, errors.NewNotFoundError("resource group", cmd.GroupSID)
		}
		groupID := group.ID()
		agent.SetGroupID(&groupID)
	}

	// Persist
	if err := uc.repo.Create(ctx, agent); err != nil {
		uc.logger.Errorw("failed to persist forward agent", "error", err)
		return nil, fmt.Errorf("failed to save forward agent: %w", err)
	}

	// Get the plain token before it's cleared
	plainToken := agent.GetAPIToken()

	result := &CreateForwardAgentResult{
		ID:            agent.SID(),
		Name:          agent.Name(),
		PublicAddress: agent.PublicAddress(),
		TunnelAddress: agent.TunnelAddress(),
		Token:         plainToken,
		Status:        string(agent.Status()),
		Remark:        agent.Remark(),
		CreatedAt:     agent.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	uc.logger.Infow("forward agent created successfully", "id", result.ID, "name", cmd.Name)
	return result, nil
}

func (uc *CreateForwardAgentUseCase) validateCommand(cmd CreateForwardAgentCommand) error {
	if cmd.Name == "" {
		return errors.NewValidationError("agent name is required")
	}
	return nil
}
