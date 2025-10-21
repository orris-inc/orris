package subscription

import (
	"testing"
	"time"

	vo "orris/internal/domain/subscription/value_objects"
)

func TestNewSubscriptionPlan(t *testing.T) {
	tests := []struct {
		name         string
		planName     string
		slug         string
		description  string
		price        uint64
		currency     string
		billingCycle vo.BillingCycle
		trialDays    int
		wantErr      bool
	}{
		{
			name:         "valid plan",
			planName:     "Basic Plan",
			slug:         "basic",
			description:  "Basic subscription",
			price:        9900,
			currency:     "USD",
			billingCycle: vo.BillingCycleMonthly,
			trialDays:    14,
			wantErr:      false,
		},
		{
			name:         "empty name",
			planName:     "",
			slug:         "basic",
			description:  "Basic subscription",
			price:        9900,
			currency:     "USD",
			billingCycle: vo.BillingCycleMonthly,
			trialDays:    14,
			wantErr:      true,
		},
		{
			name:         "empty slug",
			planName:     "Basic Plan",
			slug:         "",
			description:  "Basic subscription",
			price:        9900,
			currency:     "USD",
			billingCycle: vo.BillingCycleMonthly,
			trialDays:    14,
			wantErr:      true,
		},
		{
			name:         "invalid currency",
			planName:     "Basic Plan",
			slug:         "basic",
			description:  "Basic subscription",
			price:        9900,
			currency:     "INVALID",
			billingCycle: vo.BillingCycleMonthly,
			trialDays:    14,
			wantErr:      true,
		},
		{
			name:         "negative trial days",
			planName:     "Basic Plan",
			slug:         "basic",
			description:  "Basic subscription",
			price:        9900,
			currency:     "USD",
			billingCycle: vo.BillingCycleMonthly,
			trialDays:    -1,
			wantErr:      true,
		},
		{
			name:         "zero trial days",
			planName:     "Basic Plan",
			slug:         "basic",
			description:  "Basic subscription",
			price:        9900,
			currency:     "USD",
			billingCycle: vo.BillingCycleMonthly,
			trialDays:    0,
			wantErr:      false,
		},
		{
			name:         "name too long",
			planName:     "This is a very long plan name that exceeds the maximum allowed length of 100 characters for a plan name and should fail validation",
			slug:         "basic",
			description:  "Basic subscription",
			price:        9900,
			currency:     "USD",
			billingCycle: vo.BillingCycleMonthly,
			trialDays:    14,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := NewSubscriptionPlan(tt.planName, tt.slug, tt.description,
				tt.price, tt.currency, tt.billingCycle, tt.trialDays)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewSubscriptionPlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if plan == nil {
					t.Error("NewSubscriptionPlan() returned nil plan")
					return
				}
				if plan.Name() != tt.planName {
					t.Errorf("Name() = %v, want %v", plan.Name(), tt.planName)
				}
				if plan.Slug() != tt.slug {
					t.Errorf("Slug() = %v, want %v", plan.Slug(), tt.slug)
				}
				if plan.Price() != tt.price {
					t.Errorf("Price() = %v, want %v", plan.Price(), tt.price)
				}
				if plan.Currency() != tt.currency {
					t.Errorf("Currency() = %v, want %v", plan.Currency(), tt.currency)
				}
				if plan.Status() != PlanStatusActive {
					t.Errorf("Status() = %v, want %v", plan.Status(), PlanStatusActive)
				}
				if plan.APIRateLimit() != 60 {
					t.Errorf("APIRateLimit() = %v, want 60", plan.APIRateLimit())
				}
				if !plan.IsPublic() {
					t.Error("IsPublic() = false, want true")
				}
			}
		})
	}
}

func TestReconstructSubscriptionPlan(t *testing.T) {
	now := time.Now()
	features := vo.NewPlanFeatures([]string{"api_access"}, map[string]interface{}{"max_requests": 1000})

	tests := []struct {
		name    string
		id      uint
		status  string
		wantErr bool
	}{
		{
			name:    "valid reconstruction",
			id:      1,
			status:  "active",
			wantErr: false,
		},
		{
			name:    "zero id",
			id:      0,
			status:  "active",
			wantErr: true,
		},
		{
			name:    "invalid status",
			id:      1,
			status:  "pending",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := ReconstructSubscriptionPlan(
				tt.id, "Premium", "premium", "Premium plan",
				29900, "USD", vo.BillingCycleMonthly, 30, tt.status,
				features, "https://api.example.com", 120, 10, 5,
				1073741824, true, 1, map[string]interface{}{"category": "business"},
				now, now,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReconstructSubscriptionPlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if plan.ID() != tt.id {
					t.Errorf("ID() = %v, want %v", plan.ID(), tt.id)
				}
				if plan.Status() != PlanStatus(tt.status) {
					t.Errorf("Status() = %v, want %v", plan.Status(), tt.status)
				}
				if plan.Features() == nil {
					t.Error("Features() returned nil")
				}
			}
		})
	}
}

