package http

import (
	adminUsecases "github.com/orris-inc/orris/internal/application/admin/usecases"
	forwardUsecases "github.com/orris-inc/orris/internal/application/forward/usecases"
	nodeUsecases "github.com/orris-inc/orris/internal/application/node/usecases"
	paymentUsecases "github.com/orris-inc/orris/internal/application/payment/usecases"
	resourceUsecases "github.com/orris-inc/orris/internal/application/resource/usecases"
	subscriptionUsecases "github.com/orris-inc/orris/internal/application/subscription/usecases"
	telegramAdminUsecases "github.com/orris-inc/orris/internal/application/telegram/admin/usecases"
	"github.com/orris-inc/orris/internal/application/user/usecases"
)

// allUseCases holds all use case instances used by the application.
type allUseCases struct {
	// User / Auth
	registerUC           *usecases.RegisterWithPasswordUseCase
	loginUC              *usecases.LoginWithPasswordUseCase
	verifyEmailUC        *usecases.VerifyEmailUseCase
	requestResetUC       *usecases.RequestPasswordResetUseCase
	resetPasswordUC      *usecases.ResetPasswordUseCase
	adminResetPasswordUC *usecases.AdminResetPasswordUseCase
	initiateOAuthUC      *usecases.InitiateOAuthLoginUseCase
	handleOAuthUC        *usecases.HandleOAuthCallbackUseCase
	refreshTokenUC       *usecases.RefreshTokenUseCase
	logoutUC             *usecases.LogoutUseCase
	getDashboardUC       *usecases.GetDashboardUseCase

	// Subscription
	createSubscriptionUC        *subscriptionUsecases.CreateSubscriptionUseCase
	activateSubscriptionUC      *subscriptionUsecases.ActivateSubscriptionUseCase
	getSubscriptionUC           *subscriptionUsecases.GetSubscriptionUseCase
	listUserSubscriptionsUC     *subscriptionUsecases.ListUserSubscriptionsUseCase
	cancelSubscriptionUC        *subscriptionUsecases.CancelSubscriptionUseCase
	suspendSubscriptionUC       *subscriptionUsecases.SuspendSubscriptionUseCase
	unsuspendSubscriptionUC     *subscriptionUsecases.UnsuspendSubscriptionUseCase
	resetSubscriptionUsageUC    *subscriptionUsecases.ResetSubscriptionUsageUseCase
	deleteSubscriptionUC        *subscriptionUsecases.DeleteSubscriptionUseCase
	renewSubscriptionUC         *subscriptionUsecases.RenewSubscriptionUseCase
	changePlanUC                *subscriptionUsecases.ChangePlanUseCase
	getSubscriptionUsageStatsUC *subscriptionUsecases.GetSubscriptionUsageStatsUseCase
	resetSubscriptionLinkUC     *subscriptionUsecases.ResetSubscriptionLinkUseCase
	aggregateUsageUC            *subscriptionUsecases.AggregateUsageUseCase
	expireSubscriptionsUC       *subscriptionUsecases.ExpireSubscriptionsUseCase

	// Plan
	createPlanUC      *subscriptionUsecases.CreatePlanUseCase
	updatePlanUC      *subscriptionUsecases.UpdatePlanUseCase
	getPlanUC         *subscriptionUsecases.GetPlanUseCase
	listPlansUC       *subscriptionUsecases.ListPlansUseCase
	getPublicPlansUC  *subscriptionUsecases.GetPublicPlansUseCase
	activatePlanUC    *subscriptionUsecases.ActivatePlanUseCase
	deactivatePlanUC  *subscriptionUsecases.DeactivatePlanUseCase
	deletePlanUC      *subscriptionUsecases.DeletePlanUseCase
	getPlanPricingsUC *subscriptionUsecases.GetPlanPricingsUseCase

	// Subscription Token
	generateTokenUC            *subscriptionUsecases.GenerateSubscriptionTokenUseCase
	listTokensUC               *subscriptionUsecases.ListSubscriptionTokensUseCase
	revokeTokenUC              *subscriptionUsecases.RevokeSubscriptionTokenUseCase
	refreshSubscriptionTokenUC *subscriptionUsecases.RefreshSubscriptionTokenUseCase

	// Quota
	quotaService *subscriptionUsecases.QuotaServiceImpl

	// Payment
	createPaymentUC   *paymentUsecases.CreatePaymentUseCase
	handleCallbackUC  *paymentUsecases.HandlePaymentCallbackUseCase
	expirePaymentsUC  *paymentUsecases.ExpirePaymentsUseCase
	cancelUnpaidSubsUC *paymentUsecases.CancelUnpaidSubscriptionsUseCase
	retryActivationUC *paymentUsecases.RetrySubscriptionActivationUseCase

	// Node
	createNodeUC                *nodeUsecases.CreateNodeUseCase
	getNodeUC                   *nodeUsecases.GetNodeUseCase
	updateNodeUC                *nodeUsecases.UpdateNodeUseCase
	deleteNodeUC                *nodeUsecases.DeleteNodeUseCase
	listNodesUC                 *nodeUsecases.ListNodesUseCase
	generateNodeTokenUC         *nodeUsecases.GenerateNodeTokenUseCase
	generateNodeInstallScriptUC *nodeUsecases.GenerateNodeInstallScriptUseCase
	generateBatchInstallScriptUC *nodeUsecases.GenerateBatchInstallScriptUseCase
	validateNodeTokenUC         *nodeUsecases.ValidateNodeTokenUseCase
	getNodeConfigUC             *nodeUsecases.GetNodeConfigUseCase
	getNodeSubscriptionsUC      *nodeUsecases.GetNodeSubscriptionsUseCase
	generateSubscriptionUC      *nodeUsecases.GenerateSubscriptionUseCase
	reportSubscriptionUsageUC   *nodeUsecases.ReportSubscriptionUsageUseCase
	reportNodeStatusUC          *nodeUsecases.ReportNodeStatusUseCase
	reportOnlineSubscriptionsUC *nodeUsecases.ReportOnlineSubscriptionsUseCase

	// User Node
	createUserNodeUC             *nodeUsecases.CreateUserNodeUseCase
	listUserNodesUC              *nodeUsecases.ListUserNodesUseCase
	getUserNodeUC                *nodeUsecases.GetUserNodeUseCase
	updateUserNodeUC             *nodeUsecases.UpdateUserNodeUseCase
	deleteUserNodeUC             *nodeUsecases.DeleteUserNodeUseCase
	regenerateUserNodeTokenUC    *nodeUsecases.RegenerateUserNodeTokenUseCase
	getUserNodeUsageUC           *nodeUsecases.GetUserNodeUsageUseCase
	getUserNodeInstallScriptUC   *nodeUsecases.GetUserNodeInstallScriptUseCase
	getUserBatchInstallScriptUC  *nodeUsecases.GetUserBatchInstallScriptUseCase

	// Forward Agent
	createForwardAgentUC           *forwardUsecases.CreateForwardAgentUseCase
	getForwardAgentUC              *forwardUsecases.GetForwardAgentUseCase
	updateForwardAgentUC           *forwardUsecases.UpdateForwardAgentUseCase
	deleteForwardAgentUC           *forwardUsecases.DeleteForwardAgentUseCase
	listForwardAgentsUC            *forwardUsecases.ListForwardAgentsUseCase
	enableForwardAgentUC           *forwardUsecases.EnableForwardAgentUseCase
	disableForwardAgentUC          *forwardUsecases.DisableForwardAgentUseCase
	regenerateForwardAgentTokenUC  *forwardUsecases.RegenerateForwardAgentTokenUseCase
	validateForwardAgentTokenUC    *forwardUsecases.ValidateForwardAgentTokenUseCase
	getAgentStatusUC               *forwardUsecases.GetAgentStatusUseCase
	getRuleOverallStatusUC         *forwardUsecases.GetRuleOverallStatusUseCase
	getForwardAgentTokenUC         *forwardUsecases.GetForwardAgentTokenUseCase
	generateInstallScriptUC        *forwardUsecases.GenerateInstallScriptUseCase
	reportAgentStatusUC            *forwardUsecases.ReportAgentStatusUseCase
	reportRuleSyncStatusUC         *forwardUsecases.ReportRuleSyncStatusUseCase

	// Forward Rule
	createForwardRuleUC    *forwardUsecases.CreateForwardRuleUseCase
	getForwardRuleUC       *forwardUsecases.GetForwardRuleUseCase
	updateForwardRuleUC    *forwardUsecases.UpdateForwardRuleUseCase
	deleteForwardRuleUC    *forwardUsecases.DeleteForwardRuleUseCase
	listForwardRulesUC     *forwardUsecases.ListForwardRulesUseCase
	enableForwardRuleUC    *forwardUsecases.EnableForwardRuleUseCase
	disableForwardRuleUC   *forwardUsecases.DisableForwardRuleUseCase
	resetForwardTrafficUC  *forwardUsecases.ResetForwardRuleTrafficUseCase
	reorderForwardRulesUC  *forwardUsecases.ReorderForwardRulesUseCase
	batchForwardRuleUC     *forwardUsecases.BatchForwardRuleUseCase

	// User Forward Rule
	createUserForwardRuleUC    *forwardUsecases.CreateUserForwardRuleUseCase
	listUserForwardRulesUC     *forwardUsecases.ListUserForwardRulesUseCase
	getUserForwardUsageUC      *forwardUsecases.GetUserForwardUsageUseCase
	listUserForwardAgentsUC    *forwardUsecases.ListUserForwardAgentsUseCase

	// Subscription Forward Rule
	createSubscriptionForwardRuleUC *forwardUsecases.CreateSubscriptionForwardRuleUseCase
	listSubscriptionForwardRulesUC  *forwardUsecases.ListSubscriptionForwardRulesUseCase
	getSubscriptionForwardUsageUC   *forwardUsecases.GetSubscriptionForwardUsageUseCase

	// Resource Group
	createResourceGroupUC       *resourceUsecases.CreateResourceGroupUseCase
	getResourceGroupUC          *resourceUsecases.GetResourceGroupUseCase
	listResourceGroupsUC        *resourceUsecases.ListResourceGroupsUseCase
	updateResourceGroupUC       *resourceUsecases.UpdateResourceGroupUseCase
	deleteResourceGroupUC       *resourceUsecases.DeleteResourceGroupUseCase
	updateResourceGroupStatusUC *resourceUsecases.UpdateResourceGroupStatusUseCase
	manageNodesUC               *resourceUsecases.ManageResourceGroupNodesUseCase
	manageAgentsUC              *resourceUsecases.ManageResourceGroupForwardAgentsUseCase
	manageRulesUC               *resourceUsecases.ManageResourceGroupForwardRulesUseCase

	// Admin Dashboard & Traffic
	getAdminDashboardUC            *adminUsecases.GetAdminDashboardUseCase
	getTrafficOverviewUC           *adminUsecases.GetTrafficOverviewUseCase
	getUserTrafficStatsUC          *adminUsecases.GetUserTrafficStatsUseCase
	getSubscriptionTrafficStatsUC  *adminUsecases.GetSubscriptionTrafficStatsUseCase
	getAdminNodeTrafficStatsUC     *adminUsecases.GetAdminNodeTrafficStatsUseCase
	getTrafficRankingUC            *adminUsecases.GetTrafficRankingUseCase
	getTrafficTrendUC              *adminUsecases.GetTrafficTrendUseCase

	// Telegram Admin
	muteNotificationUC *telegramAdminUsecases.MuteNotificationUseCase
}
