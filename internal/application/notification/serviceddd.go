package notification

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/application/notification/dto"
	"github.com/orris-inc/orris/internal/application/notification/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ServiceDDD struct {
	logger logger.Interface

	createAnnouncement         *usecases.CreateAnnouncementUseCase
	updateAnnouncement         *usecases.UpdateAnnouncementUseCase
	deleteAnnouncement         *usecases.DeleteAnnouncementUseCase
	publishAnnouncement        *usecases.PublishAnnouncementUseCase
	archiveAnnouncement        *usecases.ArchiveAnnouncementUseCase
	listAnnouncements          *usecases.ListAnnouncementsUseCase
	getAnnouncement            *usecases.GetAnnouncementUseCase
	getAnnouncementUnreadCount *usecases.GetAnnouncementUnreadCountUseCase
	markAnnouncementAsRead     *usecases.MarkAnnouncementAsReadUseCase

	listNotifications      *usecases.ListNotificationsUseCase
	markNotificationAsRead *usecases.MarkNotificationAsReadUseCase
	markAllAsRead          *usecases.MarkAllAsReadUseCase
	archiveNotification    *usecases.ArchiveNotificationUseCase
	deleteNotification     *usecases.DeleteNotificationUseCase
	getUnreadCount         *usecases.GetUnreadCountUseCase

	createTemplate *usecases.CreateTemplateUseCase
	updateTemplate *usecases.UpdateTemplateUseCase
	renderTemplate *usecases.RenderTemplateUseCase
	listTemplates  *usecases.ListTemplatesUseCase
}

func NewServiceDDD(
	announcementRepo usecases.AnnouncementRepository,
	notificationRepo usecases.NotificationRepository,
	templateRepo usecases.NotificationTemplateRepository,
	userAnnouncementRepo usecases.UserAnnouncementReadRepository,
	announcementFactory usecases.AnnouncementFactory,
	templateFactory usecases.TemplateFactory,
	markdownService dto.MarkdownService,
	logger logger.Interface,
) *ServiceDDD {
	return &ServiceDDD{
		logger: logger,

		createAnnouncement:         usecases.NewCreateAnnouncementUseCase(announcementRepo, announcementFactory, markdownService, logger),
		updateAnnouncement:         usecases.NewUpdateAnnouncementUseCase(announcementRepo, markdownService, logger),
		deleteAnnouncement:         usecases.NewDeleteAnnouncementUseCase(announcementRepo, logger),
		publishAnnouncement:        usecases.NewPublishAnnouncementUseCase(announcementRepo, markdownService, logger),
		archiveAnnouncement:        usecases.NewArchiveAnnouncementUseCase(announcementRepo, markdownService, logger),
		listAnnouncements:          usecases.NewListAnnouncementsUseCase(announcementRepo, markdownService, logger),
		getAnnouncement:            usecases.NewGetAnnouncementUseCase(announcementRepo, markdownService, logger),
		getAnnouncementUnreadCount: usecases.NewGetAnnouncementUnreadCountUseCase(userAnnouncementRepo, logger),
		markAnnouncementAsRead:     usecases.NewMarkAnnouncementAsReadUseCase(announcementRepo, userAnnouncementRepo, logger),

		listNotifications:      usecases.NewListNotificationsUseCase(notificationRepo, markdownService, logger),
		markNotificationAsRead: usecases.NewMarkNotificationAsReadUseCase(notificationRepo, logger),
		markAllAsRead:          usecases.NewMarkAllAsReadUseCase(notificationRepo, logger),
		archiveNotification:    usecases.NewArchiveNotificationUseCase(notificationRepo, logger),
		deleteNotification:     usecases.NewDeleteNotificationUseCase(notificationRepo, logger),
		getUnreadCount:         usecases.NewGetUnreadCountUseCase(notificationRepo, logger),

		createTemplate: usecases.NewCreateTemplateUseCase(templateRepo, templateFactory, logger),
		updateTemplate: usecases.NewUpdateTemplateUseCase(templateRepo, logger),
		renderTemplate: usecases.NewRenderTemplateUseCase(templateRepo, markdownService, logger),
		listTemplates:  usecases.NewListTemplatesUseCase(templateRepo, logger),
	}
}

func (s *ServiceDDD) CreateAnnouncement(ctx context.Context, req dto.CreateAnnouncementRequest) (*dto.AnnouncementResponse, error) {
	return s.createAnnouncement.Execute(ctx, req)
}

