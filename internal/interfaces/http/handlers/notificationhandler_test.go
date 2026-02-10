package handlers

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appDto "github.com/orris-inc/orris/internal/application/notification/dto"
	"github.com/orris-inc/orris/internal/domain/user"
	uservo "github.com/orris-inc/orris/internal/domain/user/valueobjects"
	"github.com/orris-inc/orris/internal/interfaces/http/handlers/testutil"
	"github.com/orris-inc/orris/internal/shared/authorization"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
)

// =====================================================================
// Mock notification service
// =====================================================================

type mockNotificationService struct {
	// Announcement methods
	createAnnouncementFn  func(ctx context.Context, req appDto.CreateAnnouncementRequest) (*appDto.AnnouncementResponse, error)
	updateAnnouncementFn  func(ctx context.Context, sid string, req appDto.UpdateAnnouncementRequest) (*appDto.AnnouncementResponse, error)
	deleteAnnouncementFn  func(ctx context.Context, sid string) error
	publishAnnouncementFn func(ctx context.Context, sid string) (*appDto.AnnouncementResponse, error)
	archiveAnnouncementFn func(ctx context.Context, sid string) (*appDto.AnnouncementResponse, error)
	listAnnouncementsFn   func(ctx context.Context, limit, offset int) (*appDto.ListResponse, error)
	listPublishedFn       func(ctx context.Context, limit, offset int) (*appDto.ListResponse, error)
	getAnnouncementFn     func(ctx context.Context, sid string) (*appDto.AnnouncementResponse, error)
	getAnnUnreadCountFn   func(ctx context.Context, userID uint, userReadAt *time.Time) (int64, error)
	markAnnAsReadFn       func(ctx context.Context, userID uint, sid string) error
	getReadStatusByIDsFn  func(ctx context.Context, userID uint, announcementIDs []uint) (map[uint]bool, error)

	// Notification methods
	listNotificationsFn   func(ctx context.Context, req appDto.ListNotificationsRequest) (*appDto.ListResponse, error)
	markNotifAsReadFn     func(ctx context.Context, id uint, userID uint) error
	markAllNotifAsReadFn  func(ctx context.Context, userID uint) error
	archiveNotificationFn func(ctx context.Context, id uint, userID uint) error
	deleteNotificationFn  func(ctx context.Context, id uint, userID uint) error
	getUnreadCountFn      func(ctx context.Context, userID uint) (*appDto.UnreadCountResponse, error)

	// Template methods
	createTemplateFn func(ctx context.Context, req appDto.CreateTemplateRequest) (*appDto.TemplateResponse, error)
	renderTemplateFn func(ctx context.Context, req appDto.RenderTemplateRequest) (*appDto.RenderTemplateResponse, error)
	listTemplatesFn  func(ctx context.Context) ([]*appDto.TemplateResponse, error)
}

func (m *mockNotificationService) CreateAnnouncement(ctx context.Context, req appDto.CreateAnnouncementRequest) (*appDto.AnnouncementResponse, error) {
	if m.createAnnouncementFn != nil {
		return m.createAnnouncementFn(ctx, req)
	}
	return nil, nil
}

func (m *mockNotificationService) UpdateAnnouncement(ctx context.Context, sid string, req appDto.UpdateAnnouncementRequest) (*appDto.AnnouncementResponse, error) {
	if m.updateAnnouncementFn != nil {
		return m.updateAnnouncementFn(ctx, sid, req)
	}
	return nil, nil
}

func (m *mockNotificationService) DeleteAnnouncement(ctx context.Context, sid string) error {
	if m.deleteAnnouncementFn != nil {
		return m.deleteAnnouncementFn(ctx, sid)
	}
	return nil
}

func (m *mockNotificationService) PublishAnnouncement(ctx context.Context, sid string) (*appDto.AnnouncementResponse, error) {
	if m.publishAnnouncementFn != nil {
		return m.publishAnnouncementFn(ctx, sid)
	}
	return nil, nil
}

func (m *mockNotificationService) ArchiveAnnouncement(ctx context.Context, sid string) (*appDto.AnnouncementResponse, error) {
	if m.archiveAnnouncementFn != nil {
		return m.archiveAnnouncementFn(ctx, sid)
	}
	return nil, nil
}

