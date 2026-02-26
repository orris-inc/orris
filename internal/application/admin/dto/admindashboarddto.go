package dto

// AdminDashboardResponse represents the admin dashboard snapshot response.
type AdminDashboardResponse struct {
	Users         DashboardUsersSection         `json:"users"`
	Subscriptions DashboardSubscriptionsSection `json:"subscriptions"`
	Nodes         DashboardNodesSection         `json:"nodes"`
	Forward       DashboardForwardSection       `json:"forward"`
	TrafficToday  DashboardTrafficSection       `json:"traffic_today"`
}

// DashboardUsersSection holds user-related dashboard metrics.
type DashboardUsersSection struct {
	Total      int64 `json:"total"`
	NewToday   int64 `json:"new_today"`
	NewThisWeek int64 `json:"new_this_week"`
}

// DashboardSubscriptionsSection holds subscription-related dashboard metrics.
type DashboardSubscriptionsSection struct {
	Active         int64 `json:"active"`
	Expired        int64 `json:"expired"`
	Suspended      int64 `json:"suspended"`
	PendingPayment int64 `json:"pending_payment"`
	ExpiringIn7Days int64 `json:"expiring_in_7_days"`
}

// DashboardNodesSection holds node-related dashboard metrics.
type DashboardNodesSection struct {
	Total                   int64 `json:"total"`
	Online                  int64 `json:"online"`
	Offline                 int64 `json:"offline"`
	TotalOnlineSubscriptions int64 `json:"total_online_subscriptions"`
}

// DashboardForwardSection holds forward-related dashboard metrics.
type DashboardForwardSection struct {
	TotalRules   int64 `json:"total_rules"`
	TotalAgents  int64 `json:"total_agents"`
	OnlineAgents int64 `json:"online_agents"`
}

// DashboardTrafficSection holds today's traffic metrics.
type DashboardTrafficSection struct {
	Upload   uint64 `json:"upload"`
	Download uint64 `json:"download"`
	Total    uint64 `json:"total"`
}
