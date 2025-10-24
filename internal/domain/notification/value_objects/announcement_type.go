package value_objects

import "fmt"

type AnnouncementType string

const (
	AnnouncementTypeSystem      AnnouncementType = "system"
	AnnouncementTypeMaintenance AnnouncementType = "maintenance"
	AnnouncementTypeEvent       AnnouncementType = "event"
)

var validAnnouncementTypes = map[AnnouncementType]bool{
	AnnouncementTypeSystem:      true,
	AnnouncementTypeMaintenance: true,
	AnnouncementTypeEvent:       true,
}

func (t AnnouncementType) String() string {
	return string(t)
}

func (t AnnouncementType) IsValid() bool {
	return validAnnouncementTypes[t]
}

func (t AnnouncementType) IsSystem() bool {
	return t == AnnouncementTypeSystem
}

func (t AnnouncementType) IsMaintenance() bool {
	return t == AnnouncementTypeMaintenance
}

func (t AnnouncementType) IsEvent() bool {
	return t == AnnouncementTypeEvent
}

func NewAnnouncementType(s string) (AnnouncementType, error) {
	t := AnnouncementType(s)
	if !t.IsValid() {
		return "", fmt.Errorf("invalid announcement type: %s", s)
	}
	return t, nil
}
