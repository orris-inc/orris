package value_objects

import "fmt"

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

var validPriorities = map[Priority]bool{
	PriorityLow:    true,
	PriorityMedium: true,
	PriorityHigh:   true,
	PriorityUrgent: true,
}

var prioritySLAHours = map[Priority]int{
	PriorityLow:    72,
	PriorityMedium: 24,
	PriorityHigh:   8,
	PriorityUrgent: 2,
}

func (p Priority) String() string {
	return string(p)
}

func (p Priority) IsValid() bool {
	return validPriorities[p]
}

func (p Priority) GetSLAHours() int {
	hours, ok := prioritySLAHours[p]
	if !ok {
		return 72
	}
	return hours
}

func NewPriority(s string) (Priority, error) {
	p := Priority(s)
	if !p.IsValid() {
		return "", fmt.Errorf("invalid priority: %s", s)
	}
	return p, nil
}

func (p Priority) IsLow() bool {
	return p == PriorityLow
}

func (p Priority) IsMedium() bool {
	return p == PriorityMedium
}

func (p Priority) IsHigh() bool {
	return p == PriorityHigh
}

func (p Priority) IsUrgent() bool {
	return p == PriorityUrgent
}
