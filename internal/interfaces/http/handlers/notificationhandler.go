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

// UpdateAnnouncementStatusRequest represents a request for announcement status changes
type UpdateAnnouncementStatusRequest struct {
	Status           string `json:"status" binding:"required,oneof=draft published archived"`
	SendNotification *bool  `json:"send_notification"` // Optional: for publish action
}

func (h *NotificationHandler) UpdateAnnouncementStatus(c *gin.Context) {
	announcementID, err := dto.ParseAnnouncementID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdateAnnouncementStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update announcement status", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	switch req.Status {
	case "published":
		// Use PublishAnnouncement service method
		sendNotification := false
		if req.SendNotification != nil {
			sendNotification = *req.SendNotification
		}
		publishReq := &dto.PublishAnnouncementRequest{SendNotification: sendNotification}
		appReq := publishReq.ToApplicationDTO()
		result, err := h.serviceDDD.PublishAnnouncement(c.Request.Context(), announcementID, appReq)
		if err != nil {
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Announcement published successfully", result)

	case "draft", "archived":
		// TODO: Implement draft/archive status change when service method is available
		utils.ErrorResponse(c, http.StatusNotImplemented, "Status change to "+req.Status+" not yet implemented")

	default:
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid status value")
	}
}

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

// UpdateNotificationStatusRequest represents a request for notification status changes
type UpdateNotificationStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=read archived"`
}

func (h *NotificationHandler) UpdateNotificationStatus(c *gin.Context) {
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

	var req UpdateNotificationStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update notification status", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	switch req.Status {
	case "read":
		err = h.serviceDDD.MarkNotificationAsRead(c.Request.Context(), notificationID, uid)
		if err != nil {
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Notification marked as read", nil)

	case "archived":
		err = h.serviceDDD.ArchiveNotification(c.Request.Context(), notificationID, uid)
		if err != nil {
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Notification archived successfully", nil)

	default:
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid status value")
	}
}

// UpdateAllNotificationsStatusRequest represents a request for batch notification status changes
type UpdateAllNotificationsStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=read"`
}

func (h *NotificationHandler) UpdateAllNotificationsStatus(c *gin.Context) {
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

	var req UpdateAllNotificationsStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update all notifications status", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	switch req.Status {
	case "read":
		err := h.serviceDDD.MarkAllNotificationsAsRead(c.Request.Context(), uid)
		if err != nil {
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "All notifications marked as read", nil)

	default:
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid status value")
	}
}

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
