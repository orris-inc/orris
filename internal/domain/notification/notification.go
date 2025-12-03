package notification

import (
	"fmt"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/notification/value_objects"
)

type Notification struct {
	id               uint
	userID           uint
	notificationType vo.NotificationType
	title            string
	content          string
	relatedID        *uint
	readStatus       vo.ReadStatus
	archivedAt       *time.Time
	createdAt        time.Time
	updatedAt        time.Time
	events           []interface{}
}

func NewNotification(
	userID uint,
	notificationType vo.NotificationType,
	title string,
	content string,
	relatedID *uint,
) (*Notification, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}
	if !notificationType.IsValid() {
		return nil, fmt.Errorf("invalid notification type")
	}
	if len(title) == 0 {
		return nil, fmt.Errorf("title is required")
	}
	if len(title) > 200 {
		return nil, fmt.Errorf("title exceeds maximum length of 200 characters")
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("content is required")
	}
	if len(content) > 5000 {
		return nil, fmt.Errorf("content exceeds maximum length of 5000 characters")
	}

	now := time.Now()
	n := &Notification{
		userID:           userID,
		notificationType: notificationType,
		title:            title,
		content:          content,
		relatedID:        relatedID,
		readStatus:       vo.ReadStatusUnread,
		createdAt:        now,
		updatedAt:        now,
		events:           []interface{}{},
	}

	return n, nil
}

func ReconstructNotification(
	id uint,
	userID uint,
	notificationType vo.NotificationType,
	title string,
	content string,
	relatedID *uint,
	readStatus vo.ReadStatus,
	archivedAt *time.Time,
	createdAt, updatedAt time.Time,
) (*Notification, error) {
	if id == 0 {
		return nil, fmt.Errorf("notification ID cannot be zero")
	}
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}
	if !notificationType.IsValid() {
		return nil, fmt.Errorf("invalid notification type")
	}
	if !readStatus.IsValid() {
		return nil, fmt.Errorf("invalid read status")
	}

	return &Notification{
		id:               id,
		userID:           userID,
		notificationType: notificationType,
		title:            title,
		content:          content,
		relatedID:        relatedID,
		readStatus:       readStatus,
		archivedAt:       archivedAt,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
		events:           []interface{}{},
	}, nil
}

func (n *Notification) ID() uint {
	return n.id
}

func (n *Notification) UserID() uint {
	return n.userID
}

func (n *Notification) Type() vo.NotificationType {
	return n.notificationType
}

func (n *Notification) Title() string {
	return n.title
}

func (n *Notification) Content() string {
	return n.content
}

func (n *Notification) RelatedID() *uint {
	return n.relatedID
}

func (n *Notification) ReadStatus() vo.ReadStatus {
	return n.readStatus
}

func (n *Notification) ArchivedAt() *time.Time {
	return n.archivedAt
}

func (n *Notification) CreatedAt() time.Time {
	return n.createdAt
}

func (n *Notification) UpdatedAt() time.Time {
	return n.updatedAt
}

func (n *Notification) SetID(id uint) error {
	if n.id != 0 {
		return fmt.Errorf("notification ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("notification ID cannot be zero")
	}
	n.id = id
	return nil
}

func (n *Notification) MarkAsRead() error {
	if n.readStatus.IsRead() {
		return nil
	}

	n.readStatus = vo.ReadStatusRead
	n.updatedAt = time.Now()

	return nil
}

func (n *Notification) Archive() error {
	if n.archivedAt != nil {
		return fmt.Errorf("notification is already archived")
	}

	now := time.Now()
	n.archivedAt = &now
	n.updatedAt = now

	return nil
}

func (n *Notification) IsArchived() bool {
	return n.archivedAt != nil
}

func (n *Notification) recordEventUnsafe(event interface{}) {
	n.events = append(n.events, event)
}

func (n *Notification) GetEvents() []interface{} {
	events := make([]interface{}, len(n.events))
	copy(events, n.events)
	n.events = []interface{}{}
	return events
}

func (n *Notification) ClearEvents() {
	n.events = []interface{}{}
}
