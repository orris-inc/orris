package user

import (
	"fmt"
	"time"

	"orris/internal/domain/shared/events"
)

// Event types
const (
	EventTypeUserCreated          = "user.created"
	EventTypeUserUpdated          = "user.updated"
	EventTypeUserDeleted          = "user.deleted"
	EventTypeUserStatusChanged    = "user.status.changed"
	EventTypeUserEmailChanged     = "user.email.changed"
	EventTypeUserNameChanged      = "user.name.changed"
	EventTypeUserPasswordChanged  = "user.password.changed"
	EventTypeUserEmailVerified    = "user.email.verified"
	EventTypeUserPasswordResetReq = "user.password.reset.requested"
	EventTypeUserAccountLocked    = "user.account.locked"
)

// UserCreatedEvent is emitted when a new user is created
type UserCreatedEvent struct {
	events.BaseEvent
	UserID    uint      `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// NewUserCreatedEvent creates a new user created event
func NewUserCreatedEvent(userID uint, email, name, status string) UserCreatedEvent {
	return UserCreatedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("user:%d", userID),
			EventType:   EventTypeUserCreated,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		UserID:    userID,
		Email:     email,
		Name:      name,
		Status:    status,
		CreatedAt: time.Now(),
	}
}

// UserUpdatedEvent is emitted when a user is updated
type UserUpdatedEvent struct {
	events.BaseEvent
	UserID    uint                   `json:"user_id"`
	UpdatedAt time.Time              `json:"updated_at"`
	Changes   map[string]interface{} `json:"changes"`
}

// NewUserUpdatedEvent creates a new user updated event
func NewUserUpdatedEvent(userID uint, changes map[string]interface{}) UserUpdatedEvent {
	return UserUpdatedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("user:%d", userID),
			EventType:   EventTypeUserUpdated,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		UserID:    userID,
		UpdatedAt: time.Now(),
		Changes:   changes,
	}
}

// UserDeletedEvent is emitted when a user is deleted
type UserDeletedEvent struct {
	events.BaseEvent
	UserID    uint      `json:"user_id"`
	DeletedAt time.Time `json:"deleted_at"`
	OldStatus string    `json:"old_status"`
}

// NewUserDeletedEvent creates a new user deleted event
func NewUserDeletedEvent(userID uint, oldStatus string) UserDeletedEvent {
	return UserDeletedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("user:%d", userID),
			EventType:   EventTypeUserDeleted,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		UserID:    userID,
		DeletedAt: time.Now(),
		OldStatus: oldStatus,
	}
}

// UserStatusChangedEvent is emitted when a user's status changes
type UserStatusChangedEvent struct {
	events.BaseEvent
	UserID    uint      `json:"user_id"`
	OldStatus string    `json:"old_status"`
	NewStatus string    `json:"new_status"`
	Reason    string    `json:"reason"`
	ChangedAt time.Time `json:"changed_at"`
}

// NewUserStatusChangedEvent creates a new user status changed event
func NewUserStatusChangedEvent(userID uint, oldStatus, newStatus, reason string) UserStatusChangedEvent {
	return UserStatusChangedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("user:%d", userID),
			EventType:   EventTypeUserStatusChanged,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		UserID:    userID,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Reason:    reason,
		ChangedAt: time.Now(),
	}
}

// UserEmailChangedEvent is emitted when a user's email changes
type UserEmailChangedEvent struct {
	events.BaseEvent
	UserID    uint      `json:"user_id"`
	OldEmail  string    `json:"old_email"`
	NewEmail  string    `json:"new_email"`
	ChangedAt time.Time `json:"changed_at"`
}

// NewUserEmailChangedEvent creates a new user email changed event
func NewUserEmailChangedEvent(userID uint, oldEmail, newEmail string) UserEmailChangedEvent {
	return UserEmailChangedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("user:%d", userID),
			EventType:   EventTypeUserEmailChanged,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		UserID:    userID,
		OldEmail:  oldEmail,
		NewEmail:  newEmail,
		ChangedAt: time.Now(),
	}
}

// UserNameChangedEvent is emitted when a user's name changes
type UserNameChangedEvent struct {
	events.BaseEvent
	UserID    uint      `json:"user_id"`
	OldName   string    `json:"old_name"`
	NewName   string    `json:"new_name"`
	ChangedAt time.Time `json:"changed_at"`
}

// NewUserNameChangedEvent creates a new user name changed event
func NewUserNameChangedEvent(userID uint, oldName, newName string) UserNameChangedEvent {
	return UserNameChangedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("user:%d", userID),
			EventType:   EventTypeUserNameChanged,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		UserID:    userID,
		OldName:   oldName,
		NewName:   newName,
		ChangedAt: time.Now(),
	}
}

type UserPasswordChangedEvent struct {
	events.BaseEvent
	UserID    uint      `json:"user_id"`
	ChangedAt time.Time `json:"changed_at"`
}

func NewUserPasswordChangedEvent(userID uint) UserPasswordChangedEvent {
	return UserPasswordChangedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("user:%d", userID),
			EventType:   EventTypeUserPasswordChanged,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		UserID:    userID,
		ChangedAt: time.Now(),
	}
}

type UserEmailVerifiedEvent struct {
	events.BaseEvent
	UserID     uint      `json:"user_id"`
	Email      string    `json:"email"`
	VerifiedAt time.Time `json:"verified_at"`
}

func NewUserEmailVerifiedEvent(userID uint, email string) UserEmailVerifiedEvent {
	return UserEmailVerifiedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("user:%d", userID),
			EventType:   EventTypeUserEmailVerified,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		UserID:     userID,
		Email:      email,
		VerifiedAt: time.Now(),
	}
}

type UserPasswordResetRequestedEvent struct {
	events.BaseEvent
	UserID      uint      `json:"user_id"`
	Email       string    `json:"email"`
	RequestedAt time.Time `json:"requested_at"`
}

func NewUserPasswordResetRequestedEvent(userID uint, email string) UserPasswordResetRequestedEvent {
	return UserPasswordResetRequestedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("user:%d", userID),
			EventType:   EventTypeUserPasswordResetReq,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		UserID:      userID,
		Email:       email,
		RequestedAt: time.Now(),
	}
}

type UserAccountLockedEvent struct {
	events.BaseEvent
	UserID         uint          `json:"user_id"`
	FailedAttempts int           `json:"failed_attempts"`
	LockDuration   time.Duration `json:"lock_duration_seconds"`
	LockedAt       time.Time     `json:"locked_at"`
}

func NewUserAccountLockedEvent(userID uint, failedAttempts int, lockDuration time.Duration) UserAccountLockedEvent {
	return UserAccountLockedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("user:%d", userID),
			EventType:   EventTypeUserAccountLocked,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		UserID:         userID,
		FailedAttempts: failedAttempts,
		LockDuration:   lockDuration,
		LockedAt:       time.Now(),
	}
}
