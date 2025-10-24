package value_objects

import (
	"fmt"
	"strings"
	"time"
)

type BillingCycle string

const (
	BillingCycleMonthly   BillingCycle = "monthly"
	BillingCycleQuarterly BillingCycle = "quarterly"
	BillingCycleYearly    BillingCycle = "yearly"
	BillingCycleLifetime  BillingCycle = "lifetime"
)

var ValidBillingCycles = map[BillingCycle]bool{
	BillingCycleMonthly:   true,
	BillingCycleQuarterly: true,
	BillingCycleYearly:    true,
	BillingCycleLifetime:  true,
}

var BillingCycleDays = map[BillingCycle]int{
	BillingCycleMonthly:   30,
	BillingCycleQuarterly: 90,
	BillingCycleYearly:    365,
	BillingCycleLifetime:  0,
}

func NewBillingCycle(value string) (*BillingCycle, error) {
	cycle := BillingCycle(value)

	if value == "" {
		return nil, fmt.Errorf("billing cycle cannot be empty")
	}

	if !ValidBillingCycles[cycle] {
		return nil, fmt.Errorf("invalid billing cycle: %s", value)
	}

	return &cycle, nil
}

func ParseBillingCycle(value string) (BillingCycle, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	cycle := BillingCycle(normalized)

	if normalized == "" {
		return "", fmt.Errorf("billing cycle cannot be empty")
	}

	if !ValidBillingCycles[cycle] {
		return "", fmt.Errorf("invalid billing cycle: %s", value)
	}

	return cycle, nil
}

func (b BillingCycle) String() string {
	return string(b)
}

func (b BillingCycle) IsValid() bool {
	return ValidBillingCycles[b]
}

func (b BillingCycle) Days() int {
	days, exists := BillingCycleDays[b]
	if !exists {
		return 0
	}
	return days
}

func (b BillingCycle) NextBillingDate(from time.Time) time.Time {
	switch b {
	case BillingCycleMonthly:
		return from.AddDate(0, 1, 0)
	case BillingCycleQuarterly:
		return from.AddDate(0, 3, 0)
	case BillingCycleYearly:
		return from.AddDate(1, 0, 0)
	case BillingCycleLifetime:
		return time.Time{}
	default:
		return time.Time{}
	}
}

func (b BillingCycle) IsLifetime() bool {
	return b == BillingCycleLifetime
}

func (b BillingCycle) Equals(other BillingCycle) bool {
	return b == other
}

func (b BillingCycle) MarshalJSON() ([]byte, error) {
	return []byte(`"` + b.String() + `"`), nil
}

func (b *BillingCycle) UnmarshalJSON(data []byte) error {
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	cycle, err := NewBillingCycle(str)
	if err != nil {
		return err
	}

	*b = *cycle
	return nil
}
