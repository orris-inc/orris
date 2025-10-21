package subscription

import (
	"testing"
	"time"

	vo "orris/internal/domain/subscription/value_objects"
)

func TestNewSubscription(t *testing.T) {
	tests := []struct {
		name      string
		userID    uint
		planID    uint
		startDate time.Time
		endDate   time.Time
		autoRenew bool
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid subscription",
			userID:    1,
			planID:    1,
			startDate: time.Now(),
			endDate:   time.Now().AddDate(0, 1, 0),
			autoRenew: true,
			wantErr:   false,
		},
		{
			name:      "missing user ID",
			userID:    0,
			planID:    1,
			startDate: time.Now(),
			endDate:   time.Now().AddDate(0, 1, 0),
			autoRenew: false,
			wantErr:   true,
			errMsg:    "user ID is required",
		},
		{
			name:      "missing plan ID",
			userID:    1,
			planID:    0,
			startDate: time.Now(),
			endDate:   time.Now().AddDate(0, 1, 0),
			autoRenew: false,
			wantErr:   true,
			errMsg:    "plan ID is required",
		},
		{
			name:      "end date before start date",
			userID:    1,
			planID:    1,
			startDate: time.Now(),
			endDate:   time.Now().AddDate(0, -1, 0),
			autoRenew: false,
			wantErr:   true,
			errMsg:    "end date must be after start date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, err := NewSubscription(tt.userID, tt.planID, tt.startDate, tt.endDate, tt.autoRenew)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if sub.UserID() != tt.userID {
				t.Errorf("expected userID %d, got %d", tt.userID, sub.UserID())
			}
			if sub.PlanID() != tt.planID {
				t.Errorf("expected planID %d, got %d", tt.planID, sub.PlanID())
			}
			if sub.Status() != vo.StatusInactive {
				t.Errorf("expected status %s, got %s", vo.StatusInactive, sub.Status())
			}
			if sub.AutoRenew() != tt.autoRenew {
				t.Errorf("expected autoRenew %v, got %v", tt.autoRenew, sub.AutoRenew())
			}
			if sub.Version() != 1 {
				t.Errorf("expected version 1, got %d", sub.Version())
			}

			events := sub.GetEvents()
			if len(events) != 1 {
				t.Errorf("expected 1 event, got %d", len(events))
			}
		})
	}
}

func TestReconstructSubscription(t *testing.T) {
	now := time.Now()
	startDate := now
	endDate := now.AddDate(0, 1, 0)

	tests := []struct {
		name    string
		id      uint
		userID  uint
		planID  uint
		status  vo.SubscriptionStatus
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid reconstruction",
			id:      1,
			userID:  1,
			planID:  1,
			status:  vo.StatusActive,
			wantErr: false,
		},
		{
			name:    "zero id",
			id:      0,
			userID:  1,
			planID:  1,
			status:  vo.StatusActive,
			wantErr: true,
			errMsg:  "subscription ID cannot be zero",
		},
		{
			name:    "zero user ID",
			id:      1,
			userID:  0,
			planID:  1,
			status:  vo.StatusActive,
			wantErr: true,
			errMsg:  "user ID is required",
		},
		{
			name:    "invalid status",
			id:      1,
			userID:  1,
			planID:  1,
			status:  "invalid",
			wantErr: true,
			errMsg:  "invalid subscription status: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, err := ReconstructSubscription(
				tt.id,
				tt.userID,
				tt.planID,
				tt.status,
				startDate,
				endDate,
				true,
				startDate,
				endDate,
				nil,
				nil,
				nil,
				1,
				now,
				now,
			)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if sub.ID() != tt.id {
				t.Errorf("expected id %d, got %d", tt.id, sub.ID())
			}
			if sub.Status() != tt.status {
				t.Errorf("expected status %s, got %s", tt.status, sub.Status())
			}
		})
	}
}

