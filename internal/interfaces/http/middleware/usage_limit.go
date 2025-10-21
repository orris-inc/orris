package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"orris/internal/domain/subscription"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

type SubscriptionUsageLimitMiddleware struct {
	usageRepo subscription.SubscriptionUsageRepository
	logger    logger.Interface
}

func NewSubscriptionUsageLimitMiddleware(
	usageRepo subscription.SubscriptionUsageRepository,
	logger logger.Interface,
) *SubscriptionUsageLimitMiddleware {
	return &SubscriptionUsageLimitMiddleware{
		usageRepo: usageRepo,
		logger:    logger,
	}
}

func (m *SubscriptionUsageLimitMiddleware) CheckUsageLimits() gin.HandlerFunc {
	return func(c *gin.Context) {
		subscriptionIDValue, exists := c.Get("subscription_id")
		if !exists {
			m.logger.Warnw("subscription ID not found in context")
			utils.ErrorResponse(c, http.StatusUnauthorized, "subscription not found")
			c.Abort()
			return
		}

		subscriptionID, ok := subscriptionIDValue.(uint)
		if !ok {
			m.logger.Errorw("invalid subscription ID type in context")
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid subscription ID")
			c.Abort()
			return
		}

		planValue, exists := c.Get("subscription_plan")
		if !exists {
			m.logger.Warnw("subscription plan not found in context", "subscription_id", subscriptionID)
			utils.ErrorResponse(c, http.StatusUnauthorized, "subscription plan not found")
			c.Abort()
			return
		}

		plan, ok := planValue.(*subscription.SubscriptionPlan)
		if !ok {
			m.logger.Errorw("invalid subscription plan type in context", "subscription_id", subscriptionID)
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid subscription plan")
			c.Abort()
			return
		}

		usage, err := m.usageRepo.GetCurrentUsage(c.Request.Context(), subscriptionID)
		if err != nil {
			m.logger.Errorw("failed to get current usage",
				"error", err,
				"subscription_id", subscriptionID,
			)
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to check usage limits")
			c.Abort()
			return
		}

		if usage == nil {
			c.Next()
			return
		}

		violations := []string{}

		if plan.StorageLimit() > 0 && usage.StorageUsed() > plan.StorageLimit() {
			violations = append(violations, fmt.Sprintf("storage limit exceeded: %d/%d bytes",
				usage.StorageUsed(), plan.StorageLimit()))
		}

		if plan.MaxUsers() > 0 && usage.UsersCount() > plan.MaxUsers() {
			violations = append(violations, fmt.Sprintf("user limit exceeded: %d/%d users",
				usage.UsersCount(), plan.MaxUsers()))
		}

		if plan.MaxProjects() > 0 && usage.ProjectsCount() > plan.MaxProjects() {
			violations = append(violations, fmt.Sprintf("project limit exceeded: %d/%d projects",
				usage.ProjectsCount(), plan.MaxProjects()))
		}

		if len(violations) > 0 {
			m.logger.Warnw("usage limits exceeded",
				"subscription_id", subscriptionID,
				"plan_id", plan.ID(),
				"violations", violations,
			)
			utils.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("usage limits exceeded: %v", violations))
			c.Abort()
			return
		}

		c.Set("subscription_usage", usage)

		c.Next()
	}
}

func (m *SubscriptionUsageLimitMiddleware) CheckStorageLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		subscriptionIDValue, exists := c.Get("subscription_id")
		if !exists {
			m.logger.Warnw("subscription ID not found in context")
			utils.ErrorResponse(c, http.StatusUnauthorized, "subscription not found")
			c.Abort()
			return
		}

		subscriptionID, ok := subscriptionIDValue.(uint)
		if !ok {
			m.logger.Errorw("invalid subscription ID type in context")
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid subscription ID")
			c.Abort()
			return
		}

		planValue, exists := c.Get("subscription_plan")
		if !exists {
			m.logger.Warnw("subscription plan not found in context", "subscription_id", subscriptionID)
			utils.ErrorResponse(c, http.StatusUnauthorized, "subscription plan not found")
			c.Abort()
			return
		}

		plan, ok := planValue.(*subscription.SubscriptionPlan)
		if !ok {
			m.logger.Errorw("invalid subscription plan type in context", "subscription_id", subscriptionID)
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid subscription plan")
			c.Abort()
			return
		}

		if plan.StorageLimit() == 0 {
			c.Next()
			return
		}

		usage, err := m.usageRepo.GetCurrentUsage(c.Request.Context(), subscriptionID)
		if err != nil {
			m.logger.Errorw("failed to get current usage",
				"error", err,
				"subscription_id", subscriptionID,
			)
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to check storage limit")
			c.Abort()
			return
		}

		if usage != nil && usage.StorageUsed() > plan.StorageLimit() {
			m.logger.Warnw("storage limit exceeded",
				"subscription_id", subscriptionID,
				"plan_id", plan.ID(),
				"used", usage.StorageUsed(),
				"limit", plan.StorageLimit(),
			)
			utils.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("storage limit exceeded: %d/%d bytes",
				usage.StorageUsed(), plan.StorageLimit()))
			c.Abort()
			return
		}

		c.Next()
	}
}

