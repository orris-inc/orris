package dto

import (
	"time"
)

type CreateAnnouncementRequest struct {
	Title       string     `json:"title" binding:"required,max=255"`
	Content     string     `json:"content" binding:"required"`
	Type        string     `json:"type" binding:"required,oneof=system maintenance event"`
	Priority    int        `json:"priority" binding:"min=1,max=5"`
	ScheduledAt *time.Time `json:"scheduled_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatorID   uint       `json:"-"` // Set by handler from authenticated user
}

type UpdateAnnouncementRequest struct {
	Title     *string    `json:"title" binding:"omitempty,max=255"`
	Content   *string    `json:"content"`
	Priority  *int       `json:"priority" binding:"omitempty,min=1,max=5"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type ListNotificationsRequest struct {
	UserID uint
	Limit  int
	Offset int
	Status string `json:"status" binding:"omitempty,oneof=read unread"`
}

type CreateTemplateRequest struct {
	TemplateType string   `json:"template_type" binding:"required"`
	Name         string   `json:"name" binding:"required"`
	Title        string   `json:"title" binding:"required"`
	Content      string   `json:"content" binding:"required"`
	Variables    []string `json:"variables"`
}

type UpdateTemplateRequest struct {
	Name      *string  `json:"name" binding:"omitempty"`
	Title     *string  `json:"title" binding:"omitempty"`
	Content   *string  `json:"content" binding:"omitempty"`
	Variables []string `json:"variables"`
}

type RenderTemplateRequest struct {
	TemplateType string                 `json:"template_type" binding:"required"`
	Data         map[string]interface{} `json:"data" binding:"required"`
}

type MarkNotificationAsReadRequest struct {
	NotificationID uint `json:"notification_id" binding:"required"`
}

type AnnouncementResponse struct {
	ID          string     `json:"id"`
	InternalID  uint       `json:"-"` // Internal database ID, not exposed in JSON
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	ContentHTML string     `json:"content_html"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	Priority    int        `json:"priority"`
	ScheduledAt *time.Time `json:"scheduled_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	ViewCount   int        `json:"view_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	IsRead      *bool      `json:"is_read,omitempty"` // nil for unauthenticated users
}

type NotificationResponse struct {
	ID          uint      `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	ContentHTML string    `json:"content_html"`
	RelatedID   *uint     `json:"related_id"`
	ReadStatus  string    `json:"read_status"`
	CreatedAt   time.Time `json:"created_at"`
}

type TemplateResponse struct {
	ID           uint      `json:"id"`
	TemplateType string    `json:"template_type"`
	Name         string    `json:"name"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	Variables    []string  `json:"variables"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

type ListResponse struct {
	Items  interface{} `json:"items"`
	Total  int64       `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

type RenderTemplateResponse struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	ContentHTML string `json:"content_html"`
}
