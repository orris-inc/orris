package adapters

import (
	"context"

	"orris/internal/application/notification/usecases"
	"orris/internal/domain/notification"
	vo "orris/internal/domain/notification/value_objects"
	"orris/internal/domain/user"
)

type announcementAdapter struct {
	*notification.Announcement
}

func (a *announcementAdapter) Type() string {
	return a.Announcement.Type().String()
}

func (a *announcementAdapter) Status() string {
	return a.Announcement.Status().String()
}

func (a *announcementAdapter) Archive() error {
	a.Announcement.MarkAsExpired()
	return nil
}

func (a *announcementAdapter) Publish() error {
	return a.Announcement.Publish(true)
}

type notificationAdapter struct {
	*notification.Notification
}

func (n *notificationAdapter) Type() string {
	return n.Notification.Type().String()
}

func (n *notificationAdapter) ReadStatus() string {
	return n.Notification.ReadStatus().String()
}

func (n *notificationAdapter) Archive() error {
	return n.Notification.Archive()
}

type templateAdapter struct {
	*notification.NotificationTemplate
}

func (t *templateAdapter) TemplateType() string {
	return t.NotificationTemplate.TemplateType().String()
}

type AnnouncementRepositoryAdapter struct {
	repo notification.AnnouncementRepository
}

func NewAnnouncementRepositoryAdapter(repo notification.AnnouncementRepository) usecases.AnnouncementRepository {
	return &AnnouncementRepositoryAdapter{repo: repo}
}

func (a *AnnouncementRepositoryAdapter) Create(ctx context.Context, announcement usecases.Announcement) error {
	adapter, ok := announcement.(*announcementAdapter)
	if !ok {
		return nil
	}
	return a.repo.Create(ctx, adapter.Announcement)
}

func (a *AnnouncementRepositoryAdapter) Update(ctx context.Context, announcement usecases.Announcement) error {
	adapter, ok := announcement.(*announcementAdapter)
	if !ok {
		return nil
	}
	return a.repo.Update(ctx, adapter.Announcement)
}

func (a *AnnouncementRepositoryAdapter) Delete(ctx context.Context, id uint) error {
	return a.repo.Delete(ctx, id)
}

func (a *AnnouncementRepositoryAdapter) FindByID(ctx context.Context, id uint) (usecases.Announcement, error) {
	ann, err := a.repo.GetByID(ctx, id)
	if err != nil || ann == nil {
		return nil, err
	}
	return &announcementAdapter{ann}, nil
}

func (a *AnnouncementRepositoryAdapter) FindAll(ctx context.Context, limit, offset int) ([]usecases.Announcement, int64, error) {
	announcements, total, err := a.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	result := make([]usecases.Announcement, len(announcements))
	for i, ann := range announcements {
		result[i] = &announcementAdapter{ann}
	}
	return result, total, nil
}

func (a *AnnouncementRepositoryAdapter) FindPublished(ctx context.Context, limit, offset int) ([]usecases.Announcement, int64, error) {
	allAnnouncements, _, err := a.repo.List(ctx, 10000, 0)
	if err != nil {
		return nil, 0, err
	}

	published := make([]*notification.Announcement, 0)
	for _, ann := range allAnnouncements {
		if ann.Status().String() == "published" {
			published = append(published, ann)
		}
	}

	start := offset
	end := offset + limit
	if start > len(published) {
		start = len(published)
	}
	if end > len(published) {
		end = len(published)
	}

	result := make([]usecases.Announcement, end-start)
	for i, ann := range published[start:end] {
		result[i] = &announcementAdapter{ann}
	}
	return result, int64(len(published)), nil
}

type NotificationRepositoryAdapter struct {
	repo notification.NotificationRepository
}

func NewNotificationRepositoryAdapter(repo notification.NotificationRepository) usecases.NotificationRepository {
	return &NotificationRepositoryAdapter{repo: repo}
}

