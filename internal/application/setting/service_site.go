package setting

import (
	"context"
	"fmt"
	"strings"

	"github.com/orris-inc/orris/internal/application/setting/dto"
	userDomain "github.com/orris-inc/orris/internal/domain/user"
	userVO "github.com/orris-inc/orris/internal/domain/user/valueobjects"
)

// ============================================================================
// Branding Settings
// ============================================================================

// GetBrandingSettings retrieves branding settings (admin)
func (s *ServiceDDD) GetBrandingSettings(ctx context.Context) (*dto.BrandingSettingsResponse, error) {
	return &dto.BrandingSettingsResponse{
		AppName:    s.getSettingWithSourceDefault(ctx, "branding", "app_name", "Orris"),
		LogoURL:    s.getSettingWithSource(ctx, "branding", "logo_url"),
		FaviconURL: s.getSettingWithSource(ctx, "branding", "favicon_url"),
	}, nil
}

// UpdateBrandingSettings updates branding settings
func (s *ServiceDDD) UpdateBrandingSettings(ctx context.Context, req dto.UpdateBrandingSettingsRequest, updatedBy uint) error {
	changes := make(map[string]any)

	if req.AppName != nil {
		if err := s.upsertSetting(ctx, "branding", "app_name", *req.AppName, updatedBy); err != nil {
			return err
		}
		changes["app_name"] = *req.AppName
	}
	if req.LogoURL != nil {
		if err := validateBrandingURL(*req.LogoURL); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, "branding", "logo_url", *req.LogoURL, updatedBy); err != nil {
			return err
		}
		changes["logo_url"] = *req.LogoURL
	}
	if req.FaviconURL != nil {
		if err := validateBrandingURL(*req.FaviconURL); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, "branding", "favicon_url", *req.FaviconURL, updatedBy); err != nil {
			return err
		}
		changes["favicon_url"] = *req.FaviconURL
	}

	if len(changes) > 0 {
		if err := s.settingProvider.NotifyChange(ctx, "branding", changes); err != nil {
			s.logger.Warnw("failed to notify branding setting changes", "error", err)
		}
	}
	return nil
}

// GetPublicBranding retrieves public branding config (no auth required)
func (s *ServiceDDD) GetPublicBranding(ctx context.Context) (*dto.PublicBrandingResponse, error) {
	appName := s.getSettingWithSourceDefault(ctx, "branding", "app_name", "Orris")
	logoURL := s.getSettingWithSource(ctx, "branding", "logo_url")
	faviconURL := s.getSettingWithSource(ctx, "branding", "favicon_url")

	return &dto.PublicBrandingResponse{
		AppName:    s.valueToString(appName.Value, "Orris"),
		LogoURL:    s.valueToString(logoURL.Value, ""),
		FaviconURL: s.valueToString(faviconURL.Value, ""),
	}, nil
}

// ============================================================================
// Security Settings
// ============================================================================

// GetSecuritySettings retrieves security settings
func (s *ServiceDDD) GetSecuritySettings(ctx context.Context) (*dto.SecuritySettingsResponse, error) {
	return &dto.SecuritySettingsResponse{
		// Password policy
		PasswordMinLength:        s.getSettingWithSourceInt(ctx, "security", "password_min_length", 8),
		PasswordRequireUppercase: s.getSettingWithSourceBool(ctx, "security", "password_require_uppercase"),
		PasswordRequireLowercase: s.getSettingWithSourceBool(ctx, "security", "password_require_lowercase"),
		PasswordRequireNumber:    s.getSettingWithSourceBool(ctx, "security", "password_require_number"),
		PasswordRequireSpecial:   s.getSettingWithSourceBool(ctx, "security", "password_require_special"),
		PasswordExpiryDays:       s.getSettingWithSourceInt(ctx, "security", "password_expiry_days", 0),

		// Session settings
		SessionTimeoutMinutes: s.getSettingWithSourceInt(ctx, "security", "session_timeout_minutes", 1440),

		// Login protection
		MaxLoginAttempts:       s.getSettingWithSourceInt(ctx, "security", "max_login_attempts", 5),
		LockoutDurationMinutes: s.getSettingWithSourceInt(ctx, "security", "lockout_duration_minutes", 15),
	}, nil
}

