package email

import (
	"fmt"

	"gopkg.in/gomail.v2"
)

type SMTPConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	FromAddress string
	FromName    string
	BaseURL     string // Base URL for email links (e.g., "http://localhost:8081")
}

type SMTPEmailService struct {
	config SMTPConfig
	dialer *gomail.Dialer
}

func NewSMTPEmailService(config SMTPConfig) *SMTPEmailService {
	dialer := gomail.NewDialer(config.Host, config.Port, config.Username, config.Password)

	return &SMTPEmailService{
		config: config,
		dialer: dialer,
	}
}

func (s *SMTPEmailService) SendVerificationEmail(to, token string) error {
	verificationURL := fmt.Sprintf("%s/auth/verify-email?token=%s", s.config.BaseURL, token)

	subject := "Verify Your Email Address"
	htmlBody := fmt.Sprintf(`
		<html>
		<body>
			<h2>Welcome to Orris!</h2>
			<p>Please verify your email address by clicking the link below:</p>
			<p><a href="%s">Verify Email Address</a></p>
			<p>Or copy and paste this URL into your browser:</p>
			<p>%s</p>
			<p>This link will expire in 24 hours.</p>
			<p>If you didn't create an account, please ignore this email.</p>
		</body>
		</html>
	`, verificationURL, verificationURL)

	plainBody := fmt.Sprintf(`
Welcome to Orris!

Please verify your email address by visiting:
%s

This link will expire in 24 hours.

If you didn't create an account, please ignore this email.
	`, verificationURL)

	return s.sendEmail(to, subject, htmlBody, plainBody)
}

func (s *SMTPEmailService) SendPasswordResetEmail(to, token string) error {
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", s.config.BaseURL, token)

	subject := "Reset Your Password"
	htmlBody := fmt.Sprintf(`
		<html>
		<body>
			<h2>Password Reset Request</h2>
			<p>We received a request to reset your password. Click the link below to reset it:</p>
			<p><a href="%s">Reset Password</a></p>
			<p>Or copy and paste this URL into your browser:</p>
			<p>%s</p>
			<p>This link will expire in 30 minutes.</p>
			<p>If you didn't request a password reset, please ignore this email and your password will remain unchanged.</p>
		</body>
		</html>
	`, resetURL, resetURL)

	plainBody := fmt.Sprintf(`
Password Reset Request

We received a request to reset your password. Visit the following URL to reset it:
%s

This link will expire in 30 minutes.

If you didn't request a password reset, please ignore this email and your password will remain unchanged.
	`, resetURL)

	return s.sendEmail(to, subject, htmlBody, plainBody)
}

func (s *SMTPEmailService) SendPasswordChangedEmail(to string) error {
	subject := "Password Changed Successfully"
	htmlBody := `
		<html>
		<body>
			<h2>Password Changed</h2>
			<p>Your password has been successfully changed.</p>
			<p>If you didn't make this change, please contact support immediately.</p>
		</body>
		</html>
	`

	plainBody := `
Password Changed

Your password has been successfully changed.

If you didn't make this change, please contact support immediately.
	`

	return s.sendEmail(to, subject, htmlBody, plainBody)
}

func (s *SMTPEmailService) sendEmail(to, subject, htmlBody, plainBody string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.config.FromAddress)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", plainBody)
	m.AddAlternative("text/html", htmlBody)

	if err := s.dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
