package dto

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	appDto "github.com/orris-inc/orris/internal/application/notification/dto"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// toUTCPtr converts a *time.Time to UTC if not nil.
func toUTCPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	utc := t.UTC()
	return &utc
}

type CreateAnnouncementRequest struct {
	Title       string     `json:"title" binding:"required" validate:"required,min=1,max=255"`
	Content     string     `json:"content" binding:"required" validate:"required,min=1"`
	Type        string     `json:"type" binding:"required" validate:"required,oneof=system maintenance feature promotion"`
	Priority    int        `json:"priority" validate:"min=1,max=5"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

func (r *CreateAnnouncementRequest) ToApplicationDTO(creatorID uint) appDto.CreateAnnouncementRequest {
	return appDto.CreateAnnouncementRequest{
		Title:       r.Title,
		Content:     r.Content,
		Type:        r.Type,
		Priority:    r.Priority,
		ScheduledAt: toUTCPtr(r.ScheduledAt),
		ExpiresAt:   toUTCPtr(r.ExpiresAt),
	}
}

type UpdateAnnouncementRequest struct {
	Title       *string    `json:"title,omitempty" validate:"omitempty,min=1,max=255"`
	Content     *string    `json:"content,omitempty" validate:"omitempty,min=1"`
	Type        *string    `json:"type,omitempty" validate:"omitempty,oneof=system maintenance feature promotion"`
	Priority    *int       `json:"priority,omitempty" validate:"omitempty,min=1,max=5"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

func (r *UpdateAnnouncementRequest) ToApplicationDTO() appDto.UpdateAnnouncementRequest {
	return appDto.UpdateAnnouncementRequest{
		Title:     r.Title,
		Content:   r.Content,
		Priority:  r.Priority,
		ExpiresAt: toUTCPtr(r.ExpiresAt),
	}
}

type AnnouncementResponse struct {
	ID          uint       `json:"id"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	CreatorID   uint       `json:"creator_id"`
	Priority    int        `json:"priority"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	ViewCount   int        `json:"view_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type NotificationResponse struct {
	ID         uint       `json:"id"`
	UserID     uint       `json:"user_id"`
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	RelatedID  *uint      `json:"related_id,omitempty"`
	ReadStatus string     `json:"read_status"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type UnreadCountResponse struct {
	Count int `json:"count"`
}

type CreateTemplateRequest struct {
	TemplateType string   `json:"template_type" binding:"required" validate:"required,min=1,max=50"`
	Name         string   `json:"name" binding:"required" validate:"required,min=1,max=100"`
	Title        string   `json:"title" binding:"required" validate:"required,min=1,max=255"`
	Content      string   `json:"content" binding:"required" validate:"required,min=1"`
	Variables    []string `json:"variables,omitempty"`
	Enabled      *bool    `json:"enabled,omitempty"`
}

func (r *CreateTemplateRequest) ToApplicationDTO() appDto.CreateTemplateRequest {
	return appDto.CreateTemplateRequest{
		TemplateType: r.TemplateType,
		Name:         r.Name,
		Title:        r.Title,
		Content:      r.Content,
		Variables:    r.Variables,
	}
}

type TemplateResponse struct {
	ID           uint      `json:"id"`
	TemplateType string    `json:"template_type"`
	Name         string    `json:"name"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	Variables    []string  `json:"variables,omitempty"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RenderTemplateRequest struct {
	TemplateType string                 `json:"template_type" binding:"required"`
	Variables    map[string]interface{} `json:"variables"`
}

func (r *RenderTemplateRequest) ToApplicationDTO() appDto.RenderTemplateRequest {
	return appDto.RenderTemplateRequest{
		TemplateType: r.TemplateType,
		Data:         r.Variables,
	}
}

type PublishAnnouncementRequest struct {
	SendNotification bool `json:"send_notification"`
}

func (r *PublishAnnouncementRequest) ToApplicationDTO() appDto.PublishAnnouncementRequest {
	return appDto.PublishAnnouncementRequest{
		SendNotification: r.SendNotification,
	}
}

type RenderTemplateResponse struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func ParseAnnouncementID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	if idStr == "" {
		return 0, errors.NewValidationError("Announcement ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid announcement ID format")
	}

	if id == 0 {
		return 0, errors.NewValidationError("Announcement ID cannot be zero")
	}

	return uint(id), nil
}

func ParseNotificationID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	if idStr == "" {
		return 0, errors.NewValidationError("Notification ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid notification ID format")
	}

	if id == 0 {
		return 0, errors.NewValidationError("Notification ID cannot be zero")
	}

	return uint(id), nil
}

func ParseListAnnouncementsRequest(c *gin.Context) (*ListAnnouncementsRequest, error) {
	req := &ListAnnouncementsRequest{
		Page:     constants.DefaultPage,
		PageSize: constants.DefaultPageSize,
	}

	if pageStr := c.Query("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return nil, errors.NewValidationError("Invalid page parameter")
		}
		req.Page = page
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 {
			return nil, errors.NewValidationError("Invalid page_size parameter")
		}
		if pageSize > constants.MaxPageSize {
			pageSize = constants.MaxPageSize
		}
		req.PageSize = pageSize
	}

	req.Type = c.Query("type")
	req.Status = c.Query("status")

	if err := utils.ValidateStruct(req); err != nil {
		return nil, err
	}

	return req, nil
}

