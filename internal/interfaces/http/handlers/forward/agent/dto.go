package agent

// ForwardRuleTrafficItem represents traffic data for a single forward rule
type ForwardRuleTrafficItem struct {
	RuleID        string `json:"rule_id" binding:"required"` // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	UploadBytes   int64  `json:"upload_bytes" binding:"min=0"`
	DownloadBytes int64  `json:"download_bytes" binding:"min=0"`
}

// ReportTrafficRequest represents traffic report request from forward client
type ReportTrafficRequest struct {
	Rules []ForwardRuleTrafficItem `json:"rules" binding:"required,dive"`
}

// ReportStatusRequest represents status report request from forward client
type ReportStatusRequest struct {
	CPUPercent        float64           `json:"cpu_percent"`
	MemoryPercent     float64           `json:"memory_percent"`
	MemoryUsed        uint64            `json:"memory_used"`
	MemoryTotal       uint64            `json:"memory_total"`
	DiskPercent       float64           `json:"disk_percent"`
	DiskUsed          uint64            `json:"disk_used"`
	DiskTotal         uint64            `json:"disk_total"`
	UptimeSeconds     int64             `json:"uptime_seconds"`
	TCPConnections    int               `json:"tcp_connections"`
	UDPConnections    int               `json:"udp_connections"`
	ActiveRules       int               `json:"active_rules"`
	ActiveConnections int               `json:"active_connections"`
	TunnelStatus      map[string]string `json:"tunnel_status,omitempty"`  // Key is Stripe-style rule ID (e.g., "fr_xK9mP2vL3nQ")
	WsListenPort      uint16            `json:"ws_listen_port,omitempty"` // WebSocket listen port for exit agent tunnel connections
}