// UpdateSecuritySettings updates security settings
func (s *ServiceDDD) UpdateSecuritySettings(ctx context.Context, req dto.UpdateSecuritySettingsRequest, updatedBy uint) error {
	changes := make(map[string]any)

	// Password policy
	if req.PasswordMinLength != nil {
		if err := s.upsertSettingInt(ctx, "security", "password_min_length", *req.PasswordMinLength, updatedBy); err != nil {
			return err
		}
		changes["password_min_length"] = *req.PasswordMinLength
	}
	if req.PasswordRequireUppercase != nil {
		if err := s.upsertSettingBool(ctx, "security", "password_require_uppercase", *req.PasswordRequireUppercase, updatedBy); err != nil {
			return err
		}
		changes["password_require_uppercase"] = *req.PasswordRequireUppercase
	}
	if req.PasswordRequireLowercase != nil {
		if err := s.upsertSettingBool(ctx, "security", "password_require_lowercase", *req.PasswordRequireLowercase, updatedBy); err != nil {
			return err
		}
		changes["password_require_lowercase"] = *req.PasswordRequireLowercase
	}
	if req.PasswordRequireNumber != nil {
		if err := s.upsertSettingBool(ctx, "security", "password_require_number", *req.PasswordRequireNumber, updatedBy); err != nil {
			return err
		}
		changes["password_require_number"] = *req.PasswordRequireNumber
	}
	if req.PasswordRequireSpecial != nil {
		if err := s.upsertSettingBool(ctx, "security", "password_require_special", *req.PasswordRequireSpecial, updatedBy); err != nil {
			return err
		}
		changes["password_require_special"] = *req.PasswordRequireSpecial
	}
	if req.PasswordExpiryDays != nil {
		if err := s.upsertSettingInt(ctx, "security", "password_expiry_days", *req.PasswordExpiryDays, updatedBy); err != nil {
			return err
		}
		changes["password_expiry_days"] = *req.PasswordExpiryDays
	}

	// Session settings
	if req.SessionTimeoutMinutes != nil {
		if err := s.upsertSettingInt(ctx, "security", "session_timeout_minutes", *req.SessionTimeoutMinutes, updatedBy); err != nil {
			return err
		}
		changes["session_timeout_minutes"] = *req.SessionTimeoutMinutes
	}

	// Login protection
	if req.MaxLoginAttempts != nil {
		if err := s.upsertSettingInt(ctx, "security", "max_login_attempts", *req.MaxLoginAttempts, updatedBy); err != nil {
			return err
		}
		changes["max_login_attempts"] = *req.MaxLoginAttempts
	}
	if req.LockoutDurationMinutes != nil {
		if err := s.upsertSettingInt(ctx, "security", "lockout_duration_minutes", *req.LockoutDurationMinutes, updatedBy); err != nil {
			return err
		}
		changes["lockout_duration_minutes"] = *req.LockoutDurationMinutes
	}

	if len(changes) > 0 {
		if err := s.settingProvider.NotifyChange(ctx, "security", changes); err != nil {
			s.logger.Warnw("failed to notify security setting changes", "error", err)
		}
	}
	return nil
}

// GetPasswordPolicy retrieves password policy from settings
// Implements user.PasswordPolicyProvider interface
func (s *ServiceDDD) GetPasswordPolicy(ctx context.Context) *userVO.PasswordPolicy {
	return &userVO.PasswordPolicy{
		MinLength:        s.getIntValue(ctx, "security", "password_min_length", 8),
		RequireUppercase: s.getBoolValue(ctx, "security", "password_require_uppercase", false),
		RequireLowercase: s.getBoolValue(ctx, "security", "password_require_lowercase", false),
		RequireNumber:    s.getBoolValue(ctx, "security", "password_require_number", false),
		RequireSpecial:   s.getBoolValue(ctx, "security", "password_require_special", false),
	}
}

// GetSecurityPolicy retrieves security policy from settings
// Implements user.SecurityPolicyProvider interface
func (s *ServiceDDD) GetSecurityPolicy(ctx context.Context) *userDomain.SecurityPolicy {
	return &userDomain.SecurityPolicy{
		MaxLoginAttempts:       s.getIntValue(ctx, "security", "max_login_attempts", 5),
		LockoutDurationMinutes: s.getIntValue(ctx, "security", "lockout_duration_minutes", 15),
	}
}

