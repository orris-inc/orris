package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/ratelimit"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

type SubscriptionRateLimitMiddleware struct {
	limiter ratelimit.RateLimiter
	logger  logger.Interface
}

func NewSubscriptionRateLimitMiddleware(
	limiter ratelimit.RateLimiter,
	logger logger.Interface,
) *SubscriptionRateLimitMiddleware {
	return &SubscriptionRateLimitMiddleware{
		limiter: limiter,
		logger:  logger,
	}
}

func (m *SubscriptionRateLimitMiddleware) LimitBySubscription() gin.HandlerFunc {
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

		plan, ok := planValue.(*subscription.Plan)
		if !ok {
			m.logger.Errorw("invalid subscription plan type in context", "subscription_id", subscriptionID)
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid subscription plan")
			c.Abort()
			return
		}

		rateLimit := plan.APIRateLimit()
		if rateLimit == 0 {
			rateLimit = 60
		}

		key := fmt.Sprintf("subscription:%d", subscriptionID)
		config := ratelimit.RateLimitConfig{
			RequestsPerMinute: int(rateLimit),
			RequestsPerHour:   int(rateLimit) * 60,
			RequestsPerDay:    int(rateLimit) * 60 * 24,
			BurstSize:         int(rateLimit),
		}

		allowed, err := m.limiter.Allow(key, config)
		if err != nil {
			m.logger.Errorw("rate limit check failed",
				"error", err,
				"subscription_id", subscriptionID,
			)
			utils.ErrorResponse(c, http.StatusInternalServerError, "rate limit check failed")
			c.Abort()
			return
		}

		remaining, err := m.limiter.GetRemaining(key, time.Minute)
		if err != nil {
			m.logger.Warnw("failed to get remaining rate limit",
				"error", err,
				"subscription_id", subscriptionID,
			)
			remaining = 0
		}

		limit := int64(rateLimit)
		used := limit - remaining
		if used < 0 {
			used = 0
		}
		if remaining < 0 {
			remaining = 0
		}

		c.Header("X-RateLimit-Limit", strconv.FormatInt(limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))

		if !allowed {
			m.logger.Warnw("rate limit exceeded",
				"subscription_id", subscriptionID,
				"plan_id", plan.ID(),
				"limit", rateLimit,
			)

			retryAfter := time.Now().Add(time.Minute).Unix() - time.Now().Unix()
			c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))

			utils.ErrorResponse(c, http.StatusTooManyRequests, "rate limit exceeded")
			c.Abort()
			return
		}

		c.Next()
	}
}

func (m *SubscriptionRateLimitMiddleware) CustomLimit(requestsPerMinute int) gin.HandlerFunc {
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

		key := fmt.Sprintf("subscription:%d:custom", subscriptionID)
		config := ratelimit.RateLimitConfig{
			RequestsPerMinute: requestsPerMinute,
			RequestsPerHour:   requestsPerMinute * 60,
			RequestsPerDay:    requestsPerMinute * 60 * 24,
			BurstSize:         requestsPerMinute,
		}

		allowed, err := m.limiter.Allow(key, config)
		if err != nil {
			m.logger.Errorw("rate limit check failed",
				"error", err,
				"subscription_id", subscriptionID,
			)
			utils.ErrorResponse(c, http.StatusInternalServerError, "rate limit check failed")
			c.Abort()
			return
		}

		remaining, err := m.limiter.GetRemaining(key, time.Minute)
		if err != nil {
			m.logger.Warnw("failed to get remaining rate limit",
				"error", err,
				"subscription_id", subscriptionID,
			)
			remaining = 0
		}

		limit := int64(requestsPerMinute)
		used := limit - remaining
		if used < 0 {
			used = 0
		}
		if remaining < 0 {
			remaining = 0
		}

		c.Header("X-RateLimit-Limit", strconv.FormatInt(limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))

		if !allowed {
			m.logger.Warnw("custom rate limit exceeded",
				"subscription_id", subscriptionID,
				"limit", requestsPerMinute,
			)

			retryAfter := time.Now().Add(time.Minute).Unix() - time.Now().Unix()
			c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))

			utils.ErrorResponse(c, http.StatusTooManyRequests, "rate limit exceeded")
			c.Abort()
			return
		}

		c.Next()
	}
}
