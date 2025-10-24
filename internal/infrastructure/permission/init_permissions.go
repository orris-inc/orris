package permission

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"orris/internal/shared/logger"
)

// InitNotificationPermissions initializes notification and announcement permissions
func InitNotificationPermissions(enforcer *casbin.Enforcer, log logger.Interface) error {
	policies := [][]string{
		// Admin permissions - full access to all notification features
		{"admin", "announcement", "create"},
		{"admin", "announcement", "read"},
		{"admin", "announcement", "update"},
		{"admin", "announcement", "delete"},
		{"admin", "announcement", "publish"},
		{"admin", "template", "create"},
		{"admin", "template", "read"},
		{"admin", "template", "update"},
		{"admin", "template", "delete"},
		{"admin", "template", "use"},
		{"admin", "notification", "create"},
		{"admin", "notification", "read"},
		{"admin", "notification", "update"},
		{"admin", "notification", "delete"},
		{"admin", "notification", "send"},

		// User permissions - read and manage own notifications
		{"user", "announcement", "read"},
		{"user", "notification", "read"},
		{"user", "notification", "update"},
	}

	for _, policy := range policies {
		_, err := enforcer.AddPolicy(policy)
		if err != nil {
			log.Errorw("failed to add notification permission policy",
				"error", err,
				"role", policy[0],
				"resource", policy[1],
				"action", policy[2])
			return fmt.Errorf("failed to add policy [%s, %s, %s]: %w",
				policy[0], policy[1], policy[2], err)
		}
	}

	if err := enforcer.SavePolicy(); err != nil {
		log.Error("failed to save notification permissions", "error", err)
		return fmt.Errorf("failed to save notification permissions: %w", err)
	}

	log.Info("notification permissions initialized successfully")
	return nil
}

// InitSubscriptionPermissions initializes subscription-related permissions
func InitSubscriptionPermissions(enforcer *casbin.Enforcer, log logger.Interface) error {
	policies := [][]string{
		// Admin permissions - full access to subscription management
		{"admin", "subscription", "create"},
		{"admin", "subscription", "read"},
		{"admin", "subscription", "update"},
		{"admin", "subscription", "delete"},
		{"admin", "subscription", "cancel"},
		{"admin", "subscription_plan", "create"},
		{"admin", "subscription_plan", "read"},
		{"admin", "subscription_plan", "update"},
		{"admin", "subscription_plan", "delete"},
		{"admin", "subscription_token", "create"},
		{"admin", "subscription_token", "read"},
		{"admin", "subscription_token", "update"},
		{"admin", "subscription_token", "delete"},

		// User permissions - manage own subscriptions
		{"user", "subscription", "read"},
		{"user", "subscription", "update"},
		{"user", "subscription", "cancel"},
		{"user", "subscription_plan", "read"},
		{"user", "subscription_token", "create"},
		{"user", "subscription_token", "read"},
		{"user", "subscription_token", "delete"},
	}

	for _, policy := range policies {
		_, err := enforcer.AddPolicy(policy)
		if err != nil {
			log.Errorw("failed to add subscription permission policy",
				"error", err,
				"role", policy[0],
				"resource", policy[1],
				"action", policy[2])
			return fmt.Errorf("failed to add policy [%s, %s, %s]: %w",
				policy[0], policy[1], policy[2], err)
		}
	}

	if err := enforcer.SavePolicy(); err != nil {
		log.Error("failed to save subscription permissions", "error", err)
		return fmt.Errorf("failed to save subscription permissions: %w", err)
	}

	log.Info("subscription permissions initialized successfully")
	return nil
}

// InitAllPermissions initializes all permission policies
func InitAllPermissions(enforcer *casbin.Enforcer, log logger.Interface) error {
	if err := InitNotificationPermissions(enforcer, log); err != nil {
		return err
	}

	if err := InitSubscriptionPermissions(enforcer, log); err != nil {
		return err
	}

	log.Info("all permissions initialized successfully")
	return nil
}
