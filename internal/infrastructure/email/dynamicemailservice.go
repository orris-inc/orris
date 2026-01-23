package email

import (
	"github.com/orris-inc/orris/internal/shared/logger"
)

// DynamicEmailService wraps EmailServiceManager to provide transparent hot-reload
// It implements the same interface as SMTPEmailService but delegates to the managed service
type DynamicEmailService struct {
	manager *EmailServiceManager
	logger  logger.Interface
}

// NewDynamicEmailService creates a new DynamicEmailService
func NewDynamicEmailService(manager *EmailServiceManager, logger logger.Interface) *DynamicEmailService {
	return &DynamicEmailService{
		manager: manager,
		logger:  logger,
	}
}

// SendVerificationEmail sends a verification email
func (d *DynamicEmailService) SendVerificationEmail(to, token string) error {
	service := d.manager.GetService()
	if service == nil {
		d.logger.Warnw("email service not configured, cannot send verification email", "to", to)
		return ErrEmailServiceNotConfigured
	}
	return service.SendVerificationEmail(to, token)
}

// SendPasswordResetEmail sends a password reset email
func (d *DynamicEmailService) SendPasswordResetEmail(to, token string) error {
	service := d.manager.GetService()
	if service == nil {
		d.logger.Warnw("email service not configured, cannot send password reset email", "to", to)
		return ErrEmailServiceNotConfigured
	}
	return service.SendPasswordResetEmail(to, token)
}

// SendPasswordChangedEmail sends a password changed notification email
func (d *DynamicEmailService) SendPasswordChangedEmail(to string) error {
	service := d.manager.GetService()
	if service == nil {
		d.logger.Warnw("email service not configured, cannot send password changed email", "to", to)
		return ErrEmailServiceNotConfigured
	}
	return service.SendPasswordChangedEmail(to)
}

// SendTestEmail sends a test email to verify the configuration
func (d *DynamicEmailService) SendTestEmail(to string) error {
	service := d.manager.GetService()
	if service == nil {
		return ErrEmailServiceNotConfigured
	}
	return service.SendTestEmail(to)
}
