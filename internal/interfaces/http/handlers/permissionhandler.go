package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"orris/internal/application/permission"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

type PermissionHandler struct {
	permissionService *permission.Service
	logger            logger.Interface
}

func NewPermissionHandler(permissionService *permission.Service, logger logger.Interface) *PermissionHandler {
	return &PermissionHandler{
		permissionService: permissionService,
		logger:            logger,
	}
}

type AssignRoleRequest struct {
	RoleIDs []uint `json:"role_ids" binding:"required"`
}

type PermissionResponse struct {
	ID          uint   `json:"id"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Code        string `json:"code"`
}

type RoleResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IsSystem    bool   `json:"is_system"`
}

// AssignRolesToUser godoc
// @Summary Assign roles to user
// @Description Assign one or more roles to a user (admin only)
// @Tags permissions
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "User ID"
// @Param request body AssignRoleRequest true "Role IDs to assign"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 403 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /users/{id}/roles [post]
func (h *PermissionHandler) AssignRolesToUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.permissionService.AssignRoleToUser(c.Request.Context(), uint(userID), req.RoleIDs); err != nil {
		h.logger.Errorw("failed to assign roles to user", "error", err, "user_id", userID)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to assign roles")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "roles assigned successfully", nil)
}

// GetUserRoles godoc
// @Summary Get user roles
// @Description Get all roles assigned to a specific user
// @Tags permissions
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "User ID"
// @Success 200 {object} utils.APIResponse{data=[]RoleResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 401 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /users/{id}/roles [get]
func (h *PermissionHandler) GetUserRoles(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	roles, err := h.permissionService.GetUserRoles(c.Request.Context(), uint(userID))
	if err != nil {
		h.logger.Errorw("failed to get user roles", "error", err, "user_id", userID)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to get user roles")
		return
	}

	roleResponses := make([]RoleResponse, 0, len(roles))
	for _, role := range roles {
		roleResponses = append(roleResponses, RoleResponse{
			ID:          role.ID(),
			Name:        role.Name(),
			Slug:        role.Slug(),
			Description: role.Description(),
			Status:      string(role.Status()),
			IsSystem:    role.IsSystem(),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "success", roleResponses)
}

// GetUserPermissions godoc
// @Summary Get user permissions
// @Description Get all permissions for a specific user (based on their roles)
// @Tags permissions
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "User ID"
// @Success 200 {object} utils.APIResponse{data=[]PermissionResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 401 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /users/{id}/permissions [get]
func (h *PermissionHandler) GetUserPermissions(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	permissions, err := h.permissionService.GetUserPermissions(c.Request.Context(), uint(userID))
	if err != nil {
		h.logger.Errorw("failed to get user permissions", "error", err, "user_id", userID)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to get user permissions")
		return
	}

	permResponses := make([]PermissionResponse, 0, len(permissions))
	for _, perm := range permissions {
		permResponses = append(permResponses, PermissionResponse{
			ID:          perm.ID(),
			Resource:    perm.Resource().String(),
			Action:      perm.Action().String(),
			Description: perm.Description(),
			Code:        perm.Code(),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "success", permResponses)
}

// GetMyPermissions godoc
// @Summary Get my permissions
// @Description Get all permissions for the current authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} utils.APIResponse{data=[]PermissionResponse}
// @Failure 401 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /auth/permissions [get]
func (h *PermissionHandler) GetMyPermissions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	permissions, err := h.permissionService.GetUserPermissions(c.Request.Context(), userID.(uint))
	if err != nil {
		h.logger.Errorw("failed to get current user permissions", "error", err, "user_id", userID)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to get permissions")
		return
	}

	permResponses := make([]PermissionResponse, 0, len(permissions))
	for _, perm := range permissions {
		permResponses = append(permResponses, PermissionResponse{
			ID:          perm.ID(),
			Resource:    perm.Resource().String(),
			Action:      perm.Action().String(),
			Description: perm.Description(),
			Code:        perm.Code(),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "success", permResponses)
}

// GetMyRoles godoc
// @Summary Get my roles
// @Description Get all roles assigned to the current authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} utils.APIResponse{data=[]RoleResponse}
// @Failure 401 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /auth/roles [get]
func (h *PermissionHandler) GetMyRoles(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	roles, err := h.permissionService.GetUserRoles(c.Request.Context(), userID.(uint))
	if err != nil {
		h.logger.Errorw("failed to get current user roles", "error", err, "user_id", userID)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to get roles")
		return
	}

	roleResponses := make([]RoleResponse, 0, len(roles))
	for _, role := range roles {
		roleResponses = append(roleResponses, RoleResponse{
			ID:          role.ID(),
			Name:        role.Name(),
			Slug:        role.Slug(),
			Description: role.Description(),
			Status:      string(role.Status()),
			IsSystem:    role.IsSystem(),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "success", roleResponses)
}

// CheckPermission godoc
// @Summary Check permission
// @Description Check if the current user has a specific permission
// @Tags auth
// @Accept json
// @Produce json
// @Security Bearer
// @Param resource query string true "Resource name (e.g., user, article)"
// @Param action query string true "Action name (e.g., create, read, update, delete)"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 401 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /auth/check-permission [get]
func (h *PermissionHandler) CheckPermission(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	resource := c.Query("resource")
	action := c.Query("action")

	if resource == "" || action == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "resource and action are required")
		return
	}

	allowed, err := h.permissionService.CheckPermission(c.Request.Context(), userID.(uint), resource, action)
	if err != nil {
		h.logger.Errorw("permission check failed", "error", err)
		utils.ErrorResponse(c, http.StatusInternalServerError, "permission check failed")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "success", gin.H{
		"allowed": allowed,
		"resource": resource,
		"action": action,
	})
}
