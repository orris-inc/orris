// Package dto provides data transfer objects for the forward domain.
package dto

import (
	"time"

	"github.com/orris-inc/orris/internal/domain/forward"
)

// UserForwardAgentDTO represents a forward agent from user's perspective.
// This DTO hides sensitive fields like token_hash and api_token.
type UserForwardAgentDTO struct {
	ID            string              `json:"id"`                       // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	Name          string              `json:"name"`                     // Agent name
	PublicAddress string              `json:"public_address,omitempty"` // Public address for client connections
	Status        string              `json:"status"`                   // enabled or disabled
	Groups        []UserGroupInfoDTO  `json:"groups,omitempty"`         // Resource groups this agent belongs to
	CreatedAt     time.Time           `json:"created_at"`

	internalGroupIDs []uint `json:"-"` // internal resource group IDs for lookup
}

// UserGroupInfoDTO represents group info for user-facing DTOs.
type UserGroupInfoDTO struct {
	SID  string `json:"id"`   // Resource group SID (e.g., "rg_xK9mP2vL3nQ")
	Name string `json:"name"` // Resource group name for display
}

// GroupInfo holds resource group information for populating DTOs.
type GroupInfo struct {
	SID  string
	Name string
}

// GroupInfoMap maps group ID to GroupInfo.
type GroupInfoMap map[uint]*GroupInfo

// ToUserForwardAgentDTO converts a domain forward agent to user-facing DTO.
func ToUserForwardAgentDTO(agent *forward.ForwardAgent) *UserForwardAgentDTO {
	if agent == nil {
		return nil
	}

	dto := &UserForwardAgentDTO{
		ID:               agent.SID(),
		Name:             agent.Name(),
		PublicAddress:    agent.PublicAddress(),
		Status:           string(agent.Status()),
		CreatedAt:        agent.CreatedAt(),
		internalGroupIDs: agent.GroupIDs(),
	}

	return dto
}

// ToUserForwardAgentDTOs converts a slice of domain forward agents to user-facing DTOs.
func ToUserForwardAgentDTOs(agents []*forward.ForwardAgent) []*UserForwardAgentDTO {
	dtos := make([]*UserForwardAgentDTO, len(agents))
	for i, agent := range agents {
		dtos[i] = ToUserForwardAgentDTO(agent)
	}
	return dtos
}

// PopulateGroups fills in the groups field using the group info map.
func (d *UserForwardAgentDTO) PopulateGroups(groupInfoMap GroupInfoMap) {
	if len(d.internalGroupIDs) == 0 {
		return
	}
	d.Groups = make([]UserGroupInfoDTO, 0, len(d.internalGroupIDs))
	for _, groupID := range d.internalGroupIDs {
		if info, ok := groupInfoMap[groupID]; ok && info != nil {
			d.Groups = append(d.Groups, UserGroupInfoDTO{
				SID:  info.SID,
				Name: info.Name,
			})
		}
	}
}

// InternalGroupIDs returns the internal resource group IDs for repository lookups.
func (d *UserForwardAgentDTO) InternalGroupIDs() []uint {
	return d.internalGroupIDs
}

// CollectUserAgentGroupIDs collects unique resource group IDs from user agent DTOs for batch lookup.
func CollectUserAgentGroupIDs(dtos []*UserForwardAgentDTO) []uint {
	idSet := make(map[uint]struct{})
	for _, dto := range dtos {
		for _, groupID := range dto.internalGroupIDs {
			if groupID != 0 {
				idSet[groupID] = struct{}{}
			}
		}
	}
	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	return ids
}
