package value_objects

import (
	"encoding/json"
	"testing"
)

func TestNewPlanFeatures(t *testing.T) {
	tests := []struct {
		name     string
		features []string
		limits   map[string]interface{}
		wantLen  int
	}{
		{
			name:     "with features and limits",
			features: []string{"feature1", "feature2"},
			limits:   map[string]interface{}{"max_users": 10},
			wantLen:  2,
		},
		{
			name:     "nil features and limits",
			features: nil,
			limits:   nil,
			wantLen:  0,
		},
		{
			name:     "empty features and limits",
			features: []string{},
			limits:   map[string]interface{}{},
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf := NewPlanFeatures(tt.features, tt.limits)
			if pf == nil {
				t.Error("NewPlanFeatures() returned nil")
				return
			}
			if len(pf.Features) != tt.wantLen {
				t.Errorf("NewPlanFeatures() features length = %v, want %v", len(pf.Features), tt.wantLen)
			}
		})
	}
}

func TestPlanFeatures_HasFeature(t *testing.T) {
	pf := NewPlanFeatures([]string{"feature1", "feature2"}, nil)

	tests := []struct {
		name    string
		feature string
		want    bool
	}{
		{
			name:    "existing feature",
			feature: "feature1",
			want:    true,
		},
		{
			name:    "non-existing feature",
			feature: "feature3",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pf.HasFeature(tt.feature); got != tt.want {
				t.Errorf("PlanFeatures.HasFeature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlanFeatures_GetLimit(t *testing.T) {
	pf := NewPlanFeatures(nil, map[string]interface{}{
		"max_users":   10,
		"max_storage": 100.5,
		"enabled":     true,
	})

	tests := []struct {
		name       string
		key        string
		wantValue  interface{}
		wantExists bool
	}{
		{
			name:       "existing int limit",
			key:        "max_users",
			wantValue:  10,
			wantExists: true,
		},
		{
			name:       "existing float limit",
			key:        "max_storage",
			wantValue:  100.5,
			wantExists: true,
		},
		{
			name:       "existing bool limit",
			key:        "enabled",
			wantValue:  true,
			wantExists: true,
		},
		{
			name:       "non-existing limit",
			key:        "max_projects",
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotExists := pf.GetLimit(tt.key)
			if gotExists != tt.wantExists {
				t.Errorf("PlanFeatures.GetLimit() exists = %v, want %v", gotExists, tt.wantExists)
			}
			if tt.wantExists && gotValue != tt.wantValue {
				t.Errorf("PlanFeatures.GetLimit() value = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}

func TestPlanFeatures_IsWithinLimit(t *testing.T) {
	pf := NewPlanFeatures(nil, map[string]interface{}{
		"max_users":   10,
		"max_storage": 100.5,
		"plan_type":   "premium",
		"enabled":     true,
	})

	tests := []struct {
		name  string
		key   string
		value interface{}
		want  bool
	}{
		{
			name:  "within int limit",
			key:   "max_users",
			value: 5,
			want:  true,
		},
		{
			name:  "at int limit",
			key:   "max_users",
			value: 10,
			want:  true,
		},
		{
			name:  "exceed int limit",
			key:   "max_users",
			value: 15,
			want:  false,
		},
		{
			name:  "within float limit",
			key:   "max_storage",
			value: 50.0,
			want:  true,
		},
		{
			name:  "exceed float limit",
			key:   "max_storage",
			value: 150.0,
			want:  false,
		},
		{
			name:  "int value against float limit",
			key:   "max_storage",
			value: 50,
			want:  true,
		},
		{
			name:  "matching string limit",
			key:   "plan_type",
			value: "premium",
			want:  true,
		},
		{
			name:  "non-matching string limit",
			key:   "plan_type",
			value: "basic",
			want:  false,
		},
		{
			name:  "matching bool limit",
			key:   "enabled",
			value: true,
			want:  true,
		},
		{
			name:  "non-existing limit",
			key:   "non_existing",
			value: 100,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pf.IsWithinLimit(tt.key, tt.value); got != tt.want {
				t.Errorf("PlanFeatures.IsWithinLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlanFeatures_AddFeature(t *testing.T) {
	pf := NewPlanFeatures([]string{"feature1"}, nil)

	pf.AddFeature("feature2")
	if !pf.HasFeature("feature2") {
		t.Error("PlanFeatures.AddFeature() failed to add feature2")
	}

	pf.AddFeature("feature1")
	count := 0
	for _, f := range pf.Features {
		if f == "feature1" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("PlanFeatures.AddFeature() added duplicate feature, count = %v", count)
	}
}

func TestPlanFeatures_RemoveFeature(t *testing.T) {
	pf := NewPlanFeatures([]string{"feature1", "feature2", "feature3"}, nil)

	pf.RemoveFeature("feature2")
	if pf.HasFeature("feature2") {
		t.Error("PlanFeatures.RemoveFeature() failed to remove feature2")
	}

	if !pf.HasFeature("feature1") || !pf.HasFeature("feature3") {
		t.Error("PlanFeatures.RemoveFeature() removed wrong features")
	}

	pf.RemoveFeature("non_existing")
	if len(pf.Features) != 2 {
		t.Errorf("PlanFeatures.RemoveFeature() changed length unexpectedly, got %v", len(pf.Features))
	}
}

func TestPlanFeatures_SetLimit(t *testing.T) {
	pf := NewPlanFeatures(nil, nil)

	pf.SetLimit("max_users", 20)
	value, exists := pf.GetLimit("max_users")
	if !exists || value != 20 {
		t.Errorf("PlanFeatures.SetLimit() failed, got %v, exists %v", value, exists)
	}

	pf.SetLimit("max_users", 30)
	value, exists = pf.GetLimit("max_users")
	if !exists || value != 30 {
		t.Errorf("PlanFeatures.SetLimit() failed to update, got %v, exists %v", value, exists)
	}
}

func TestPlanFeatures_RemoveLimit(t *testing.T) {
	pf := NewPlanFeatures(nil, map[string]interface{}{
		"max_users":   10,
		"max_storage": 100,
	})

	pf.RemoveLimit("max_users")
	_, exists := pf.GetLimit("max_users")
	if exists {
		t.Error("PlanFeatures.RemoveLimit() failed to remove limit")
	}

	_, exists = pf.GetLimit("max_storage")
	if !exists {
		t.Error("PlanFeatures.RemoveLimit() removed wrong limit")
	}
}

func TestPlanFeatures_GetIntLimit(t *testing.T) {
	pf := NewPlanFeatures(nil, map[string]interface{}{
		"max_users":   10,
		"max_storage": 100.5,
		"plan_type":   "premium",
	})

	tests := []struct {
		name    string
		key     string
		want    int
		wantErr bool
	}{
		{
			name:    "get int limit",
			key:     "max_users",
			want:    10,
			wantErr: false,
		},
		{
			name:    "get float as int limit",
			key:     "max_storage",
			want:    100,
			wantErr: false,
		},
		{
			name:    "non-int limit",
			key:     "plan_type",
			wantErr: true,
		},
		{
			name:    "non-existing limit",
			key:     "non_existing",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pf.GetIntLimit(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("PlanFeatures.GetIntLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("PlanFeatures.GetIntLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlanFeatures_GetStringLimit(t *testing.T) {
	pf := NewPlanFeatures(nil, map[string]interface{}{
		"plan_type": "premium",
		"max_users": 10,
	})

	tests := []struct {
		name    string
		key     string
		want    string
		wantErr bool
	}{
		{
			name:    "get string limit",
			key:     "plan_type",
			want:    "premium",
			wantErr: false,
		},
		{
			name:    "non-string limit",
			key:     "max_users",
			wantErr: true,
		},
		{
			name:    "non-existing limit",
			key:     "non_existing",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pf.GetStringLimit(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("PlanFeatures.GetStringLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("PlanFeatures.GetStringLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlanFeatures_GetBoolLimit(t *testing.T) {
	pf := NewPlanFeatures(nil, map[string]interface{}{
		"enabled":   true,
		"max_users": 10,
	})

	tests := []struct {
		name    string
		key     string
		want    bool
		wantErr bool
	}{
		{
			name:    "get bool limit",
			key:     "enabled",
			want:    true,
			wantErr: false,
		},
		{
			name:    "non-bool limit",
			key:     "max_users",
			wantErr: true,
		},
		{
			name:    "non-existing limit",
			key:     "non_existing",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pf.GetBoolLimit(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("PlanFeatures.GetBoolLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("PlanFeatures.GetBoolLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlanFeatures_Clone(t *testing.T) {
	original := NewPlanFeatures(
		[]string{"feature1", "feature2"},
		map[string]interface{}{"max_users": 10},
	)

	cloned := original.Clone()

	if !original.Equals(cloned) {
		t.Error("PlanFeatures.Clone() created unequal clone")
	}

	cloned.AddFeature("feature3")
	if original.HasFeature("feature3") {
		t.Error("PlanFeatures.Clone() did not create deep copy of features")
	}

	cloned.SetLimit("max_storage", 100)
	_, exists := original.GetLimit("max_storage")
	if exists {
		t.Error("PlanFeatures.Clone() did not create deep copy of limits")
	}
}

func TestPlanFeatures_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		features []string
		limits   map[string]interface{}
		want     bool
	}{
		{
			name:     "empty plan features",
			features: []string{},
			limits:   map[string]interface{}{},
			want:     true,
		},
		{
			name:     "with features",
			features: []string{"feature1"},
			limits:   map[string]interface{}{},
			want:     false,
		},
		{
			name:     "with limits",
			features: []string{},
			limits:   map[string]interface{}{"max_users": 10},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf := NewPlanFeatures(tt.features, tt.limits)
			if got := pf.IsEmpty(); got != tt.want {
				t.Errorf("PlanFeatures.IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlanFeatures_Counts(t *testing.T) {
	pf := NewPlanFeatures(
		[]string{"feature1", "feature2", "feature3"},
		map[string]interface{}{"max_users": 10, "max_storage": 100},
	)

	if got := pf.FeatureCount(); got != 3 {
		t.Errorf("PlanFeatures.FeatureCount() = %v, want 3", got)
	}

	if got := pf.LimitCount(); got != 2 {
		t.Errorf("PlanFeatures.LimitCount() = %v, want 2", got)
	}
}

func TestPlanFeatures_Equals(t *testing.T) {
	pf1 := NewPlanFeatures(
		[]string{"feature1", "feature2"},
		map[string]interface{}{"max_users": 10},
	)

	pf2 := NewPlanFeatures(
		[]string{"feature2", "feature1"},
		map[string]interface{}{"max_users": 10},
	)

	pf3 := NewPlanFeatures(
		[]string{"feature1"},
		map[string]interface{}{"max_users": 10},
	)

	tests := []struct {
		name string
		pf1  *PlanFeatures
		pf2  *PlanFeatures
		want bool
	}{
		{
			name: "equal plan features",
			pf1:  pf1,
			pf2:  pf2,
			want: true,
		},
		{
			name: "different features",
			pf1:  pf1,
			pf2:  pf3,
			want: false,
		},
		{
			name: "nil comparison",
			pf1:  pf1,
			pf2:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pf1.Equals(tt.pf2); got != tt.want {
				t.Errorf("PlanFeatures.Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlanFeatures_JSON(t *testing.T) {
	pf := NewPlanFeatures(
		[]string{"feature1", "feature2"},
		map[string]interface{}{"max_users": float64(10), "enabled": true},
	)

	data, err := json.Marshal(pf)
	if err != nil {
		t.Errorf("json.Marshal() error = %v", err)
		return
	}

	var unmarshaled PlanFeatures
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
		return
	}

	if !pf.Equals(&unmarshaled) {
		t.Errorf("json round trip failed: plan features not equal\noriginal: %+v\nunmarshaled: %+v", pf, &unmarshaled)
	}
}
