package dto

import "time"

// NewUserAlertData represents data for new user alert
type NewUserAlertData struct {
	UserID    string    `json:"user_id"`
	UserEmail string    `json:"user_email"`
	UserName  string    `json:"user_name"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

// PaymentSuccessAlertData represents data for payment success alert
type PaymentSuccessAlertData struct {
	OrderID          string    `json:"order_id"`
	UserID           string    `json:"user_id"`
	UserEmail        string    `json:"user_email"`
	PlanName         string    `json:"plan_name"`
	Amount           int64     `json:"amount"`
	Currency         string    `json:"currency"`
	PaymentMethod    string    `json:"payment_method"`
	SubscriptionType string    `json:"subscription_type"`
	PaidAt           time.Time `json:"paid_at"`
}

// NodeOfflineAlertData represents data for node offline alert
type NodeOfflineAlertData struct {
	NodeID       string    `json:"node_id"`
	NodeName     string    `json:"node_name"`
	NodeIP       string    `json:"node_ip"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	OfflineSince time.Time `json:"offline_since"`
	Duration     int64     `json:"duration"`
}

// AgentOfflineAlertData represents data for agent offline alert
type AgentOfflineAlertData struct {
	AgentID      string    `json:"agent_id"`
	AgentName    string    `json:"agent_name"`
	AgentIP      string    `json:"agent_ip"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	OfflineSince time.Time `json:"offline_since"`
	Duration     int64     `json:"duration"`
}

// DailySummaryData represents daily summary report data
type DailySummaryData struct {
	Date          string `json:"date"`
	NewUserCount  int64  `json:"new_user_count"`
	OrderCount    int64  `json:"order_count"`
	Revenue       int64  `json:"revenue"`
	Currency      string `json:"currency"`
	OnlineNodes   int64  `json:"online_nodes"`
	OfflineNodes  int64  `json:"offline_nodes"`
	TotalNodes    int64  `json:"total_nodes"`
	OnlineAgents  int64  `json:"online_agents"`
	OfflineAgents int64  `json:"offline_agents"`
	TotalAgents   int64  `json:"total_agents"`
	UploadBytes   uint64 `json:"upload_bytes"`
	DownloadBytes uint64 `json:"download_bytes"`
	TotalBytes    uint64 `json:"total_bytes"`
}

// WeeklySummaryData represents weekly summary report data with comparison
type WeeklySummaryData struct {
	WeekStart          string  `json:"week_start"`
	WeekEnd            string  `json:"week_end"`
	NewUserCount       int64   `json:"new_user_count"`
	NewUserCountChange float64 `json:"new_user_count_change"`
	OrderCount         int64   `json:"order_count"`
	OrderCountChange   float64 `json:"order_count_change"`
	Revenue            int64   `json:"revenue"`
	RevenueChange      float64 `json:"revenue_change"`
	Currency           string  `json:"currency"`
	OnlineNodes        int64   `json:"online_nodes"`
	OfflineNodes       int64   `json:"offline_nodes"`
	TotalNodes         int64   `json:"total_nodes"`
	TotalNodesChange   float64 `json:"total_nodes_change"`
	OnlineAgents       int64   `json:"online_agents"`
	OfflineAgents      int64   `json:"offline_agents"`
	TotalAgents        int64   `json:"total_agents"`
	TotalAgentsChange  float64 `json:"total_agents_change"`
	UploadBytes        uint64  `json:"upload_bytes"`
	DownloadBytes      uint64  `json:"download_bytes"`
	TotalBytes         uint64  `json:"total_bytes"`
	TotalBytesChange   float64 `json:"total_bytes_change"`
}