func TestSubscriptionPlan_SetID(t *testing.T) {
	plan, _ := NewSubscriptionPlan("Basic", "basic", "Basic plan",
		9900, "USD", vo.BillingCycleMonthly, 14)

	if err := plan.SetID(1); err != nil {
		t.Errorf("SetID() error = %v", err)
	}

	if plan.ID() != 1 {
		t.Errorf("ID() = %v, want 1", plan.ID())
	}

	if err := plan.SetID(2); err == nil {
		t.Error("SetID() should fail when ID is already set")
	}

	if err := plan.SetID(0); err == nil {
		t.Error("SetID() should fail with zero ID")
	}
}

func TestSubscriptionPlan_Activate(t *testing.T) {
	plan, _ := NewSubscriptionPlan("Basic", "basic", "Basic plan",
		9900, "USD", vo.BillingCycleMonthly, 14)

	plan.Deactivate()

	if err := plan.Activate(); err != nil {
		t.Errorf("Activate() error = %v", err)
	}

	if !plan.IsActive() {
		t.Error("IsActive() = false, want true")
	}

	if plan.Status() != PlanStatusActive {
		t.Errorf("Status() = %v, want %v", plan.Status(), PlanStatusActive)
	}
}

func TestSubscriptionPlan_Deactivate(t *testing.T) {
	plan, _ := NewSubscriptionPlan("Basic", "basic", "Basic plan",
		9900, "USD", vo.BillingCycleMonthly, 14)

	if err := plan.Deactivate(); err != nil {
		t.Errorf("Deactivate() error = %v", err)
	}

	if plan.IsActive() {
		t.Error("IsActive() = true, want false")
	}

	if plan.Status() != PlanStatusInactive {
		t.Errorf("Status() = %v, want %v", plan.Status(), PlanStatusInactive)
	}
}

func TestSubscriptionPlan_UpdatePrice(t *testing.T) {
	plan, _ := NewSubscriptionPlan("Basic", "basic", "Basic plan",
		9900, "USD", vo.BillingCycleMonthly, 14)

	tests := []struct {
		name     string
		price    uint64
		currency string
		wantErr  bool
	}{
		{
			name:     "valid price update",
			price:    14900,
			currency: "USD",
			wantErr:  false,
		},
		{
			name:     "change currency",
			price:    99900,
			currency: "CNY",
			wantErr:  false,
		},
		{
			name:     "invalid currency",
			price:    9900,
			currency: "XXX",
			wantErr:  true,
		},
		{
			name:     "zero price",
			price:    0,
			currency: "USD",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plan.UpdatePrice(tt.price, tt.currency)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdatePrice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if plan.Price() != tt.price {
					t.Errorf("Price() = %v, want %v", plan.Price(), tt.price)
				}
				if plan.Currency() != tt.currency {
					t.Errorf("Currency() = %v, want %v", plan.Currency(), tt.currency)
				}
			}
		})
	}
}

func TestSubscriptionPlan_UpdateDescription(t *testing.T) {
	plan, _ := NewSubscriptionPlan("Basic", "basic", "Basic plan",
		9900, "USD", vo.BillingCycleMonthly, 14)

	newDesc := "Updated description"
	plan.UpdateDescription(newDesc)

	if plan.Description() != newDesc {
		t.Errorf("Description() = %v, want %v", plan.Description(), newDesc)
	}
}

func TestSubscriptionPlan_UpdateFeatures(t *testing.T) {
	plan, _ := NewSubscriptionPlan("Basic", "basic", "Basic plan",
		9900, "USD", vo.BillingCycleMonthly, 14)

	features := vo.NewPlanFeatures(
		[]string{"api_access", "priority_support"},
		map[string]interface{}{"max_requests": 5000},
	)

	if err := plan.UpdateFeatures(features); err != nil {
		t.Errorf("UpdateFeatures() error = %v", err)
	}

	if plan.Features() == nil {
		t.Error("Features() returned nil")
	}

	if !plan.HasFeature("api_access") {
		t.Error("HasFeature(api_access) = false, want true")
	}

	if err := plan.UpdateFeatures(nil); err == nil {
		t.Error("UpdateFeatures(nil) should fail")
	}
}

