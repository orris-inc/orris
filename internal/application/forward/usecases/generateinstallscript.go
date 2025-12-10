package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// InstallScriptURL is the URL of the install script hosted on GitHub
	InstallScriptURL = "https://raw.githubusercontent.com/orris-inc/orris-client/main/scripts/install.sh"
)

// GenerateInstallScriptQuery represents the input for generating install script.
type GenerateInstallScriptQuery struct {
	ShortID   string // External API identifier
	ServerURL string // WebSocket server URL (e.g., wss://example.com)
	Token     string // Optional: API token for the agent. If not provided, uses agent's current token
}

// GenerateInstallScriptResult represents the output of generating install script.
type GenerateInstallScriptResult struct {
	InstallCommand   string `json:"install_command"`
	UninstallCommand string `json:"uninstall_command"`
	ScriptURL        string `json:"script_url"`
	ServerURL        string `json:"server_url"`
	Token            string `json:"token"`
}

// GenerateInstallScriptUseCase handles generating install script for forward agent.
type GenerateInstallScriptUseCase struct {
	repo   forward.AgentRepository
	logger logger.Interface
}

// NewGenerateInstallScriptUseCase creates a new GenerateInstallScriptUseCase.
func NewGenerateInstallScriptUseCase(
	repo forward.AgentRepository,
	logger logger.Interface,
) *GenerateInstallScriptUseCase {
	return &GenerateInstallScriptUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute generates the install command for a forward agent.
func (uc *GenerateInstallScriptUseCase) Execute(ctx context.Context, query GenerateInstallScriptQuery) (*GenerateInstallScriptResult, error) {
	if query.ShortID == "" {
		return nil, errors.NewValidationError("short_id is required")
	}

	if query.ServerURL == "" {
		return nil, errors.NewValidationError("server URL is required")
	}

	uc.logger.Infow("executing generate install script use case", "short_id", query.ShortID)

	agent, err := uc.repo.GetByShortID(ctx, query.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "short_id", query.ShortID, "error", err)
		return nil, fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return nil, errors.NewNotFoundError("forward agent", query.ShortID)
	}

	// Use provided token or fall back to agent's stored token
	token := query.Token
	if token == "" {
		token = agent.GetAPIToken()
		if token == "" {
			return nil, errors.NewValidationError("agent has no token, please call regenerate-token endpoint first")
		}
	}

	// Generate install and uninstall commands
	installCmd := fmt.Sprintf("curl -fsSL %s | sudo bash -s -- --server %s --token %s", InstallScriptURL, query.ServerURL, token)
	uninstallCmd := fmt.Sprintf("curl -fsSL %s | sudo bash -s -- uninstall", InstallScriptURL)

	result := &GenerateInstallScriptResult{
		InstallCommand:   installCmd,
		UninstallCommand: uninstallCmd,
		ScriptURL:        InstallScriptURL,
		ServerURL:        query.ServerURL,
		Token:            token,
	}

	uc.logger.Infow("install command generated successfully", "agent_id", agent.ID(), "short_id", agent.ShortID())
	return result, nil
}
