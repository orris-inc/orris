package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

type SubscriptionFeatureMiddleware struct {
	logger logger.Interface
}

func NewSubscriptionFeatureMiddleware(logger logger.Interface) *SubscriptionFeatureMiddleware {
	return &SubscriptionFeatureMiddleware{
		logger: logger,
	}
}

func (m *SubscriptionFeatureMiddleware) RequireSubscriptionFeature(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		planValue, exists := c.Get("subscription_plan")
		if !exists {
			m.logger.Warnw("subscription plan not found in context", "feature", feature)
			utils.ErrorResponse(c, http.StatusUnauthorized, "subscription plan not found")
			c.Abort()
			return
		}

		plan, ok := planValue.(*subscription.Plan)
		if !ok {
			m.logger.Errorw("invalid subscription plan type in context", "feature", feature)
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid subscription plan")
			c.Abort()
			return
		}

		if !plan.HasFeature(feature) {
			subscriptionID, _ := c.Get("subscription_id")
			m.logger.Warnw("subscription lacks required feature",
				"subscription_id", subscriptionID,
				"plan_id", plan.ID(),
				"required_feature", feature,
			)
			utils.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("feature not available: %s", feature))
			c.Abort()
			return
		}

		c.Next()
	}
}

func (m *SubscriptionFeatureMiddleware) RequireAnyFeature(features ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		planValue, exists := c.Get("subscription_plan")
		if !exists {
			m.logger.Warnw("subscription plan not found in context", "features", features)
			utils.ErrorResponse(c, http.StatusUnauthorized, "subscription plan not found")
			c.Abort()
			return
		}

		plan, ok := planValue.(*subscription.Plan)
		if !ok {
			m.logger.Errorw("invalid subscription plan type in context", "features", features)
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid subscription plan")
			c.Abort()
			return
		}

		for _, feature := range features {
			if plan.HasFeature(feature) {
				c.Next()
				return
			}
		}

		subscriptionID, _ := c.Get("subscription_id")
		m.logger.Warnw("subscription lacks any of required features",
			"subscription_id", subscriptionID,
			"plan_id", plan.ID(),
			"required_features", features,
		)
		utils.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("none of required features available: %v", features))
		c.Abort()
	}
}

func (m *SubscriptionFeatureMiddleware) RequireAllFeatures(features ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		planValue, exists := c.Get("subscription_plan")
		if !exists {
			m.logger.Warnw("subscription plan not found in context", "features", features)
			utils.ErrorResponse(c, http.StatusUnauthorized, "subscription plan not found")
			c.Abort()
			return
		}

		plan, ok := planValue.(*subscription.Plan)
		if !ok {
			m.logger.Errorw("invalid subscription plan type in context", "features", features)
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid subscription plan")
			c.Abort()
			return
		}

		missingFeatures := []string{}
		for _, feature := range features {
			if !plan.HasFeature(feature) {
				missingFeatures = append(missingFeatures, feature)
			}
		}

		if len(missingFeatures) > 0 {
			subscriptionID, _ := c.Get("subscription_id")
			m.logger.Warnw("subscription lacks some required features",
				"subscription_id", subscriptionID,
				"plan_id", plan.ID(),
				"missing_features", missingFeatures,
			)
			utils.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("missing features: %v", missingFeatures))
			c.Abort()
			return
		}

		c.Next()
	}
}
