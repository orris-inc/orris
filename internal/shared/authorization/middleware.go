package authorization

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/constants"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.GetString(constants.ContextKeyUserRole)
		if userRole != string(RoleAdmin) {
			c.JSON(403, gin.H{
				"error": "admin access required",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireOwnerOrAdmin checks if the user is either the resource owner or an admin
// paramName is the URL parameter name that contains the resource owner ID (e.g., "id")
func RequireOwnerOrAdmin(paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get current user info from context (set by auth middleware)
		currentUserID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{
				"error": "user not authenticated",
			})
			c.Abort()
			return
		}

		userRole := c.GetString(constants.ContextKeyUserRole)

		// Admin has full access
		if userRole == string(RoleAdmin) {
			c.Next()
			return
		}

		// Get resource owner ID from URL parameter
		resourceIDStr := c.Param(paramName)
		resourceID, err := strconv.ParseUint(resourceIDStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{
				"error": "invalid resource ID",
			})
			c.Abort()
			return
		}

		// Check if current user is the resource owner
		currentID, ok := currentUserID.(uint)
		if !ok {
			c.JSON(500, gin.H{
				"error": "invalid user ID type",
			})
			c.Abort()
			return
		}

		if currentID != uint(resourceID) {
			c.JSON(403, gin.H{
				"error": "access denied - requires owner or admin access",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

type OwnedResource interface {
	GetOwnerID() uint
}

func CanAccessResource(userID uint, userRole UserRole, resource OwnedResource) bool {
	if userRole.IsAdmin() {
		return true
	}
	return userID == resource.GetOwnerID()
}

func CanAccessResourceByOwnerID(userID uint, userRole UserRole, resourceOwnerID uint) bool {
	if userRole.IsAdmin() {
		return true
	}
	return userID == resourceOwnerID
}