func TestSubscriptionPlan_SetCustomEndpoint(t *testing.T) {
	plan, _ := NewSubscriptionPlan("Basic", "basic", "Basic plan",
		9900, "USD", vo.BillingCycleMonthly, 14)

	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{
			name:     "valid endpoint",
			endpoint: "https://api.example.com/v1",
			wantErr:  false,
		},
		{
			name:     "empty endpoint",
			endpoint: "",
			wantErr:  false,
		},
		{
			name:     "endpoint too long",
			endpoint: string(make([]byte, 501)),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plan.SetCustomEndpoint(tt.endpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetCustomEndpoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && plan.CustomEndpoint() != tt.endpoint {
				t.Errorf("CustomEndpoint() = %v, want %v", plan.CustomEndpoint(), tt.endpoint)
			}
		})
	}
}

func TestSubscriptionPlan_SetAPIRateLimit(t *testing.T) {
	plan, _ := NewSubscriptionPlan("Basic", "basic", "Basic plan",
		9900, "USD", vo.BillingCycleMonthly, 14)

	tests := []struct {
		name    string
		limit   uint
		wantErr bool
	}{
		{
			name:    "valid rate limit",
			limit:   120,
			wantErr: false,
		},
		{
			name:    "zero rate limit",
			limit:   0,
			wantErr: true,
		},
		{
			name:    "high rate limit",
			limit:   10000,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plan.SetAPIRateLimit(tt.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetAPIRateLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && plan.APIRateLimit() != tt.limit {
				t.Errorf("APIRateLimit() = %v, want %v", plan.APIRateLimit(), tt.limit)
			}
		})
	}
}

func TestSubscriptionPlan_SetLimits(t *testing.T) {
	plan, _ := NewSubscriptionPlan("Basic", "basic", "Basic plan",
		9900, "USD", vo.BillingCycleMonthly, 14)

	plan.SetMaxUsers(50)
	if plan.MaxUsers() != 50 {
		t.Errorf("MaxUsers() = %v, want 50", plan.MaxUsers())
	}

	plan.SetMaxProjects(100)
	if plan.MaxProjects() != 100 {
		t.Errorf("MaxProjects() = %v, want 100", plan.MaxProjects())
	}

	var storageLimit uint64 = 10737418240
	plan.SetStorageLimit(storageLimit)
	if plan.StorageLimit() != storageLimit {
		t.Errorf("StorageLimit() = %v, want %v", plan.StorageLimit(), storageLimit)
	}
}

func TestSubscriptionPlan_SetSortOrder(t *testing.T) {
	plan, _ := NewSubscriptionPlan("Basic", "basic", "Basic plan",
		9900, "USD", vo.BillingCycleMonthly, 14)

	plan.SetSortOrder(5)
	if plan.SortOrder() != 5 {
		t.Errorf("SortOrder() = %v, want 5", plan.SortOrder())
	}

	plan.SetSortOrder(-1)
	if plan.SortOrder() != -1 {
		t.Errorf("SortOrder() = %v, want -1", plan.SortOrder())
	}
}

func TestSubscriptionPlan_SetPublic(t *testing.T) {
	plan, _ := NewSubscriptionPlan("Basic", "basic", "Basic plan",
		9900, "USD", vo.BillingCycleMonthly, 14)

	if !plan.IsPublic() {
		t.Error("IsPublic() = false, want true (default)")
	}

	plan.SetPublic(false)
	if plan.IsPublic() {
		t.Error("IsPublic() = true, want false")
	}

	plan.SetPublic(true)
	if !plan.IsPublic() {
		t.Error("IsPublic() = false, want true")
	}
}

func TestSubscriptionPlan_HasFeature(t *testing.T) {
	plan, _ := NewSubscriptionPlan("Basic", "basic", "Basic plan",
		9900, "USD", vo.BillingCycleMonthly, 14)

	if plan.HasFeature("any_feature") {
		t.Error("HasFeature() should return false when features is nil")
	}

	features := vo.NewPlanFeatures(
		[]string{"api_access", "priority_support"},
		map[string]interface{}{},
	)
	plan.UpdateFeatures(features)

	if !plan.HasFeature("api_access") {
		t.Error("HasFeature(api_access) = false, want true")
	}

	if plan.HasFeature("non_existent") {
		t.Error("HasFeature(non_existent) = true, want false")
	}
}

