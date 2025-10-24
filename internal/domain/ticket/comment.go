package ticket

import (
	"fmt"
	"sync"
	"time"
)

type Comment struct {
	id         uint
	ticketID   uint
	userID     uint
	content    string
	isInternal bool
	createdAt  time.Time
	updatedAt  time.Time
	mu         sync.RWMutex
}

func NewComment(
	ticketID uint,
	userID uint,
	content string,
	isInternal bool,
) (*Comment, error) {
	if ticketID == 0 {
		return nil, fmt.Errorf("ticket ID is required")
	}
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("content cannot be empty")
	}
	if len(content) > 5000 {
		return nil, fmt.Errorf("content exceeds maximum length of 5000 characters")
	}

	now := time.Now()
	return &Comment{
		ticketID:   ticketID,
		userID:     userID,
		content:    content,
		isInternal: isInternal,
		createdAt:  now,
		updatedAt:  now,
	}, nil
}

func ReconstructComment(
	id uint,
	ticketID uint,
	userID uint,
	content string,
	isInternal bool,
	createdAt, updatedAt time.Time,
) (*Comment, error) {
	if id == 0 {
		return nil, fmt.Errorf("comment ID cannot be zero")
	}
	if ticketID == 0 {
		return nil, fmt.Errorf("ticket ID is required")
	}
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}

	return &Comment{
		id:         id,
		ticketID:   ticketID,
		userID:     userID,
		content:    content,
		isInternal: isInternal,
		createdAt:  createdAt,
		updatedAt:  updatedAt,
	}, nil
}

func (c *Comment) ID() uint {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.id
}

func (c *Comment) TicketID() uint {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ticketID
}

func (c *Comment) UserID() uint {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.userID
}

func (c *Comment) Content() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.content
}

func (c *Comment) IsInternal() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isInternal
}

func (c *Comment) CreatedAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.createdAt
}

func (c *Comment) UpdatedAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.updatedAt
}

func (c *Comment) SetID(id uint) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.id != 0 {
		return fmt.Errorf("comment ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("comment ID cannot be zero")
	}
	c.id = id
	return nil
}

func (c *Comment) UpdateContent(content string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(content) == 0 {
		return fmt.Errorf("content cannot be empty")
	}
	if len(content) > 5000 {
		return fmt.Errorf("content exceeds maximum length of 5000 characters")
	}

	c.content = content
	c.updatedAt = time.Now()
	return nil
}
