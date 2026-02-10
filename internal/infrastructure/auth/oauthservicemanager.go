package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/orris-inc/orris/internal/domain/setting"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ErrOAuthNotConfigured is returned when OAuth client is not configured
var ErrOAuthNotConfigured = errors.New("oauth provider not configured")

// OAuthServiceManager manages OAuth clients with hot-reload support.
// It implements SettingChangeSubscriber to receive configuration change notifications.
type OAuthServiceManager struct {
	provider setting.SettingProvider
	logger   logger.Interface

	mu           sync.RWMutex
	googleClient *GoogleOAuthClient
	githubClient *GitHubOAuthClient
}

// NewOAuthServiceManager creates a new OAuthServiceManager instance.
func NewOAuthServiceManager(
	provider setting.SettingProvider,
	logger logger.Interface,
) *OAuthServiceManager {
	return &OAuthServiceManager{
		provider: provider,
		logger:   logger,
	}
}

// Initialize creates OAuth clients based on current configuration.
func (m *OAuthServiceManager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.initializeClientsLocked(ctx)
}

// initializeClientsLocked initializes OAuth clients. Must be called with m.mu held.
func (m *OAuthServiceManager) initializeClientsLocked(ctx context.Context) error {
	apiBaseURLSetting := m.provider.GetAPIBaseURL(ctx)
	apiBaseURL := ""
	if val, ok := apiBaseURLSetting.Value.(string); ok {
		apiBaseURL = val
	} else if apiBaseURLSetting.Value != nil {
		m.logger.Warnw("unexpected type for api_base_url setting",
			"expected", "string",
			"got_type", fmt.Sprintf("%T", apiBaseURLSetting.Value),
		)
	}

	// Initialize Google OAuth client
	googleCfg := m.provider.GetGoogleOAuthConfig(ctx)
	if googleCfg.ClientID != "" && googleCfg.ClientSecret != "" {
		redirectURL := googleCfg.GetRedirectURL(apiBaseURL)
		m.googleClient = NewGoogleOAuthClient(GoogleOAuthConfig{
			ClientID:     googleCfg.ClientID,
			ClientSecret: googleCfg.ClientSecret,
			RedirectURL:  redirectURL,
		})
		m.logger.Infow("google oauth client initialized",
			"redirect_url", redirectURL,
		)
	} else {
		m.googleClient = nil
		m.logger.Debugw("google oauth client not configured")
	}

	// Initialize GitHub OAuth client
	githubCfg := m.provider.GetGitHubOAuthConfig(ctx)
	if githubCfg.ClientID != "" && githubCfg.ClientSecret != "" {
		redirectURL := githubCfg.GetRedirectURL(apiBaseURL)
		m.githubClient = NewGitHubOAuthClient(GitHubOAuthConfig{
			ClientID:     githubCfg.ClientID,
			ClientSecret: githubCfg.ClientSecret,
			RedirectURL:  redirectURL,
		})
		m.logger.Infow("github oauth client initialized",
			"redirect_url", redirectURL,
		)
	} else {
		m.githubClient = nil
		m.logger.Debugw("github oauth client not configured")
	}

	return nil
}

// OnSettingChange handles configuration changes from the SettingProvider.
// It implements the SettingChangeSubscriber interface.
func (m *OAuthServiceManager) OnSettingChange(ctx context.Context, category string, changes map[string]any) error {
	needsReload := false

	switch category {
	case "system":
		// api_base_url change affects redirect URLs
		if _, ok := changes["api_base_url"]; ok {
			needsReload = true
		}
	case "oauth_google", "oauth_github":
		needsReload = true
	}

	if !needsReload {
		return nil
	}

	m.logger.Infow("oauth configuration changed, reinitializing clients",
		"category", category,
	)

	m.mu.Lock()
	defer m.mu.Unlock()
	return m.initializeClientsLocked(ctx)
}

// GetGoogleClient returns the current Google OAuth client.
// Returns nil if Google OAuth is not configured.
func (m *OAuthServiceManager) GetGoogleClient() *GoogleOAuthClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.googleClient
}

// GetGitHubClient returns the current GitHub OAuth client.
// Returns nil if GitHub OAuth is not configured.
func (m *OAuthServiceManager) GetGitHubClient() *GitHubOAuthClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.githubClient
}

// IsGoogleEnabled checks if Google OAuth is configured and available.
func (m *OAuthServiceManager) IsGoogleEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.googleClient != nil
}

// IsGitHubEnabled checks if GitHub OAuth is configured and available.
func (m *OAuthServiceManager) IsGitHubEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.githubClient != nil
}