// GetPublicPasswordPolicy retrieves public password policy (no auth required)
// Used by frontend to display password requirements during registration/password change
func (s *ServiceDDD) GetPublicPasswordPolicy(ctx context.Context) (*dto.PublicPasswordPolicyResponse, error) {
	minLength := s.getIntValue(ctx, "security", "password_min_length", 8)
	maxLength := 72 // bcrypt limitation
	requireUppercase := s.getBoolValue(ctx, "security", "password_require_uppercase", false)
	requireLowercase := s.getBoolValue(ctx, "security", "password_require_lowercase", false)
	requireNumber := s.getBoolValue(ctx, "security", "password_require_number", false)
	requireSpecial := s.getBoolValue(ctx, "security", "password_require_special", false)

	// Build rules array for frontend display
	rules := []dto.PasswordPolicyRule{
		{Type: "min_length", Value: minLength, Message: fmt.Sprintf("At least %d characters", minLength)},
		{Type: "max_length", Value: maxLength, Message: fmt.Sprintf("At most %d characters", maxLength)},
	}

	if requireUppercase {
		rules = append(rules, dto.PasswordPolicyRule{
			Type: "uppercase", Required: true, Message: "At least one uppercase letter (A-Z)",
		})
	}
	if requireLowercase {
		rules = append(rules, dto.PasswordPolicyRule{
			Type: "lowercase", Required: true, Message: "At least one lowercase letter (a-z)",
		})
	}
	if requireNumber {
		rules = append(rules, dto.PasswordPolicyRule{
			Type: "number", Required: true, Message: "At least one number (0-9)",
		})
	}
	if requireSpecial {
		rules = append(rules, dto.PasswordPolicyRule{
			Type: "special", Required: true, Message: "At least one special character",
		})
	}

	return &dto.PublicPasswordPolicyResponse{
		MinLength:         minLength,
		MaxLength:         maxLength,
		RequireUppercase:  requireUppercase,
		RequireLowercase:  requireLowercase,
		RequireNumber:     requireNumber,
		RequireSpecial:    requireSpecial,
		Rules:             rules,
		SpecialCharacters: "!@#$%^&*()_+-=[]{}|;':\",./<>?`~",
	}, nil
}

// ============================================================================
// Registration Settings
// ============================================================================

// GetRegistrationSettings retrieves registration settings
func (s *ServiceDDD) GetRegistrationSettings(ctx context.Context) (*dto.RegistrationSettingsResponse, error) {
	return &dto.RegistrationSettingsResponse{
		RegistrationEnabled:       s.getSettingWithSourceBoolDefault(ctx, "registration", "registration_enabled", true),
		EmailVerificationRequired: s.getSettingWithSourceBoolDefault(ctx, "registration", "email_verification_required", true),
	}, nil
}

// UpdateRegistrationSettings updates registration settings
func (s *ServiceDDD) UpdateRegistrationSettings(ctx context.Context, req dto.UpdateRegistrationSettingsRequest, updatedBy uint) error {
	changes := make(map[string]any)

	if req.RegistrationEnabled != nil {
		if err := s.upsertSettingBool(ctx, "registration", "registration_enabled", *req.RegistrationEnabled, updatedBy); err != nil {
			return err
		}
		changes["registration_enabled"] = *req.RegistrationEnabled
	}
	if req.EmailVerificationRequired != nil {
		if err := s.upsertSettingBool(ctx, "registration", "email_verification_required", *req.EmailVerificationRequired, updatedBy); err != nil {
			return err
		}
		changes["email_verification_required"] = *req.EmailVerificationRequired
	}

	if len(changes) > 0 {
		if err := s.settingProvider.NotifyChange(ctx, "registration", changes); err != nil {
			s.logger.Warnw("failed to notify registration setting changes", "error", err)
		}
	}
	return nil
}

// GetPublicRegistration retrieves public registration settings (no auth required)
func (s *ServiceDDD) GetPublicRegistration(ctx context.Context) (*dto.PublicRegistrationResponse, error) {
	regEnabled := s.getSettingWithSourceBoolDefault(ctx, "registration", "registration_enabled", true)
	emailRequired := s.getSettingWithSourceBoolDefault(ctx, "registration", "email_verification_required", true)

	return &dto.PublicRegistrationResponse{
		RegistrationEnabled:       s.valueToBool(regEnabled.Value, true),
		EmailVerificationRequired: s.valueToBool(emailRequired.Value, true),
	}, nil
}

// ============================================================================
// Legal Settings
// ============================================================================

// GetLegalSettings retrieves legal settings
func (s *ServiceDDD) GetLegalSettings(ctx context.Context) (*dto.LegalSettingsResponse, error) {
	return &dto.LegalSettingsResponse{
		TermsOfServiceURL: s.getSettingWithSource(ctx, "legal", "terms_of_service_url"),
		PrivacyPolicyURL:  s.getSettingWithSource(ctx, "legal", "privacy_policy_url"),
	}, nil
}