func (a *NotificationRepositoryAdapter) Create(ctx context.Context, notif usecases.Notification) error {
	adapter, ok := notif.(*notificationAdapter)
	if !ok {
		return nil
	}
	return a.repo.Create(ctx, adapter.Notification)
}

func (a *NotificationRepositoryAdapter) BulkCreate(ctx context.Context, notifications []usecases.Notification) error {
	domainNotifs := make([]*notification.Notification, 0, len(notifications))
	for _, n := range notifications {
		adapter, ok := n.(*notificationAdapter)
		if !ok {
			continue
		}
		domainNotifs = append(domainNotifs, adapter.Notification)
	}
	return a.repo.BulkCreate(ctx, domainNotifs)
}

func (a *NotificationRepositoryAdapter) FindByID(ctx context.Context, id uint) (usecases.Notification, error) {
	notif, err := a.repo.GetByID(ctx, id)
	if err != nil || notif == nil {
		return nil, err
	}
	return &notificationAdapter{notif}, nil
}

func (a *NotificationRepositoryAdapter) FindByUserID(ctx context.Context, userID uint, limit, offset int) ([]usecases.Notification, int64, error) {
	notifications, total, err := a.repo.ListByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	result := make([]usecases.Notification, len(notifications))
	for i, n := range notifications {
		result[i] = &notificationAdapter{n}
	}
	return result, total, nil
}

func (a *NotificationRepositoryAdapter) FindUnreadByUserID(ctx context.Context, userID uint, limit, offset int) ([]usecases.Notification, int64, error) {
	allNotifications, _, err := a.repo.ListByUserID(ctx, userID, 10000, 0)
	if err != nil {
		return nil, 0, err
	}

	unread := make([]*notification.Notification, 0)
	for _, n := range allNotifications {
		if n.ReadStatus().String() == "unread" {
			unread = append(unread, n)
		}
	}

	start := offset
	end := offset + limit
	if start > len(unread) {
		start = len(unread)
	}
	if end > len(unread) {
		end = len(unread)
	}

	result := make([]usecases.Notification, end-start)
	for i, n := range unread[start:end] {
		result[i] = &notificationAdapter{n}
	}
	return result, int64(len(unread)), nil
}

func (a *NotificationRepositoryAdapter) CountUnreadByUserID(ctx context.Context, userID uint) (int64, error) {
	allNotifications, _, err := a.repo.ListByUserID(ctx, userID, 10000, 0)
	if err != nil {
		return 0, err
	}

	count := int64(0)
	for _, n := range allNotifications {
		if n.ReadStatus().String() == "unread" {
			count++
		}
	}
	return count, nil
}

func (a *NotificationRepositoryAdapter) Update(ctx context.Context, notif usecases.Notification) error {
	adapter, ok := notif.(*notificationAdapter)
	if !ok {
		return nil
	}
	return a.repo.Update(ctx, adapter.Notification)
}

