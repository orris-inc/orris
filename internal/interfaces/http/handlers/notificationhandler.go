package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"orris/internal/application/notification"
	"orris/internal/interfaces/dto"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

type NotificationHandler struct {
	serviceDDD *notification.ServiceDDD
	logger     logger.Interface
}

func NewNotificationHandler(serviceDDD *notification.ServiceDDD, logger logger.Interface) *NotificationHandler {
	return &NotificationHandler{
		serviceDDD: serviceDDD,
		logger:     logger,
	}
}

// CreateAnnouncement godoc
// @Summary Create announcement
// @Description Create a new announcement with the provided data
// @Security Bearer
// @Tags announcements
// @Accept json
// @Produce json
// @Param request body dto.CreateAnnouncementRequest true "Announcement data"
// @Success 201 {object} utils.APIResponse{data=dto.AnnouncementResponse} "Announcement created successfully"
// @Failure 400 {object} utils.APIResponse "Bad request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden - Requires admin role"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /announcements [post]
func (h *NotificationHandler) CreateAnnouncement(c *gin.Context) {
	var req dto.CreateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create announcement", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := utils.ValidateStruct(&req); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	creatorID, ok := userID.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type", "user_id", userID)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Internal error"))
		return
	}

	appReq := req.ToApplicationDTO(creatorID)

	result, err := h.serviceDDD.CreateAnnouncement(c.Request.Context(), appReq)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Announcement created successfully")
}

// UpdateAnnouncement godoc
// @Summary Update announcement
// @Description Update an existing announcement by ID
// @Security Bearer
// @Tags announcements
// @Accept json
// @Produce json
// @Param id path int true "Announcement ID"
// @Param request body dto.UpdateAnnouncementRequest true "Updated announcement data"
// @Success 200 {object} utils.APIResponse{data=dto.AnnouncementResponse} "Announcement updated successfully"
// @Failure 400 {object} utils.APIResponse "Bad request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden - Requires admin role"
// @Failure 404 {object} utils.APIResponse "Announcement not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /announcements/{id} [put]
func (h *NotificationHandler) UpdateAnnouncement(c *gin.Context) {
	announcementID, err := dto.ParseAnnouncementID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req dto.UpdateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update announcement",
			"announcement_id", announcementID,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := utils.ValidateStruct(&req); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	appReq := req.ToApplicationDTO()

	result, err := h.serviceDDD.UpdateAnnouncement(c.Request.Context(), announcementID, appReq)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Announcement updated successfully", result)
}

// DeleteAnnouncement godoc
// @Summary Delete announcement
// @Description Delete an announcement by ID
// @Security Bearer
// @Tags announcements
// @Accept json
// @Produce json
// @Param id path int true "Announcement ID"
// @Success 204 "Announcement deleted successfully"
// @Failure 400 {object} utils.APIResponse "Invalid announcement ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Announcement not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /announcements/{id} [delete]
func (h *NotificationHandler) DeleteAnnouncement(c *gin.Context) {
	announcementID, err := dto.ParseAnnouncementID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	err = h.serviceDDD.DeleteAnnouncement(c.Request.Context(), announcementID)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// PublishAnnouncement godoc
// @Summary Publish announcement
// @Description Publish a draft announcement to make it visible to users
// @Security Bearer
// @Tags announcements
// @Accept json
// @Produce json
// @Param id path int true "Announcement ID"
// @Success 200 {object} utils.APIResponse{data=dto.AnnouncementResponse} "Announcement published successfully"
// @Failure 400 {object} utils.APIResponse "Invalid announcement ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Announcement not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /announcements/{id}/publish [post]
func (h *NotificationHandler) PublishAnnouncement(c *gin.Context) {
	announcementID, err := dto.ParseAnnouncementID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req dto.PublishAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.SendNotification = false
	}

	appReq := req.ToApplicationDTO()

	result, err := h.serviceDDD.PublishAnnouncement(c.Request.Context(), announcementID, appReq)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Announcement published successfully", result)
}

