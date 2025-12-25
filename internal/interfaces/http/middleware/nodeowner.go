package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// NodeOwnerMiddleware validates that the user owns the node being accessed
type NodeOwnerMiddleware struct {
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

// NewNodeOwnerMiddleware creates a new node owner middleware
func NewNodeOwnerMiddleware(nodeRepo node.NodeRepository) *NodeOwnerMiddleware {
	return &NodeOwnerMiddleware{
		nodeRepo: nodeRepo,
		logger:   logger.NewLogger(),
	}
}

// RequireOwnership ensures the user owns the node specified in the URL parameter
func (m *NodeOwnerMiddleware) RequireOwnership() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context
		userIDInterface, exists := c.Get("user_id")
		if !exists {
			utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("user not authenticated"))
			c.Abort()
			return
		}
		userID, ok := userIDInterface.(uint)
		if !ok {
			utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("invalid user ID in context"))
			c.Abort()
			return
		}

		// Get user role (admin can access all nodes)
		roleInterface, _ := c.Get("user_role")
		role, _ := roleInterface.(string)
		if role == "admin" {
			c.Next()
			return
		}

		// Get node ID from URL parameter
		nodeID := c.Param("id")
		if nodeID == "" {
			utils.ErrorResponseWithError(c, errors.NewValidationError("node ID is required"))
			c.Abort()
			return
		}

		// Validate node SID format (e.g., "node_xxx")
		if err := id.ValidatePrefix(nodeID, id.PrefixNode); err != nil {
			m.logger.Warnw("invalid node ID format", "node_id", nodeID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid node ID format"))
			c.Abort()
			return
		}

		// Get node from repository (database stores full prefixed ID like "node_xxx")
		nodeEntity, err := m.nodeRepo.GetBySID(c.Request.Context(), nodeID)
		if err != nil {
			utils.ErrorResponseWithError(c, err)
			c.Abort()
			return
		}

		// Check ownership
		if !nodeEntity.IsOwnedBy(userID) {
			m.logger.Warnw("user attempted to access node they don't own",
				"user_id", userID,
				"node_id", nodeID,
				"node_owner", nodeEntity.UserID(),
			)
			utils.ErrorResponseWithError(c, errors.NewForbiddenError("access denied to this node"))
			c.Abort()
			return
		}

		// Store node in context for handler use
		c.Set("node", nodeEntity)
		c.Set("node_id", nodeID)

		c.Next()
	}
}
