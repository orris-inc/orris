package subscription

import (
	"testing"
	"time"
)

func TestNewSubscriptionHistory(t *testing.T) {
	tests := []struct {
		name           string
		subscriptionID uint
		eventType      string
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "valid created event",
			subscriptionID: 1,
			eventType:      EventTypeCreated,
			wantErr:        false,
		},
		{
			name:           "valid activated event",
			subscriptionID: 1,
			eventType:      EventTypeActivated,
			wantErr:        false,
		},
		{
			name:           "valid cancelled event",
			subscriptionID: 1,
			eventType:      EventTypeCancelled,
			wantErr:        false,
		},
		{
			name:           "valid renewed event",
			subscriptionID: 1,
			eventType:      EventTypeRenewed,
			wantErr:        false,
		},
		{
			name:           "valid plan changed event",
			subscriptionID: 1,
			eventType:      EventTypePlanChanged,
			wantErr:        false,
		},
		{
			name:           "zero subscription ID",
			subscriptionID: 0,
			eventType:      EventTypeCreated,
			wantErr:        true,
			errMsg:         "subscription ID cannot be zero",
		},
		{
			name:           "empty event type",
			subscriptionID: 1,
			eventType:      "",
			wantErr:        true,
			errMsg:         "event type cannot be empty",
		},
		{
			name:           "invalid event type",
			subscriptionID: 1,
			eventType:      "invalid_event",
			wantErr:        true,
			errMsg:         "invalid event type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history, err := NewSubscriptionHistory(tt.subscriptionID, tt.eventType)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewSubscriptionHistory() expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("NewSubscriptionHistory() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("NewSubscriptionHistory() unexpected error = %v", err)
				return
			}

			if history.subscriptionID != tt.subscriptionID {
				t.Errorf("subscriptionID = %v, want %v", history.subscriptionID, tt.subscriptionID)
			}

			if history.eventType != tt.eventType {
				t.Errorf("eventType = %v, want %v", history.eventType, tt.eventType)
			}

			if history.metadata == nil {
				t.Errorf("metadata should be initialized")
			}

			if history.createdAt.IsZero() {
				t.Errorf("createdAt should be set")
			}
		})
	}
}

func TestReconstructSubscriptionHistory(t *testing.T) {
	now := time.Now()
	reason := "User requested cancellation"
	oldPlanID := uint(1)
	newPlanID := uint(2)
	metadata := map[string]interface{}{
		"user_id": 123,
		"ip":      "192.168.1.1",
	}

	tests := []struct {
		name           string
		id             uint
		subscriptionID uint
		eventType      string
		oldPlanID      *uint
		newPlanID      *uint
		reason         *string
		metadata       map[string]interface{}
		createdAt      time.Time
		wantErr        bool
	}{
		{
			name:           "valid reconstruction",
			id:             1,
			subscriptionID: 100,
			eventType:      EventTypePlanChanged,
			oldPlanID:      &oldPlanID,
			newPlanID:      &newPlanID,
			reason:         &reason,
			metadata:       metadata,
			createdAt:      now,
			wantErr:        false,
		},
		{
			name:           "reconstruction with nil metadata",
			id:             2,
			subscriptionID: 100,
			eventType:      EventTypeCreated,
			oldPlanID:      nil,
			newPlanID:      nil,
			reason:         nil,
			metadata:       nil,
			createdAt:      now,
			wantErr:        false,
		},
		{
			name:           "zero ID",
			id:             0,
			subscriptionID: 100,
			eventType:      EventTypeCreated,
			oldPlanID:      nil,
			newPlanID:      nil,
			reason:         nil,
			metadata:       nil,
			createdAt:      now,
			wantErr:        true,
		},
		{
			name:           "zero subscription ID",
			id:             1,
			subscriptionID: 0,
			eventType:      EventTypeCreated,
			oldPlanID:      nil,
			newPlanID:      nil,
			reason:         nil,
			metadata:       nil,
			createdAt:      now,
			wantErr:        true,
		},
		{
			name:           "invalid event type",
			id:             1,
			subscriptionID: 100,
			eventType:      "invalid",
			oldPlanID:      nil,
			newPlanID:      nil,
			reason:         nil,
			metadata:       nil,
			createdAt:      now,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history, err := ReconstructSubscriptionHistory(
				tt.id,
				tt.subscriptionID,
				tt.eventType,
				tt.oldPlanID,
				tt.newPlanID,
				tt.reason,
				tt.metadata,
				tt.createdAt,
			)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ReconstructSubscriptionHistory() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ReconstructSubscriptionHistory() unexpected error = %v", err)
				return
			}

			if history.id != tt.id {
				t.Errorf("id = %v, want %v", history.id, tt.id)
			}

			if history.metadata == nil {
				t.Errorf("metadata should be initialized")
			}
		})
	}
}

func TestSubscriptionHistory_SetPlanChange(t *testing.T) {
	history, _ := NewSubscriptionHistory(1, EventTypePlanChanged)

	oldPlanID := uint(1)
	newPlanID := uint(2)

	history.SetPlanChange(oldPlanID, newPlanID)

	if history.oldPlanID == nil || *history.oldPlanID != oldPlanID {
		t.Errorf("oldPlanID = %v, want %v", history.oldPlanID, oldPlanID)
	}

	if history.newPlanID == nil || *history.newPlanID != newPlanID {
		t.Errorf("newPlanID = %v, want %v", history.newPlanID, newPlanID)
	}
}