// ListAnnouncements godoc
// @Summary List announcements
// @Description Get a paginated list of announcements with optional filters
// @Security Bearer
// @Tags announcements
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param type query string false "Filter by type" Enums(system, maintenance, feature, promotion)
// @Param status query string false "Filter by status" Enums(draft, published, archived)
// @Success 200 {object} utils.APIResponse{data=utils.ListResponse} "Announcements list"
// @Failure 400 {object} utils.APIResponse "Invalid query parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /announcements [get]
func (h *NotificationHandler) ListAnnouncements(c *gin.Context) {
	req, err := dto.ParseListAnnouncementsRequest(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	limit := req.PageSize
	offset := (req.Page - 1) * req.PageSize

	result, err := h.serviceDDD.ListAnnouncements(c.Request.Context(), limit, offset)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// GetAnnouncement godoc
// @Summary Get announcement
// @Description Get detailed information of an announcement by ID
// @Security Bearer
// @Tags announcements
// @Accept json
// @Produce json
// @Param id path int true "Announcement ID"
// @Success 200 {object} utils.APIResponse{data=dto.AnnouncementResponse} "Announcement details"
// @Failure 400 {object} utils.APIResponse "Invalid announcement ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Announcement not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /announcements/{id} [get]
func (h *NotificationHandler) GetAnnouncement(c *gin.Context) {
	announcementID, err := dto.ParseAnnouncementID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.serviceDDD.GetAnnouncement(c.Request.Context(), announcementID)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// ListNotifications godoc
// @Summary List notifications for current user
// @Description Get a list of notifications for the authenticated user
// @Security Bearer
// @Tags notifications
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Param status query string false "Filter by status" Enums(read, unread)
// @Success 200 {object} utils.APIResponse{data=[]dto.NotificationResponse} "Notifications list"
// @Failure 400 {object} utils.APIResponse "Invalid query parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /notifications [get]
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	req, err := dto.ParseListNotificationsRequest(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type", "user_id", userID)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Internal error"))
		return
	}

	appReq := req.ToApplicationDTO(uid)

	result, err := h.serviceDDD.ListNotifications(c.Request.Context(), appReq)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// GetUnreadCount godoc
// @Summary Get unread notifications count
// @Description Get the count of unread notifications for the authenticated user
// @Security Bearer
// @Tags notifications
// @Accept json
// @Produce json
// @Success 200 {object} utils.APIResponse{data=dto.UnreadCountResponse} "Unread count"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /notifications/unread-count [get]
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type", "user_id", userID)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Internal error"))
		return
	}

	result, err := h.serviceDDD.GetUnreadCount(c.Request.Context(), uid)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// MarkAsRead godoc
// @Summary Mark notification as read
// @Description Mark a specific notification as read for the authenticated user
// @Security Bearer
// @Tags notifications
// @Accept json
// @Produce json
// @Param id path int true "Notification ID"
// @Success 200 {object} utils.APIResponse{data=dto.NotificationResponse} "Notification marked as read"
// @Failure 400 {object} utils.APIResponse "Invalid notification ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Notification not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /notifications/{id}/read [put]
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	notificationID, err := dto.ParseNotificationID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type", "user_id", userID)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Internal error"))
		return
	}

	err = h.serviceDDD.MarkNotificationAsRead(c.Request.Context(), notificationID, uid)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Notification marked as read", nil)
}

// MarkAllAsRead godoc
// @Summary Mark all notifications as read
// @Description Mark all notifications as read for the authenticated user
// @Security Bearer
// @Tags notifications
// @Accept json
// @Produce json
// @Success 200 {object} utils.APIResponse "All notifications marked as read"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /notifications/read-all [put]
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type", "user_id", userID)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Internal error"))
		return
	}

	err := h.serviceDDD.MarkAllNotificationsAsRead(c.Request.Context(), uid)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "All notifications marked as read", nil)
}

