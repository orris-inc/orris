package email

import (
	"context"
	"fmt"
	"sync"

	settingUsecases "github.com/orris-inc/orris/internal/application/setting/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// EmailServiceManager manages email service with hot-reload support
type EmailServiceManager struct {
	provider *settingUsecases.SettingProvider
	logger   logger.Interface

	mu      sync.RWMutex
	service *SMTPEmailService
}

// NewEmailServiceManager creates a new EmailServiceManager
func NewEmailServiceManager(
	provider *settingUsecases.SettingProvider,
	logger logger.Interface,
) *EmailServiceManager {
	return &EmailServiceManager{
		provider: provider,
		logger:   logger,
	}
}

// Initialize creates email service based on current configuration
func (m *EmailServiceManager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.initializeServiceLocked(ctx)
}

func (m *EmailServiceManager) initializeServiceLocked(ctx context.Context) error {
	emailCfg := m.provider.GetEmailConfig(ctx)
	apiBaseURL := m.provider.GetAPIBaseURL(ctx)

	// Only initialize if SMTP host is configured
	if emailCfg.SMTPHost == "" {
		m.service = nil
		m.logger.Debugw("email service not configured, smtp_host is empty")
		return nil
	}

	// Safe type assertion for baseURL with logging
	baseURL := ""
	if val, ok := apiBaseURL.Value.(string); ok {
		baseURL = val
	} else if apiBaseURL.Value != nil {
		m.logger.Warnw("unexpected type for api_base_url setting",
			"expected", "string",
			"got_type", fmt.Sprintf("%T", apiBaseURL.Value),
		)
	}

	smtpCfg := SMTPConfig{
		Host:        emailCfg.SMTPHost,
		Port:        emailCfg.SMTPPort,
		Username:    emailCfg.SMTPUser,
		Password:    emailCfg.SMTPPassword,
		FromAddress: emailCfg.FromAddress,
		FromName:    emailCfg.FromName,
		BaseURL:     baseURL,
	}

	m.service = NewSMTPEmailService(smtpCfg)
	m.logger.Infow("email service initialized",
		"host", smtpCfg.Host,
		"port", smtpCfg.Port,
		"from", smtpCfg.FromAddress,
	)

	return nil
}

// OnSettingChange handles configuration changes
// Implements SettingChangeSubscriber interface
func (m *EmailServiceManager) OnSettingChange(ctx context.Context, category string, changes map[string]any) error {
	needsReload := false
	switch category {
	case "system":
		if _, ok := changes["api_base_url"]; ok {
			needsReload = true
		}
	case "email":
		needsReload = true
	}

	if needsReload {
		m.logger.Infow("email configuration changed, reinitializing service",
			"category", category,
		)
		m.mu.Lock()
		defer m.mu.Unlock()
		return m.initializeServiceLocked(ctx)
	}

	return nil
}

// GetService returns the current email service
func (m *EmailServiceManager) GetService() *SMTPEmailService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.service
}

// IsConfigured checks if email service is configured
func (m *EmailServiceManager) IsConfigured() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.service != nil
}
