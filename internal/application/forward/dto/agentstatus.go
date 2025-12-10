// Package dto provides data transfer objects for the forward domain.
package dto

// AgentStatusDTO represents the status data reported by a forward agent.
type AgentStatusDTO struct {
	// System resources
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryUsed    uint64  `json:"memory_used"`
	MemoryTotal   uint64  `json:"memory_total"`
	DiskPercent   float64 `json:"disk_percent"`
	DiskUsed      uint64  `json:"disk_used"`
	DiskTotal     uint64  `json:"disk_total"`
	UptimeSeconds int64   `json:"uptime_seconds"`

	// Network connections
	TCPConnections int `json:"tcp_connections"`
	UDPConnections int `json:"udp_connections"`

	// Forward status
	ActiveRules       int               `json:"active_rules"`
	ActiveConnections int               `json:"active_connections"`
	TunnelStatus      map[string]string `json:"tunnel_status,omitempty"` // Key is Stripe-style rule ID (e.g., "fr_xK9mP2vL3nQ")
}

// ReportAgentStatusInput represents the input for ReportAgentStatus use case.
type ReportAgentStatusInput struct {
	AgentID uint
	Status  *AgentStatusDTO
}