func (m *SubscriptionUsageLimitMiddleware) CheckUserLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		subscriptionIDValue, exists := c.Get("subscription_id")
		if !exists {
			m.logger.Warnw("subscription ID not found in context")
			utils.ErrorResponse(c, http.StatusUnauthorized, "subscription not found")
			c.Abort()
			return
		}

		subscriptionID, ok := subscriptionIDValue.(uint)
		if !ok {
			m.logger.Errorw("invalid subscription ID type in context")
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid subscription ID")
			c.Abort()
			return
		}

		planValue, exists := c.Get("subscription_plan")
		if !exists {
			m.logger.Warnw("subscription plan not found in context", "subscription_id", subscriptionID)
			utils.ErrorResponse(c, http.StatusUnauthorized, "subscription plan not found")
			c.Abort()
			return
		}

		plan, ok := planValue.(*subscription.SubscriptionPlan)
		if !ok {
			m.logger.Errorw("invalid subscription plan type in context", "subscription_id", subscriptionID)
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid subscription plan")
			c.Abort()
			return
		}

		if plan.MaxUsers() == 0 {
			c.Next()
			return
		}

		usage, err := m.usageRepo.GetCurrentUsage(c.Request.Context(), subscriptionID)
		if err != nil {
			m.logger.Errorw("failed to get current usage",
				"error", err,
				"subscription_id", subscriptionID,
			)
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to check user limit")
			c.Abort()
			return
		}

		if usage != nil && usage.UsersCount() > plan.MaxUsers() {
			m.logger.Warnw("user limit exceeded",
				"subscription_id", subscriptionID,
				"plan_id", plan.ID(),
				"users", usage.UsersCount(),
				"limit", plan.MaxUsers(),
			)
			utils.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("user limit exceeded: %d/%d users",
				usage.UsersCount(), plan.MaxUsers()))
			c.Abort()
			return
		}

		c.Next()
	}
}

func (m *SubscriptionUsageLimitMiddleware) CheckProjectLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		subscriptionIDValue, exists := c.Get("subscription_id")
		if !exists {
			m.logger.Warnw("subscription ID not found in context")
			utils.ErrorResponse(c, http.StatusUnauthorized, "subscription not found")
			c.Abort()
			return
		}

		subscriptionID, ok := subscriptionIDValue.(uint)
		if !ok {
			m.logger.Errorw("invalid subscription ID type in context")
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid subscription ID")
			c.Abort()
			return
		}

		planValue, exists := c.Get("subscription_plan")
		if !exists {
			m.logger.Warnw("subscription plan not found in context", "subscription_id", subscriptionID)
			utils.ErrorResponse(c, http.StatusUnauthorized, "subscription plan not found")
			c.Abort()
			return
		}

		plan, ok := planValue.(*subscription.SubscriptionPlan)
		if !ok {
			m.logger.Errorw("invalid subscription plan type in context", "subscription_id", subscriptionID)
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid subscription plan")
			c.Abort()
			return
		}

		if plan.MaxProjects() == 0 {
			c.Next()
			return
		}

		usage, err := m.usageRepo.GetCurrentUsage(c.Request.Context(), subscriptionID)
		if err != nil {
			m.logger.Errorw("failed to get current usage",
				"error", err,
				"subscription_id", subscriptionID,
			)
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to check project limit")
			c.Abort()
			return
		}

		if usage != nil && usage.ProjectsCount() > plan.MaxProjects() {
			m.logger.Warnw("project limit exceeded",
				"subscription_id", subscriptionID,
				"plan_id", plan.ID(),
				"projects", usage.ProjectsCount(),
				"limit", plan.MaxProjects(),
			)
			utils.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("project limit exceeded: %d/%d projects",
				usage.ProjectsCount(), plan.MaxProjects()))
			c.Abort()
			return
		}

		c.Next()
	}
}
