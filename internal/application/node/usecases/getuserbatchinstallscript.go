package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetUserBatchInstallScriptQuery represents the input for getting user batch install script.
type GetUserBatchInstallScriptQuery struct {
	UserID uint     // User ID for ownership verification
	SIDs   []string // Node SIDs to include in the batch install
	APIURL string   // API server URL (e.g., https://api.example.com)
}

// GetUserBatchInstallScriptResult represents the output of getting user batch install script.
type GetUserBatchInstallScriptResult struct {
	InstallCommand   string            `json:"install_command"`
	UninstallCommand string            `json:"uninstall_command"`
	ScriptURL        string            `json:"script_url"`
	APIURL           string            `json:"api_url"`
	Nodes            []NodeInstallInfo `json:"nodes"`
}

// GetUserBatchInstallScriptExecutor defines the interface for getting user batch install script.
type GetUserBatchInstallScriptExecutor interface {
	Execute(ctx context.Context, query GetUserBatchInstallScriptQuery) (*GetUserBatchInstallScriptResult, error)
}

// GetUserBatchInstallScriptUseCase handles getting batch install script for user-owned nodes.
type GetUserBatchInstallScriptUseCase struct {
	repo   node.NodeRepository
	logger logger.Interface
}

// NewGetUserBatchInstallScriptUseCase creates a new GetUserBatchInstallScriptUseCase.
func NewGetUserBatchInstallScriptUseCase(
	repo node.NodeRepository,
	logger logger.Interface,
) *GetUserBatchInstallScriptUseCase {
	return &GetUserBatchInstallScriptUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute generates the batch install script for multiple user-owned nodes.
func (uc *GetUserBatchInstallScriptUseCase) Execute(ctx context.Context, query GetUserBatchInstallScriptQuery) (*GetUserBatchInstallScriptResult, error) {
	uc.logger.Infow("executing get user batch install script use case", "user_id", query.UserID, "node_count", len(query.SIDs))

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid get user batch install script query", "error", err, "user_id", query.UserID)
		return nil, err
	}

	// Deduplicate SIDs to avoid duplicate entries in error messages and results
	sids := deduplicateSIDs(query.SIDs)

	// Validate all node SIDs belong to the user
	invalidSIDs, err := uc.repo.ValidateNodeSIDsForUser(ctx, sids, query.UserID)
	if err != nil {
		uc.logger.Errorw("failed to validate node SIDs for user", "error", err, "user_id", query.UserID)
		return nil, fmt.Errorf("failed to validate node SIDs: %w", err)
	}
	if len(invalidSIDs) > 0 {
		return nil, errors.NewForbiddenError(fmt.Sprintf("you do not have permission to access these nodes: %s", strings.Join(invalidSIDs, ", ")))
	}

	// Get all nodes
	nodes, err := uc.repo.GetBySIDs(ctx, sids)
	if err != nil {
		uc.logger.Errorw("failed to get nodes by SIDs", "error", err)
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	// Check all nodes have tokens
	var nodesWithoutToken []string
	nodeInfos := make([]NodeInstallInfo, 0, len(nodes))
	var nodeArgs []string

	for _, n := range nodes {
		token := n.GetAPIToken()
		if token == "" {
			nodesWithoutToken = append(nodesWithoutToken, n.SID())
			continue
		}
		nodeInfos = append(nodeInfos, NodeInstallInfo{
			NodeSID: n.SID(),
			Token:   token,
		})
		nodeArgs = append(nodeArgs, fmt.Sprintf("--node %s:%s", n.SID(), token))
	}

	if len(nodesWithoutToken) > 0 {
		return nil, errors.NewValidationError(fmt.Sprintf("nodes have no token: %s, please regenerate tokens first", strings.Join(nodesWithoutToken, ", ")))
	}

	// Generate install and uninstall commands
	// Format: curl -fsSL <script_url> | sudo bash -s -- --api-url <api_url> --node node_xxx:token1 --node node_yyy:token2
	installCmd := fmt.Sprintf("curl -fsSL %s | sudo bash -s -- --api-url %s %s", NodeInstallScriptURL, query.APIURL, strings.Join(nodeArgs, " "))
	uninstallCmd := fmt.Sprintf("curl -fsSL %s | sudo bash -s -- uninstall", NodeInstallScriptURL)

	result := &GetUserBatchInstallScriptResult{
		InstallCommand:   installCmd,
		UninstallCommand: uninstallCmd,
		ScriptURL:        NodeInstallScriptURL,
		APIURL:           query.APIURL,
		Nodes:            nodeInfos,
	}

	uc.logger.Infow("user batch install command generated successfully", "user_id", query.UserID, "node_count", len(nodes))
	return result, nil
}

func (uc *GetUserBatchInstallScriptUseCase) validateQuery(query GetUserBatchInstallScriptQuery) error {
	if query.UserID == 0 {
		return errors.NewValidationError("user_id is required")
	}
	if len(query.SIDs) == 0 {
		return errors.NewValidationError("at least one node SID is required")
	}
	if len(query.SIDs) > MaxBatchSize {
		return errors.NewValidationError(fmt.Sprintf("batch size exceeds limit of %d", MaxBatchSize))
	}
	if query.APIURL == "" {
		return errors.NewValidationError("API URL is required")
	}

	// Validate all node ID formats
	for _, sid := range query.SIDs {
		if err := id.ValidatePrefix(sid, id.PrefixNode); err != nil {
			return errors.NewValidationError(fmt.Sprintf("invalid node ID format: %s", sid))
		}
	}

	return nil
}
