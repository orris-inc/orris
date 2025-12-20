package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetUserNodeInstallScriptQuery represents the input for getting user node install script.
type GetUserNodeInstallScriptQuery struct {
	UserID  uint   // User ID for ownership verification
	NodeSID string // Node SID
	APIURL  string // API server URL (e.g., https://api.example.com)
}

// GetUserNodeInstallScriptResult represents the output of getting user node install script.
type GetUserNodeInstallScriptResult struct {
	InstallCommand   string `json:"install_command"`
	UninstallCommand string `json:"uninstall_command"`
	ScriptURL        string `json:"script_url"`
	APIURL           string `json:"api_url"`
	NodeSID          string `json:"node_sid"`
	Token            string `json:"token"`
}

// GetUserNodeInstallScriptExecutor defines the interface for getting user node install script.
type GetUserNodeInstallScriptExecutor interface {
	Execute(ctx context.Context, query GetUserNodeInstallScriptQuery) (*GetUserNodeInstallScriptResult, error)
}

// GetUserNodeInstallScriptUseCase handles getting install script for user node.
type GetUserNodeInstallScriptUseCase struct {
	repo   node.NodeRepository
	logger logger.Interface
}

// NewGetUserNodeInstallScriptUseCase creates a new GetUserNodeInstallScriptUseCase.
func NewGetUserNodeInstallScriptUseCase(
	repo node.NodeRepository,
	logger logger.Interface,
) *GetUserNodeInstallScriptUseCase {
	return &GetUserNodeInstallScriptUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute generates the install script for a user-owned node.
func (uc *GetUserNodeInstallScriptUseCase) Execute(ctx context.Context, query GetUserNodeInstallScriptQuery) (*GetUserNodeInstallScriptResult, error) {
	uc.logger.Infow("executing get user node install script use case", "user_id", query.UserID, "node_sid", query.NodeSID)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid get user node install script query", "error", err, "node_sid", query.NodeSID)
		return nil, err
	}

	// Get the node by SID
	n, err := uc.repo.GetBySID(ctx, query.NodeSID)
	if err != nil {
		uc.logger.Errorw("failed to get node by SID", "node_sid", query.NodeSID, "error", err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	if n == nil {
		return nil, errors.NewNotFoundError("node", query.NodeSID)
	}

	// Verify ownership
	if n.UserID() == nil || *n.UserID() != query.UserID {
		uc.logger.Warnw("user does not own this node", "user_id", query.UserID, "node_sid", query.NodeSID)
		return nil, errors.NewForbiddenError("you do not have permission to access this node")
	}

	// Get the token
	token := n.GetAPIToken()
	if token == "" {
		return nil, errors.NewValidationError("node has no token, please regenerate token first")
	}

	nodeID := n.ID()

	// Generate install and uninstall commands
	nodeArg := fmt.Sprintf("%d:%s", nodeID, token)
	installCmd := fmt.Sprintf("curl -fsSL %s | sudo bash -s -- --api-url %s --node %s", NodeInstallScriptURL, query.APIURL, nodeArg)
	uninstallCmd := fmt.Sprintf("curl -fsSL %s | sudo bash -s -- uninstall", NodeInstallScriptURL)

	result := &GetUserNodeInstallScriptResult{
		InstallCommand:   installCmd,
		UninstallCommand: uninstallCmd,
		ScriptURL:        NodeInstallScriptURL,
		APIURL:           query.APIURL,
		NodeSID:          n.SID(),
		Token:            token,
	}

	uc.logger.Infow("user node install script generated successfully", "user_id", query.UserID, "node_sid", query.NodeSID)
	return result, nil
}

func (uc *GetUserNodeInstallScriptUseCase) validateQuery(query GetUserNodeInstallScriptQuery) error {
	if query.UserID == 0 {
		return errors.NewValidationError("user_id is required")
	}
	if query.NodeSID == "" {
		return errors.NewValidationError("node SID is required")
	}
	if query.APIURL == "" {
		return errors.NewValidationError("API URL is required")
	}
	return nil
}
