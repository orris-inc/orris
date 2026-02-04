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

const (
	// MaxBatchSize is the maximum number of nodes that can be included in a batch install script.
	MaxBatchSize = 100
)

// GenerateBatchInstallScriptQuery represents the input for generating batch install script.
type GenerateBatchInstallScriptQuery struct {
	SIDs   []string // Node SIDs to include in the batch install
	APIURL string   // API server URL (e.g., https://api.example.com)
}

// NodeInstallInfo represents install information for a single node.
type NodeInstallInfo struct {
	NodeSID string `json:"node_sid"`
	Token   string `json:"token"`
}

// GenerateBatchInstallScriptResult represents the output of generating batch install script.
type GenerateBatchInstallScriptResult struct {
	InstallCommand   string            `json:"install_command"`
	UninstallCommand string            `json:"uninstall_command"`
	ScriptURL        string            `json:"script_url"`
	APIURL           string            `json:"api_url"`
	Nodes            []NodeInstallInfo `json:"nodes"`
}

// GenerateBatchInstallScriptExecutor defines the interface for generating batch install script.
type GenerateBatchInstallScriptExecutor interface {
	Execute(ctx context.Context, query GenerateBatchInstallScriptQuery) (*GenerateBatchInstallScriptResult, error)
}

// GenerateBatchInstallScriptUseCase handles generating batch install script for nodes.
type GenerateBatchInstallScriptUseCase struct {
	repo   node.NodeRepository
	logger logger.Interface
}

// NewGenerateBatchInstallScriptUseCase creates a new GenerateBatchInstallScriptUseCase.
func NewGenerateBatchInstallScriptUseCase(
	repo node.NodeRepository,
	logger logger.Interface,
) *GenerateBatchInstallScriptUseCase {
	return &GenerateBatchInstallScriptUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute generates the batch install command for multiple nodes.
func (uc *GenerateBatchInstallScriptUseCase) Execute(ctx context.Context, query GenerateBatchInstallScriptQuery) (*GenerateBatchInstallScriptResult, error) {
	uc.logger.Infow("executing generate batch install script use case", "node_count", len(query.SIDs))

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid generate batch install script query", "error", err)
		return nil, err
	}

	// Deduplicate SIDs to avoid duplicate entries in error messages and results
	sids := deduplicateSIDs(query.SIDs)

	// Validate all node SIDs exist
	invalidSIDs, err := uc.repo.ValidateNodeSIDsExist(ctx, sids)
	if err != nil {
		uc.logger.Errorw("failed to validate node SIDs", "error", err)
		return nil, fmt.Errorf("failed to validate node SIDs: %w", err)
	}
	if len(invalidSIDs) > 0 {
		return nil, errors.NewNotFoundError("nodes", strings.Join(invalidSIDs, ", "))
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
		return nil, errors.NewValidationError(fmt.Sprintf("nodes have no token: %s, please generate tokens first", strings.Join(nodesWithoutToken, ", ")))
	}

	// Generate install and uninstall commands
	// Format: curl -fsSL <script_url> | sudo bash -s -- --api-url <api_url> --node node_xxx:token1 --node node_yyy:token2
	installCmd := fmt.Sprintf("curl -fsSL %s | sudo bash -s -- --api-url %s %s", NodeInstallScriptURL, query.APIURL, strings.Join(nodeArgs, " "))
	uninstallCmd := fmt.Sprintf("curl -fsSL %s | sudo bash -s -- uninstall", NodeInstallScriptURL)

	result := &GenerateBatchInstallScriptResult{
		InstallCommand:   installCmd,
		UninstallCommand: uninstallCmd,
		ScriptURL:        NodeInstallScriptURL,
		APIURL:           query.APIURL,
		Nodes:            nodeInfos,
	}

	uc.logger.Infow("batch install command generated successfully", "node_count", len(nodes))
	return result, nil
}

func (uc *GenerateBatchInstallScriptUseCase) validateQuery(query GenerateBatchInstallScriptQuery) error {
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

// deduplicateSIDs removes duplicate SIDs while preserving order.
func deduplicateSIDs(sids []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(sids))
	for _, sid := range sids {
		if !seen[sid] {
			seen[sid] = true
			result = append(result, sid)
		}
	}
	return result
}
