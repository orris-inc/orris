package http

import (
	"context"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	forwardServices "github.com/orris-inc/orris/internal/application/forward/services"
	nodeServices "github.com/orris-inc/orris/internal/application/node/services"
	settingApp "github.com/orris-inc/orris/internal/application/setting"
	telegramApp "github.com/orris-inc/orris/internal/application/telegram"
	telegramAdminApp "github.com/orris-inc/orris/internal/application/telegram/admin"
	"github.com/orris-inc/orris/internal/application/user"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/infrastructure/config"
	"github.com/orris-inc/orris/internal/infrastructure/email"
	infraPayment "github.com/orris-inc/orris/internal/infrastructure/payment"
	"github.com/orris-inc/orris/internal/infrastructure/scheduler"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	telegramInfra "github.com/orris-inc/orris/internal/infrastructure/telegram"
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
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// Router represents the HTTP router configuration.
// Handler and middleware fields are accessed by router_methods.go for route setup.
type Router struct {
	engine                         *gin.Engine
	userHandler                    *handlers.UserHandler
	authHandler                    *handlers.AuthHandler
	passkeyHandler                 *handlers.PasskeyHandler
	profileHandler                 *handlers.ProfileHandler
	dashboardHandler               *handlers.DashboardHandler
	subscriptionHandler            *handlers.SubscriptionHandler
	adminSubscriptionHandler       *adminSubscriptionHandlers.Handler
	adminResourceGroupHandler      *adminResourceGroupHandlers.Handler
	adminDashboardHandler          *adminHandlers.AdminDashboardHandler
	adminTrafficStatsHandler       *adminHandlers.TrafficStatsHandler
	adminTelegramHandler           *adminHandlers.AdminTelegramHandler
	adminNotificationService       *telegramAdminApp.ServiceDDD
	settingHandler                 *adminHandlers.SettingHandler
	settingService                 *settingApp.ServiceDDD
	planHandler                    *handlers.PlanHandler
	subscriptionTokenHandler       *handlers.SubscriptionTokenHandler
	paymentHandler                 *handlers.PaymentHandler
	nodeHandler                    *handlers.NodeHandler
	nodeSubscriptionHandler        *handlers.NodeSubscriptionHandler
	userNodeHandler                *nodeHandlers.UserNodeHandler
	agentHandler                   *nodeHandlers.AgentHandler
	ticketHandler                  *ticketHandlers.TicketHandler
	notificationHandler            *handlers.NotificationHandler
	telegramHandler                *telegramHandlers.Handler
	telegramService                *telegramApp.ServiceDDD
	telegramBotManager             *telegramInfra.BotServiceManager
	forwardRuleHandler             *forwardRuleHandlers.Handler
	forwardAgentHandler            *forwardAgentCrudHandlers.Handler
	forwardAgentVersionHandler     *forwardAgentCrudHandlers.VersionHandler
	forwardAgentSSEHandler         *forwardAgentCrudHandlers.ForwardAgentSSEHandler
	forwardAgentAPIHandler         *forwardAgentAPIHandlers.Handler
	userForwardRuleHandler         *forwardUserHandlers.Handler
	subscriptionForwardRuleHandler *forwardSubscriptionHandlers.Handler
	agentHub                       *services.AgentHub
	agentHubHandler                *forwardAgentHubHandlers.Handler
	nodeHubHandler                 *nodeHandlers.NodeHubHandler
	nodeVersionHandler             *nodeHandlers.NodeVersionHandler
	nodeSSEHandler                 *nodeHandlers.NodeSSEHandler
	adminHub                       *services.AdminHub
	configSyncService              *forwardServices.ConfigSyncService
	trafficLimitEnforcementSvc     *forwardServices.TrafficLimitEnforcementService
	forwardTrafficCache            cache.ForwardTrafficCache
	ruleTrafficBuffer              *forwardServices.RuleTrafficBuffer
	ruleTrafficFlushDone           chan struct{}
	subscriptionTrafficCache       cache.SubscriptionTrafficCache
	subscriptionTrafficBuffer      *nodeServices.SubscriptionTrafficBuffer
	subscriptionTrafficFlushDone   chan struct{}
	schedulerManager               *scheduler.SchedulerManager
	usdtServiceManager             *infraPayment.USDTServiceManager
	hubEventBusCancel              context.CancelFunc
	hubEventBusCancelMu            *sync.Mutex
	logger                         logger.Interface
	authMiddleware                 *middleware.AuthMiddleware
	subscriptionOwnerMiddleware    *middleware.SubscriptionOwnerMiddleware
	nodeTokenMiddleware            *middleware.NodeTokenMiddleware
	nodeOwnerMiddleware            *middleware.NodeOwnerMiddleware
	nodeQuotaMiddleware            *middleware.NodeQuotaMiddleware
	forwardAgentTokenMiddleware    *middleware.ForwardAgentTokenMiddleware
	forwardRuleOwnerMiddleware     *middleware.ForwardRuleOwnerMiddleware
	forwardQuotaMiddleware         *middleware.ForwardQuotaMiddleware
	rateLimiter                    *middleware.RateLimiter
	oauthManager                   *auth.OAuthServiceManager
	emailManager                   *email.EmailServiceManager
}

// NewRouter creates a new HTTP router with all dependencies.
// The function signature is preserved for backward compatibility with command.go.
func NewRouter(userService *user.ServiceDDD, db *gorm.DB, cfg *config.Config, log logger.Interface) *Router {
	c := NewContainer(userService, db, cfg, log)

	return &Router{
		engine:                         c.engine,
		userHandler:                    c.hdlrs.userHandler,
		authHandler:                    c.hdlrs.authHandler,
		passkeyHandler:                 c.hdlrs.passkeyHandler,
		profileHandler:                 c.hdlrs.profileHandler,
		dashboardHandler:               c.hdlrs.dashboardHandler,
		subscriptionHandler:            c.hdlrs.subscriptionHandler,
		adminSubscriptionHandler:       c.hdlrs.adminSubscriptionHandler,
		adminResourceGroupHandler:      c.hdlrs.adminResourceGroupHandler,
		adminDashboardHandler:          c.hdlrs.adminDashboardHandler,
		adminTrafficStatsHandler:       c.hdlrs.adminTrafficStatsHandler,
		adminTelegramHandler:           c.hdlrs.adminTelegramHandler,
		adminNotificationService:       c.adminNotificationServiceDDD,
		settingHandler:                 c.hdlrs.settingHandler,
		settingService:                 c.settingServiceDDD,
		planHandler:                    c.hdlrs.planHandler,
		subscriptionTokenHandler:       c.hdlrs.subscriptionTokenHandler,
		paymentHandler:                 c.hdlrs.paymentHandler,
		nodeHandler:                    c.hdlrs.nodeHandler,
		nodeSubscriptionHandler:        c.hdlrs.nodeSubscriptionHandler,
		userNodeHandler:                c.hdlrs.userNodeHandler,
		agentHandler:                   c.hdlrs.agentHandler,
		ticketHandler:                  c.hdlrs.ticketHandler,
		notificationHandler:            c.hdlrs.notificationHandler,
		telegramHandler:                c.hdlrs.telegramHandler,
		telegramService:                c.telegramServiceDDD,
		telegramBotManager:             c.telegramBotManager,
		forwardRuleHandler:             c.hdlrs.forwardRuleHandler,
		forwardAgentHandler:            c.hdlrs.forwardAgentHandler,
		forwardAgentVersionHandler:     c.hdlrs.forwardAgentVersionHandler,
		forwardAgentSSEHandler:         c.hdlrs.forwardAgentSSEHandler,
		forwardAgentAPIHandler:         c.hdlrs.forwardAgentAPIHandler,
		userForwardRuleHandler:         c.hdlrs.userForwardRuleHandler,
		subscriptionForwardRuleHandler: c.hdlrs.subscriptionForwardRuleHandler,
		agentHub:                       c.agentHub,
		agentHubHandler:                c.hdlrs.agentHubHandler,
		nodeHubHandler:                 c.hdlrs.nodeHubHandler,
		nodeVersionHandler:             c.hdlrs.nodeVersionHandler,
		nodeSSEHandler:                 c.hdlrs.nodeSSEHandler,
		adminHub:                       c.adminHub,
		configSyncService:              c.configSyncService,
		trafficLimitEnforcementSvc:     c.trafficLimitEnforcementSvc,
		forwardTrafficCache:            c.forwardTrafficCache,
		ruleTrafficBuffer:              c.ruleTrafficBuffer,
		ruleTrafficFlushDone:           c.ruleTrafficFlushDone,
		subscriptionTrafficCache:       c.subscriptionTrafficCache,
		subscriptionTrafficBuffer:      c.subscriptionTrafficBuffer,
		subscriptionTrafficFlushDone:   c.subscriptionTrafficFlushDone,
		schedulerManager:               c.schedulerManager,
		usdtServiceManager:             c.usdtServiceManager,
		hubEventBusCancel:              c.hubEventBusCancel,
		hubEventBusCancelMu:            &c.hubEventBusCancelMu,
		logger:                         log,
		authMiddleware:                 c.authMiddleware,
		subscriptionOwnerMiddleware:    c.subscriptionOwnerMiddleware,
		nodeTokenMiddleware:            c.nodeTokenMiddleware,
		nodeOwnerMiddleware:            c.nodeOwnerMiddleware,
		nodeQuotaMiddleware:            c.nodeQuotaMiddleware,
		forwardAgentTokenMiddleware:    c.forwardAgentTokenMiddleware,
		forwardRuleOwnerMiddleware:     c.forwardRuleOwnerMiddleware,
		forwardQuotaMiddleware:         c.forwardQuotaMiddleware,
		rateLimiter:                    c.rateLimiter,
		oauthManager:                   c.oauthManager,
		emailManager:                   c.emailManager,
	}
}

// GetEngine returns the Gin engine.
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

// Run starts the HTTP server.
func (r *Router) Run(addr string) error {
	return r.engine.Run(addr)
}

// Shutdown gracefully shuts down the router by delegating to the Container.
func (r *Router) Shutdown() {
	// Stop hub event bus subscribers first to prevent new cross-instance events
	r.hubEventBusCancelMu.Lock()
	cancel := r.hubEventBusCancel
	r.hubEventBusCancelMu.Unlock()

	if cancel != nil {
		cancel()
	}

	// Stop unified scheduler manager (includes all scheduled jobs)
	if r.schedulerManager != nil {
		if err := r.schedulerManager.Stop(); err != nil {
			r.logger.Errorw("failed to stop scheduler manager", "error", err)
		}
	}

	// Stop USDT monitor scheduler (managed by USDTServiceManager separately)
	if r.usdtServiceManager != nil {
		r.usdtServiceManager.StopScheduler()
	}

	// Stop telegram bot service manager if running
	if r.telegramBotManager != nil {
		r.telegramBotManager.Stop()
	}

	// Close all SSE connections first to allow HTTP server shutdown to proceed quickly
	if r.adminHub != nil {
		r.adminHub.Shutdown()
	}

	// Stop rule traffic flush scheduler goroutine
	if r.ruleTrafficFlushDone != nil {
		close(r.ruleTrafficFlushDone)
	}

	// Stop rule traffic buffer (flushes remaining data to Redis)
	if r.ruleTrafficBuffer != nil {
		r.ruleTrafficBuffer.Stop()
	}

	// Final flush forward traffic from Redis to MySQL
	if r.forwardTrafficCache != nil {
		ctx := context.Background()
		if err := r.forwardTrafficCache.FlushToDatabase(ctx); err != nil {
			r.logger.Errorw("failed to flush forward traffic to database on shutdown", "error", err)
		}
	}

	// Stop subscription traffic flush scheduler goroutine
	if r.subscriptionTrafficFlushDone != nil {
		close(r.subscriptionTrafficFlushDone)
	}

	// Stop subscription traffic buffer (flushes remaining data to Redis)
	if r.subscriptionTrafficBuffer != nil {
		r.subscriptionTrafficBuffer.Stop()
	}

	// Final flush subscription traffic from Redis to MySQL
	if r.subscriptionTrafficCache != nil {
		ctx := context.Background()
		if err := r.subscriptionTrafficCache.FlushToDatabase(ctx); err != nil {
			r.logger.Errorw("failed to flush subscription traffic to database on shutdown", "error", err)
		}
	}
}

// GetTelegramService returns the telegram service for scheduler use.
func (r *Router) GetTelegramService() *telegramApp.ServiceDDD {
	return r.telegramService
}

// GetAdminNotificationService returns the admin notification service for scheduler use.
func (r *Router) GetAdminNotificationService() *telegramAdminApp.ServiceDDD {
	return r.adminNotificationService
}

// GetTelegramBotManager returns the telegram bot service manager.
func (r *Router) GetTelegramBotManager() *telegramInfra.BotServiceManager {
	return r.telegramBotManager
}

// GetSettingService returns the setting service.
func (r *Router) GetSettingService() *settingApp.ServiceDDD {
	return r.settingService
}

// StartTelegramPolling starts the telegram bot service using BotServiceManager.
func (r *Router) StartTelegramPolling(ctx context.Context) error {
	if r.telegramBotManager == nil {
		return nil
	}
	if err := r.telegramBotManager.Start(ctx); err != nil {
		return err
	}
	return nil
}

// StartScheduler starts the unified scheduler manager (all registered jobs).
func (r *Router) StartScheduler() {
	if r.schedulerManager != nil {
		r.schedulerManager.Start()
	}
}

// StartUSDTMonitorScheduler starts the USDT payment monitor scheduler
// (managed by USDTServiceManager separately from the unified scheduler).
func (r *Router) StartUSDTMonitorScheduler(ctx context.Context) {
	if r.usdtServiceManager != nil {
		r.usdtServiceManager.StartScheduler(ctx)
	}
}

// GetSchedulerManager returns the scheduler manager for external job registration.
func (r *Router) GetSchedulerManager() *scheduler.SchedulerManager {
	return r.schedulerManager
}
