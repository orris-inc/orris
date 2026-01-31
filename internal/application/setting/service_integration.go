package setting

import (
	"context"

	"github.com/orris-inc/orris/internal/application/setting/dto"
)

// ============================================================================
// System Settings
// ============================================================================

// GetSystemSettings retrieves system settings with source tracking
func (s *ServiceDDD) GetSystemSettings(ctx context.Context) (*dto.SystemSettingsResponse, error) {
	provider := s.settingProvider

	// api_base_url and timezone are read-only (environment variable only)
	apiBaseURL := provider.GetAPIBaseURL(ctx)
	apiBaseURL.IsReadOnly = true

	timezone := provider.GetTimezone(ctx)
	timezone.IsReadOnly = true

	return &dto.SystemSettingsResponse{
		APIBaseURL:          apiBaseURL,
		SubscriptionBaseURL: provider.GetSubscriptionBaseURL(ctx),
		FrontendURL:         provider.GetFrontendURL(ctx),
		Timezone:            timezone,
	}, nil
}

// UpdateSystemSettings updates system settings
// Note: api_base_url and timezone are read-only and cannot be modified via API
func (s *ServiceDDD) UpdateSystemSettings(ctx context.Context, req dto.UpdateSystemSettingsRequest, updatedBy uint) error {
	changes := make(map[string]any)

	if req.SubscriptionBaseURL != nil {
		if err := s.upsertSetting(ctx, "system", "subscription_base_url", *req.SubscriptionBaseURL, updatedBy); err != nil {
			return err
		}
		changes["subscription_base_url"] = *req.SubscriptionBaseURL
	}
	if req.FrontendURL != nil {
		if err := s.upsertSetting(ctx, "system", "frontend_url", *req.FrontendURL, updatedBy); err != nil {
			return err
		}
		changes["frontend_url"] = *req.FrontendURL
	}

	if len(changes) > 0 {
		// Notify subscribers for hot-reload; log warning if fails but don't return error
		// since database update was successful
		if err := s.settingProvider.NotifyChange(ctx, "system", changes); err != nil {
			s.logger.Warnw("failed to notify system setting changes", "error", err)
		}
	}
	return nil
}

// ============================================================================
// OAuth Settings
// ============================================================================

// GetOAuthSettings retrieves OAuth settings
func (s *ServiceDDD) GetOAuthSettings(ctx context.Context) (*dto.OAuthSettingsResponse, error) {
	provider := s.settingProvider

	googleCfg := provider.GetGoogleOAuthConfig(ctx)
	githubCfg := provider.GetGitHubOAuthConfig(ctx)

	return &dto.OAuthSettingsResponse{
		Google: dto.OAuthProviderSettings{
			Enabled:      googleCfg.ClientID != "" && googleCfg.ClientSecret != "",
			ClientID:     s.getSettingWithSource(ctx, "oauth_google", "client_id"),
			ClientSecret: s.getSettingWithSourceMasked(ctx, "oauth_google", "client_secret"),
			RedirectURL:  s.getSettingWithSource(ctx, "oauth_google", "redirect_url"),
		},
		GitHub: dto.OAuthProviderSettings{
			Enabled:      githubCfg.ClientID != "" && githubCfg.ClientSecret != "",
			ClientID:     s.getSettingWithSource(ctx, "oauth_github", "client_id"),
			ClientSecret: s.getSettingWithSourceMasked(ctx, "oauth_github", "client_secret"),
			RedirectURL:  s.getSettingWithSource(ctx, "oauth_github", "redirect_url"),
		},
	}, nil
}

// UpdateOAuthSettings updates OAuth settings
func (s *ServiceDDD) UpdateOAuthSettings(ctx context.Context, req dto.UpdateOAuthSettingsRequest, updatedBy uint) error {
	if req.Google != nil {
		changes := make(map[string]any)
		if req.Google.ClientID != nil {
			if err := s.upsertSetting(ctx, "oauth_google", "client_id", *req.Google.ClientID, updatedBy); err != nil {
				return err
			}
			changes["client_id"] = *req.Google.ClientID
		}
		if req.Google.ClientSecret != nil {
			if err := s.upsertSetting(ctx, "oauth_google", "client_secret", *req.Google.ClientSecret, updatedBy); err != nil {
				return err
			}
			changes["client_secret"] = "[REDACTED]"
		}
		if req.Google.RedirectURL != nil {
			if err := s.upsertSetting(ctx, "oauth_google", "redirect_url", *req.Google.RedirectURL, updatedBy); err != nil {
				return err
			}
			changes["redirect_url"] = *req.Google.RedirectURL
		}
		if len(changes) > 0 {
			if err := s.settingProvider.NotifyChange(ctx, "oauth_google", changes); err != nil {
				s.logger.Warnw("failed to notify oauth_google changes", "error", err)
			}
		}
	}

	if req.GitHub != nil {
		changes := make(map[string]any)
		if req.GitHub.ClientID != nil {
			if err := s.upsertSetting(ctx, "oauth_github", "client_id", *req.GitHub.ClientID, updatedBy); err != nil {
				return err
			}
			changes["client_id"] = *req.GitHub.ClientID
		}
		if req.GitHub.ClientSecret != nil {
			if err := s.upsertSetting(ctx, "oauth_github", "client_secret", *req.GitHub.ClientSecret, updatedBy); err != nil {
				return err
			}
			changes["client_secret"] = "[REDACTED]"
		}
		if req.GitHub.RedirectURL != nil {
			if err := s.upsertSetting(ctx, "oauth_github", "redirect_url", *req.GitHub.RedirectURL, updatedBy); err != nil {
				return err
			}
			changes["redirect_url"] = *req.GitHub.RedirectURL
		}
		if len(changes) > 0 {
			if err := s.settingProvider.NotifyChange(ctx, "oauth_github", changes); err != nil {
				s.logger.Warnw("failed to notify oauth_github changes", "error", err)
			}
		}
	}

	return nil
}