type ListAnnouncementsRequest struct {
	Page     int    `json:"page" validate:"min=1"`
	PageSize int    `json:"page_size" validate:"min=1,max=100"`
	Type     string `json:"type,omitempty" validate:"omitempty,oneof=system maintenance feature promotion"`
	Status   string `json:"status,omitempty" validate:"omitempty,oneof=draft published archived"`
}

func ParseListNotificationsRequest(c *gin.Context) (*ListNotificationsRequest, error) {
	req := &ListNotificationsRequest{
		Limit:  20,
		Offset: 0,
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			return nil, errors.NewValidationError("Invalid limit parameter")
		}
		if limit > 100 {
			limit = 100
		}
		req.Limit = limit
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return nil, errors.NewValidationError("Invalid offset parameter")
		}
		req.Offset = offset
	}

	req.Status = c.Query("status")

	if err := utils.ValidateStruct(req); err != nil {
		return nil, err
	}

	return req, nil
}

type ListNotificationsRequest struct {
	Limit  int    `json:"limit" validate:"min=1,max=100"`
	Offset int    `json:"offset" validate:"min=0"`
	Status string `json:"status,omitempty" validate:"omitempty,oneof=read unread"`
}

func (r *ListNotificationsRequest) ToApplicationDTO(userID uint) appDto.ListNotificationsRequest {
	return appDto.ListNotificationsRequest{
		UserID: userID,
		Limit:  r.Limit,
		Offset: r.Offset,
		Status: r.Status,
	}
}

func ParseListTemplatesRequest(c *gin.Context) (*ListTemplatesRequest, error) {
	req := &ListTemplatesRequest{
		Page:     constants.DefaultPage,
		PageSize: constants.DefaultPageSize,
	}

	if pageStr := c.Query("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return nil, errors.NewValidationError("Invalid page parameter")
		}
		req.Page = page
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 {
			return nil, errors.NewValidationError("Invalid page_size parameter")
		}
		if pageSize > constants.MaxPageSize {
			pageSize = constants.MaxPageSize
		}
		req.PageSize = pageSize
	}

	if enabledStr := c.Query("enabled"); enabledStr != "" {
		enabled := enabledStr == "true"
		req.Enabled = &enabled
	}

	if err := utils.ValidateStruct(req); err != nil {
		return nil, err
	}

	return req, nil
}

type ListTemplatesRequest struct {
	Page     int   `json:"page" validate:"min=1"`
	PageSize int   `json:"page_size" validate:"min=1,max=100"`
	Enabled  *bool `json:"enabled,omitempty"`
}