func TestActivate(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus vo.SubscriptionStatus
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "activate from inactive",
			initialStatus: vo.StatusInactive,
			wantErr:       false,
		},
		{
			name:          "activate from trialing",
			initialStatus: vo.StatusTrialing,
			wantErr:       false,
		},
		{
			name:          "already active",
			initialStatus: vo.StatusActive,
			wantErr:       false,
		},
		{
			name:          "cannot activate from cancelled",
			initialStatus: vo.StatusCancelled,
			wantErr:       true,
			errMsg:        "cannot activate subscription with status cancelled",
		},
		{
			name:          "cannot activate from past_due",
			initialStatus: vo.StatusPastDue,
			wantErr:       true,
			errMsg:        "cannot activate subscription with status past_due",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			sub, _ := ReconstructSubscription(
				1, 1, 1,
				tt.initialStatus,
				now, now.AddDate(0, 1, 0),
				true,
				now, now.AddDate(0, 1, 0),
				nil, nil, nil,
				1, now, now,
			)

			initialVersion := sub.Version()
			err := sub.Activate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if sub.Status() != vo.StatusActive {
				t.Errorf("expected status active, got %s", sub.Status())
			}

			if tt.initialStatus != vo.StatusActive && sub.Version() != initialVersion+1 {
				t.Errorf("expected version to increment")
			}

			if tt.initialStatus != vo.StatusActive {
				events := sub.GetEvents()
				if len(events) != 1 {
					t.Errorf("expected 1 event, got %d", len(events))
				}
			}
		})
	}
}

func TestCancel(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus vo.SubscriptionStatus
		reason        string
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "cancel active subscription",
			initialStatus: vo.StatusActive,
			reason:        "user requested",
			wantErr:       false,
		},
		{
			name:          "cancel trialing subscription",
			initialStatus: vo.StatusTrialing,
			reason:        "user requested",
			wantErr:       false,
		},
		{
			name:          "already cancelled",
			initialStatus: vo.StatusCancelled,
			reason:        "user requested",
			wantErr:       false,
		},
		{
			name:          "missing cancel reason",
			initialStatus: vo.StatusActive,
			reason:        "",
			wantErr:       true,
			errMsg:        "cancel reason is required",
		},
		{
			name:          "cannot cancel inactive",
			initialStatus: vo.StatusInactive,
			reason:        "user requested",
			wantErr:       true,
			errMsg:        "cannot cancel subscription with status inactive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			sub, _ := ReconstructSubscription(
				1, 1, 1,
				tt.initialStatus,
				now, now.AddDate(0, 1, 0),
				true,
				now, now.AddDate(0, 1, 0),
				nil, nil, nil,
				1, now, now,
			)

			initialVersion := sub.Version()
			err := sub.Cancel(tt.reason)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if sub.Status() != vo.StatusCancelled {
				t.Errorf("expected status cancelled, got %s", sub.Status())
			}

			if tt.initialStatus != vo.StatusCancelled {
				if sub.CancelledAt() == nil {
					t.Errorf("expected cancelledAt to be set")
				}
				if sub.CancelReason() == nil || *sub.CancelReason() != tt.reason {
					t.Errorf("expected cancel reason %q", tt.reason)
				}
				if sub.Version() != initialVersion+1 {
					t.Errorf("expected version to increment")
				}

				events := sub.GetEvents()
				if len(events) != 1 {
					t.Errorf("expected 1 event, got %d", len(events))
				}
			}
		})
	}
}

