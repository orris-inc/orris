package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UpdateForwardAgentCommand represents the input for updating a forward agent.
type UpdateForwardAgentCommand struct {
	ShortID          string // External API identifier
	Name             *string
	PublicAddress    *string
	TunnelAddress    *string
	Remark           *string
	GroupSIDs        []string   // Resource group SIDs (empty slice to remove all associations)
	AllowedPortRange *string    // nil: no update, empty string: clear (allow all), non-empty: set new range
	BlockedProtocols *[]string  // nil: no update, empty slice: clear (allow all), non-empty: set new protocols
	SortOrder        *int       // nil: no update, non-nil: set new sort order
	MuteNotification *bool      // nil: no update, non-nil: set mute notification flag
	ExpiresAt        *time.Time // nil: no update, set to update expiration time
	ClearExpiresAt   bool       // true: clear expiration time
	RenewalAmount    *float64   // nil: no update, set to update renewal amount
	ClearRenewal     bool       // true: clear renewal amount
}

// UpdateForwardAgentUseCase handles forward agent updates.
type UpdateForwardAgentUseCase struct {
	repo                  forward.AgentRepository
	resourceGroupRepo     resource.Repository
	addressChangeNotifier AgentAddressChangeNotifier
	agentConfigNotifier   AgentConfigChangeNotifier
	logger                logger.Interface
}

// NewUpdateForwardAgentUseCase creates a new UpdateForwardAgentUseCase.
func NewUpdateForwardAgentUseCase(
	repo forward.AgentRepository,
	resourceGroupRepo resource.Repository,
	addressChangeNotifier AgentAddressChangeNotifier,
	agentConfigNotifier AgentConfigChangeNotifier,
	logger logger.Interface,
) *UpdateForwardAgentUseCase {
	return &UpdateForwardAgentUseCase{
		repo:                  repo,
		resourceGroupRepo:     resourceGroupRepo,
		addressChangeNotifier: addressChangeNotifier,
		agentConfigNotifier:   agentConfigNotifier,
		logger:                logger,
	}
}

