package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/notification"
	appDto "github.com/orris-inc/orris/internal/application/notification/dto"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/interfaces/dto"
	"github.com/orris-inc/orris/internal/shared/authorization"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

type NotificationHandler struct {
	serviceDDD *notification.ServiceDDD
	userRepo   user.Repository
	logger     logger.Interface
}

func NewNotificationHandler(serviceDDD *notification.ServiceDDD, userRepo user.Repository, logger logger.Interface) *NotificationHandler {
	return &NotificationHandler{
		serviceDDD: serviceDDD,
		userRepo:   userRepo,
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
	sid, err := dto.ParseAnnouncementSID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req dto.UpdateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update announcement",
			"announcement_sid", sid,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := utils.ValidateStruct(&req); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	appReq := req.ToApplicationDTO()

	result, err := h.serviceDDD.UpdateAnnouncement(c.Request.Context(), sid, appReq)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Announcement updated successfully", result)
}

func (h *NotificationHandler) DeleteAnnouncement(c *gin.Context) {
	sid, err := dto.ParseAnnouncementSID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	err = h.serviceDDD.DeleteAnnouncement(c.Request.Context(), sid)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateAnnouncementStatusRequest represents a request for announcement status changes
type UpdateAnnouncementStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=draft published archived"`
}

func (h *NotificationHandler) UpdateAnnouncementStatus(c *gin.Context) {
	sid, err := dto.ParseAnnouncementSID(c)
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
		result, err := h.serviceDDD.PublishAnnouncement(c.Request.Context(), sid)
		if err != nil {
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Announcement published successfully", result)

	case "archived":
		result, err := h.serviceDDD.ArchiveAnnouncement(c.Request.Context(), sid)
		if err != nil {
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Announcement archived successfully", result)

	case "draft":
		utils.ErrorResponse(c, http.StatusNotImplemented, "Status change to draft not yet implemented")

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
	sid, err := dto.ParseAnnouncementSID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.serviceDDD.GetAnnouncement(c.Request.Context(), sid)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Non-admin users can only access published announcements
	userRole := authorization.ParseUserRole(c.GetString(constants.ContextKeyUserRole))
	if !userRole.IsAdmin() && result.Status != "published" {
		utils.ErrorResponseWithError(c, errors.NewNotFoundError("announcement not found"))
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

	// If user is authenticated, calculate is_read for each announcement
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uint); ok {
			h.enrichAnnouncementsWithReadStatus(c.Request.Context(), result, uid)
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// enrichAnnouncementsWithReadStatus calculates is_read for each announcement
// based on two conditions:
// 1. Global: user's announcements_read_at timestamp (marks all old announcements as read)
// 2. Individual: user_announcement_reads table (marks specific announcements as read)
func (h *NotificationHandler) enrichAnnouncementsWithReadStatus(ctx context.Context, result *appDto.ListResponse, userID uint) {
	if result == nil || result.Items == nil {
		return
	}

	u, err := h.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return
	}

	userReadAt := u.AnnouncementsReadAt()

	// Type assert to []*appDto.AnnouncementResponse
	items, ok := result.Items.([]*appDto.AnnouncementResponse)
	if !ok {
		return
	}

	// Collect announcement IDs for batch query
	announcementIDs := make([]uint, 0, len(items))
	for _, item := range items {
		if item != nil {
			announcementIDs = append(announcementIDs, item.InternalID)
		}
	}

	// Get read status for only the current page announcements (optimized query)
	readStatusMap, err := h.serviceDDD.GetReadStatusByIDs(ctx, userID, announcementIDs)
	if err != nil {
		h.logger.Warnw("failed to get read status", "user_id", userID, "error", err)
		readStatusMap = make(map[uint]bool) // Continue with empty map
	}

	for _, item := range items {
		if item == nil {
			continue
		}

		// An announcement is read if:
		// 1. It was published before user's global read timestamp, OR
		// 2. It was individually marked as read
		globalRead := userReadAt != nil && !item.UpdatedAt.After(*userReadAt)
		individualRead := readStatusMap[item.InternalID]
		isRead := globalRead || individualRead
		item.IsRead = &isRead
	}
}

// MarkAnnouncementsAsRead marks all announcements as read for the current user.
// This updates the user's announcements_read_at timestamp to the current time.
func (h *NotificationHandler) MarkAnnouncementsAsRead(c *gin.Context) {
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

	ctx := c.Request.Context()

	u, err := h.userRepo.GetByID(ctx, uid)
	if err != nil {
		h.logger.Errorw("failed to get user", "user_id", uid, "error", err)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Failed to get user"))
		return
	}

	if u == nil {
		utils.ErrorResponseWithError(c, errors.NewNotFoundError("User not found"))
		return
	}

	u.MarkAnnouncementsAsRead()

	if err := h.userRepo.Update(ctx, u); err != nil {
		h.logger.Errorw("failed to update user announcements read time", "user_id", uid, "error", err)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Failed to mark announcements as read"))
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Announcements marked as read", nil)
}

// GetAnnouncementUnreadCount returns the count of unread announcements for the current user.
func (h *NotificationHandler) GetAnnouncementUnreadCount(c *gin.Context) {
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

	ctx := c.Request.Context()

	u, err := h.userRepo.GetByID(ctx, uid)
	if err != nil {
		h.logger.Errorw("failed to get user", "user_id", uid, "error", err)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Failed to get user"))
		return
	}

	if u == nil {
		utils.ErrorResponseWithError(c, errors.NewNotFoundError("User not found"))
		return
	}

	count, err := h.serviceDDD.GetAnnouncementUnreadCount(ctx, uid, u.AnnouncementsReadAt())
	if err != nil {
		h.logger.Errorw("failed to get announcement unread count", "user_id", uid, "error", err)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Failed to get unread count"))
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Success", appDto.UnreadCountResponse{Count: count})
}

// MarkAnnouncementAsRead marks a specific announcement as read for the current user.
func (h *NotificationHandler) MarkAnnouncementAsRead(c *gin.Context) {
	sid, err := dto.ParseAnnouncementSID(c)
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

	if err := h.serviceDDD.MarkAnnouncementAsRead(c.Request.Context(), uid, sid); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Announcement marked as read", nil)
}