func TestRenew(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus vo.SubscriptionStatus
		endDate       time.Time
		newEndDate    time.Time
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "renew active subscription",
			initialStatus: vo.StatusActive,
			endDate:       time.Now().AddDate(0, 1, 0),
			newEndDate:    time.Now().AddDate(0, 2, 0),
			wantErr:       false,
		},
		{
			name:          "renew expired subscription",
			initialStatus: vo.StatusExpired,
			endDate:       time.Now().AddDate(0, -1, 0),
			newEndDate:    time.Now().AddDate(0, 1, 0),
			wantErr:       false,
		},
		{
			name:          "new end date before current",
			initialStatus: vo.StatusActive,
			endDate:       time.Now().AddDate(0, 1, 0),
			newEndDate:    time.Now(),
			wantErr:       true,
			errMsg:        "new end date must be after current end date",
		},
		{
			name:          "cannot renew cancelled",
			initialStatus: vo.StatusCancelled,
			endDate:       time.Now().AddDate(0, 1, 0),
			newEndDate:    time.Now().AddDate(0, 2, 0),
			wantErr:       true,
			errMsg:        "cannot renew subscription with status cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			sub, _ := ReconstructSubscription(
				1, 1, 1,
				tt.initialStatus,
				now, tt.endDate,
				true,
				now, tt.endDate,
				nil, nil, nil,
				1, now, now,
			)

			initialVersion := sub.Version()
			err := sub.Renew(tt.newEndDate)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !sub.EndDate().Equal(tt.newEndDate) {
				t.Errorf("expected end date to be updated")
			}
			if sub.Version() != initialVersion+1 {
				t.Errorf("expected version to increment")
			}

			if tt.initialStatus == vo.StatusExpired && sub.Status() != vo.StatusActive {
				t.Errorf("expected status to be active after renewing expired subscription")
			}

			events := sub.GetEvents()
			if len(events) != 1 {
				t.Errorf("expected 1 event, got %d", len(events))
			}
		})
	}
}

func TestUpgradePlan(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus vo.SubscriptionStatus
		currentPlanID uint
		newPlanID     uint
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "upgrade active subscription",
			initialStatus: vo.StatusActive,
			currentPlanID: 1,
			newPlanID:     2,
			wantErr:       false,
		},
		{
			name:          "same plan ID",
			initialStatus: vo.StatusActive,
			currentPlanID: 1,
			newPlanID:     1,
			wantErr:       false,
		},
		{
			name:          "zero new plan ID",
			initialStatus: vo.StatusActive,
			currentPlanID: 1,
			newPlanID:     0,
			wantErr:       true,
			errMsg:        "new plan ID is required",
		},
		{
			name:          "cannot upgrade cancelled",
			initialStatus: vo.StatusCancelled,
			currentPlanID: 1,
			newPlanID:     2,
			wantErr:       true,
			errMsg:        "cannot upgrade plan for subscription with status cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			sub, _ := ReconstructSubscription(
				1, 1, tt.currentPlanID,
				tt.initialStatus,
				now, now.AddDate(0, 1, 0),
				true,
				now, now.AddDate(0, 1, 0),
				nil, nil, nil,
				1, now, now,
			)

			initialVersion := sub.Version()
			err := sub.UpgradePlan(tt.newPlanID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.newPlanID != tt.currentPlanID {
				if sub.PlanID() != tt.newPlanID {
					t.Errorf("expected plan ID %d, got %d", tt.newPlanID, sub.PlanID())
				}
				if sub.Version() != initialVersion+1 {
					t.Errorf("expected version to increment")
				}

				events := sub.GetEvents()
				if len(events) != 1 {
					t.Errorf("expected 1 event, got %d", len(events))
				}
			}
		})
	}
}

func TestDowngradePlan(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus vo.SubscriptionStatus
		currentPlanID uint
		newPlanID     uint
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "downgrade active subscription",
			initialStatus: vo.StatusActive,
			currentPlanID: 2,
			newPlanID:     1,
			wantErr:       false,
		},
		{
			name:          "same plan ID",
			initialStatus: vo.StatusActive,
			currentPlanID: 1,
			newPlanID:     1,
			wantErr:       false,
		},
		{
			name:          "cannot downgrade inactive",
			initialStatus: vo.StatusInactive,
			currentPlanID: 2,
			newPlanID:     1,
			wantErr:       true,
			errMsg:        "cannot downgrade plan for subscription with status inactive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			sub, _ := ReconstructSubscription(
				1, 1, tt.currentPlanID,
				tt.initialStatus,
				now, now.AddDate(0, 1, 0),
				true,
				now, now.AddDate(0, 1, 0),
				nil, nil, nil,
				1, now, now,
			)

			err := sub.DowngradePlan(tt.newPlanID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.newPlanID != tt.currentPlanID && sub.PlanID() != tt.newPlanID {
				t.Errorf("expected plan ID %d, got %d", tt.newPlanID, sub.PlanID())
			}
		})
	}
}