func (a *NotificationRepositoryAdapter) MarkAllAsReadByUserID(ctx context.Context, userID uint) error {
	allNotifications, _, err := a.repo.ListByUserID(ctx, userID, 10000, 0)
	if err != nil {
		return err
	}

	for _, n := range allNotifications {
		if n.ReadStatus().String() == "unread" {
			if err := n.MarkAsRead(); err != nil {
				return err
			}
			if err := a.repo.Update(ctx, n); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *NotificationRepositoryAdapter) Delete(ctx context.Context, id uint) error {
	return a.repo.Delete(ctx, id)
}

type TemplateRepositoryAdapter struct {
	repo notification.NotificationTemplateRepository
}

func NewTemplateRepositoryAdapter(repo notification.NotificationTemplateRepository) usecases.NotificationTemplateRepository {
	return &TemplateRepositoryAdapter{repo: repo}
}

func (a *TemplateRepositoryAdapter) Create(ctx context.Context, template usecases.NotificationTemplate) error {
	adapter, ok := template.(*templateAdapter)
	if !ok {
		return nil
	}
	return a.repo.Create(ctx, adapter.NotificationTemplate)
}

func (a *TemplateRepositoryAdapter) Update(ctx context.Context, template usecases.NotificationTemplate) error {
	adapter, ok := template.(*templateAdapter)
	if !ok {
		return nil
	}
	return a.repo.Update(ctx, adapter.NotificationTemplate)
}

func (a *TemplateRepositoryAdapter) Delete(ctx context.Context, id uint) error {
	return a.repo.Delete(ctx, id)
}

func (a *TemplateRepositoryAdapter) FindByType(ctx context.Context, templateType string) (usecases.NotificationTemplate, error) {
	tmplType, err := vo.NewTemplateType(templateType)
	if err != nil {
		return nil, err
	}

	tmpl, err := a.repo.GetByTemplateType(ctx, tmplType)
	if err != nil || tmpl == nil {
		return nil, err
	}
	return &templateAdapter{tmpl}, nil
}

func (a *TemplateRepositoryAdapter) FindByID(ctx context.Context, id uint) (usecases.NotificationTemplate, error) {
	tmpl, err := a.repo.GetByID(ctx, id)
	if err != nil || tmpl == nil {
		return nil, err
	}
	return &templateAdapter{tmpl}, nil
}

func (a *TemplateRepositoryAdapter) FindAll(ctx context.Context) ([]usecases.NotificationTemplate, error) {
	templates, _, err := a.repo.List(ctx, 1000, 0)
	if err != nil {
		return nil, err
	}

	result := make([]usecases.NotificationTemplate, len(templates))
	for i, t := range templates {
		result[i] = &templateAdapter{t}
	}
	return result, nil
}

type UserRepositoryAdapter struct {
	repo user.RepositoryWithSpecifications
}

func NewUserRepositoryAdapter(repo user.RepositoryWithSpecifications) usecases.UserRepository {
	return &UserRepositoryAdapter{repo: repo}
}

func (a *UserRepositoryAdapter) FindAllActiveUserIDs(ctx context.Context) ([]uint, error) {
	users, err := a.repo.FindBySpecification(ctx, nil, 10000)
	if err != nil {
		return nil, err
	}

	userIDs := make([]uint, 0, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.ID())
	}
	return userIDs, nil
}

type AnnouncementFactoryAdapter struct{}

func NewAnnouncementFactoryAdapter() usecases.AnnouncementFactory {
	return &AnnouncementFactoryAdapter{}
}

func (f *AnnouncementFactoryAdapter) CreateAnnouncement(title, content, announcementType string, priority int) (usecases.Announcement, error) {
	annType, err := vo.NewAnnouncementType(announcementType)
	if err != nil {
		return nil, err
	}

	ann, err := notification.CreateAnnouncement(
		title,
		content,
		annType,
		0,
		priority,
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &announcementAdapter{ann}, nil
}

type NotificationFactoryAdapter struct{}

func NewNotificationFactoryAdapter() usecases.NotificationFactory {
	return &NotificationFactoryAdapter{}
}

func (f *NotificationFactoryAdapter) CreateNotification(userID uint, notificationType, title, content string, relatedID *uint) (usecases.Notification, error) {
	notifType, err := vo.NewNotificationType(notificationType)
	if err != nil {
		return nil, err
	}

	notif, err := notification.CreateNotification(
		userID,
		notifType,
		title,
		content,
		relatedID,
	)
	if err != nil {
		return nil, err
	}

	return &notificationAdapter{notif}, nil
}

type TemplateFactoryAdapter struct{}

func NewTemplateFactoryAdapter() usecases.TemplateFactory {
	return &TemplateFactoryAdapter{}
}

func (f *TemplateFactoryAdapter) CreateTemplate(templateType, name, title, content string, variables []string) (usecases.NotificationTemplate, error) {
	tmplType, err := vo.NewTemplateType(templateType)
	if err != nil {
		return nil, err
	}

	tmpl, err := notification.CreateNotificationTemplate(
		tmplType,
		name,
		title,
		content,
		variables,
	)
	if err != nil {
		return nil, err
	}

	return &templateAdapter{tmpl}, nil
}
