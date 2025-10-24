package subscription

import (
	"errors"
	"time"
)

const (
	EventTypeCreated     = "created"
	EventTypeActivated   = "activated"
	EventTypeCancelled   = "cancelled"
	EventTypeRenewed     = "renewed"
	EventTypePlanChanged = "plan_changed"
	EventTypeSuspended   = "suspended"
	EventTypeReactivated = "reactivated"
	EventTypeExpired     = "expired"
)

var ValidEventTypes = map[string]bool{
	EventTypeCreated:     true,
	EventTypeActivated:   true,
	EventTypeCancelled:   true,
	EventTypeRenewed:     true,
	EventTypePlanChanged: true,
	EventTypeSuspended:   true,
	EventTypeReactivated: true,
	EventTypeExpired:     true,
}

var (
	ErrInvalidEventType = errors.New("invalid event type")
	ErrHistoryImmutable = errors.New("history record is immutable")
)

type SubscriptionHistory struct {
	id             uint
	subscriptionID uint
	eventType      string
	oldPlanID      *uint
	newPlanID      *uint
	reason         *string
	metadata       map[string]interface{}
	createdAt      time.Time
}

func NewSubscriptionHistory(subscriptionID uint, eventType string) (*SubscriptionHistory, error) {
	if subscriptionID == 0 {
		return nil, errors.New("subscription ID cannot be zero")
	}

	if eventType == "" {
		return nil, errors.New("event type cannot be empty")
	}

	if !ValidEventTypes[eventType] {
		return nil, ErrInvalidEventType
	}

	return &SubscriptionHistory{
		subscriptionID: subscriptionID,
		eventType:      eventType,
		metadata:       make(map[string]interface{}),
		createdAt:      time.Now(),
	}, nil
}

func ReconstructSubscriptionHistory(
	id uint,
	subscriptionID uint,
	eventType string,
	oldPlanID *uint,
	newPlanID *uint,
	reason *string,
	metadata map[string]interface{},
	createdAt time.Time,
) (*SubscriptionHistory, error) {
	if id == 0 {
		return nil, errors.New("history ID cannot be zero")
	}

	if subscriptionID == 0 {
		return nil, errors.New("subscription ID cannot be zero")
	}

	if eventType == "" {
		return nil, errors.New("event type cannot be empty")
	}

	if !ValidEventTypes[eventType] {
		return nil, ErrInvalidEventType
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &SubscriptionHistory{
		id:             id,
		subscriptionID: subscriptionID,
		eventType:      eventType,
		oldPlanID:      oldPlanID,
		newPlanID:      newPlanID,
		reason:         reason,
		metadata:       metadata,
		createdAt:      createdAt,
	}, nil
}

func (h *SubscriptionHistory) SetPlanChange(oldPlanID, newPlanID uint) {
	h.oldPlanID = &oldPlanID
	h.newPlanID = &newPlanID
}

func (h *SubscriptionHistory) SetReason(reason string) {
	h.reason = &reason
}

func (h *SubscriptionHistory) SetMetadata(metadata map[string]interface{}) {
	if metadata == nil {
		h.metadata = make(map[string]interface{})
		return
	}
	h.metadata = metadata
}

func (h *SubscriptionHistory) AddMetadata(key string, value interface{}) {
	if h.metadata == nil {
		h.metadata = make(map[string]interface{})
	}
	h.metadata[key] = value
}

func (h *SubscriptionHistory) ID() uint {
	return h.id
}

func (h *SubscriptionHistory) SubscriptionID() uint {
	return h.subscriptionID
}

func (h *SubscriptionHistory) EventType() string {
	return h.eventType
}

func (h *SubscriptionHistory) OldPlanID() *uint {
	return h.oldPlanID
}

func (h *SubscriptionHistory) NewPlanID() *uint {
	return h.newPlanID
}

func (h *SubscriptionHistory) Reason() *string {
	return h.reason
}

func (h *SubscriptionHistory) Metadata() map[string]interface{} {
	if h.metadata == nil {
		return make(map[string]interface{})
	}
	metadata := make(map[string]interface{}, len(h.metadata))
	for k, v := range h.metadata {
		metadata[k] = v
	}
	return metadata
}

func (h *SubscriptionHistory) CreatedAt() time.Time {
	return h.createdAt
}

func (h *SubscriptionHistory) IsPlanChange() bool {
	return h.eventType == EventTypePlanChanged
}

func (h *SubscriptionHistory) IsCancellation() bool {
	return h.eventType == EventTypeCancelled
}

func (h *SubscriptionHistory) IsRenewal() bool {
	return h.eventType == EventTypeRenewed
}

func (h *SubscriptionHistory) IsActivation() bool {
	return h.eventType == EventTypeActivated
}

func (h *SubscriptionHistory) IsCreation() bool {
	return h.eventType == EventTypeCreated
}

func (h *SubscriptionHistory) IsSuspension() bool {
	return h.eventType == EventTypeSuspended
}

func (h *SubscriptionHistory) IsReactivation() bool {
	return h.eventType == EventTypeReactivated
}

func (h *SubscriptionHistory) IsExpiration() bool {
	return h.eventType == EventTypeExpired
}