func TestIsExpired(t *testing.T) {
	tests := []struct {
		name        string
		endDate     time.Time
		wantExpired bool
	}{
		{
			name:        "not expired",
			endDate:     time.Now().AddDate(0, 1, 0),
			wantExpired: false,
		},
		{
			name:        "expired",
			endDate:     time.Now().AddDate(0, -1, 0),
			wantExpired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			sub, _ := ReconstructSubscription(
				1, 1, 1,
				vo.StatusActive,
				now, tt.endDate,
				true,
				now, tt.endDate,
				nil, nil, nil,
				1, now, now,
			)

			if sub.IsExpired() != tt.wantExpired {
				t.Errorf("expected IsExpired() = %v, got %v", tt.wantExpired, sub.IsExpired())
			}
		})
	}
}

func TestIsActive(t *testing.T) {
	tests := []struct {
		name       string
		status     vo.SubscriptionStatus
		endDate    time.Time
		wantActive bool
	}{
		{
			name:       "active and not expired",
			status:     vo.StatusActive,
			endDate:    time.Now().AddDate(0, 1, 0),
			wantActive: true,
		},
		{
			name:       "active but expired",
			status:     vo.StatusActive,
			endDate:    time.Now().AddDate(0, -1, 0),
			wantActive: false,
		},
		{
			name:       "cancelled",
			status:     vo.StatusCancelled,
			endDate:    time.Now().AddDate(0, 1, 0),
			wantActive: false,
		},
		{
			name:       "trialing and not expired",
			status:     vo.StatusTrialing,
			endDate:    time.Now().AddDate(0, 1, 0),
			wantActive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			sub, _ := ReconstructSubscription(
				1, 1, 1,
				tt.status,
				now, tt.endDate,
				true,
				now, tt.endDate,
				nil, nil, nil,
				1, now, now,
			)

			if sub.IsActive() != tt.wantActive {
				t.Errorf("expected IsActive() = %v, got %v", tt.wantActive, sub.IsActive())
			}
		})
	}
}

func TestMarkAsExpired(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus vo.SubscriptionStatus
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "mark active as expired",
			initialStatus: vo.StatusActive,
			wantErr:       false,
		},
		{
			name:          "already expired",
			initialStatus: vo.StatusExpired,
			wantErr:       false,
		},
		{
			name:          "cannot expire cancelled",
			initialStatus: vo.StatusCancelled,
			wantErr:       true,
			errMsg:        "cannot mark subscription as expired with status cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			sub, _ := ReconstructSubscription(
				1, 1, 1,
				tt.initialStatus,
				now, now.AddDate(0, -1, 0),
				true,
				now, now.AddDate(0, -1, 0),
				nil, nil, nil,
				1, now, now,
			)

			initialVersion := sub.Version()
			err := sub.MarkAsExpired()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if sub.Status() != vo.StatusExpired {
				t.Errorf("expected status expired, got %s", sub.Status())
			}

			if tt.initialStatus != vo.StatusExpired && sub.Version() != initialVersion+1 {
				t.Errorf("expected version to increment")
			}

			if tt.initialStatus != vo.StatusExpired {
				events := sub.GetEvents()
				if len(events) != 1 {
					t.Errorf("expected 1 event, got %d", len(events))
				}
			}
		})
	}
}

