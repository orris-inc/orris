// Package dto provides data transfer objects for the forward domain.
package dto

import commondto "github.com/orris-inc/orris/internal/application/common/dto"

// AgentStatusDTO extends SystemStatus with forward-specific fields.
type AgentStatusDTO struct {
	commondto.SystemStatus

	// Forward-specific fields
	ActiveRules       int               `json:"active_rules"`
	ActiveConnections int               `json:"active_connections"`
	TunnelStatus      map[string]string `json:"tunnel_status,omitempty"`
	WsListenPort      uint16            `json:"ws_listen_port,omitempty"`
	TlsListenPort     uint16            `json:"tls_listen_port,omitempty"`
}

// ReportAgentStatusInput represents the input for ReportAgentStatus use case.
type ReportAgentStatusInput struct {
	AgentID uint
	Status  *AgentStatusDTO
}
