package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// ValidateForwardAgentTokenExecutor defines the interface for validating forward agent tokens.
type ValidateForwardAgentTokenExecutor interface {
	Execute(ctx context.Context, cmd usecases.ValidateForwardAgentTokenCommand) (*usecases.ValidateForwardAgentTokenResult, error)
}

// ForwardAgentTokenMiddleware provides authentication middleware for forward agents.
type ForwardAgentTokenMiddleware struct {
	validateTokenUC ValidateForwardAgentTokenExecutor
	logger          logger.Interface
}

// NewForwardAgentTokenMiddleware creates a new instance of ForwardAgentTokenMiddleware.
func NewForwardAgentTokenMiddleware(
	validateTokenUC ValidateForwardAgentTokenExecutor,
	logger logger.Interface,
) *ForwardAgentTokenMiddleware {
	return &ForwardAgentTokenMiddleware{
		validateTokenUC: validateTokenUC,
		logger:          logger,
	}
}

// RequireForwardAgentToken is a middleware that validates forward agent tokens.
// It supports both Authorization header (Bearer token) and query parameter (?token=xxx).
// On successful validation, it sets forward_agent_id and forward_agent_name in the context.
func (m *ForwardAgentTokenMiddleware) RequireForwardAgentToken() gin.HandlerFunc {
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
			m.logger.Warnw("forward agent request without token", "ip", c.ClientIP())
			utils.ErrorResponse(c, http.StatusUnauthorized, "missing authorization token")
			c.Abort()
			return
		}

		cmd := usecases.ValidateForwardAgentTokenCommand{
			PlainToken: token,
			IPAddress:  c.ClientIP(),
		}

		result, err := m.validateTokenUC.Execute(c.Request.Context(), cmd)
		if err != nil {
			m.logger.Warnw("forward agent token validation failed",
				"error", err,
				"ip", c.ClientIP(),
			)
			utils.ErrorResponse(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		// Set agent information in context for downstream handlers
		c.Set("forward_agent_id", result.AgentID)
		c.Set("forward_agent_name", result.AgentName)

		m.logger.Infow("forward agent authenticated",
			"agent_id", result.AgentID,
			"agent_name", result.AgentName,
			"ip", c.ClientIP(),
		)

		c.Next()
	}
}