func TestSubscriptionHistory_SetReason(t *testing.T) {
	history, _ := NewSubscriptionHistory(1, EventTypeCancelled)

	reason := "User requested cancellation"
	history.SetReason(reason)

	if history.reason == nil || *history.reason != reason {
		t.Errorf("reason = %v, want %v", history.reason, reason)
	}
}

func TestSubscriptionHistory_SetMetadata(t *testing.T) {
	history, _ := NewSubscriptionHistory(1, EventTypeCreated)

	tests := []struct {
		name     string
		metadata map[string]interface{}
	}{
		{
			name: "set valid metadata",
			metadata: map[string]interface{}{
				"user_id": 123,
				"ip":      "192.168.1.1",
			},
		},
		{
			name:     "set nil metadata",
			metadata: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history.SetMetadata(tt.metadata)

			if history.metadata == nil {
				t.Errorf("metadata should be initialized")
			}

			if tt.metadata != nil {
				for k, v := range tt.metadata {
					if history.metadata[k] != v {
						t.Errorf("metadata[%s] = %v, want %v", k, history.metadata[k], v)
					}
				}
			}
		})
	}
}

func TestSubscriptionHistory_AddMetadata(t *testing.T) {
	history, _ := NewSubscriptionHistory(1, EventTypeCreated)

	history.AddMetadata("user_id", 123)
	history.AddMetadata("ip", "192.168.1.1")

	if history.metadata["user_id"] != 123 {
		t.Errorf("metadata[user_id] = %v, want 123", history.metadata["user_id"])
	}

	if history.metadata["ip"] != "192.168.1.1" {
		t.Errorf("metadata[ip] = %v, want 192.168.1.1", history.metadata["ip"])
	}
}

func TestSubscriptionHistory_Metadata(t *testing.T) {
	history, _ := NewSubscriptionHistory(1, EventTypeCreated)

	originalMetadata := map[string]interface{}{
		"user_id": 123,
		"ip":      "192.168.1.1",
	}

	history.SetMetadata(originalMetadata)

	copiedMetadata := history.Metadata()

	copiedMetadata["modified"] = true

	if _, exists := history.metadata["modified"]; exists {
		t.Errorf("original metadata should not be modified")
	}
}

func TestSubscriptionHistory_EventTypeCheckers(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		checker   func(*SubscriptionHistory) bool
		want      bool
	}{
		{
			name:      "is plan change",
			eventType: EventTypePlanChanged,
			checker:   (*SubscriptionHistory).IsPlanChange,
			want:      true,
		},
		{
			name:      "is not plan change",
			eventType: EventTypeCreated,
			checker:   (*SubscriptionHistory).IsPlanChange,
			want:      false,
		},
		{
			name:      "is cancellation",
			eventType: EventTypeCancelled,
			checker:   (*SubscriptionHistory).IsCancellation,
			want:      true,
		},
		{
			name:      "is renewal",
			eventType: EventTypeRenewed,
			checker:   (*SubscriptionHistory).IsRenewal,
			want:      true,
		},
		{
			name:      "is activation",
			eventType: EventTypeActivated,
			checker:   (*SubscriptionHistory).IsActivation,
			want:      true,
		},
		{
			name:      "is creation",
			eventType: EventTypeCreated,
			checker:   (*SubscriptionHistory).IsCreation,
			want:      true,
		},
		{
			name:      "is suspension",
			eventType: EventTypeSuspended,
			checker:   (*SubscriptionHistory).IsSuspension,
			want:      true,
		},
		{
			name:      "is reactivation",
			eventType: EventTypeReactivated,
			checker:   (*SubscriptionHistory).IsReactivation,
			want:      true,
		},
		{
			name:      "is expiration",
			eventType: EventTypeExpired,
			checker:   (*SubscriptionHistory).IsExpiration,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history, _ := NewSubscriptionHistory(1, tt.eventType)

			if got := tt.checker(history); got != tt.want {
				t.Errorf("checker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubscriptionHistory_Getters(t *testing.T) {
	now := time.Now()
	reason := "Test reason"
	oldPlanID := uint(1)
	newPlanID := uint(2)
	metadata := map[string]interface{}{
		"user_id": 123,
	}

	history, _ := ReconstructSubscriptionHistory(
		1,
		100,
		EventTypePlanChanged,
		&oldPlanID,
		&newPlanID,
		&reason,
		metadata,
		now,
	)

	if history.ID() != 1 {
		t.Errorf("ID() = %v, want 1", history.ID())
	}

	if history.SubscriptionID() != 100 {
		t.Errorf("SubscriptionID() = %v, want 100", history.SubscriptionID())
	}

	if history.EventType() != EventTypePlanChanged {
		t.Errorf("EventType() = %v, want %v", history.EventType(), EventTypePlanChanged)
	}

	if history.OldPlanID() == nil || *history.OldPlanID() != oldPlanID {
		t.Errorf("OldPlanID() = %v, want %v", history.OldPlanID(), oldPlanID)
	}

	if history.NewPlanID() == nil || *history.NewPlanID() != newPlanID {
		t.Errorf("NewPlanID() = %v, want %v", history.NewPlanID(), newPlanID)
	}

	if history.Reason() == nil || *history.Reason() != reason {
		t.Errorf("Reason() = %v, want %v", history.Reason(), reason)
	}

	if !history.CreatedAt().Equal(now) {
		t.Errorf("CreatedAt() = %v, want %v", history.CreatedAt(), now)
	}
}
