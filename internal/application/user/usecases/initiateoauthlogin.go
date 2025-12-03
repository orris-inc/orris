package usecases

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// StateStore defines the interface for OAuth state storage
type StateStore interface {
	Set(ctx context.Context, state string, codeVerifier string) error
	VerifyAndGet(ctx context.Context, state string) (*cache.StateInfo, error)
}

type OAuthClient interface {
	GetAuthURL(state string) (authURL string, codeVerifier string, err error)
}

type InitiateOAuthLoginCommand struct {
	Provider string
}

type InitiateOAuthLoginResult struct {
	AuthURL string
	State   string
}

type InitiateOAuthLoginUseCase struct {
	googleClient OAuthClient
	githubClient OAuthClient
	logger       logger.Interface
	stateStore   StateStore
}

func NewInitiateOAuthLoginUseCase(
	googleClient OAuthClient,
	githubClient OAuthClient,
	logger logger.Interface,
	stateStore StateStore,
) *InitiateOAuthLoginUseCase {
	return &InitiateOAuthLoginUseCase{
		googleClient: googleClient,
		githubClient: githubClient,
		logger:       logger,
		stateStore:   stateStore,
	}
}

func (uc *InitiateOAuthLoginUseCase) Execute(cmd InitiateOAuthLoginCommand) (*InitiateOAuthLoginResult, error) {
	state, err := generateState()
	if err != nil {
		uc.logger.Errorw("failed to generate state", "error", err)
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	var client OAuthClient
	switch cmd.Provider {
	case "google":
		client = uc.googleClient
	case "github":
		client = uc.githubClient
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", cmd.Provider)
	}

	// Get auth URL with PKCE parameters
	authURL, codeVerifier, err := client.GetAuthURL(state)
	if err != nil {
		uc.logger.Errorw("failed to get auth URL", "error", err, "provider", cmd.Provider)
		return nil, fmt.Errorf("failed to get auth URL: %w", err)
	}

	// Store state and code_verifier in Redis
	ctx := context.TODO()
	if err := uc.stateStore.Set(ctx, state, codeVerifier); err != nil {
		uc.logger.Errorw("failed to store OAuth state", "error", err, "state", state)
		return nil, fmt.Errorf("failed to store state: %w", err)
	}

	uc.logger.Infow("OAuth login initiated", "provider", cmd.Provider, "state", state)

	return &InitiateOAuthLoginResult{
		AuthURL: authURL,
		State:   state,
	}, nil
}

// VerifyStateAndGetVerifier verifies state and retrieves code_verifier from Redis
func (uc *InitiateOAuthLoginUseCase) VerifyStateAndGetVerifier(ctx context.Context, state string) (*cache.StateInfo, error) {
	stateInfo, err := uc.stateStore.VerifyAndGet(ctx, state)
	if err != nil {
		uc.logger.Warnw("invalid or expired OAuth state", "state", state, "error", err)
		return nil, fmt.Errorf("invalid or expired state parameter")
	}
	return stateInfo, nil
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