// ArchiveNotification godoc
// @Summary Archive notification
// @Description Archive a notification for the authenticated user
// @Security Bearer
// @Tags notifications
// @Accept json
// @Produce json
// @Param id path int true "Notification ID"
// @Success 200 {object} utils.APIResponse{data=dto.NotificationResponse} "Notification archived successfully"
// @Failure 400 {object} utils.APIResponse "Invalid notification ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Notification not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /notifications/{id}/archive [post]
func (h *NotificationHandler) ArchiveNotification(c *gin.Context) {
	notificationID, err := dto.ParseNotificationID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type", "user_id", userID)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Internal error"))
		return
	}

	err = h.serviceDDD.ArchiveNotification(c.Request.Context(), notificationID, uid)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Notification archived successfully", nil)
}

// DeleteNotification godoc
// @Summary Delete notification
// @Description Delete a notification for the authenticated user
// @Security Bearer
// @Tags notifications
// @Accept json
// @Produce json
// @Param id path int true "Notification ID"
// @Success 204 "Notification deleted successfully"
// @Failure 400 {object} utils.APIResponse "Invalid notification ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Notification not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /notifications/{id} [delete]
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	notificationID, err := dto.ParseNotificationID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type", "user_id", userID)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Internal error"))
		return
	}

	err = h.serviceDDD.DeleteNotification(c.Request.Context(), notificationID, uid)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateTemplate godoc
// @Summary Create notification template
// @Description Create a new notification template
// @Security Bearer
// @Tags notification-templates
// @Accept json
// @Produce json
// @Param request body dto.CreateTemplateRequest true "Template data"
// @Success 201 {object} utils.APIResponse{data=dto.TemplateResponse} "Template created successfully"
// @Failure 400 {object} utils.APIResponse "Bad request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 409 {object} utils.APIResponse "Template type already exists"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /notification-templates [post]
func (h *NotificationHandler) CreateTemplate(c *gin.Context) {
	var req dto.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create template", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := utils.ValidateStruct(&req); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	appReq := req.ToApplicationDTO()

	result, err := h.serviceDDD.CreateTemplate(c.Request.Context(), appReq)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Template created successfully")
}

// ListTemplates godoc
// @Summary List notification templates
// @Description Get a paginated list of notification templates
// @Security Bearer
// @Tags notification-templates
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param enabled query boolean false "Filter by enabled status"
// @Success 200 {object} utils.APIResponse{data=utils.ListResponse} "Templates list"
// @Failure 400 {object} utils.APIResponse "Invalid query parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /notification-templates [get]
func (h *NotificationHandler) ListTemplates(c *gin.Context) {
	_, err := dto.ParseListTemplatesRequest(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.serviceDDD.ListTemplates(c.Request.Context())
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// RenderTemplate godoc
// @Summary Render notification template
// @Description Render a notification template with provided variables
// @Security Bearer
// @Tags notification-templates
// @Accept json
// @Produce json
// @Param request body dto.RenderTemplateRequest true "Render data"
// @Success 200 {object} utils.APIResponse{data=dto.RenderTemplateResponse} "Template rendered successfully"
// @Failure 400 {object} utils.APIResponse "Bad request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Template not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /notification-templates/render [post]
func (h *NotificationHandler) RenderTemplate(c *gin.Context) {
	var req dto.RenderTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for render template", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := utils.ValidateStruct(&req); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	appReq := req.ToApplicationDTO()

	result, err := h.serviceDDD.RenderTemplate(c.Request.Context(), appReq)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Template rendered successfully", result)
}

// ListPublicAnnouncements godoc
// @Summary List public announcements
// @Description Get a paginated list of published announcements (public endpoint, no authentication required)
// @Tags announcements
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param type query string false "Filter by type" Enums(system, maintenance, feature, promotion)
// @Success 200 {object} utils.APIResponse{data=utils.ListResponse} "Public announcements list"
// @Failure 400 {object} utils.APIResponse "Invalid query parameters"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /public/announcements [get]
func (h *NotificationHandler) ListPublicAnnouncements(c *gin.Context) {
	req, err := dto.ParseListAnnouncementsRequest(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	limit := req.PageSize
	offset := (req.Page - 1) * req.PageSize

	result, err := h.serviceDDD.ListPublishedAnnouncements(c.Request.Context(), limit, offset)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}
