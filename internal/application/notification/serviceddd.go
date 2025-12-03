package notification

import (
	"context"

	"github.com/orris-inc/orris/internal/application/notification/dto"
	"github.com/orris-inc/orris/internal/application/notification/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ServiceDDD struct {
	logger logger.Interface

	createAnnouncement  *usecases.CreateAnnouncementUseCase
	updateAnnouncement  *usecases.UpdateAnnouncementUseCase
	deleteAnnouncement  *usecases.DeleteAnnouncementUseCase
	publishAnnouncement *usecases.PublishAnnouncementUseCase
	listAnnouncements   *usecases.ListAnnouncementsUseCase
	getAnnouncement     *usecases.GetAnnouncementUseCase

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
	userRepo usecases.UserRepository,
	announcementFactory usecases.AnnouncementFactory,
	notificationFactory usecases.NotificationFactory,
	templateFactory usecases.TemplateFactory,
	markdownService dto.MarkdownService,
	logger logger.Interface,
) *ServiceDDD {
	return &ServiceDDD{
		logger: logger,

		createAnnouncement:  usecases.NewCreateAnnouncementUseCase(announcementRepo, announcementFactory, markdownService, logger),
		updateAnnouncement:  usecases.NewUpdateAnnouncementUseCase(announcementRepo, markdownService, logger),
		deleteAnnouncement:  usecases.NewDeleteAnnouncementUseCase(announcementRepo, logger),
		publishAnnouncement: usecases.NewPublishAnnouncementUseCase(announcementRepo, notificationRepo, userRepo, notificationFactory, markdownService, logger),
		listAnnouncements:   usecases.NewListAnnouncementsUseCase(announcementRepo, markdownService, logger),
		getAnnouncement:     usecases.NewGetAnnouncementUseCase(announcementRepo, markdownService, logger),

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

func (s *ServiceDDD) UpdateAnnouncement(ctx context.Context, id uint, req dto.UpdateAnnouncementRequest) (*dto.AnnouncementResponse, error) {
	return s.updateAnnouncement.Execute(ctx, id, req)
}

func (s *ServiceDDD) DeleteAnnouncement(ctx context.Context, id uint) error {
	return s.deleteAnnouncement.Execute(ctx, id)
}

func (s *ServiceDDD) PublishAnnouncement(ctx context.Context, id uint, req dto.PublishAnnouncementRequest) (*dto.AnnouncementResponse, error) {
	return s.publishAnnouncement.Execute(ctx, id, req)
}

func (s *ServiceDDD) ListAnnouncements(ctx context.Context, limit, offset int) (*dto.ListResponse, error) {
	return s.listAnnouncements.Execute(ctx, limit, offset)
}

func (s *ServiceDDD) ListPublishedAnnouncements(ctx context.Context, limit, offset int) (*dto.ListResponse, error) {
	return s.listAnnouncements.ExecutePublished(ctx, limit, offset)
}

func (s *ServiceDDD) GetAnnouncement(ctx context.Context, id uint) (*dto.AnnouncementResponse, error) {
	return s.getAnnouncement.Execute(ctx, id)
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