func (m *mockNotificationService) ListAnnouncements(ctx context.Context, limit, offset int) (*appDto.ListResponse, error) {
	if m.listAnnouncementsFn != nil {
		return m.listAnnouncementsFn(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockNotificationService) ListPublishedAnnouncements(ctx context.Context, limit, offset int) (*appDto.ListResponse, error) {
	if m.listPublishedFn != nil {
		return m.listPublishedFn(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockNotificationService) GetAnnouncement(ctx context.Context, sid string) (*appDto.AnnouncementResponse, error) {
	if m.getAnnouncementFn != nil {
		return m.getAnnouncementFn(ctx, sid)
	}
	return nil, nil
}

func (m *mockNotificationService) GetAnnouncementUnreadCount(ctx context.Context, userID uint, userReadAt *time.Time) (int64, error) {
	if m.getAnnUnreadCountFn != nil {
		return m.getAnnUnreadCountFn(ctx, userID, userReadAt)
	}
	return 0, nil
}

func (m *mockNotificationService) MarkAnnouncementAsRead(ctx context.Context, userID uint, sid string) error {
	if m.markAnnAsReadFn != nil {
		return m.markAnnAsReadFn(ctx, userID, sid)
	}
	return nil
}

func (m *mockNotificationService) GetReadStatusByIDs(ctx context.Context, userID uint, announcementIDs []uint) (map[uint]bool, error) {
	if m.getReadStatusByIDsFn != nil {
		return m.getReadStatusByIDsFn(ctx, userID, announcementIDs)
	}
	return nil, nil
}

func (m *mockNotificationService) ListNotifications(ctx context.Context, req appDto.ListNotificationsRequest) (*appDto.ListResponse, error) {
	if m.listNotificationsFn != nil {
		return m.listNotificationsFn(ctx, req)
	}
	return nil, nil
}

func (m *mockNotificationService) MarkNotificationAsRead(ctx context.Context, id uint, userID uint) error {
	if m.markNotifAsReadFn != nil {
		return m.markNotifAsReadFn(ctx, id, userID)
	}
	return nil
}

func (m *mockNotificationService) MarkAllNotificationsAsRead(ctx context.Context, userID uint) error {
	if m.markAllNotifAsReadFn != nil {
		return m.markAllNotifAsReadFn(ctx, userID)
	}
	return nil
}

func (m *mockNotificationService) ArchiveNotification(ctx context.Context, id uint, userID uint) error {
	if m.archiveNotificationFn != nil {
		return m.archiveNotificationFn(ctx, id, userID)
	}
	return nil
}

func (m *mockNotificationService) DeleteNotification(ctx context.Context, id uint, userID uint) error {
	if m.deleteNotificationFn != nil {
		return m.deleteNotificationFn(ctx, id, userID)
	}
	return nil
}

func (m *mockNotificationService) GetUnreadCount(ctx context.Context, userID uint) (*appDto.UnreadCountResponse, error) {
	if m.getUnreadCountFn != nil {
		return m.getUnreadCountFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockNotificationService) CreateTemplate(ctx context.Context, req appDto.CreateTemplateRequest) (*appDto.TemplateResponse, error) {
	if m.createTemplateFn != nil {
		return m.createTemplateFn(ctx, req)
	}
	return nil, nil
}

func (m *mockNotificationService) RenderTemplate(ctx context.Context, req appDto.RenderTemplateRequest) (*appDto.RenderTemplateResponse, error) {
	if m.renderTemplateFn != nil {
		return m.renderTemplateFn(ctx, req)
	}
	return nil, nil
}

func (m *mockNotificationService) ListTemplates(ctx context.Context) ([]*appDto.TemplateResponse, error) {
	if m.listTemplatesFn != nil {
		return m.listTemplatesFn(ctx)
	}
	return nil, nil
}

// =====================================================================
// Mock user repository
// =====================================================================

type mockNotifUserRepo struct {
	getByIDFn func(ctx context.Context, id uint) (*user.User, error)
	updateFn  func(ctx context.Context, u *user.User) error
}

func (m *mockNotifUserRepo) GetByID(ctx context.Context, id uint) (*user.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockNotifUserRepo) Update(ctx context.Context, u *user.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, u)
	}
	return nil
}

// =====================================================================
// Test helpers
// =====================================================================

func createNotifTestUser() *user.User {
	email, _ := uservo.NewEmail("test@example.com")
	name, _ := uservo.NewName("Test User")
	now := time.Now().UTC()

	u, _ := user.ReconstructUser(
		1, "usr_test123",
		email, name,
		authorization.RoleUser, uservo.StatusActive,
		now, now,
		1,
	)
	return u
}

func newTestNotificationHandler(svc notificationService, userRepo notificationUserRepo) *NotificationHandler {
	return NewNotificationHandler(svc, userRepo, testutil.NewMockLogger())
}

// =====================================================================
// TestNotificationHandler_ListNotifications
// =====================================================================

func TestNotificationHandler_ListNotifications_Success(t *testing.T) {
	mockSvc := &mockNotificationService{
		listNotificationsFn: func(ctx context.Context, req appDto.ListNotificationsRequest) (*appDto.ListResponse, error) {
			return &appDto.ListResponse{
				Items:  []*appDto.NotificationResponse{},
				Total:  0,
				Limit:  20,
				Offset: 0,
			}, nil
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/notifications", nil)
	testutil.SetAuthContext(c, 1)

	handler.ListNotifications(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNotificationHandler_ListNotifications_InvalidLimit(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/notifications", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetQueryParams(c, map[string]string{"limit": "abc"})

	handler.ListNotifications(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestNotificationHandler_ListNotifications_NotAuthenticated(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/notifications", nil)
	// No auth context set

	handler.ListNotifications(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestNotificationHandler_ListNotifications_ServiceError(t *testing.T) {
	mockSvc := &mockNotificationService{
		listNotificationsFn: func(ctx context.Context, req appDto.ListNotificationsRequest) (*appDto.ListResponse, error) {
			return nil, stderrors.New("database error")
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/notifications", nil)
	testutil.SetAuthContext(c, 1)

	handler.ListNotifications(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestNotificationHandler_GetUnreadCount
// =====================================================================

func TestNotificationHandler_GetUnreadCount_Success(t *testing.T) {
	mockSvc := &mockNotificationService{
		getUnreadCountFn: func(ctx context.Context, userID uint) (*appDto.UnreadCountResponse, error) {
			return &appDto.UnreadCountResponse{Count: 5}, nil
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/notifications/unread-count", nil)
	testutil.SetAuthContext(c, 1)

	handler.GetUnreadCount(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	var data appDto.UnreadCountResponse
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Equal(t, int64(5), data.Count)
}

func TestNotificationHandler_GetUnreadCount_NotAuthenticated(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/notifications/unread-count", nil)
	// No auth context

	handler.GetUnreadCount(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestNotificationHandler_GetUnreadCount_ServiceError(t *testing.T) {
	mockSvc := &mockNotificationService{
		getUnreadCountFn: func(ctx context.Context, userID uint) (*appDto.UnreadCountResponse, error) {
			return nil, stderrors.New("database error")
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/notifications/unread-count", nil)
	testutil.SetAuthContext(c, 1)

	handler.GetUnreadCount(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// =====================================================================
// TestNotificationHandler_UpdateNotificationStatus (MarkAsRead)
// =====================================================================

func TestNotificationHandler_MarkAsRead_Success(t *testing.T) {
	mockSvc := &mockNotificationService{
		markNotifAsReadFn: func(ctx context.Context, id uint, userID uint) error {
			return nil
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	reqBody := UpdateNotificationStatusRequest{Status: "read"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/notifications/1/status", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "1")

	handler.UpdateNotificationStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNotificationHandler_MarkAsRead_InvalidID(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	reqBody := UpdateNotificationStatusRequest{Status: "read"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/notifications/abc/status", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "abc")

	handler.UpdateNotificationStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestNotificationHandler_MarkAsRead_ZeroID(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	reqBody := UpdateNotificationStatusRequest{Status: "read"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/notifications/0/status", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "0")

	handler.UpdateNotificationStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestNotificationHandler_MarkAsRead_NotAuthenticated(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	reqBody := UpdateNotificationStatusRequest{Status: "read"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/notifications/1/status", reqBody)
	// No auth context
	testutil.SetURLParam(c, "id", "1")

	handler.UpdateNotificationStatus(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNotificationHandler_MarkAsRead_ServiceError(t *testing.T) {
	mockSvc := &mockNotificationService{
		markNotifAsReadFn: func(ctx context.Context, id uint, userID uint) error {
			return errors.NewNotFoundError("notification not found")
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	reqBody := UpdateNotificationStatusRequest{Status: "read"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/notifications/999/status", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "999")

	handler.UpdateNotificationStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestNotificationHandler_UpdateNotificationStatus (Archive)
// =====================================================================

func TestNotificationHandler_ArchiveNotification_Success(t *testing.T) {
	mockSvc := &mockNotificationService{
		archiveNotificationFn: func(ctx context.Context, id uint, userID uint) error {
			return nil
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	reqBody := UpdateNotificationStatusRequest{Status: "archived"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/notifications/1/status", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "1")

	handler.UpdateNotificationStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNotificationHandler_ArchiveNotification_InvalidID(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	reqBody := UpdateNotificationStatusRequest{Status: "archived"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/notifications/bad/status", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "bad")

	handler.UpdateNotificationStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNotificationHandler_ArchiveNotification_ServiceError(t *testing.T) {
	mockSvc := &mockNotificationService{
		archiveNotificationFn: func(ctx context.Context, id uint, userID uint) error {
			return errors.NewNotFoundError("notification not found")
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	reqBody := UpdateNotificationStatusRequest{Status: "archived"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/notifications/42/status", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "42")

	handler.UpdateNotificationStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =====================================================================
// TestNotificationHandler_UpdateAllNotificationsStatus (MarkAllAsRead)
// =====================================================================

func TestNotificationHandler_MarkAllAsRead_Success(t *testing.T) {
	mockSvc := &mockNotificationService{
		markAllNotifAsReadFn: func(ctx context.Context, userID uint) error {
			return nil
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	reqBody := UpdateAllNotificationsStatusRequest{Status: "read"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/notifications/status", reqBody)
	testutil.SetAuthContext(c, 1)

	handler.UpdateAllNotificationsStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNotificationHandler_MarkAllAsRead_NotAuthenticated(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	reqBody := UpdateAllNotificationsStatusRequest{Status: "read"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/notifications/status", reqBody)
	// No auth context

	handler.UpdateAllNotificationsStatus(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNotificationHandler_MarkAllAsRead_ServiceError(t *testing.T) {
	mockSvc := &mockNotificationService{
		markAllNotifAsReadFn: func(ctx context.Context, userID uint) error {
			return stderrors.New("database error")
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	reqBody := UpdateAllNotificationsStatusRequest{Status: "read"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/notifications/status", reqBody)
	testutil.SetAuthContext(c, 1)

	handler.UpdateAllNotificationsStatus(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestNotificationHandler_MarkAllAsRead_InvalidBody(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	// Send empty body (missing required "status" field)
	c, w := testutil.NewTestContext(http.MethodPatch, "/notifications/status", map[string]string{})
	testutil.SetAuthContext(c, 1)

	handler.UpdateAllNotificationsStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =====================================================================
// TestNotificationHandler_DeleteNotification
// =====================================================================

func TestNotificationHandler_DeleteNotification_Success(t *testing.T) {
	mockSvc := &mockNotificationService{
		deleteNotificationFn: func(ctx context.Context, id uint, userID uint) error {
			return nil
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/notifications/1", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "1")

	handler.DeleteNotification(c)

	// gin's c.Status() sets the status on the writer; use Writer.Status() for reliable check.
	assert.Equal(t, http.StatusNoContent, c.Writer.Status())
	assert.Empty(t, w.Body.String())
}

func TestNotificationHandler_DeleteNotification_InvalidID(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/notifications/xyz", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "xyz")

	handler.DeleteNotification(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNotificationHandler_DeleteNotification_NotAuthenticated(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/notifications/1", nil)
	// No auth context
	testutil.SetURLParam(c, "id", "1")

	handler.DeleteNotification(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNotificationHandler_DeleteNotification_ServiceError(t *testing.T) {
	mockSvc := &mockNotificationService{
		deleteNotificationFn: func(ctx context.Context, id uint, userID uint) error {
			return errors.NewNotFoundError("notification not found")
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodDelete, "/notifications/999", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "999")

	handler.DeleteNotification(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =====================================================================
// TestNotificationHandler_ListAnnouncements (admin)
// =====================================================================

func TestNotificationHandler_ListAnnouncements_Success(t *testing.T) {
	mockSvc := &mockNotificationService{
		listAnnouncementsFn: func(ctx context.Context, limit, offset int) (*appDto.ListResponse, error) {
			return &appDto.ListResponse{
				Items:  []*appDto.AnnouncementResponse{},
				Total:  0,
				Limit:  20,
				Offset: 0,
			}, nil
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/admin/announcements", nil)
	testutil.SetAuthContext(c, 1)

	handler.ListAnnouncements(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNotificationHandler_ListAnnouncements_ServiceError(t *testing.T) {
	mockSvc := &mockNotificationService{
		listAnnouncementsFn: func(ctx context.Context, limit, offset int) (*appDto.ListResponse, error) {
			return nil, stderrors.New("database error")
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/admin/announcements", nil)
	testutil.SetAuthContext(c, 1)

	handler.ListAnnouncements(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// =====================================================================
// TestNotificationHandler_ListPublicAnnouncements
// =====================================================================

func TestNotificationHandler_ListPublicAnnouncements_Success(t *testing.T) {
	mockSvc := &mockNotificationService{
		listPublishedFn: func(ctx context.Context, limit, offset int) (*appDto.ListResponse, error) {
			return &appDto.ListResponse{
				Items:  []*appDto.AnnouncementResponse{},
				Total:  0,
				Limit:  20,
				Offset: 0,
			}, nil
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/announcements", nil)

	handler.ListPublicAnnouncements(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNotificationHandler_ListPublicAnnouncements_WithAuthUser(t *testing.T) {
	now := time.Now().UTC()
	mockSvc := &mockNotificationService{
		listPublishedFn: func(ctx context.Context, limit, offset int) (*appDto.ListResponse, error) {
			return &appDto.ListResponse{
				Items: []*appDto.AnnouncementResponse{
					{
						ID:         "ann_xK9mP2vL3nQR",
						InternalID: 10,
						Title:      "Test Announcement",
						Status:     "published",
						UpdatedAt:  now.Add(-1 * time.Hour),
					},
				},
				Total:  1,
				Limit:  20,
				Offset: 0,
			}, nil
		},
		getReadStatusByIDsFn: func(ctx context.Context, userID uint, announcementIDs []uint) (map[uint]bool, error) {
			return map[uint]bool{10: false}, nil
		},
	}
	testUser := createNotifTestUser()
	mockRepo := &mockNotifUserRepo{
		getByIDFn: func(ctx context.Context, id uint) (*user.User, error) {
			return testUser, nil
		},
	}
	handler := newTestNotificationHandler(mockSvc, mockRepo)

	c, w := testutil.NewTestContext(http.MethodGet, "/announcements", nil)
	testutil.SetAuthContext(c, 1)

	handler.ListPublicAnnouncements(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

// =====================================================================
// TestNotificationHandler_GetAnnouncement
// =====================================================================

func TestNotificationHandler_GetAnnouncement_SuccessPublished(t *testing.T) {
	mockSvc := &mockNotificationService{
		getAnnouncementFn: func(ctx context.Context, sid string) (*appDto.AnnouncementResponse, error) {
			return &appDto.AnnouncementResponse{
				ID:     "ann_xK9mP2vL3nQR",
				Title:  "Test",
				Status: "published",
			}, nil
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/announcements/ann_xK9mP2vL3nQR", nil)
	testutil.SetURLParam(c, "id", "ann_xK9mP2vL3nQR")
	// Regular user
	c.Set(constants.ContextKeyUserRole, "user")

	handler.GetAnnouncement(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNotificationHandler_GetAnnouncement_DraftAsNonAdmin(t *testing.T) {
	mockSvc := &mockNotificationService{
		getAnnouncementFn: func(ctx context.Context, sid string) (*appDto.AnnouncementResponse, error) {
			return &appDto.AnnouncementResponse{
				ID:     "ann_xK9mP2vL3nQR",
				Title:  "Draft",
				Status: "draft",
			}, nil
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/announcements/ann_xK9mP2vL3nQR", nil)
	testutil.SetURLParam(c, "id", "ann_xK9mP2vL3nQR")
	c.Set(constants.ContextKeyUserRole, "user")

	handler.GetAnnouncement(c)

	// Non-admin should get 404 for draft announcements
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestNotificationHandler_GetAnnouncement_DraftAsAdmin(t *testing.T) {
	mockSvc := &mockNotificationService{
		getAnnouncementFn: func(ctx context.Context, sid string) (*appDto.AnnouncementResponse, error) {
			return &appDto.AnnouncementResponse{
				ID:     "ann_xK9mP2vL3nQR",
				Title:  "Draft",
				Status: "draft",
			}, nil
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/announcements/ann_xK9mP2vL3nQR", nil)
	testutil.SetURLParam(c, "id", "ann_xK9mP2vL3nQR")
	c.Set(constants.ContextKeyUserRole, "admin")

	handler.GetAnnouncement(c)

	// Admin should be able to see draft announcements
	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNotificationHandler_GetAnnouncement_InvalidSID(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/announcements/invalid_id", nil)
	testutil.SetURLParam(c, "id", "invalid_id")

	handler.GetAnnouncement(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNotificationHandler_GetAnnouncement_NotFound(t *testing.T) {
	mockSvc := &mockNotificationService{
		getAnnouncementFn: func(ctx context.Context, sid string) (*appDto.AnnouncementResponse, error) {
			return nil, errors.NewNotFoundError("announcement not found")
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/announcements/ann_xK9mP2vL3nQR", nil)
	testutil.SetURLParam(c, "id", "ann_xK9mP2vL3nQR")

	handler.GetAnnouncement(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =====================================================================
// TestNotificationHandler_MarkAnnouncementsAsRead
// =====================================================================

func TestNotificationHandler_MarkAnnouncementsAsRead_Success(t *testing.T) {
	testUser := createNotifTestUser()
	mockRepo := &mockNotifUserRepo{
		getByIDFn: func(ctx context.Context, id uint) (*user.User, error) {
			return testUser, nil
		},
		updateFn: func(ctx context.Context, u *user.User) error {
			return nil
		},
	}
	handler := newTestNotificationHandler(&mockNotificationService{}, mockRepo)

	c, w := testutil.NewTestContext(http.MethodPost, "/announcements/read", nil)
	testutil.SetAuthContext(c, 1)

	handler.MarkAnnouncementsAsRead(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNotificationHandler_MarkAnnouncementsAsRead_NotAuthenticated(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/announcements/read", nil)
	// No auth context

	handler.MarkAnnouncementsAsRead(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNotificationHandler_MarkAnnouncementsAsRead_UserNotFound(t *testing.T) {
	mockRepo := &mockNotifUserRepo{
		getByIDFn: func(ctx context.Context, id uint) (*user.User, error) {
			return nil, nil
		},
	}
	handler := newTestNotificationHandler(&mockNotificationService{}, mockRepo)

	c, w := testutil.NewTestContext(http.MethodPost, "/announcements/read", nil)
	testutil.SetAuthContext(c, 1)

	handler.MarkAnnouncementsAsRead(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestNotificationHandler_MarkAnnouncementsAsRead_UpdateError(t *testing.T) {
	testUser := createNotifTestUser()
	mockRepo := &mockNotifUserRepo{
		getByIDFn: func(ctx context.Context, id uint) (*user.User, error) {
			return testUser, nil
		},
		updateFn: func(ctx context.Context, u *user.User) error {
			return stderrors.New("database error")
		},
	}
	handler := newTestNotificationHandler(&mockNotificationService{}, mockRepo)

	c, w := testutil.NewTestContext(http.MethodPost, "/announcements/read", nil)
	testutil.SetAuthContext(c, 1)

	handler.MarkAnnouncementsAsRead(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// =====================================================================
// TestNotificationHandler_GetAnnouncementUnreadCount
// =====================================================================

func TestNotificationHandler_GetAnnouncementUnreadCount_Success(t *testing.T) {
	testUser := createNotifTestUser()
	mockSvc := &mockNotificationService{
		getAnnUnreadCountFn: func(ctx context.Context, userID uint, userReadAt *time.Time) (int64, error) {
			return 3, nil
		},
	}
	mockRepo := &mockNotifUserRepo{
		getByIDFn: func(ctx context.Context, id uint) (*user.User, error) {
			return testUser, nil
		},
	}
	handler := newTestNotificationHandler(mockSvc, mockRepo)

	c, w := testutil.NewTestContext(http.MethodGet, "/announcements/unread-count", nil)
	testutil.SetAuthContext(c, 1)

	handler.GetAnnouncementUnreadCount(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	var data appDto.UnreadCountResponse
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Equal(t, int64(3), data.Count)
}

func TestNotificationHandler_GetAnnouncementUnreadCount_NotAuthenticated(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/announcements/unread-count", nil)
	// No auth context

	handler.GetAnnouncementUnreadCount(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNotificationHandler_GetAnnouncementUnreadCount_UserNotFound(t *testing.T) {
	mockRepo := &mockNotifUserRepo{
		getByIDFn: func(ctx context.Context, id uint) (*user.User, error) {
			return nil, nil
		},
	}
	handler := newTestNotificationHandler(&mockNotificationService{}, mockRepo)

	c, w := testutil.NewTestContext(http.MethodGet, "/announcements/unread-count", nil)
	testutil.SetAuthContext(c, 1)

	handler.GetAnnouncementUnreadCount(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =====================================================================
// TestNotificationHandler_MarkAnnouncementAsRead
// =====================================================================

func TestNotificationHandler_MarkAnnouncementAsRead_Success(t *testing.T) {
	mockSvc := &mockNotificationService{
		markAnnAsReadFn: func(ctx context.Context, userID uint, sid string) error {
			return nil
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/announcements/ann_xK9mP2vL3nQR/read", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "ann_xK9mP2vL3nQR")

	handler.MarkAnnouncementAsRead(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNotificationHandler_MarkAnnouncementAsRead_InvalidSID(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/announcements/invalid/read", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "invalid")

	handler.MarkAnnouncementAsRead(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNotificationHandler_MarkAnnouncementAsRead_NotAuthenticated(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/announcements/ann_xK9mP2vL3nQR/read", nil)
	// No auth context
	testutil.SetURLParam(c, "id", "ann_xK9mP2vL3nQR")

	handler.MarkAnnouncementAsRead(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNotificationHandler_MarkAnnouncementAsRead_ServiceError(t *testing.T) {
	mockSvc := &mockNotificationService{
		markAnnAsReadFn: func(ctx context.Context, userID uint, sid string) error {
			return errors.NewNotFoundError("announcement not found")
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodPost, "/announcements/ann_xK9mP2vL3nQR/read", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "ann_xK9mP2vL3nQR")

	handler.MarkAnnouncementAsRead(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =====================================================================
// TestNotificationHandler_UpdateNotificationStatus_InvalidBody
// =====================================================================

func TestNotificationHandler_UpdateNotificationStatus_InvalidBody(t *testing.T) {
	handler := newTestNotificationHandler(&mockNotificationService{}, nil)

	// Missing required "status" field
	c, w := testutil.NewTestContext(http.MethodPatch, "/notifications/1/status", map[string]string{})
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "1")

	handler.UpdateNotificationStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// =====================================================================
// TestNotificationHandler_ListTemplates
// =====================================================================

func TestNotificationHandler_ListTemplates_Success(t *testing.T) {
	mockSvc := &mockNotificationService{
		listTemplatesFn: func(ctx context.Context) ([]*appDto.TemplateResponse, error) {
			return []*appDto.TemplateResponse{
				{
					ID:           1,
					TemplateType: "welcome",
					Name:         "Welcome Template",
					Title:        "Welcome",
					Content:      "Welcome to our service",
				},
			}, nil
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/admin/notification-templates", nil)
	testutil.SetAuthContext(c, 1)

	handler.ListTemplates(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestNotificationHandler_ListTemplates_ServiceError(t *testing.T) {
	mockSvc := &mockNotificationService{
		listTemplatesFn: func(ctx context.Context) ([]*appDto.TemplateResponse, error) {
			return nil, stderrors.New("database error")
		},
	}
	handler := newTestNotificationHandler(mockSvc, nil)

	c, w := testutil.NewTestContext(http.MethodGet, "/admin/notification-templates", nil)
	testutil.SetAuthContext(c, 1)

	handler.ListTemplates(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
