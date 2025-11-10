package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"orris/internal/application/node/usecases"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

type ValidateNodeTokenExecutor interface {
	Execute(ctx context.Context, cmd usecases.ValidateNodeTokenCommand) (*usecases.ValidateNodeTokenResult, error)
}

type NodeTokenMiddleware struct {
	validateTokenUC ValidateNodeTokenExecutor
	logger          logger.Interface
}

func NewNodeTokenMiddleware(
	validateTokenUC ValidateNodeTokenExecutor,
	logger logger.Interface,
) *NodeTokenMiddleware {
	return &NodeTokenMiddleware{
		validateTokenUC: validateTokenUC,
		logger:          logger,
	}
}

func (m *NodeTokenMiddleware) RequireNodeToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get token from Authorization header first (Bearer token)
		authHeader := c.GetHeader("Authorization")
		var token string

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}

		// If no token in header, try query parameter (for XrayR compatibility)
		if token == "" {
			token = c.Query("token")
		}

		if token == "" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "missing authorization token")
			c.Abort()
			return
		}

		cmd := usecases.ValidateNodeTokenCommand{
			PlainToken: token,
		}

		result, err := m.validateTokenUC.Execute(c.Request.Context(), cmd)
		if err != nil {
			m.logger.Warnw("node token validation failed", "error", err, "ip", c.ClientIP())
			utils.ErrorResponse(c, http.StatusUnauthorized, "invalid or expired node token")
			c.Abort()
			return
		}

		c.Set("node_id", result.NodeID)
		c.Set("node_name", result.Name)
		c.Next()
	}
}

// RequireNodeTokenXrayR is a middleware for XrayR API that validates node token
// Returns v2raysocks-formatted error responses for compatibility
func (m *NodeTokenMiddleware) RequireNodeTokenXrayR() gin.HandlerFunc {
	return func(c *gin.Context) {
		// XrayR sends token as query parameter
		token := c.Query("token")

		if token == "" {
			m.logger.Warnw("XrayR request without token", "ip", c.ClientIP())
			utils.V2RaySocksError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		cmd := usecases.ValidateNodeTokenCommand{
			PlainToken: token,
		}

		result, err := m.validateTokenUC.Execute(c.Request.Context(), cmd)
		if err != nil {
			m.logger.Warnw("XrayR node token validation failed", "error", err, "ip", c.ClientIP())
			utils.V2RaySocksError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		// Store node info in context
		c.Set("node_id", result.NodeID)
		c.Set("node_name", result.Name)

		m.logger.Infow("XrayR node authenticated",
			"node_id", result.NodeID,
			"node_name", result.Name,
			"ip", c.ClientIP(),
		)

		c.Next()
	}
}

