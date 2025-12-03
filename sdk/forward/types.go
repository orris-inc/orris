// Package forward provides a Go SDK for interacting with the Orris Forward Agent API.
package forward

// Rule represents a forward rule returned by the API.
type Rule struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	ListenPort    uint16 `json:"listen_port"`
	TargetAddress string `json:"target_address"`
	TargetPort    uint16 `json:"target_port"`
	Protocol      string `json:"protocol"`
	Status        string `json:"status"`
	Remark        string `json:"remark,omitempty"`
	UploadBytes   int64  `json:"upload_bytes"`
	DownloadBytes int64  `json:"download_bytes"`
	TotalBytes    int64  `json:"total_bytes"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// TrafficItem represents traffic data for a single rule.
type TrafficItem struct {
	RuleID        uint  `json:"rule_id"`
	UploadBytes   int64 `json:"upload_bytes"`
	DownloadBytes int64 `json:"download_bytes"`
}

// TrafficReportResult represents the result of a traffic report.
type TrafficReportResult struct {
	RulesUpdated int `json:"rules_updated"`
	RulesFailed  int `json:"rules_failed"`
}

// apiResponse represents the standard API response structure.
type apiResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}
