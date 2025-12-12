package valueobjects

import "fmt"

type NotificationType string

const (
	NotificationTypeSystem       NotificationType = "system"
	NotificationTypeActivity     NotificationType = "activity"
	NotificationTypeSubscription NotificationType = "subscription"
	NotificationTypeTemplate     NotificationType = "template"
)

var validNotificationTypes = map[NotificationType]bool{
	NotificationTypeSystem:       true,
	NotificationTypeActivity:     true,
	NotificationTypeSubscription: true,
	NotificationTypeTemplate:     true,
}

func (t NotificationType) String() string {
	return string(t)
}

func (t NotificationType) IsValid() bool {
	return validNotificationTypes[t]
}

func (t NotificationType) IsSystem() bool {
	return t == NotificationTypeSystem
}

func (t NotificationType) IsActivity() bool {
	return t == NotificationTypeActivity
}

func (t NotificationType) IsSubscription() bool {
	return t == NotificationTypeSubscription
}

func (t NotificationType) IsTemplate() bool {
	return t == NotificationTypeTemplate
}

func NewNotificationType(s string) (NotificationType, error) {
	t := NotificationType(s)
	if !t.IsValid() {
		return "", fmt.Errorf("invalid notification type: %s", s)
	}
	return t, nil
}
