package seeds

import (
	"gorm.io/gorm"
	"orris/internal/infrastructure/persistence/models"
)

// SeedNotificationTemplates seeds the notification templates table with default templates
func SeedNotificationTemplates(db *gorm.DB) error {
	templates := []models.NotificationTemplateModel{
		{
			TemplateType: "subscription_expiring",
			Name:         "Subscription Expiring Reminder",
			Title:        "Your subscription is expiring soon",
			Content:      "Your subscription **{{.SubscriptionName}}** will expire on {{.ExpiryDate}}.",
			Variables:    `["SubscriptionName", "ExpiryDate"]`,
			Enabled:      true,
		},
		{
			TemplateType: "system_maintenance",
			Name:         "System Maintenance Notification",
			Title:        "Scheduled maintenance",
			Content:      "System maintenance is scheduled at {{.MaintenanceTime}}. Expected downtime: {{.Duration}}.",
			Variables:    `["MaintenanceTime", "Duration"]`,
			Enabled:      true,
		},
		{
			TemplateType: "welcome",
			Name:         "Welcome New User",
			Title:        "Welcome to {{.AppName}}!",
			Content:      "Hello {{.Username}}, welcome aboard!",
			Variables:    `["AppName", "Username"]`,
			Enabled:      true,
		},
		{
			TemplateType: "subscription_activated",
			Name:         "Subscription Activated",
			Title:        "Your subscription has been activated",
			Content:      "Your subscription **{{.PlanName}}** is now active until {{.EndDate}}.",
			Variables:    `["PlanName", "EndDate"]`,
			Enabled:      true,
		},
		{
			TemplateType: "subscription_cancelled",
			Name:         "Subscription Cancelled",
			Title:        "Your subscription has been cancelled",
			Content:      "Your subscription **{{.PlanName}}** has been cancelled. It will remain active until {{.EndDate}}.",
			Variables:    `["PlanName", "EndDate"]`,
			Enabled:      true,
		},
		{
			TemplateType: "password_reset",
			Name:         "Password Reset Request",
			Title:        "Reset your password",
			Content:      "Click the link to reset your password: {{.ResetLink}}. This link will expire in {{.ExpiryMinutes}} minutes.",
			Variables:    `["ResetLink", "ExpiryMinutes"]`,
			Enabled:      true,
		},
		{
			TemplateType: "email_verification",
			Name:         "Email Verification",
			Title:        "Verify your email address",
			Content:      "Please verify your email by clicking: {{.VerificationLink}}",
			Variables:    `["VerificationLink"]`,
			Enabled:      true,
		},
	}

	for _, template := range templates {
		if err := db.FirstOrCreate(&template, models.NotificationTemplateModel{
			TemplateType: template.TemplateType,
		}).Error; err != nil {
			return err
		}
	}

	return nil
}
