package http

import (
	"context"

	settingUsecases "github.com/orris-inc/orris/internal/application/setting/usecases"
	telegramAdminUsecases "github.com/orris-inc/orris/internal/application/telegram/admin/usecases"
	"github.com/orris-inc/orris/internal/application/user/usecases"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/setting"
	sharedConfig "github.com/orris-inc/orris/internal/shared/config"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	telegramInfra "github.com/orris-inc/orris/internal/infrastructure/telegram"
	"github.com/orris-inc/orris/internal/shared/authorization"
)

// jwtServiceAdapter adapts JWTService to usecases.JWTService interface.
type jwtServiceAdapter struct {
	*auth.JWTService
}

func (a *jwtServiceAdapter) Generate(userUUID string, sessionID string, role authorization.UserRole) (*usecases.TokenPair, error) {
	pair, err := a.JWTService.Generate(userUUID, sessionID, role)
	if err != nil {
		return nil, err
	}
	return &usecases.TokenPair{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	}, nil
}

func (a *jwtServiceAdapter) Refresh(refreshToken string) (*usecases.TokenPair, error) {
	pair, err := a.JWTService.Refresh(refreshToken)
	if err != nil {
		return nil, err
	}
	return &usecases.TokenPair{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	}, nil
}

// dynamicOAuthClientAdapter wraps OAuthServiceManager to provide dynamic OAuth client access.
// This adapter fetches the current OAuth client from manager on each call, enabling hot-reload support.
type dynamicOAuthClientAdapter struct {
	manager  *auth.OAuthServiceManager
	provider string // "google" or "github"
}

func (a *dynamicOAuthClientAdapter) getClient() interface {
	GetAuthURL(state string) (authURL string, codeVerifier string, err error)
	ExchangeCode(ctx context.Context, code string, codeVerifier string) (string, error)
	GetUserInfo(ctx context.Context, accessToken string) (*auth.OAuthUserInfo, error)
} {
	switch a.provider {
	case "google":
		return a.manager.GetGoogleClient()
	case "github":
		return a.manager.GetGitHubClient()
	default:
		return nil
	}
}

func (a *dynamicOAuthClientAdapter) GetAuthURL(state string) (string, string, error) {
	client := a.getClient()
	if client == nil {
		return "", "", auth.ErrOAuthNotConfigured
	}
	return client.GetAuthURL(state)
}

func (a *dynamicOAuthClientAdapter) ExchangeCode(ctx context.Context, code string, codeVerifier string) (string, error) {
	client := a.getClient()
	if client == nil {
		return "", auth.ErrOAuthNotConfigured
	}
	return client.ExchangeCode(ctx, code, codeVerifier)
}

func (a *dynamicOAuthClientAdapter) GetUserInfo(ctx context.Context, accessToken string) (*usecases.OAuthUserInfo, error) {
	client := a.getClient()
	if client == nil {
		return nil, auth.ErrOAuthNotConfigured
	}
	info, err := client.GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	return &usecases.OAuthUserInfo{
		Email:         info.Email,
		Name:          info.Name,
		Picture:       info.Picture,
		EmailVerified: info.EmailVerified,
		Provider:      info.Provider,
		ProviderID:    info.ProviderID,
	}, nil
}

// nodeSIDResolverAdapter adapts node repository for SID resolution.
type nodeSIDResolverAdapter struct {
	repo node.NodeRepository
}

// GetSIDByID resolves node internal ID to Stripe-style SID.
func (a *nodeSIDResolverAdapter) GetSIDByID(nodeID uint) (string, bool) {
	ctx := context.Background()
	n, err := a.repo.GetByID(ctx, nodeID)
	if err != nil || n == nil {
		return "", false
	}
	return n.SID(), true
}

// agentSIDResolverAdapter adapts forward agent repository for SID resolution.
type agentSIDResolverAdapter struct {
	repo forward.AgentRepository
}

// GetSIDByID resolves forward agent internal ID to Stripe-style SID and name.
func (a *agentSIDResolverAdapter) GetSIDByID(agentID uint) (string, string, bool) {
	ctx := context.Background()
	agent, err := a.repo.GetByID(ctx, agentID)
	if err != nil || agent == nil {
		return "", "", false
	}
	return agent.SID(), agent.Name(), true
}

// botServiceProviderAdapter adapts BotServiceManager to satisfy telegramAdminApp.BotServiceProvider interface.
type botServiceProviderAdapter struct {
	manager *telegramInfra.BotServiceManager
}

// GetBotService returns the BotService as TelegramMessageSender interface.
func (a *botServiceProviderAdapter) GetBotService() telegramAdminUsecases.TelegramMessageSender {
	bs := a.manager.GetBotService()
	if bs == nil {
		return nil
	}
	return bs
}

// settingProviderAdapter adapts the application-layer *usecases.SettingProvider
// to the domain-layer setting.SettingProvider interface.
// This breaks the reverse dependency from infrastructure to application.
type settingProviderAdapter struct {
	provider *settingUsecases.SettingProvider
}

// Ensure compile-time interface compliance.
var _ setting.SettingProvider = (*settingProviderAdapter)(nil)

func (a *settingProviderAdapter) GetEmailConfig(ctx context.Context) sharedConfig.EmailConfig {
	return a.provider.GetEmailConfig(ctx)
}

func (a *settingProviderAdapter) GetAPIBaseURL(ctx context.Context) setting.ConfigValue {
	result := a.provider.GetAPIBaseURL(ctx)
	return setting.ConfigValue{
		Value:  result.Value,
		Source: string(result.Source),
	}
}

func (a *settingProviderAdapter) GetGoogleOAuthConfig(ctx context.Context) sharedConfig.GoogleOAuthConfig {
	return a.provider.GetGoogleOAuthConfig(ctx)
}

func (a *settingProviderAdapter) GetGitHubOAuthConfig(ctx context.Context) sharedConfig.GitHubOAuthConfig {
	return a.provider.GetGitHubOAuthConfig(ctx)
}

func (a *settingProviderAdapter) GetTelegramConfig(ctx context.Context) sharedConfig.TelegramConfig {
	return a.provider.GetTelegramConfig(ctx)
}

func (a *settingProviderAdapter) IsTelegramEnabled(ctx context.Context) bool {
	return a.provider.IsTelegramEnabled(ctx)
}
