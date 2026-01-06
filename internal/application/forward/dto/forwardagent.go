// Package dto provides data transfer objects for the forward domain.
package dto

import (
	"time"

	"github.com/orris-inc/orris/internal/domain/forward"
)

// ForwardAgentDTO represents the data transfer object for forward agents.
// An agent can participate in multiple rules with different roles (entry/relay/exit) simultaneously.
type ForwardAgentDTO struct {
	ID               string          `json:"id"` // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	Name             string          `json:"name"`
	PublicAddress    string          `json:"public_address"`
	TunnelAddress    string          `json:"tunnel_address,omitempty"` // IP or hostname only (no port), configure if agent may serve as relay/exit in any rule
	Status           string          `json:"status"`
	Remark           string          `json:"remark"`
	GroupSID         *string         `json:"group_id,omitempty"` // Resource group SID this agent belongs to
	AgentVersion     string          `json:"agent_version"`      // Agent software version (e.g., "1.2.3"), extracted from system_status for easy display
	HasUpdate        bool            `json:"has_update"`         // True if a newer version is available
	AllowedPortRange string          `json:"allowed_port_range,omitempty"`
	BlockedProtocols []string        `json:"blocked_protocols,omitempty"` // Protocols blocked by this agent
	SortOrder        int             `json:"sort_order"`                  // Custom sort order for UI display
	MuteNotification bool            `json:"mute_notification"`           // Mute online/offline notifications for this agent
	IsOnline         bool            `json:"is_online"`                   // Indicates if the agent is online (reported within 5 minutes)
	LastSeenAt       *time.Time      `json:"last_seen_at,omitempty"`      // Last time the agent reported status
	CreatedAt        string          `json:"created_at"`
	UpdatedAt        string          `json:"updated_at"`
	SystemStatus     *AgentStatusDTO `json:"system_status,omitempty"`
}

// ToForwardAgentDTO converts a domain forward agent to DTO.
func ToForwardAgentDTO(agent *forward.ForwardAgent) *ForwardAgentDTO {
	if agent == nil {
		return nil
	}

	var allowedPortRange string
	if agent.AllowedPortRange() != nil {
		allowedPortRange = agent.AllowedPortRange().String()
	}

	return &ForwardAgentDTO{
		ID:               agent.SID(),
		Name:             agent.Name(),
		PublicAddress:    agent.PublicAddress(),
		TunnelAddress:    agent.TunnelAddress(),
		Status:           string(agent.Status()),
		Remark:           agent.Remark(),
		AgentVersion:     agent.AgentVersion(),
		AllowedPortRange: allowedPortRange,
		BlockedProtocols: agent.BlockedProtocols().ToStringSlice(),
		SortOrder:        agent.SortOrder(),
		MuteNotification: agent.MuteNotification(),
		IsOnline:         agent.IsOnline(),
		LastSeenAt:       agent.LastSeenAt(),
		CreatedAt:        agent.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        agent.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
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