// Execute updates an existing forward agent.
func (uc *UpdateForwardAgentUseCase) Execute(ctx context.Context, cmd UpdateForwardAgentCommand) error {
	if cmd.ShortID == "" {
		return errors.NewValidationError("short_id is required")
	}

	uc.logger.Infow("executing update forward agent use case", "short_id", cmd.ShortID)

	agent, err := uc.repo.GetBySID(ctx, cmd.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return errors.NewNotFoundError("forward agent", cmd.ShortID)
	}

	// Track original values to detect changes
	originalPublicAddress := agent.PublicAddress()
	originalTunnelAddress := agent.TunnelAddress()
	originalBlockedProtocols := agent.BlockedProtocols().ToStringSlice()

	// Update fields
	if cmd.Name != nil {
		if err := agent.UpdateName(*cmd.Name); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.PublicAddress != nil {
		if err := agent.UpdatePublicAddress(*cmd.PublicAddress); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.TunnelAddress != nil {
		if err := agent.UpdateTunnelAddress(*cmd.TunnelAddress); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.Remark != nil {
		if err := agent.UpdateRemark(*cmd.Remark); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Handle GroupSIDs update (resolve SIDs to internal IDs)
	// Limit to 10 groups to prevent DoS attacks
	const maxGroupSIDs = 10
	if len(cmd.GroupSIDs) > maxGroupSIDs {
		return errors.NewValidationError(fmt.Sprintf("too many group_sids, maximum allowed is %d", maxGroupSIDs))
	}
	// Note: We always update if GroupSIDs is provided (even empty slice means clear all)
	if cmd.GroupSIDs != nil {
		if len(cmd.GroupSIDs) == 0 {
			// Empty slice means remove all associations
			agent.SetGroupIDs(nil)
		} else {
			// Deduplicate and filter empty SIDs
			uniqueSIDs := make([]string, 0, len(cmd.GroupSIDs))
			seenSIDs := make(map[string]struct{}, len(cmd.GroupSIDs))
			for _, sid := range cmd.GroupSIDs {
				if sid == "" {
					continue
				}
				if _, exists := seenSIDs[sid]; exists {
					continue
				}
				seenSIDs[sid] = struct{}{}
				uniqueSIDs = append(uniqueSIDs, sid)
			}

			// Batch fetch all groups to avoid N+1 queries
			groupMap, err := uc.resourceGroupRepo.GetBySIDs(ctx, uniqueSIDs)
			if err != nil {
				uc.logger.Errorw("failed to batch get resource groups", "error", err)
				return fmt.Errorf("failed to get resource groups: %w", err)
			}

			// Resolve SIDs to internal IDs
			resolvedIDs := make([]uint, 0, len(uniqueSIDs))
			for _, sid := range uniqueSIDs {
				group, ok := groupMap[sid]
				if !ok || group == nil {
					return errors.NewNotFoundError("resource group", sid)
				}
				resolvedIDs = append(resolvedIDs, group.ID())
			}
			agent.SetGroupIDs(resolvedIDs)
		}
	}

	// Handle AllowedPortRange update
	// Note: We use a "forward-compatible" approach - existing rules continue to work,
	// only new/updated rules must comply with the new port range.
	// This prevents service disruption when adjusting port policies.
	if cmd.AllowedPortRange != nil {
		if *cmd.AllowedPortRange == "" {
			// Empty string means clear the port range (allow all ports)
			if err := agent.SetAllowedPortRange(nil); err != nil {
				return errors.NewValidationError(err.Error())
			}
		} else {
			// Parse and set the new port range
			portRange, err := vo.ParsePortRange(*cmd.AllowedPortRange)
			if err != nil {
				uc.logger.Errorw("invalid allowed port range", "range", *cmd.AllowedPortRange, "error", err)
				return errors.NewValidationError(fmt.Sprintf("invalid allowed port range: %v", err))
			}

			if err := agent.SetAllowedPortRange(portRange); err != nil {
				return errors.NewValidationError(err.Error())
			}
		}
	}

	// Handle BlockedProtocols update
	if cmd.BlockedProtocols != nil {
		if len(*cmd.BlockedProtocols) == 0 {
			// Empty slice means clear blocked protocols (allow all protocols)
			uc.logger.Infow("clearing blocked protocols",
				"short_id", cmd.ShortID,
				"old_blocked_protocols", originalBlockedProtocols,
			)
			agent.SetBlockedProtocols(nil)
		} else {
			// Validate protocols
			invalidProtocols := vo.ValidateBlockedProtocols(*cmd.BlockedProtocols)
			if len(invalidProtocols) > 0 {
				uc.logger.Errorw("invalid blocked protocols", "protocols", *cmd.BlockedProtocols, "invalid", invalidProtocols)
				return errors.NewValidationError(fmt.Sprintf("invalid blocked protocols: %v, valid protocols are: %v", invalidProtocols, vo.ValidBlockedProtocolNames()))
			}
			// Set the new blocked protocols
			uc.logger.Infow("updating blocked protocols",
				"short_id", cmd.ShortID,
				"old_blocked_protocols", originalBlockedProtocols,
				"new_blocked_protocols", *cmd.BlockedProtocols,
			)
			blockedProtocols := vo.NewBlockedProtocols(*cmd.BlockedProtocols)
			agent.SetBlockedProtocols(blockedProtocols)
		}
	}

	// Handle SortOrder update
	if cmd.SortOrder != nil {
		agent.UpdateSortOrder(*cmd.SortOrder)
	}

	// Handle MuteNotification update
	if cmd.MuteNotification != nil {
		agent.SetMuteNotification(*cmd.MuteNotification)
	}

	// Handle ExpiresAt update
	if cmd.ClearExpiresAt {
		agent.SetExpiresAt(nil)
	} else if cmd.ExpiresAt != nil {
		agent.SetExpiresAt(cmd.ExpiresAt)
	}

	// Handle RenewalAmount update
	if cmd.ClearRenewal {
		agent.SetRenewalAmount(nil)
	} else if cmd.RenewalAmount != nil {
		agent.SetRenewalAmount(cmd.RenewalAmount)
	}

	// Persist changes
	if err := uc.repo.Update(ctx, agent); err != nil {
		uc.logger.Errorw("failed to update forward agent", "id", agent.ID(), "short_id", agent.SID(), "error", err)
		return fmt.Errorf("failed to update forward agent: %w", err)
	}

	uc.logger.Infow("forward agent updated successfully",
		"id", agent.ID(),
		"short_id", agent.SID(),
		"blocked_protocols", agent.BlockedProtocols().ToStringSlice(),
	)

	// Check if address changed and notify related agents
	addressChanged := (cmd.PublicAddress != nil && *cmd.PublicAddress != originalPublicAddress) ||
		(cmd.TunnelAddress != nil && *cmd.TunnelAddress != originalTunnelAddress)

	if addressChanged && uc.addressChangeNotifier != nil {
		agentID := agent.ID()
		uc.logger.Infow("agent address changed, notifying related agents",
			"agent_id", agentID,
			"short_id", agent.SID(),
			"old_public_address", originalPublicAddress,
			"new_public_address", agent.PublicAddress(),
			"old_tunnel_address", originalTunnelAddress,
			"new_tunnel_address", agent.TunnelAddress(),
		)

		// Notify asynchronously to avoid blocking the API response
		go func() {
			if err := uc.addressChangeNotifier.NotifyAgentAddressChange(context.Background(), agentID); err != nil {
				uc.logger.Warnw("failed to notify agent address change",
					"agent_id", agentID,
					"error", err,
				)
			}
		}()
	}

	// Check if blocked protocols changed and notify the agent
	blockedProtocolsChanged := cmd.BlockedProtocols != nil && !slicesEqual(originalBlockedProtocols, agent.BlockedProtocols().ToStringSlice())
	if blockedProtocolsChanged && uc.agentConfigNotifier != nil {
		agentID := agent.ID()
		uc.logger.Infow("agent blocked protocols changed, notifying agent",
			"agent_id", agentID,
			"short_id", agent.SID(),
			"old_blocked_protocols", originalBlockedProtocols,
			"new_blocked_protocols", agent.BlockedProtocols().ToStringSlice(),
		)

		// Notify asynchronously to avoid blocking the API response
		go func() {
			if err := uc.agentConfigNotifier.NotifyAgentBlockedProtocolsChange(context.Background(), agentID); err != nil {
				uc.logger.Warnw("failed to notify agent blocked protocols change",
					"agent_id", agentID,
					"error", err,
				)
			}
		}()
	}

	return nil
}

// slicesEqual checks if two string slices are equal (order matters).
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
