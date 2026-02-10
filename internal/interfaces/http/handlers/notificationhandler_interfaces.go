package handlers

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/application/notification/dto"
	"github.com/orris-inc/orris/internal/domain/user"
)

// Service interface for NotificationHandler - enables unit testing with mocks.

type notificationService interface {
	CreateAnnouncement(ctx context.Context, req dto.CreateAnnouncementRequest) (*dto.AnnouncementResponse, error)
	UpdateAnnouncement(ctx context.Context, sid string, req dto.UpdateAnnouncementRequest) (*dto.AnnouncementResponse, error)
	DeleteAnnouncement(ctx context.Context, sid string) error
	PublishAnnouncement(ctx context.Context, sid string) (*dto.AnnouncementResponse, error)
	ArchiveAnnouncement(ctx context.Context, sid string) (*dto.AnnouncementResponse, error)
	ListAnnouncements(ctx context.Context, limit, offset int) (*dto.ListResponse, error)
	ListPublishedAnnouncements(ctx context.Context, limit, offset int) (*dto.ListResponse, error)
	GetAnnouncement(ctx context.Context, sid string) (*dto.AnnouncementResponse, error)
	GetAnnouncementUnreadCount(ctx context.Context, userID uint, userReadAt *time.Time) (int64, error)
	MarkAnnouncementAsRead(ctx context.Context, userID uint, sid string) error
	GetReadStatusByIDs(ctx context.Context, userID uint, announcementIDs []uint) (map[uint]bool, error)

	ListNotifications(ctx context.Context, req dto.ListNotificationsRequest) (*dto.ListResponse, error)
	MarkNotificationAsRead(ctx context.Context, id uint, userID uint) error
	MarkAllNotificationsAsRead(ctx context.Context, userID uint) error
	ArchiveNotification(ctx context.Context, id uint, userID uint) error
	DeleteNotification(ctx context.Context, id uint, userID uint) error
	GetUnreadCount(ctx context.Context, userID uint) (*dto.UnreadCountResponse, error)

	CreateTemplate(ctx context.Context, req dto.CreateTemplateRequest) (*dto.TemplateResponse, error)
	RenderTemplate(ctx context.Context, req dto.RenderTemplateRequest) (*dto.RenderTemplateResponse, error)
	ListTemplates(ctx context.Context) ([]*dto.TemplateResponse, error)
}

// notificationUserRepo is the subset of user.Repository used by NotificationHandler.
type notificationUserRepo interface {
	GetByID(ctx context.Context, id uint) (*user.User, error)
	Update(ctx context.Context, u *user.User) error
}
