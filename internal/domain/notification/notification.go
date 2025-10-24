package notification

import (
	"fmt"
	"sync"
	"time"

	vo "orris/internal/domain/notification/value_objects"
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
	version          int
	createdAt        time.Time
	updatedAt        time.Time
	events           []interface{}
	mu               sync.RWMutex
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
		version:          1,
		createdAt:        now,
		updatedAt:        now,
		events:           []interface{}{},
	}

	n.recordEventUnsafe(NotificationCreatedEvent{
		NotificationID: n.id,
		UserID:         userID,
		CreatedAt:      now,
	})

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
	version int,
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
		version:          version,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
		events:           []interface{}{},
	}, nil
}

func (n *Notification) ID() uint {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.id
}

func (n *Notification) UserID() uint {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.userID
}

func (n *Notification) Type() vo.NotificationType {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.notificationType
}

func (n *Notification) Title() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.title
}

func (n *Notification) Content() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.content
}

func (n *Notification) RelatedID() *uint {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.relatedID
}

func (n *Notification) ReadStatus() vo.ReadStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.readStatus
}

func (n *Notification) ArchivedAt() *time.Time {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.archivedAt
}

func (n *Notification) Version() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.version
}

func (n *Notification) CreatedAt() time.Time {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.createdAt
}

func (n *Notification) UpdatedAt() time.Time {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.updatedAt
}

func (n *Notification) SetID(id uint) error {
	n.mu.Lock()
	defer n.mu.Unlock()

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
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.readStatus.IsRead() {
		return nil
	}

	n.readStatus = vo.ReadStatusRead
	n.updatedAt = time.Now()
	n.version++

	return nil
}

func (n *Notification) Archive() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.archivedAt != nil {
		return fmt.Errorf("notification is already archived")
	}

	now := time.Now()
	n.archivedAt = &now
	n.updatedAt = now
	n.version++

	return nil
}

func (n *Notification) IsArchived() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.archivedAt != nil
}

func (n *Notification) recordEventUnsafe(event interface{}) {
	n.events = append(n.events, event)
}

func (n *Notification) GetEvents() []interface{} {
	n.mu.Lock()
	defer n.mu.Unlock()
	events := make([]interface{}, len(n.events))
	copy(events, n.events)
	n.events = []interface{}{}
	return events
}

func (n *Notification) ClearEvents() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.events = []interface{}{}
}
