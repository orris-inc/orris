package constants

const (
	// Default pagination
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100

	// Context keys
	ContextKeyUserID   = "user_id"
	ContextKeyUserRole = "user_role"

	// User status
	UserStatusActive   = "active"
	UserStatusInactive = "inactive"
	UserStatusPending  = "pending"

	// User roles
	RoleAdmin        = "admin"
	RoleUser         = "user"
	RoleSupportAgent = "support_agent"

	// Database table names
	TableUsers                  = "users"
	TableNodes                  = "nodes"
	TableNodeShadowsocksConfigs = "node_shadowsocks_configs"
	TableNodeTrojanConfigs      = "node_trojan_configs"
	TablePlans                  = "plans"
	TablePlanPricings           = "plan_pricings"
	TableSubscriptions          = "subscriptions"
	TableSubscriptionTokens     = "subscription_tokens"
	TableSubscriptionUsages     = "subscription_usages"
	TablePayments               = "payments"
	TableNotifications          = "notifications"
	TableNotificationTemplates  = "notification_templates"
	TableAnnouncements          = "announcements"
	TableTickets                = "tickets"
	TableTicketComments         = "ticket_comments"
	TableForwardRules           = "forward_rules"
	TableForwardAgents          = "forward_agents"
	TableResourceGroups         = "resource_groups"

	// Default values
	DefaultCurrency = "CNY"
)
