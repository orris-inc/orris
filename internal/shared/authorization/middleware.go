package authorization

import (
	"github.com/gin-gonic/gin"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.GetString("user_role")
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
