package dto

import "time"

// ========== Request DTOs ==========

// TrafficStatsQueryRequest represents common query parameters for traffic statistics
type TrafficStatsQueryRequest struct {
	From         time.Time `form:"from" binding:"required" time_format:"2006-01-02"`
	To           time.Time `form:"to" binding:"required" time_format:"2006-01-02"`
	ResourceType *string   `form:"resource_type"`
	Page         int       `form:"page,default=1" binding:"min=1"`
	PageSize     int       `form:"page_size,default=20" binding:"min=1,max=100"`
}

// TrafficRankingRequest represents query parameters for traffic ranking
type TrafficRankingRequest struct {
	From         time.Time `form:"from" binding:"required" time_format:"2006-01-02"`
	To           time.Time `form:"to" binding:"required" time_format:"2006-01-02"`
	ResourceType *string   `form:"resource_type"`
	Limit        int       `form:"limit,default=10" binding:"min=1,max=100"`
}

// TrafficTrendRequest represents query parameters for traffic trend
type TrafficTrendRequest struct {
	From         time.Time `form:"from" binding:"required" time_format:"2006-01-02"`
	To           time.Time `form:"to" binding:"required" time_format:"2006-01-02"`
	ResourceType *string   `form:"resource_type"`
	Granularity  string    `form:"granularity" binding:"required,oneof=hour day month"`
}

// ========== Response DTOs ==========

// TrafficOverviewResponse represents global traffic overview
type TrafficOverviewResponse struct {
	TotalUpload   uint64 `json:"total_upload"`
	TotalDownload uint64 `json:"total_download"`
	TotalTraffic  uint64 `json:"total_traffic"`
}

// UserTrafficStatsItem represents traffic statistics for a single user
type UserTrafficStatsItem struct {
	UserSID            string `json:"user_id"`
	UserEmail          string `json:"user_email"`
	UserName           string `json:"user_name"`
	Upload             uint64 `json:"upload"`
	Download           uint64 `json:"download"`
	Total              uint64 `json:"total"`
	SubscriptionsCount int    `json:"subscriptions_count"`
}

// UserTrafficStatsResponse represents paginated user traffic statistics
type UserTrafficStatsResponse struct {
	Items    []UserTrafficStatsItem `json:"items"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

// SubscriptionTrafficStatsItem represents traffic statistics for a single subscription
type SubscriptionTrafficStatsItem struct {
	SubscriptionSID string `json:"subscription_id"`
	UserSID         string `json:"user_id"`
	UserEmail       string `json:"user_email"`
	PlanName        string `json:"plan_name"`
	Status          string `json:"status"`
	Upload          uint64 `json:"upload"`
	Download        uint64 `json:"download"`
	Total           uint64 `json:"total"`
}

// SubscriptionTrafficStatsResponse represents paginated subscription traffic statistics
type SubscriptionTrafficStatsResponse struct {
	Items    []SubscriptionTrafficStatsItem `json:"items"`
	Total    int64                          `json:"total"`
	Page     int                            `json:"page"`
	PageSize int                            `json:"page_size"`
}

// NodeTrafficStatsItem represents traffic statistics for a single node
type NodeTrafficStatsItem struct {
	NodeSID                 string `json:"node_id"`
	NodeName                string `json:"node_name"`
	Status                  string `json:"status"`
	Upload                  uint64 `json:"upload"`
	Download                uint64 `json:"download"`
	Total                   uint64 `json:"total"`
	OnlineSubscriptionCount int    `json:"online_subscription_count"`
}

// NodeTrafficStatsResponse represents paginated node traffic statistics
type NodeTrafficStatsResponse struct {
	Items    []NodeTrafficStatsItem `json:"items"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

// TrafficRankingItem represents a single item in traffic ranking
type TrafficRankingItem struct {
	Rank     int    `json:"rank"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Upload   uint64 `json:"upload"`
	Download uint64 `json:"download"`
	Total    uint64 `json:"total"`
}

// TrafficRankingResponse represents traffic ranking response
type TrafficRankingResponse struct {
	Items []TrafficRankingItem `json:"items"`
}

// TrafficTrendPoint represents a single data point in traffic trend
type TrafficTrendPoint struct {
	Period   string `json:"period"`
	Upload   uint64 `json:"upload"`
	Download uint64 `json:"download"`
	Total    uint64 `json:"total"`
}

// TrafficTrendResponse represents traffic trend response
type TrafficTrendResponse struct {
	Points      []TrafficTrendPoint `json:"points"`
	Granularity string              `json:"granularity"`
}
