package value_objects

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewBillingCycle(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    BillingCycle
		wantErr bool
	}{
		{
			name:    "valid monthly",
			value:   "monthly",
			want:    BillingCycleMonthly,
			wantErr: false,
		},
		{
			name:    "valid quarterly",
			value:   "quarterly",
			want:    BillingCycleQuarterly,
			wantErr: false,
		},
		{
			name:    "valid yearly",
			value:   "yearly",
			want:    BillingCycleYearly,
			wantErr: false,
		},
		{
			name:    "valid lifetime",
			value:   "lifetime",
			want:    BillingCycleLifetime,
			wantErr: false,
		},
		{
			name:    "invalid cycle",
			value:   "weekly",
			wantErr: true,
		},
		{
			name:    "empty cycle",
			value:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewBillingCycle(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBillingCycle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && *got != tt.want {
				t.Errorf("NewBillingCycle() = %v, want %v", *got, tt.want)
			}
		})
	}
}

func TestParseBillingCycle(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    BillingCycle
		wantErr bool
	}{
		{
			name:    "parse monthly with spaces",
			value:   "  monthly  ",
			want:    BillingCycleMonthly,
			wantErr: false,
		},
		{
			name:    "parse uppercase",
			value:   "MONTHLY",
			want:    BillingCycleMonthly,
			wantErr: false,
		},
		{
			name:    "parse mixed case",
			value:   "MoNtHlY",
			want:    BillingCycleMonthly,
			wantErr: false,
		},
		{
			name:    "parse empty",
			value:   "",
			wantErr: true,
		},
		{
			name:    "parse invalid",
			value:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBillingCycle(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBillingCycle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseBillingCycle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBillingCycle_Days(t *testing.T) {
	tests := []struct {
		name  string
		cycle BillingCycle
		want  int
	}{
		{
			name:  "monthly days",
			cycle: BillingCycleMonthly,
			want:  30,
		},
		{
			name:  "quarterly days",
			cycle: BillingCycleQuarterly,
			want:  90,
		},
		{
			name:  "yearly days",
			cycle: BillingCycleYearly,
			want:  365,
		},
		{
			name:  "lifetime days",
			cycle: BillingCycleLifetime,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cycle.Days(); got != tt.want {
				t.Errorf("BillingCycle.Days() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBillingCycle_NextBillingDate(t *testing.T) {
	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		cycle BillingCycle
		from  time.Time
		want  time.Time
	}{
		{
			name:  "monthly next billing",
			cycle: BillingCycleMonthly,
			from:  baseDate,
			want:  time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "quarterly next billing",
			cycle: BillingCycleQuarterly,
			from:  baseDate,
			want:  time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "yearly next billing",
			cycle: BillingCycleYearly,
			from:  baseDate,
			want:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "lifetime next billing",
			cycle: BillingCycleLifetime,
			from:  baseDate,
			want:  time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cycle.NextBillingDate(tt.from)
			if !got.Equal(tt.want) {
				t.Errorf("BillingCycle.NextBillingDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBillingCycle_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		cycle BillingCycle
		want  bool
	}{
		{
			name:  "valid monthly",
			cycle: BillingCycleMonthly,
			want:  true,
		},
		{
			name:  "invalid cycle",
			cycle: BillingCycle("invalid"),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cycle.IsValid(); got != tt.want {
				t.Errorf("BillingCycle.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBillingCycle_IsLifetime(t *testing.T) {
	tests := []struct {
		name  string
		cycle BillingCycle
		want  bool
	}{
		{
			name:  "lifetime is lifetime",
			cycle: BillingCycleLifetime,
			want:  true,
		},
		{
			name:  "monthly is not lifetime",
			cycle: BillingCycleMonthly,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cycle.IsLifetime(); got != tt.want {
				t.Errorf("BillingCycle.IsLifetime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBillingCycle_Equals(t *testing.T) {
	tests := []struct {
		name  string
		cycle BillingCycle
		other BillingCycle
		want  bool
	}{
		{
			name:  "equal cycles",
			cycle: BillingCycleMonthly,
			other: BillingCycleMonthly,
			want:  true,
		},
		{
			name:  "different cycles",
			cycle: BillingCycleMonthly,
			other: BillingCycleYearly,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cycle.Equals(tt.other); got != tt.want {
				t.Errorf("BillingCycle.Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBillingCycle_JSON(t *testing.T) {
	tests := []struct {
		name    string
		cycle   BillingCycle
		wantErr bool
	}{
		{
			name:    "marshal monthly",
			cycle:   BillingCycleMonthly,
			wantErr: false,
		},
		{
			name:    "marshal yearly",
			cycle:   BillingCycleYearly,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.cycle)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				var unmarshaled BillingCycle
				err = json.Unmarshal(data, &unmarshaled)
				if err != nil {
					t.Errorf("json.Unmarshal() error = %v", err)
					return
				}
				if unmarshaled != tt.cycle {
					t.Errorf("json round trip failed: got %v, want %v", unmarshaled, tt.cycle)
				}
			}
		})
	}
}
