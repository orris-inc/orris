package value_objects

import "fmt"

type TemplateType string

const (
	TemplateTypeSubscriptionExpiring TemplateType = "subscription_expiring"
	TemplateTypeSystemMaintenance    TemplateType = "system_maintenance"
	TemplateTypeWelcome              TemplateType = "welcome"
)

var validTemplateTypes = map[TemplateType]bool{
	TemplateTypeSubscriptionExpiring: true,
	TemplateTypeSystemMaintenance:    true,
	TemplateTypeWelcome:              true,
}

func (t TemplateType) String() string {
	return string(t)
}

func (t TemplateType) IsValid() bool {
	return validTemplateTypes[t]
}

func (t TemplateType) IsSubscriptionExpiring() bool {
	return t == TemplateTypeSubscriptionExpiring
}

func (t TemplateType) IsSystemMaintenance() bool {
	return t == TemplateTypeSystemMaintenance
}

func (t TemplateType) IsWelcome() bool {
	return t == TemplateTypeWelcome
}

func NewTemplateType(s string) (TemplateType, error) {
	t := TemplateType(s)
	if !t.IsValid() {
		return "", fmt.Errorf("invalid template type: %s", s)
	}
	return t, nil
}