func TestSubscriptionPlan_GetLimit(t *testing.T) {
	plan, _ := NewSubscriptionPlan("Basic", "basic", "Basic plan",
		9900, "USD", vo.BillingCycleMonthly, 14)

	if _, exists := plan.GetLimit("max_requests"); exists {
		t.Error("GetLimit() should return false when features is nil")
	}

	features := vo.NewPlanFeatures(
		[]string{},
		map[string]interface{}{"max_requests": 1000},
	)
	plan.UpdateFeatures(features)

	value, exists := plan.GetLimit("max_requests")
	if !exists {
		t.Error("GetLimit(max_requests) should return true")
	}
	if value != 1000 {
		t.Errorf("GetLimit(max_requests) = %v, want 1000", value)
	}

	_, exists = plan.GetLimit("non_existent")
	if exists {
		t.Error("GetLimit(non_existent) should return false")
	}
}

func TestSubscriptionPlan_Getters(t *testing.T) {
	now := time.Now()
	metadata := map[string]interface{}{
		"category": "business",
		"tier":     2,
	}

	plan, err := ReconstructSubscriptionPlan(
		123, "Enterprise", "enterprise", "Enterprise plan",
		99900, "USD", vo.BillingCycleYearly, 30, "active",
		nil, "https://api.enterprise.com", 500, 100, 50,
		107374182400, false, 10, metadata,
		now, now,
	)

	if err != nil {
		t.Fatalf("ReconstructSubscriptionPlan() error = %v", err)
	}

	if plan.ID() != 123 {
		t.Errorf("ID() = %v, want 123", plan.ID())
	}
	if plan.Name() != "Enterprise" {
		t.Errorf("Name() = %v, want Enterprise", plan.Name())
	}
	if plan.Slug() != "enterprise" {
		t.Errorf("Slug() = %v, want enterprise", plan.Slug())
	}
	if plan.Description() != "Enterprise plan" {
		t.Errorf("Description() = %v, want Enterprise plan", plan.Description())
	}
	if plan.Price() != 99900 {
		t.Errorf("Price() = %v, want 99900", plan.Price())
	}
	if plan.Currency() != "USD" {
		t.Errorf("Currency() = %v, want USD", plan.Currency())
	}
	if plan.BillingCycle() != vo.BillingCycleYearly {
		t.Errorf("BillingCycle() = %v, want %v", plan.BillingCycle(), vo.BillingCycleYearly)
	}
	if plan.TrialDays() != 30 {
		t.Errorf("TrialDays() = %v, want 30", plan.TrialDays())
	}
	if plan.Status() != PlanStatusActive {
		t.Errorf("Status() = %v, want %v", plan.Status(), PlanStatusActive)
	}
	if plan.CustomEndpoint() != "https://api.enterprise.com" {
		t.Errorf("CustomEndpoint() = %v, want https://api.enterprise.com", plan.CustomEndpoint())
	}
	if plan.APIRateLimit() != 500 {
		t.Errorf("APIRateLimit() = %v, want 500", plan.APIRateLimit())
	}
	if plan.MaxUsers() != 100 {
		t.Errorf("MaxUsers() = %v, want 100", plan.MaxUsers())
	}
	if plan.MaxProjects() != 50 {
		t.Errorf("MaxProjects() = %v, want 50", plan.MaxProjects())
	}
	if plan.StorageLimit() != 107374182400 {
		t.Errorf("StorageLimit() = %v, want 107374182400", plan.StorageLimit())
	}
	if plan.IsPublic() {
		t.Error("IsPublic() = true, want false")
	}
	if plan.SortOrder() != 10 {
		t.Errorf("SortOrder() = %v, want 10", plan.SortOrder())
	}

	meta := plan.Metadata()
	if meta["category"] != "business" {
		t.Errorf("Metadata()[category] = %v, want business", meta["category"])
	}

	if !plan.CreatedAt().Equal(now) {
		t.Errorf("CreatedAt() = %v, want %v", plan.CreatedAt(), now)
	}
	if !plan.UpdatedAt().Equal(now) {
		t.Errorf("UpdatedAt() = %v, want %v", plan.UpdatedAt(), now)
	}
}
