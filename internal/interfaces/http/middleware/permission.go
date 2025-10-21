package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"orris/internal/application/permission"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

type PermissionMiddleware struct {
	permissionService *permission.Service
	logger            logger.Interface
}

func NewPermissionMiddleware(permissionService *permission.Service, logger logger.Interface) *PermissionMiddleware {
	return &PermissionMiddleware{
		permissionService: permissionService,
		logger:            logger,
	}
}

func (m *PermissionMiddleware) RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
			c.Abort()
			return
		}

		allowed, err := m.permissionService.CheckPermission(c.Request.Context(), userID.(uint), resource, action)
		if err != nil {
			m.logger.Errorw("permission check failed", "error", err, "user_id", userID, "resource", resource, "action", action)
			utils.ErrorResponse(c, http.StatusInternalServerError, "permission check failed")
			c.Abort()
			return
		}

		if !allowed {
			m.logger.Warnw("permission denied", "user_id", userID, "resource", resource, "action", action)
			utils.ErrorResponse(c, http.StatusForbidden, "insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

func (m *PermissionMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
			c.Abort()
			return
		}

		userRoles, err := m.permissionService.GetUserRoles(c.Request.Context(), userID.(uint))
		if err != nil {
			m.logger.Errorw("failed to get user roles", "error", err, "user_id", userID)
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to check user roles")
			c.Abort()
			return
		}

		userRoleMap := make(map[string]bool)
		for _, role := range userRoles {
			userRoleMap[role.Slug()] = true
		}

		for _, requiredRole := range roles {
			if userRoleMap[requiredRole] {
				c.Next()
				return
			}
		}

		m.logger.Warnw("role check failed", "user_id", userID, "required_roles", roles)
		utils.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("required role: %v", roles))
		c.Abort()
	}
}
