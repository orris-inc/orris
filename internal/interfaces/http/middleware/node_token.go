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
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "invalid authorization header format")
			c.Abort()
			return
		}

		token := parts[1]
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
