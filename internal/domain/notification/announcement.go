package notification

import (
	"fmt"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/notification/valueobjects"
)

type Announcement struct {
	id               uint
	title            string
	content          string
	announcementType vo.AnnouncementType
	status           vo.AnnouncementStatus
	creatorID        uint
	priority         int
	scheduledAt      *time.Time
	expiresAt        *time.Time
	viewCount        int
	createdAt        time.Time
	updatedAt        time.Time
	events           []interface{}
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
		createdAt:        createdAt,
		updatedAt:        updatedAt,
		events:           []interface{}{},
	}, nil
}

func (a *Announcement) ID() uint {
	return a.id
}

func (a *Announcement) Title() string {
	return a.title
}

func (a *Announcement) Content() string {
	return a.content
}

func (a *Announcement) Type() vo.AnnouncementType {
	return a.announcementType
}

func (a *Announcement) Status() vo.AnnouncementStatus {
	return a.status
}

func (a *Announcement) CreatorID() uint {
	return a.creatorID
}

func (a *Announcement) Priority() int {
	return a.priority
}

func (a *Announcement) ScheduledAt() *time.Time {
	return a.scheduledAt
}

func (a *Announcement) ExpiresAt() *time.Time {
	return a.expiresAt
}

func (a *Announcement) ViewCount() int {
	return a.viewCount
}

func (a *Announcement) CreatedAt() time.Time {
	return a.createdAt
}

func (a *Announcement) UpdatedAt() time.Time {
	return a.updatedAt
}

func (a *Announcement) SetID(id uint) error {
	if a.id != 0 {
		return fmt.Errorf("announcement ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("announcement ID cannot be zero")
	}
	a.id = id
	return nil
}

func (a *Announcement) Publish() error {
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

	return nil
}

func (a *Announcement) Update(title, content string, priority int, expiresAt *time.Time) error {
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

	return nil
}

func (a *Announcement) MarkAsExpired() error {
	if a.status.IsExpired() {
		return nil
	}

	if !a.status.CanTransitionTo(vo.AnnouncementStatusExpired) {
		return fmt.Errorf("cannot mark announcement with status %s as expired", a.status)
	}

	a.status = vo.AnnouncementStatusExpired
	a.updatedAt = time.Now()

	return nil
}

func (a *Announcement) IncrementViewCount() {
	a.viewCount++
}

func (a *Announcement) IsExpired() bool {
	if a.expiresAt == nil {
		return false
	}

	return time.Now().After(*a.expiresAt)
}

func (a *Announcement) GetEvents() []interface{} {
	events := make([]interface{}, len(a.events))
	copy(events, a.events)
	a.events = []interface{}{}
	return events
}

func (a *Announcement) ClearEvents() {
	a.events = []interface{}{}
}