// UpdateLegalSettings updates legal settings
func (s *ServiceDDD) UpdateLegalSettings(ctx context.Context, req dto.UpdateLegalSettingsRequest, updatedBy uint) error {
	changes := make(map[string]any)

	if req.TermsOfServiceURL != nil {
		if err := s.upsertSetting(ctx, "legal", "terms_of_service_url", *req.TermsOfServiceURL, updatedBy); err != nil {
			return err
		}
		changes["terms_of_service_url"] = *req.TermsOfServiceURL
	}
	if req.PrivacyPolicyURL != nil {
		if err := s.upsertSetting(ctx, "legal", "privacy_policy_url", *req.PrivacyPolicyURL, updatedBy); err != nil {
			return err
		}
		changes["privacy_policy_url"] = *req.PrivacyPolicyURL
	}

	if len(changes) > 0 {
		if err := s.settingProvider.NotifyChange(ctx, "legal", changes); err != nil {
			s.logger.Warnw("failed to notify legal setting changes", "error", err)
		}
	}
	return nil
}

// GetPublicLegal retrieves public legal URLs (no auth required)
func (s *ServiceDDD) GetPublicLegal(ctx context.Context) (*dto.PublicLegalResponse, error) {
	tosURL := s.getSettingWithSource(ctx, "legal", "terms_of_service_url")
	privacyURL := s.getSettingWithSource(ctx, "legal", "privacy_policy_url")

	return &dto.PublicLegalResponse{
		TermsOfServiceURL: s.valueToString(tosURL.Value, ""),
		PrivacyPolicyURL:  s.valueToString(privacyURL.Value, ""),
	}, nil
}

// ============================================================================
// Helper Methods (site-specific)
// ============================================================================

// getSettingWithSourceDefault retrieves a setting with a default value
func (s *ServiceDDD) getSettingWithSourceDefault(ctx context.Context, category, key, defaultValue string) dto.SettingWithSource {
	result := s.getSettingWithSource(ctx, category, key)
	if result.Source == dto.SourceDefault {
		if result.Value == nil {
			result.Value = defaultValue
		} else if strVal, ok := result.Value.(string); ok && strVal == "" {
			result.Value = defaultValue
		}
	}
	return result
}

// valueToString converts any value to string with default
func (s *ServiceDDD) valueToString(value any, defaultValue string) string {
	if value == nil {
		return defaultValue
	}
	if str, ok := value.(string); ok {
		if str == "" {
			return defaultValue
		}
		return str
	}
	return defaultValue
}

// getSettingWithSourceBoolDefault retrieves a bool setting value with its source and default
func (s *ServiceDDD) getSettingWithSourceBoolDefault(ctx context.Context, category, key string, defaultVal bool) dto.SettingWithSource {
	existing, err := s.getSettingsUC.GetSettingByKey(ctx, category, key)
	if err == nil && existing != nil && existing.HasValue() {
		val, err := existing.GetBoolValue()
		if err == nil {
			return dto.SettingWithSource{
				Value:  val,
				Source: dto.SourceDatabase,
			}
		}
	}
	return dto.SettingWithSource{
		Value:  defaultVal,
		Source: dto.SourceDefault,
	}
}

// valueToBool converts any value to bool with default
func (s *ServiceDDD) valueToBool(value any, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	if b, ok := value.(bool); ok {
		return b
	}
	return defaultValue
}

// getIntValue is a helper to get int setting value with default
func (s *ServiceDDD) getIntValue(ctx context.Context, category, key string, defaultVal int) int {
	existing, err := s.getSettingsUC.GetSettingByKey(ctx, category, key)
	if err == nil && existing != nil && existing.HasValue() {
		val, err := existing.GetIntValue()
		if err == nil {
			return val
		}
	}
	return defaultVal
}

// getBoolValue is a helper to get bool setting value with default
func (s *ServiceDDD) getBoolValue(ctx context.Context, category, key string, defaultVal bool) bool {
	existing, err := s.getSettingsUC.GetSettingByKey(ctx, category, key)
	if err == nil && existing != nil && existing.HasValue() {
		val, err := existing.GetBoolValue()
		if err == nil {
			return val
		}
	}
	return defaultVal
}

// validateBrandingURL validates branding URL to prevent security issues
// Only allows empty string or paths starting with /uploads/branding/
func validateBrandingURL(url string) error {
	// Empty URL is allowed (to clear the setting)
	if url == "" {
		return nil
	}

	// Must start with /uploads/branding/ (local upload path only)
	if !strings.HasPrefix(url, "/uploads/branding/") {
		return fmt.Errorf("invalid URL: must be a local upload path starting with /uploads/branding/")
	}

	// Prevent path traversal attacks
	if strings.Contains(url, "..") {
		return fmt.Errorf("invalid URL: path traversal not allowed")
	}

	// Validate filename format (timestamp_random.ext)
	filename := strings.TrimPrefix(url, "/uploads/branding/")
	if filename == "" || strings.Contains(filename, "/") {
		return fmt.Errorf("invalid URL: invalid filename")
	}

	return nil
}
