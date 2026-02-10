package http

import (
	"github.com/orris-inc/orris/internal/interfaces/http/handlers"
	adminHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/admin"
	adminResourceGroupHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/admin/resourcegroup"
	adminSubscriptionHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/admin/subscription"
	forwardAgentAPIHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/agent/api"
	forwardAgentCrudHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/agent/crud"
	forwardAgentHubHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/agent/hub"
	forwardRuleHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/rule"
	forwardSubscriptionHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/subscription"
	forwardUserHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/user"
	nodeHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/node"
	telegramHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/telegram"
	ticketHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/ticket"
)

// allHandlers holds all HTTP handler instances used by the application.
type allHandlers struct {
	// User & Auth
	userHandler    *handlers.UserHandler
	authHandler    *handlers.AuthHandler
	passkeyHandler *handlers.PasskeyHandler
	profileHandler *handlers.ProfileHandler

	// Dashboard
	dashboardHandler *handlers.DashboardHandler

	// Subscription
	subscriptionHandler      *handlers.SubscriptionHandler
	adminSubscriptionHandler *adminSubscriptionHandlers.Handler
	subscriptionTokenHandler *handlers.SubscriptionTokenHandler

	// Plan
	planHandler *handlers.PlanHandler

	// Payment
	paymentHandler *handlers.PaymentHandler

	// Node
	nodeHandler             *handlers.NodeHandler
	nodeSubscriptionHandler *handlers.NodeSubscriptionHandler
	userNodeHandler         *nodeHandlers.UserNodeHandler
	agentHandler            *nodeHandlers.AgentHandler
	nodeHubHandler          *nodeHandlers.NodeHubHandler
	nodeVersionHandler      *nodeHandlers.NodeVersionHandler
	nodeSSEHandler          *nodeHandlers.NodeSSEHandler

	// Forward
	forwardRuleHandler             *forwardRuleHandlers.Handler
	forwardAgentHandler            *forwardAgentCrudHandlers.Handler
	forwardAgentVersionHandler     *forwardAgentCrudHandlers.VersionHandler
	forwardAgentSSEHandler         *forwardAgentCrudHandlers.ForwardAgentSSEHandler
	forwardAgentAPIHandler         *forwardAgentAPIHandlers.Handler
	userForwardRuleHandler         *forwardUserHandlers.Handler
	subscriptionForwardRuleHandler *forwardSubscriptionHandlers.Handler
	agentHubHandler                *forwardAgentHubHandlers.Handler

	// Ticket
	ticketHandler *ticketHandlers.TicketHandler

	// Notification
	notificationHandler *handlers.NotificationHandler

	// Telegram
	telegramHandler *telegramHandlers.Handler

	// Admin
	adminResourceGroupHandler *adminResourceGroupHandlers.Handler
	adminTrafficStatsHandler  *adminHandlers.TrafficStatsHandler
	adminTelegramHandler      *adminHandlers.AdminTelegramHandler
	settingHandler            *adminHandlers.SettingHandler
}
