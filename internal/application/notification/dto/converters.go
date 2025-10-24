package dto

import "time"

type MarkdownService interface {
	ToHTML(markdown string) (string, error)
}

type Announcement interface {
	ID() uint
	Title() string
	Content() string
	Type() string
	Status() string
	Priority() int
	ScheduledAt() *time.Time
	ExpiresAt() *time.Time
	ViewCount() int
	CreatedAt() time.Time
	UpdatedAt() time.Time
}

type Notification interface {
	ID() uint
	Type() string
	Title() string
	Content() string
	RelatedID() *uint
	ReadStatus() string
	CreatedAt() time.Time
}

type NotificationTemplate interface {
	ID() uint
	TemplateType() string
	Name() string
	Title() string
	Content() string
	Variables() []string
	CreatedAt() time.Time
	UpdatedAt() time.Time
}

func ToAnnouncementResponse(announcement Announcement, markdownSvc MarkdownService) (*AnnouncementResponse, error) {
	if announcement == nil {
		return nil, nil
	}

	contentHTML := ""
	if markdownSvc != nil {
		html, err := markdownSvc.ToHTML(announcement.Content())
		if err == nil {
			contentHTML = html
		}
	}

	return &AnnouncementResponse{
		ID:          announcement.ID(),
		Title:       announcement.Title(),
		Content:     announcement.Content(),
		ContentHTML: contentHTML,
		Type:        announcement.Type(),
		Status:      announcement.Status(),
		Priority:    announcement.Priority(),
		ScheduledAt: announcement.ScheduledAt(),
		ExpiresAt:   announcement.ExpiresAt(),
		ViewCount:   announcement.ViewCount(),
		CreatedAt:   announcement.CreatedAt(),
		UpdatedAt:   announcement.UpdatedAt(),
	}, nil
}

func ToAnnouncementResponseList[T Announcement](announcements []T, markdownSvc MarkdownService) ([]*AnnouncementResponse, error) {
	responses := make([]*AnnouncementResponse, 0, len(announcements))
	for _, announcement := range announcements {
		resp, err := ToAnnouncementResponse(announcement, markdownSvc)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}
	return responses, nil
}

func ToNotificationResponse(notification Notification, markdownSvc MarkdownService) (*NotificationResponse, error) {
	if notification == nil {
		return nil, nil
	}

	contentHTML := ""
	if markdownSvc != nil {
		html, err := markdownSvc.ToHTML(notification.Content())
		if err == nil {
			contentHTML = html
		}
	}

	return &NotificationResponse{
		ID:          notification.ID(),
		Type:        notification.Type(),
		Title:       notification.Title(),
		Content:     notification.Content(),
		ContentHTML: contentHTML,
		RelatedID:   notification.RelatedID(),
		ReadStatus:  notification.ReadStatus(),
		CreatedAt:   notification.CreatedAt(),
	}, nil
}

func ToNotificationResponseList[T Notification](notifications []T, markdownSvc MarkdownService) ([]*NotificationResponse, error) {
	responses := make([]*NotificationResponse, 0, len(notifications))
	for _, notification := range notifications {
		resp, err := ToNotificationResponse(notification, markdownSvc)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}
	return responses, nil
}

func ToTemplateResponse(template NotificationTemplate) *TemplateResponse {
	if template == nil {
		return nil
	}

	return &TemplateResponse{
		ID:           template.ID(),
		TemplateType: template.TemplateType(),
		Name:         template.Name(),
		Title:        template.Title(),
		Content:      template.Content(),
		Variables:    template.Variables(),
		CreatedAt:    template.CreatedAt(),
		UpdatedAt:    template.UpdatedAt(),
	}
}

func ToTemplateResponseList[T NotificationTemplate](templates []T) []*TemplateResponse {
	responses := make([]*TemplateResponse, 0, len(templates))
	for _, template := range templates {
		responses = append(responses, ToTemplateResponse(template))
	}
	return responses
}