func (s *ServiceDDD) UpdateAnnouncement(ctx context.Context, sid string, req dto.UpdateAnnouncementRequest) (*dto.AnnouncementResponse, error) {
	return s.updateAnnouncement.Execute(ctx, sid, req)
}

func (s *ServiceDDD) DeleteAnnouncement(ctx context.Context, sid string) error {
	return s.deleteAnnouncement.Execute(ctx, sid)
}

func (s *ServiceDDD) PublishAnnouncement(ctx context.Context, sid string) (*dto.AnnouncementResponse, error) {
	return s.publishAnnouncement.Execute(ctx, sid)
}

func (s *ServiceDDD) ArchiveAnnouncement(ctx context.Context, sid string) (*dto.AnnouncementResponse, error) {
	return s.archiveAnnouncement.Execute(ctx, sid)
}

func (s *ServiceDDD) ListAnnouncements(ctx context.Context, limit, offset int) (*dto.ListResponse, error) {
	return s.listAnnouncements.Execute(ctx, limit, offset)
}

func (s *ServiceDDD) ListPublishedAnnouncements(ctx context.Context, limit, offset int) (*dto.ListResponse, error) {
	return s.listAnnouncements.ExecutePublished(ctx, limit, offset)
}

func (s *ServiceDDD) GetAnnouncement(ctx context.Context, sid string) (*dto.AnnouncementResponse, error) {
	return s.getAnnouncement.Execute(ctx, sid)
}

func (s *ServiceDDD) GetAnnouncementUnreadCount(ctx context.Context, userID uint, userReadAt *time.Time) (int64, error) {
	return s.getAnnouncementUnreadCount.Execute(ctx, userID, userReadAt)
}

func (s *ServiceDDD) MarkAnnouncementAsRead(ctx context.Context, userID uint, sid string) error {
	return s.markAnnouncementAsRead.Execute(ctx, userID, sid)
}

func (s *ServiceDDD) GetReadAnnouncementIDs(ctx context.Context, userID uint) ([]uint, error) {
	return s.markAnnouncementAsRead.GetReadAnnouncementIDs(ctx, userID)
}

func (s *ServiceDDD) GetReadStatusByIDs(ctx context.Context, userID uint, announcementIDs []uint) (map[uint]bool, error) {
	return s.markAnnouncementAsRead.GetReadStatusByIDs(ctx, userID, announcementIDs)
}

func (s *ServiceDDD) ListNotifications(ctx context.Context, req dto.ListNotificationsRequest) (*dto.ListResponse, error) {
	return s.listNotifications.Execute(ctx, req)
}

func (s *ServiceDDD) MarkNotificationAsRead(ctx context.Context, id uint, userID uint) error {
	return s.markNotificationAsRead.Execute(ctx, id, userID)
}

func (s *ServiceDDD) MarkAllNotificationsAsRead(ctx context.Context, userID uint) error {
	return s.markAllAsRead.Execute(ctx, userID)
}

func (s *ServiceDDD) ArchiveNotification(ctx context.Context, id uint, userID uint) error {
	return s.archiveNotification.Execute(ctx, id, userID)
}

func (s *ServiceDDD) DeleteNotification(ctx context.Context, id uint, userID uint) error {
	return s.deleteNotification.Execute(ctx, id, userID)
}

func (s *ServiceDDD) GetUnreadCount(ctx context.Context, userID uint) (*dto.UnreadCountResponse, error) {
	return s.getUnreadCount.Execute(ctx, userID)
}

func (s *ServiceDDD) CreateTemplate(ctx context.Context, req dto.CreateTemplateRequest) (*dto.TemplateResponse, error) {
	return s.createTemplate.Execute(ctx, req)
}

func (s *ServiceDDD) UpdateTemplate(ctx context.Context, id uint, req dto.UpdateTemplateRequest) (*dto.TemplateResponse, error) {
	return s.updateTemplate.Execute(ctx, id, req)
}

func (s *ServiceDDD) RenderTemplate(ctx context.Context, req dto.RenderTemplateRequest) (*dto.RenderTemplateResponse, error) {
	return s.renderTemplate.Execute(ctx, req)
}

func (s *ServiceDDD) ListTemplates(ctx context.Context) ([]*dto.TemplateResponse, error) {
	return s.listTemplates.Execute(ctx)
}
