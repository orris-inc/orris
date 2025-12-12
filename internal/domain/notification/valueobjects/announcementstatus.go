package valueobjects

import "fmt"

type AnnouncementStatus string

const (
	AnnouncementStatusDraft     AnnouncementStatus = "draft"
	AnnouncementStatusPublished AnnouncementStatus = "published"
	AnnouncementStatusExpired   AnnouncementStatus = "expired"
	AnnouncementStatusDeleted   AnnouncementStatus = "deleted"
)

var validAnnouncementStatuses = map[AnnouncementStatus]bool{
	AnnouncementStatusDraft:     true,
	AnnouncementStatusPublished: true,
	AnnouncementStatusExpired:   true,
	AnnouncementStatusDeleted:   true,
}

var announcementStatusTransitions = map[AnnouncementStatus][]AnnouncementStatus{
	AnnouncementStatusDraft: {
		AnnouncementStatusPublished,
		AnnouncementStatusDeleted,
	},
	AnnouncementStatusPublished: {
		AnnouncementStatusExpired,
		AnnouncementStatusDeleted,
	},
	AnnouncementStatusExpired: {
		AnnouncementStatusPublished,
		AnnouncementStatusDeleted,
	},
	AnnouncementStatusDeleted: {},
}

func (s AnnouncementStatus) String() string {
	return string(s)
}

func (s AnnouncementStatus) IsValid() bool {
	return validAnnouncementStatuses[s]
}

func (s AnnouncementStatus) IsDraft() bool {
	return s == AnnouncementStatusDraft
}

func (s AnnouncementStatus) IsPublished() bool {
	return s == AnnouncementStatusPublished
}

func (s AnnouncementStatus) IsExpired() bool {
	return s == AnnouncementStatusExpired
}

func (s AnnouncementStatus) IsDeleted() bool {
	return s == AnnouncementStatusDeleted
}

func (s AnnouncementStatus) CanTransitionTo(target AnnouncementStatus) bool {
	allowedTransitions, exists := announcementStatusTransitions[s]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == target {
			return true
		}
	}

	return false
}

func NewAnnouncementStatus(str string) (AnnouncementStatus, error) {
	s := AnnouncementStatus(str)
	if !s.IsValid() {
		return "", fmt.Errorf("invalid announcement status: %s", str)
	}
	return s, nil
}