func TestSetAutoRenew(t *testing.T) {
	now := time.Now()
	sub, _ := ReconstructSubscription(
		1, 1, 1,
		vo.StatusActive,
		now, now.AddDate(0, 1, 0),
		false,
		now, now.AddDate(0, 1, 0),
		nil, nil, nil,
		1, now, now,
	)

	initialVersion := sub.Version()
	sub.SetAutoRenew(true)

	if !sub.AutoRenew() {
		t.Errorf("expected auto-renew to be true")
	}
	if sub.Version() != initialVersion+1 {
		t.Errorf("expected version to increment")
	}

	sub.SetAutoRenew(true)
	if sub.Version() != initialVersion+1 {
		t.Errorf("expected version to not increment when setting same value")
	}
}

func TestUpdateCurrentPeriod(t *testing.T) {
	tests := []struct {
		name    string
		start   time.Time
		end     time.Time
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid period",
			start:   time.Now(),
			end:     time.Now().AddDate(0, 1, 0),
			wantErr: false,
		},
		{
			name:    "end before start",
			start:   time.Now(),
			end:     time.Now().AddDate(0, -1, 0),
			wantErr: true,
			errMsg:  "period end must be after period start",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			sub, _ := ReconstructSubscription(
				1, 1, 1,
				vo.StatusActive,
				now, now.AddDate(0, 1, 0),
				true,
				now, now.AddDate(0, 1, 0),
				nil, nil, nil,
				1, now, now,
			)

			initialVersion := sub.Version()
			err := sub.UpdateCurrentPeriod(tt.start, tt.end)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if sub.Version() != initialVersion+1 {
				t.Errorf("expected version to increment")
			}
		})
	}
}

func TestValidate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		setupFunc func() *Subscription
		wantErr   bool
		errMsg    string
	}{
		{
			name: "valid subscription",
			setupFunc: func() *Subscription {
				sub, _ := ReconstructSubscription(
					1, 1, 1,
					vo.StatusActive,
					now, now.AddDate(0, 1, 0),
					true,
					now, now.AddDate(0, 1, 0),
					nil, nil, nil,
					1, now, now,
				)
				return sub
			},
			wantErr: false,
		},
		{
			name: "invalid status",
			setupFunc: func() *Subscription {
				sub, _ := ReconstructSubscription(
					1, 1, 1,
					vo.StatusActive,
					now, now.AddDate(0, 1, 0),
					true,
					now, now.AddDate(0, 1, 0),
					nil, nil, nil,
					1, now, now,
				)
				sub.status = "invalid"
				return sub
			},
			wantErr: true,
			errMsg:  "invalid status: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := tt.setupFunc()
			err := sub.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestSetID(t *testing.T) {
	sub, _ := NewSubscription(1, 1, time.Now(), time.Now().AddDate(0, 1, 0), true)

	err := sub.SetID(100)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if sub.ID() != 100 {
		t.Errorf("expected ID 100, got %d", sub.ID())
	}

	err = sub.SetID(200)
	if err == nil {
		t.Errorf("expected error when setting ID twice")
	}

	err = sub.SetID(0)
	if err == nil {
		t.Errorf("expected error when setting ID to zero")
	}
}

func TestEventManagement(t *testing.T) {
	sub, _ := NewSubscription(1, 1, time.Now(), time.Now().AddDate(0, 1, 0), true)

	events := sub.GetEvents()
	if len(events) != 1 {
		t.Errorf("expected 1 event after creation, got %d", len(events))
	}

	events = sub.GetEvents()
	if len(events) != 0 {
		t.Errorf("expected events to be cleared after GetEvents(), got %d", len(events))
	}

	_ = sub.Activate()
	events = sub.GetEvents()
	if len(events) != 1 {
		t.Errorf("expected 1 event after activation, got %d", len(events))
	}

	sub.ClearEvents()
	events = sub.GetEvents()
	if len(events) != 0 {
		t.Errorf("expected events to be cleared, got %d", len(events))
	}
}
