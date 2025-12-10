package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// NodeInstallScriptURL is the URL of the install script hosted on GitHub
	NodeInstallScriptURL = "https://raw.githubusercontent.com/orris-inc/orrisp/main/scripts/install.sh"
)

// GenerateNodeInstallScriptQuery represents the input for generating node install script.
type GenerateNodeInstallScriptQuery struct {
	ShortID string // External API identifier
	APIURL  string // API server URL (e.g., https://api.example.com)
	Token   string // Optional: API token for the node. If not provided, uses node's current token
}

// GenerateNodeInstallScriptResult represents the output of generating node install script.
type GenerateNodeInstallScriptResult struct {
	InstallCommand   string `json:"install_command"`
	UninstallCommand string `json:"uninstall_command"`
	ScriptURL        string `json:"script_url"`
	APIURL           string `json:"api_url"`
	NodeID           uint   `json:"node_id"`
	Token            string `json:"token"`
}

// GenerateNodeInstallScriptExecutor defines the interface for generating node install script.
type GenerateNodeInstallScriptExecutor interface {
	Execute(ctx context.Context, query GenerateNodeInstallScriptQuery) (*GenerateNodeInstallScriptResult, error)
}

// GenerateNodeInstallScriptUseCase handles generating install script for node.
type GenerateNodeInstallScriptUseCase struct {
	repo   node.NodeRepository
	logger logger.Interface
}

// NewGenerateNodeInstallScriptUseCase creates a new GenerateNodeInstallScriptUseCase.
func NewGenerateNodeInstallScriptUseCase(
	repo node.NodeRepository,
	logger logger.Interface,
) *GenerateNodeInstallScriptUseCase {
	return &GenerateNodeInstallScriptUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute generates the install command for a node.
func (uc *GenerateNodeInstallScriptUseCase) Execute(ctx context.Context, query GenerateNodeInstallScriptQuery) (*GenerateNodeInstallScriptResult, error) {
	uc.logger.Infow("executing generate node install script use case", "short_id", query.ShortID)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid generate node install script query", "error", err, "short_id", query.ShortID)
		return nil, err
	}

	// Get the node
	n, err := uc.repo.GetByShortID(ctx, query.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get node by short ID", "short_id", query.ShortID, "error", err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	if n == nil {
		return nil, errors.NewNotFoundError("node", query.ShortID)
	}

	// Use provided token or fall back to node's stored token
	token := query.Token
	if token == "" {
		token = n.GetAPIToken()
		if token == "" {
			return nil, errors.NewValidationError("node has no token, please call generate token endpoint first")
		}
	}

	nodeID := n.ID()

	// Generate install and uninstall commands
	// Format: curl -fsSL <script_url> | sudo bash -s -- --api-url <api_url> --node <id>:<token>
	nodeArg := fmt.Sprintf("%d:%s", nodeID, token)
	installCmd := fmt.Sprintf("curl -fsSL %s | sudo bash -s -- --api-url %s --node %s", NodeInstallScriptURL, query.APIURL, nodeArg)
	uninstallCmd := fmt.Sprintf("curl -fsSL %s | sudo bash -s -- uninstall", NodeInstallScriptURL)

	result := &GenerateNodeInstallScriptResult{
		InstallCommand:   installCmd,
		UninstallCommand: uninstallCmd,
		ScriptURL:        NodeInstallScriptURL,
		APIURL:           query.APIURL,
		NodeID:           nodeID,
		Token:            token,
	}

	uc.logger.Infow("node install command generated successfully", "short_id", query.ShortID)
	return result, nil
}

func (uc *GenerateNodeInstallScriptUseCase) validateQuery(query GenerateNodeInstallScriptQuery) error {
	if query.ShortID == "" {
		return errors.NewValidationError("short ID must be provided")
	}
	if query.APIURL == "" {
		return errors.NewValidationError("API URL is required")
	}
	return nil
}
