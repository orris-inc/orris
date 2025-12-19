// Package dto provides data transfer objects for the forward domain.
package dto

import (
	"time"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/id"
)

// UserForwardAgentDTO represents a forward agent from user's perspective.
// This DTO hides sensitive fields like token_hash and api_token.
type UserForwardAgentDTO struct {
	ID            string    `json:"id"`                       // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	Name          string    `json:"name"`                     // Agent name
	PublicAddress string    `json:"public_address,omitempty"` // Public address for client connections
	Status        string    `json:"status"`                   // enabled or disabled
	GroupSID      string    `json:"group_id,omitempty"`       // Resource group SID (e.g., "rg_xK9mP2vL3nQ")
	GroupName     string    `json:"group_name,omitempty"`     // Resource group name for display
	CreatedAt     time.Time `json:"created_at"`
}

// GroupInfo holds resource group information for populating DTOs.
type GroupInfo struct {
	SID  string
	Name string
}

// GroupInfoMap maps group ID to GroupInfo.
type GroupInfoMap map[uint]*GroupInfo

// ToUserForwardAgentDTO converts a domain forward agent to user-facing DTO.
func ToUserForwardAgentDTO(agent *forward.ForwardAgent, groupInfo *GroupInfo) *UserForwardAgentDTO {
	if agent == nil {
		return nil
	}

	dto := &UserForwardAgentDTO{
		ID:            id.FormatForwardAgentID(agent.ShortID()),
		Name:          agent.Name(),
		PublicAddress: agent.PublicAddress(),
		Status:        string(agent.Status()),
		CreatedAt:     agent.CreatedAt(),
	}

	if groupInfo != nil {
		dto.GroupSID = groupInfo.SID
		dto.GroupName = groupInfo.Name
	}

	return dto
}

// ToUserForwardAgentDTOs converts a slice of domain forward agents to user-facing DTOs.
func ToUserForwardAgentDTOs(agents []*forward.ForwardAgent, groupInfoMap GroupInfoMap) []*UserForwardAgentDTO {
	dtos := make([]*UserForwardAgentDTO, len(agents))
	for i, agent := range agents {
		var groupInfo *GroupInfo
		if agent.GroupID() != nil {
			groupInfo = groupInfoMap[*agent.GroupID()]
		}
		dtos[i] = ToUserForwardAgentDTO(agent, groupInfo)
	}
	return dtos
}
