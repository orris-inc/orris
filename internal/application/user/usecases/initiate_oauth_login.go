package usecases

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"orris/internal/shared/logger"
)

type OAuthClient interface {
	GetAuthURL(state string) string
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
	stateStore   map[string]time.Time
}

func NewInitiateOAuthLoginUseCase(
	googleClient OAuthClient,
	githubClient OAuthClient,
	logger logger.Interface,
) *InitiateOAuthLoginUseCase {
	return &InitiateOAuthLoginUseCase{
		googleClient: googleClient,
		githubClient: githubClient,
		logger:       logger,
		stateStore:   make(map[string]time.Time),
	}
}

func (uc *InitiateOAuthLoginUseCase) Execute(cmd InitiateOAuthLoginCommand) (*InitiateOAuthLoginResult, error) {
	state, err := generateState()
	if err != nil {
		uc.logger.Errorw("failed to generate state", "error", err)
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	uc.stateStore[state] = time.Now().Add(10 * time.Minute)

	var client OAuthClient
	switch cmd.Provider {
	case "google":
		client = uc.googleClient
	case "github":
		client = uc.githubClient
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", cmd.Provider)
	}

	authURL := client.GetAuthURL(state)

	uc.logger.Infow("OAuth login initiated", "provider", cmd.Provider)

	return &InitiateOAuthLoginResult{
		AuthURL: authURL,
		State:   state,
	}, nil
}

func (uc *InitiateOAuthLoginUseCase) VerifyState(state string) bool {
	expiry, exists := uc.stateStore[state]
	if !exists {
		return false
	}

	if time.Now().After(expiry) {
		delete(uc.stateStore, state)
		return false
	}

	delete(uc.stateStore, state)
	return true
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
