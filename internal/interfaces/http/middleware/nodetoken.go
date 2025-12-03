package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
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

		// If no token in header, try query parameter
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

// RequireNodeTokenQuery is a middleware that validates node token from query parameter
func (m *NodeTokenMiddleware) RequireNodeTokenQuery() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from query parameter
		token := c.Query("token")

		if token == "" {
			m.logger.Warnw("agent request without token", "ip", c.ClientIP())
			utils.AgentAPIError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		cmd := usecases.ValidateNodeTokenCommand{
			PlainToken: token,
		}

		result, err := m.validateTokenUC.Execute(c.Request.Context(), cmd)
		if err != nil {
			m.logger.Warnw("agent node token validation failed", "error", err, "ip", c.ClientIP())
			utils.AgentAPIError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		// Store node info in context
		c.Set("node_id", result.NodeID)
		c.Set("node_name", result.Name)

		m.logger.Infow("agent node authenticated",
			"node_id", result.NodeID,
			"node_name", result.Name,
			"ip", c.ClientIP(),
		)

		c.Next()
	}
}

// RequireNodeTokenHeader is a middleware for RESTful API that validates X-Node-Token header
func (m *NodeTokenMiddleware) RequireNodeTokenHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		// RESTful API uses X-Node-Token header
		token := c.GetHeader("X-Node-Token")

		if token == "" {
			m.logger.Warnw("RESTful API request without X-Node-Token header", "ip", c.ClientIP())
			utils.AgentAPIError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		cmd := usecases.ValidateNodeTokenCommand{
			PlainToken: token,
		}

		result, err := m.validateTokenUC.Execute(c.Request.Context(), cmd)
		if err != nil {
			m.logger.Warnw("node token validation failed", "error", err, "ip", c.ClientIP())
			utils.AgentAPIError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		// Store node info in context
		c.Set("node_id", result.NodeID)
		c.Set("node_name", result.Name)

		m.logger.Infow("node authenticated via X-Node-Token header",
			"node_id", result.NodeID,
			"node_name", result.Name,
			"ip", c.ClientIP(),
		)

		c.Next()
	}
}