// ============================================================================
// Email Settings
// ============================================================================

// GetEmailSettings retrieves email settings
func (s *ServiceDDD) GetEmailSettings(ctx context.Context) (*dto.EmailSettingsResponse, error) {
	return &dto.EmailSettingsResponse{
		SMTPHost:     s.getSettingWithSource(ctx, "email", "smtp_host"),
		SMTPPort:     s.getSettingWithSource(ctx, "email", "smtp_port"),
		SMTPUser:     s.getSettingWithSource(ctx, "email", "smtp_user"),
		SMTPPassword: s.getSettingWithSourceMasked(ctx, "email", "smtp_password"),
		FromAddress:  s.getSettingWithSource(ctx, "email", "from_address"),
		FromName:     s.getSettingWithSource(ctx, "email", "from_name"),
	}, nil
}

// UpdateEmailSettings updates email settings
func (s *ServiceDDD) UpdateEmailSettings(ctx context.Context, req dto.UpdateEmailSettingsRequest, updatedBy uint) error {
	changes := make(map[string]any)

	if req.SMTPHost != nil {
		if err := s.upsertSetting(ctx, "email", "smtp_host", *req.SMTPHost, updatedBy); err != nil {
			return err
		}
		changes["smtp_host"] = *req.SMTPHost
	}
	if req.SMTPPort != nil {
		if err := s.upsertSettingInt(ctx, "email", "smtp_port", *req.SMTPPort, updatedBy); err != nil {
			return err
		}
		changes["smtp_port"] = *req.SMTPPort
	}
	if req.SMTPUser != nil {
		if err := s.upsertSetting(ctx, "email", "smtp_user", *req.SMTPUser, updatedBy); err != nil {
			return err
		}
		changes["smtp_user"] = *req.SMTPUser
	}
	if req.SMTPPassword != nil {
		if err := s.upsertSetting(ctx, "email", "smtp_password", *req.SMTPPassword, updatedBy); err != nil {
			return err
		}
		changes["smtp_password"] = "[REDACTED]"
	}
	if req.FromAddress != nil {
		if err := s.upsertSetting(ctx, "email", "from_address", *req.FromAddress, updatedBy); err != nil {
			return err
		}
		changes["from_address"] = *req.FromAddress
	}
	if req.FromName != nil {
		if err := s.upsertSetting(ctx, "email", "from_name", *req.FromName, updatedBy); err != nil {
			return err
		}
		changes["from_name"] = *req.FromName
	}

	if len(changes) > 0 {
		return s.settingProvider.NotifyChange(ctx, "email", changes)
	}
	return nil
}

// TestEmailConnection tests email SMTP connection by sending a test email
func (s *ServiceDDD) TestEmailConnection(_ context.Context, recipientEmail string) (*dto.EmailTestResponse, error) {
	if s.emailTester == nil {
		return &dto.EmailTestResponse{
			Success: false,
			Error:   "Email service not configured",
		}, nil
	}

	err := s.emailTester.SendTestEmail(recipientEmail)
	if err != nil {
		return &dto.EmailTestResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &dto.EmailTestResponse{
		Success: true,
	}, nil
}

// ============================================================================
// Setup Status
// ============================================================================

// GetSetupStatus checks if the system is configured
func (s *ServiceDDD) GetSetupStatus(ctx context.Context) (*dto.SetupStatusResponse, error) {
	missingSettings := []string{}

	apiBaseURL := s.settingProvider.GetAPIBaseURL(ctx)
	if apiBaseURL.Source == dto.SourceDefault {
		missingSettings = append(missingSettings, "api_base_url")
	}

	isConfigured := len(missingSettings) == 0

	return &dto.SetupStatusResponse{
		IsConfigured:    isConfigured,
		RequiresSetup:   !isConfigured,
		MissingSettings: missingSettings,
	}, nil
}
