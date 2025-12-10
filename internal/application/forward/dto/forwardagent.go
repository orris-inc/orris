// Package dto provides data transfer objects for the forward domain.
package dto

import (
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/id"
)

// ForwardAgentDTO represents the data transfer object for forward agents.
type ForwardAgentDTO struct {
	ID            string          `json:"id"` // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	Name          string          `json:"name"`
	PublicAddress string          `json:"public_address"`
	Status        string          `json:"status"`
	Remark        string          `json:"remark"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
	SystemStatus  *AgentStatusDTO `json:"system_status,omitempty"`
}

// ToForwardAgentDTO converts a domain forward agent to DTO.
func ToForwardAgentDTO(agent *forward.ForwardAgent) *ForwardAgentDTO {
	if agent == nil {
		return nil
	}

	return &ForwardAgentDTO{
		ID:            id.FormatForwardAgentID(agent.ShortID()),
		Name:          agent.Name(),
		PublicAddress: agent.PublicAddress(),
		Status:        string(agent.Status()),
		Remark:        agent.Remark(),
		CreatedAt:     agent.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     agent.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ToForwardAgentDTOs converts a slice of domain forward agents to DTOs.
func ToForwardAgentDTOs(agents []*forward.ForwardAgent) []*ForwardAgentDTO {
	dtos := make([]*ForwardAgentDTO, len(agents))
	for i, agent := range agents {
		dtos[i] = ToForwardAgentDTO(agent)
	}
	return dtos
}
