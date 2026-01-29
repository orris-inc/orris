package usecases

import (
	"context"
	"time"
)

type Announcement interface {
	ID() uint
	SID() string
	Title() string
	Content() string
	Type() string
	Status() string
	Priority() int
	ScheduledAt() *time.Time
	ExpiresAt() *time.Time
	ViewCount() int
	CreatedAt() time.Time
	UpdatedAt() time.Time
	Update(title, content string, priority int, expiresAt *time.Time) error
	Publish() error
	Archive() error
	IncrementViewCount()
}

type Notification interface {
	ID() uint
	UserID() uint
	Type() string
	Title() string
	Content() string
	RelatedID() *uint
	ReadStatus() string
	CreatedAt() time.Time
	MarkAsRead() error
	Archive() error
}

type NotificationTemplate interface {
	ID() uint
	TemplateType() string
	Name() string
	Title() string
	Content() string
	Variables() []string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	Render(data map[string]interface{}) (title string, content string, err error)
}

type AnnouncementRepository interface {
	Create(ctx context.Context, announcement Announcement) error
	Update(ctx context.Context, announcement Announcement) error
	Delete(ctx context.Context, id uint) error
	DeleteBySID(ctx context.Context, sid string) error
	FindByID(ctx context.Context, id uint) (Announcement, error)
	FindBySID(ctx context.Context, sid string) (Announcement, error)
	FindAll(ctx context.Context, limit, offset int) ([]Announcement, int64, error)
	FindPublished(ctx context.Context, limit, offset int) ([]Announcement, int64, error)
}

type NotificationRepository interface {
	Create(ctx context.Context, notification Notification) error
	BulkCreate(ctx context.Context, notifications []Notification) error
	Update(ctx context.Context, notification Notification) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (Notification, error)
	FindByUserID(ctx context.Context, userID uint, limit, offset int) ([]Notification, int64, error)
	FindUnreadByUserID(ctx context.Context, userID uint, limit, offset int) ([]Notification, int64, error)
	CountUnreadByUserID(ctx context.Context, userID uint) (int64, error)
	MarkAllAsReadByUserID(ctx context.Context, userID uint) error
}

type NotificationTemplateRepository interface {
	Create(ctx context.Context, template NotificationTemplate) error
	Update(ctx context.Context, template NotificationTemplate) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (NotificationTemplate, error)
	FindByType(ctx context.Context, templateType string) (NotificationTemplate, error)
	FindAll(ctx context.Context) ([]NotificationTemplate, error)
}

type UserRepository interface {
	FindAllActiveUserIDs(ctx context.Context) ([]uint, error)
}
