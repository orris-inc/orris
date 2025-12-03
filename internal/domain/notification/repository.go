package notification

import (
	"context"

	vo "github.com/orris-inc/orris/internal/domain/notification/value_objects"
)

type AnnouncementRepository interface {
	Create(ctx context.Context, announcement *Announcement) error
	GetByID(ctx context.Context, id uint) (*Announcement, error)
	Update(ctx context.Context, announcement *Announcement) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, limit, offset int) ([]*Announcement, int64, error)
	FindBySpecification(ctx context.Context, spec Specification, limit, offset int) ([]*Announcement, int64, error)
	IncrementViewCount(ctx context.Context, id uint) error
	FindByStatus(ctx context.Context, status vo.AnnouncementStatus, limit, offset int) ([]*Announcement, int64, error)
}

type NotificationRepository interface {
	Create(ctx context.Context, notification *Notification) error
	GetByID(ctx context.Context, id uint) (*Notification, error)
	Update(ctx context.Context, notification *Notification) error
	Delete(ctx context.Context, id uint) error
	ListByUserID(ctx context.Context, userID uint, limit, offset int) ([]*Notification, int64, error)
	CountUnread(ctx context.Context, userID uint) (int64, error)
	MarkAsRead(ctx context.Context, id uint) error
	BulkCreate(ctx context.Context, notifications []*Notification) error
	FindBySpecification(ctx context.Context, spec Specification, limit, offset int) ([]*Notification, int64, error)
}

type NotificationTemplateRepository interface {
	Create(ctx context.Context, template *NotificationTemplate) error
	GetByID(ctx context.Context, id uint) (*NotificationTemplate, error)
	Update(ctx context.Context, template *NotificationTemplate) error
	Delete(ctx context.Context, id uint) error
	GetByTemplateType(ctx context.Context, templateType vo.TemplateType) (*NotificationTemplate, error)
	ListEnabled(ctx context.Context) ([]*NotificationTemplate, error)
	List(ctx context.Context, limit, offset int) ([]*NotificationTemplate, int64, error)
}
