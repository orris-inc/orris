package notification

import (
	"fmt"
	"sync"
	"time"

	vo "orris/internal/domain/notification/value_objects"
)

type Announcement struct {
	id          uint
	title       string
	content     string
	announcementType vo.AnnouncementType
	status      vo.AnnouncementStatus
	creatorID   uint
	priority    int
	scheduledAt *time.Time
	expiresAt   *time.Time
	viewCount   int
	version     int
	createdAt   time.Time
	updatedAt   time.Time
	events      []interface{}
	mu          sync.RWMutex
}

func NewAnnouncement(
	title string,
	content string,
	announcementType vo.AnnouncementType,
	creatorID uint,
	priority int,
	scheduledAt *time.Time,
	expiresAt *time.Time,
) (*Announcement, error) {
	if len(title) == 0 {
		return nil, fmt.Errorf("title is required")
	}
	if len(title) > 200 {
		return nil, fmt.Errorf("title exceeds maximum length of 200 characters")
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("content is required")
	}
	if len(content) > 10000 {
		return nil, fmt.Errorf("content exceeds maximum length of 10000 characters")
	}
	if !announcementType.IsValid() {
		return nil, fmt.Errorf("invalid announcement type")
	}
	if creatorID == 0 {
		return nil, fmt.Errorf("creator ID is required")
	}
	if priority < 1 || priority > 5 {
		return nil, fmt.Errorf("priority must be between 1 and 5")
	}
	if expiresAt != nil && scheduledAt != nil && expiresAt.Before(*scheduledAt) {
		return nil, fmt.Errorf("expires at must be after scheduled at")
	}

	now := time.Now()
	a := &Announcement{
		title:            title,
		content:          content,
		announcementType: announcementType,
		status:           vo.AnnouncementStatusDraft,
		creatorID:        creatorID,
		priority:         priority,
		scheduledAt:      scheduledAt,
		expiresAt:        expiresAt,
		viewCount:        0,
		version:          1,
		createdAt:        now,
		updatedAt:        now,
		events:           []interface{}{},
	}

	return a, nil
}

func ReconstructAnnouncement(
	id uint,
	title string,
	content string,
	announcementType vo.AnnouncementType,
	status vo.AnnouncementStatus,
	creatorID uint,
	priority int,
	scheduledAt *time.Time,
	expiresAt *time.Time,
	viewCount int,
	version int,
	createdAt, updatedAt time.Time,
) (*Announcement, error) {
	if id == 0 {
		return nil, fmt.Errorf("announcement ID cannot be zero")
	}
	if len(title) == 0 {
		return nil, fmt.Errorf("title is required")
	}
	if !announcementType.IsValid() {
		return nil, fmt.Errorf("invalid announcement type")
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid status")
	}
	if priority < 1 || priority > 5 {
		return nil, fmt.Errorf("priority must be between 1 and 5")
	}

	return &Announcement{
		id:               id,
		title:            title,
		content:          content,
		announcementType: announcementType,
		status:           status,
		creatorID:        creatorID,
		priority:         priority,
		scheduledAt:      scheduledAt,
		expiresAt:        expiresAt,
		viewCount:        viewCount,
		version:          version,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
		events:           []interface{}{},
	}, nil
}

func (a *Announcement) ID() uint {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.id
}

func (a *Announcement) Title() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.title
}

func (a *Announcement) Content() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.content
}

func (a *Announcement) Type() vo.AnnouncementType {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.announcementType
}

func (a *Announcement) Status() vo.AnnouncementStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

func (a *Announcement) CreatorID() uint {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.creatorID
}

func (a *Announcement) Priority() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.priority
}

func (a *Announcement) ScheduledAt() *time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.scheduledAt
}

func (a *Announcement) ExpiresAt() *time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.expiresAt
}

func (a *Announcement) ViewCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.viewCount
}

func (a *Announcement) Version() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.version
}

func (a *Announcement) CreatedAt() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.createdAt
}

func (a *Announcement) UpdatedAt() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.updatedAt
}

func (a *Announcement) SetID(id uint) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.id != 0 {
		return fmt.Errorf("announcement ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("announcement ID cannot be zero")
	}
	a.id = id
	return nil
}

func (a *Announcement) Publish(sendNotification bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.status.CanTransitionTo(vo.AnnouncementStatusPublished) {
		return fmt.Errorf("cannot publish announcement with status %s", a.status)
	}

	now := time.Now()
	if a.scheduledAt != nil && now.Before(*a.scheduledAt) {
		return fmt.Errorf("cannot publish before scheduled time")
	}

	if a.expiresAt != nil && now.After(*a.expiresAt) {
		return fmt.Errorf("cannot publish expired announcement")
	}

	a.status = vo.AnnouncementStatusPublished
	a.updatedAt = now
	a.version++

	a.recordEventUnsafe(AnnouncementPublishedEvent{
		AnnouncementID:   a.id,
		SendNotification: sendNotification,
		PublishedAt:      now,
	})

	return nil
}

func (a *Announcement) Update(title, content string, priority int, expiresAt *time.Time) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(title) == 0 {
		return fmt.Errorf("title is required")
	}
	if len(title) > 200 {
		return fmt.Errorf("title exceeds maximum length of 200 characters")
	}
	if len(content) == 0 {
		return fmt.Errorf("content is required")
	}
	if len(content) > 10000 {
		return fmt.Errorf("content exceeds maximum length of 10000 characters")
	}
	if priority < 1 || priority > 5 {
		return fmt.Errorf("priority must be between 1 and 5")
	}

	a.title = title
	a.content = content
	a.priority = priority
	a.expiresAt = expiresAt
	a.updatedAt = time.Now()
	a.version++

	return nil
}

func (a *Announcement) MarkAsExpired() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status.IsExpired() {
		return nil
	}

	if !a.status.CanTransitionTo(vo.AnnouncementStatusExpired) {
		return fmt.Errorf("cannot mark announcement with status %s as expired", a.status)
	}

	a.status = vo.AnnouncementStatusExpired
	a.updatedAt = time.Now()
	a.version++

	return nil
}

func (a *Announcement) IncrementViewCount() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.viewCount++
}

func (a *Announcement) IsExpired() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.expiresAt == nil {
		return false
	}

	return time.Now().After(*a.expiresAt)
}

func (a *Announcement) recordEventUnsafe(event interface{}) {
	a.events = append(a.events, event)
}

func (a *Announcement) GetEvents() []interface{} {
	a.mu.Lock()
	defer a.mu.Unlock()
	events := make([]interface{}, len(a.events))
	copy(events, a.events)
	a.events = []interface{}{}
	return events
}

func (a *Announcement) ClearEvents() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = []interface{}{}
}
