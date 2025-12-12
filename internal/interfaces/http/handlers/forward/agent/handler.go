// Package agent provides HTTP handlers for forward agent API requests.
package agent

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// Handler handles RESTful agent API requests for forward client
type Handler struct {
	repo               forward.Repository
	agentRepo          forward.AgentRepository
	nodeRepo           node.NodeRepository
	reportStatusUC     *usecases.ReportAgentStatusUseCase
	statusQuerier      usecases.AgentStatusQuerier
	tokenSigningSecret string
	agentTokenService  *auth.AgentTokenService
	logger             logger.Interface
}

// NewHandler creates a new Handler instance
func NewHandler(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	reportStatusUC *usecases.ReportAgentStatusUseCase,
	statusQuerier usecases.AgentStatusQuerier,
	tokenSigningSecret string,
	logger logger.Interface,
) *Handler {
	return &Handler{
		repo:               repo,
		agentRepo:          agentRepo,
		nodeRepo:           nodeRepo,
		reportStatusUC:     reportStatusUC,
		statusQuerier:      statusQuerier,
		tokenSigningSecret: tokenSigningSecret,
		agentTokenService:  auth.NewAgentTokenService(tokenSigningSecret),
		logger:             logger,
	}
}

// getAuthenticatedAgentID extracts the authenticated forward agent ID from context.
// Returns the agent ID or an error if not found.
func (h *Handler) getAuthenticatedAgentID(c *gin.Context) (uint, error) {
	agentID, exists := c.Get("forward_agent_id")
	if !exists {
		return 0, fmt.Errorf("forward_agent_id not found in context")
	}
	id, ok := agentID.(uint)
	if !ok {
		return 0, fmt.Errorf("invalid forward_agent_id type in context")
	}
	return id, nil
}

// resolveNodeAddress selects the appropriate node address based on IP version preference.
// ipVersion: "auto", "ipv4", or "ipv6"
func (h *Handler) resolveNodeAddress(targetNode *node.Node, ipVersion string) string {
	serverAddr := targetNode.ServerAddress().Value()
	ipv4 := ""
	ipv6 := ""

	if targetNode.PublicIPv4() != nil {
		ipv4 = *targetNode.PublicIPv4()
	}
	if targetNode.PublicIPv6() != nil {
		ipv6 = *targetNode.PublicIPv6()
	}

	// Check if server_address is a valid usable address
	isValidServerAddr := serverAddr != "" && serverAddr != "0.0.0.0" && serverAddr != "::"

	switch ipVersion {
	case "ipv6":
		// Prefer IPv6: ipv6 > server_address > ipv4
		if ipv6 != "" {
			h.logger.Debugw("using IPv6 address per ip_version setting",
				"ipv6", ipv6,
			)
			return ipv6
		}
		if isValidServerAddr {
			return serverAddr
		}
		if ipv4 != "" {
			h.logger.Debugw("falling back to IPv4 (IPv6 not available)",
				"ipv4", ipv4,
			)
			return ipv4
		}

	case "ipv4":
		// Prefer IPv4: ipv4 > server_address > ipv6
		if ipv4 != "" {
			h.logger.Debugw("using IPv4 address per ip_version setting",
				"ipv4", ipv4,
			)
			return ipv4
		}
		if isValidServerAddr {
			return serverAddr
		}
		if ipv6 != "" {
			h.logger.Debugw("falling back to IPv6 (IPv4 not available)",
				"ipv6", ipv6,
			)
			return ipv6
		}

	default: // "auto" or unknown
		// Default priority: server_address > ipv4 > ipv6
		if isValidServerAddr {
			return serverAddr
		}
		if ipv4 != "" {
			h.logger.Debugw("using public IPv4 as fallback",
				"ipv4", ipv4,
			)
			return ipv4
		}
		if ipv6 != "" {
			h.logger.Debugw("using public IPv6 as fallback",
				"ipv6", ipv6,
			)
			return ipv6
		}
	}

	return serverAddr
}
