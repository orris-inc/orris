package notification

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	vo "github.com/orris-inc/orris/internal/domain/notification/valueobjects"
)

// =============================================================================
// NewNotification - Valid Input Tests
// =============================================================================

// TestNewNotification_ValidSystemNotification verifies creating a notification
// with all valid fields and no relatedID.
func TestNewNotification_ValidSystemNotification(t *testing.T) {
	n, err := NewNotification(1, vo.NotificationTypeSystem, "System Alert", "Server rebooted", nil)

	require.NoError(t, err)
	require.NotNil(t, n)
	assert.Equal(t, uint(0), n.ID(), "new notification should have zero ID")
	assert.Equal(t, uint(1), n.UserID())
	assert.Equal(t, vo.NotificationTypeSystem, n.Type())
	assert.Equal(t, "System Alert", n.Title())
	assert.Equal(t, "Server rebooted", n.Content())
	assert.Nil(t, n.RelatedID())
	assert.Equal(t, vo.ReadStatusUnread, n.ReadStatus())
	assert.Nil(t, n.ArchivedAt())
	assert.False(t, n.IsArchived())
	assert.WithinDuration(t, time.Now().UTC(), n.CreatedAt(), 2*time.Second)
	assert.WithinDuration(t, time.Now().UTC(), n.UpdatedAt(), 2*time.Second)
}

// TestNewNotification_ValidWithRelatedID verifies creating a notification
// with a relatedID pointer set.
func TestNewNotification_ValidWithRelatedID(t *testing.T) {
	relatedID := uint(42)
	n, err := NewNotification(1, vo.NotificationTypeSubscription, "Expiring", "Your plan expires", &relatedID)

	require.NoError(t, err)
	require.NotNil(t, n)
	require.NotNil(t, n.RelatedID())
	assert.Equal(t, uint(42), *n.RelatedID())
}

// TestNewNotification_AllValidTypes verifies that each notification type is accepted.
func TestNewNotification_AllValidTypes(t *testing.T) {
	types := []vo.NotificationType{
		vo.NotificationTypeSystem,
		vo.NotificationTypeActivity,
		vo.NotificationTypeSubscription,
		vo.NotificationTypeTemplate,
	}

	for _, nt := range types {
		t.Run(string(nt), func(t *testing.T) {
			n, err := NewNotification(1, nt, "Title", "Content", nil)
			require.NoError(t, err)
			require.NotNil(t, n)
			assert.Equal(t, nt, n.Type())
		})
	}
}

// TestNewNotification_InitializesEventsSlice verifies that the events slice
// is initialized to empty (not nil).
func TestNewNotification_InitializesEventsSlice(t *testing.T) {
	n, err := NewNotification(1, vo.NotificationTypeSystem, "Title", "Content", nil)

	require.NoError(t, err)
	events := n.GetEvents()
	assert.NotNil(t, events)
	assert.Empty(t, events)
}

// =============================================================================
// NewNotification - Validation Error Tests
// =============================================================================

// TestNewNotification_ZeroUserID verifies that a zero userID is rejected.
func TestNewNotification_ZeroUserID(t *testing.T) {
	n, err := NewNotification(0, vo.NotificationTypeSystem, "Title", "Content", nil)

	assert.Error(t, err)
	assert.Nil(t, n)
	assert.Contains(t, err.Error(), "user ID is required")
}

// TestNewNotification_InvalidType verifies that an invalid notification type is rejected.
func TestNewNotification_InvalidType(t *testing.T) {
	n, err := NewNotification(1, vo.NotificationType("invalid"), "Title", "Content", nil)

	assert.Error(t, err)
	assert.Nil(t, n)
	assert.Contains(t, err.Error(), "invalid notification type")
}

// TestNewNotification_EmptyTitle verifies that an empty title is rejected.
func TestNewNotification_EmptyTitle(t *testing.T) {
	n, err := NewNotification(1, vo.NotificationTypeSystem, "", "Content", nil)

	assert.Error(t, err)
	assert.Nil(t, n)
	assert.Contains(t, err.Error(), "title is required")
}

// TestNewNotification_TitleExceedsMaxLength verifies that a title longer
// than 200 characters is rejected.
func TestNewNotification_TitleExceedsMaxLength(t *testing.T) {
	longTitle := strings.Repeat("a", 201)
	n, err := NewNotification(1, vo.NotificationTypeSystem, longTitle, "Content", nil)

	assert.Error(t, err)
	assert.Nil(t, n)
	assert.Contains(t, err.Error(), "title exceeds maximum length of 200 characters")
}

// TestNewNotification_TitleAtMaxLength verifies that a title at exactly
// 200 characters is accepted.
func TestNewNotification_TitleAtMaxLength(t *testing.T) {
	title := strings.Repeat("a", 200)
	n, err := NewNotification(1, vo.NotificationTypeSystem, title, "Content", nil)

	require.NoError(t, err)
	require.NotNil(t, n)
	assert.Len(t, n.Title(), 200)
}

// TestNewNotification_EmptyContent verifies that empty content is rejected.
func TestNewNotification_EmptyContent(t *testing.T) {
	n, err := NewNotification(1, vo.NotificationTypeSystem, "Title", "", nil)

	assert.Error(t, err)
	assert.Nil(t, n)
	assert.Contains(t, err.Error(), "content is required")
}

// TestNewNotification_ContentExceedsMaxLength verifies that content longer
// than 5000 characters is rejected.
func TestNewNotification_ContentExceedsMaxLength(t *testing.T) {
	longContent := strings.Repeat("b", 5001)
	n, err := NewNotification(1, vo.NotificationTypeSystem, "Title", longContent, nil)

	assert.Error(t, err)
	assert.Nil(t, n)
	assert.Contains(t, err.Error(), "content exceeds maximum length of 5000 characters")
}

// TestNewNotification_ContentAtMaxLength verifies that content at exactly
// 5000 characters is accepted.
func TestNewNotification_ContentAtMaxLength(t *testing.T) {
	content := strings.Repeat("b", 5000)
	n, err := NewNotification(1, vo.NotificationTypeSystem, "Title", content, nil)

	require.NoError(t, err)
	require.NotNil(t, n)
	assert.Len(t, n.Content(), 5000)
}

// =============================================================================
// ReconstructNotification Tests
// =============================================================================

// TestReconstructNotification_ValidInput verifies reconstructing a notification
// from persisted data with all fields.
func TestReconstructNotification_ValidInput(t *testing.T) {
	now := time.Now().UTC()
	archivedAt := now.Add(-time.Hour)
	relatedID := uint(10)

	n, err := ReconstructNotification(
		1, 2, vo.NotificationTypeActivity, "Title", "Content", &relatedID,
		vo.ReadStatusRead, &archivedAt, now, now,
	)

	require.NoError(t, err)
	require.NotNil(t, n)
	assert.Equal(t, uint(1), n.ID())
	assert.Equal(t, uint(2), n.UserID())
	assert.Equal(t, vo.NotificationTypeActivity, n.Type())
	assert.Equal(t, vo.ReadStatusRead, n.ReadStatus())
	assert.NotNil(t, n.ArchivedAt())
	assert.True(t, n.IsArchived())
}

// TestReconstructNotification_ZeroID verifies that zero ID is rejected.
func TestReconstructNotification_ZeroID(t *testing.T) {
	now := time.Now().UTC()
	n, err := ReconstructNotification(
		0, 1, vo.NotificationTypeSystem, "Title", "Content", nil,
		vo.ReadStatusUnread, nil, now, now,
	)

	assert.Error(t, err)
	assert.Nil(t, n)
	assert.Contains(t, err.Error(), "notification ID cannot be zero")
}

// TestReconstructNotification_ZeroUserID verifies that zero userID is rejected.
func TestReconstructNotification_ZeroUserID(t *testing.T) {
	now := time.Now().UTC()
	n, err := ReconstructNotification(
		1, 0, vo.NotificationTypeSystem, "Title", "Content", nil,
		vo.ReadStatusUnread, nil, now, now,
	)

	assert.Error(t, err)
	assert.Nil(t, n)
	assert.Contains(t, err.Error(), "user ID is required")
}

// TestReconstructNotification_InvalidType verifies that invalid type is rejected.
func TestReconstructNotification_InvalidType(t *testing.T) {
	now := time.Now().UTC()
	n, err := ReconstructNotification(
		1, 1, vo.NotificationType("bogus"), "Title", "Content", nil,
		vo.ReadStatusUnread, nil, now, now,
	)

	assert.Error(t, err)
	assert.Nil(t, n)
	assert.Contains(t, err.Error(), "invalid notification type")
}

// TestReconstructNotification_InvalidReadStatus verifies that invalid read status is rejected.
func TestReconstructNotification_InvalidReadStatus(t *testing.T) {
	now := time.Now().UTC()
	n, err := ReconstructNotification(
		1, 1, vo.NotificationTypeSystem, "Title", "Content", nil,
		vo.ReadStatus("unknown"), nil, now, now,
	)

	assert.Error(t, err)
	assert.Nil(t, n)
	assert.Contains(t, err.Error(), "invalid read status")
}

// =============================================================================
// SetID Tests
// =============================================================================

// TestNotification_SetID_Success verifies setting ID on a new notification.
func TestNotification_SetID_Success(t *testing.T) {
	n, _ := NewNotification(1, vo.NotificationTypeSystem, "Title", "Content", nil)

	err := n.SetID(5)

	require.NoError(t, err)
	assert.Equal(t, uint(5), n.ID())
}

// TestNotification_SetID_AlreadySet verifies that setting ID twice fails.
func TestNotification_SetID_AlreadySet(t *testing.T) {
	n, _ := NewNotification(1, vo.NotificationTypeSystem, "Title", "Content", nil)
	_ = n.SetID(5)

	err := n.SetID(10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "notification ID is already set")
	assert.Equal(t, uint(5), n.ID(), "ID should not change after failed SetID")
}

// TestNotification_SetID_Zero verifies that setting ID to zero fails.
func TestNotification_SetID_Zero(t *testing.T) {
	n, _ := NewNotification(1, vo.NotificationTypeSystem, "Title", "Content", nil)

	err := n.SetID(0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "notification ID cannot be zero")
}

// =============================================================================
// MarkAsRead Tests
// =============================================================================

// TestNotification_MarkAsRead_FromUnread verifies transitioning from unread to read.
func TestNotification_MarkAsRead_FromUnread(t *testing.T) {
	n, _ := NewNotification(1, vo.NotificationTypeSystem, "Title", "Content", nil)
	assert.Equal(t, vo.ReadStatusUnread, n.ReadStatus())

	beforeMark := n.UpdatedAt()
	time.Sleep(5 * time.Millisecond)

	err := n.MarkAsRead()

	require.NoError(t, err)
	assert.Equal(t, vo.ReadStatusRead, n.ReadStatus())
	assert.True(t, n.UpdatedAt().After(beforeMark), "updatedAt should advance after MarkAsRead")
}

// TestNotification_MarkAsRead_Idempotent verifies that marking an already read
// notification as read is idempotent (no error, no timestamp change).
func TestNotification_MarkAsRead_Idempotent(t *testing.T) {
	n, _ := NewNotification(1, vo.NotificationTypeSystem, "Title", "Content", nil)
	_ = n.MarkAsRead()
	afterFirstMark := n.UpdatedAt()

	err := n.MarkAsRead()

	require.NoError(t, err)
	assert.Equal(t, vo.ReadStatusRead, n.ReadStatus())
	assert.Equal(t, afterFirstMark, n.UpdatedAt(), "updatedAt should not change on duplicate MarkAsRead")
}

// =============================================================================
// Archive Tests
// =============================================================================

// TestNotification_Archive_Success verifies archiving a non-archived notification.
func TestNotification_Archive_Success(t *testing.T) {
	n, _ := NewNotification(1, vo.NotificationTypeSystem, "Title", "Content", nil)
	assert.False(t, n.IsArchived())

	beforeArchive := n.UpdatedAt()
	time.Sleep(5 * time.Millisecond)

	err := n.Archive()

	require.NoError(t, err)
	assert.True(t, n.IsArchived())
	assert.NotNil(t, n.ArchivedAt())
	assert.True(t, n.UpdatedAt().After(beforeArchive), "updatedAt should advance after Archive")
}

// TestNotification_Archive_AlreadyArchived verifies that archiving an already
// archived notification returns an error.
func TestNotification_Archive_AlreadyArchived(t *testing.T) {
	n, _ := NewNotification(1, vo.NotificationTypeSystem, "Title", "Content", nil)
	_ = n.Archive()
	archivedTime := n.ArchivedAt()

	err := n.Archive()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "notification is already archived")
	assert.Equal(t, archivedTime, n.ArchivedAt(), "archivedAt should not change on duplicate Archive")
}

// TestNotification_IsArchived_FalseForNewNotification verifies that a new
// notification is not archived.
func TestNotification_IsArchived_FalseForNewNotification(t *testing.T) {
	n, _ := NewNotification(1, vo.NotificationTypeSystem, "Title", "Content", nil)

	assert.False(t, n.IsArchived())
	assert.Nil(t, n.ArchivedAt())
}

// TestNotification_IsArchived_TrueAfterArchive verifies that IsArchived returns
// true after the notification is archived.
func TestNotification_IsArchived_TrueAfterArchive(t *testing.T) {
	n, _ := NewNotification(1, vo.NotificationTypeSystem, "Title", "Content", nil)
	_ = n.Archive()

	assert.True(t, n.IsArchived())
}

// =============================================================================
// Domain Event Tests
// =============================================================================

// TestNotification_GetEvents_ReturnsEmptyAndDrains verifies that GetEvents
// returns a copy and clears internal events.
func TestNotification_GetEvents_ReturnsEmptyAndDrains(t *testing.T) {
	n, _ := NewNotification(1, vo.NotificationTypeSystem, "Title", "Content", nil)

	events := n.GetEvents()

	assert.NotNil(t, events)
	assert.Empty(t, events)

	// Second call should also return empty
	events2 := n.GetEvents()
	assert.Empty(t, events2)
}

// TestNotification_ClearEvents_ResetsSlice verifies that ClearEvents resets
// the internal events slice.
func TestNotification_ClearEvents_ResetsSlice(t *testing.T) {
	n, _ := NewNotification(1, vo.NotificationTypeSystem, "Title", "Content", nil)

	n.ClearEvents()
	events := n.GetEvents()

	assert.NotNil(t, events)
	assert.Empty(t, events)
}

// TestNotification_GetEvents_DrainsOnCall verifies that GetEvents drains events
// so a subsequent GetEvents returns empty.
func TestNotification_GetEvents_DrainsOnCall(t *testing.T) {
	n, _ := NewNotification(1, vo.NotificationTypeSystem, "Title", "Content", nil)

	// First call: should drain
	_ = n.GetEvents()
	// Second call: should be empty
	events := n.GetEvents()

	assert.Empty(t, events)
}

// =============================================================================
// Edge Cases
// =============================================================================

// TestNotification_MarkAsRead_ThenArchive verifies the combined workflow of
// marking as read then archiving.
func TestNotification_MarkAsRead_ThenArchive(t *testing.T) {
	n, _ := NewNotification(1, vo.NotificationTypeSystem, "Title", "Content", nil)

	err := n.MarkAsRead()
	require.NoError(t, err)
	assert.Equal(t, vo.ReadStatusRead, n.ReadStatus())

	err = n.Archive()
	require.NoError(t, err)
	assert.True(t, n.IsArchived())
	assert.Equal(t, vo.ReadStatusRead, n.ReadStatus())
}

// TestNotification_Archive_ThenMarkAsRead verifies that archiving and then
// marking as read both succeed (Archive does not block MarkAsRead).
func TestNotification_Archive_ThenMarkAsRead(t *testing.T) {
	n, _ := NewNotification(1, vo.NotificationTypeSystem, "Title", "Content", nil)

	err := n.Archive()
	require.NoError(t, err)

	err = n.MarkAsRead()
	require.NoError(t, err)
	assert.True(t, n.IsArchived())
	assert.Equal(t, vo.ReadStatusRead, n.ReadStatus())
}

// =============================================================================
// Factory Function Tests
// =============================================================================

// TestCreateSystemNotification verifies the convenience factory for system notifications.
func TestCreateSystemNotification(t *testing.T) {
	n, err := CreateSystemNotification(1, "System Alert", "Check logs")

	require.NoError(t, err)
	require.NotNil(t, n)
	assert.Equal(t, vo.NotificationTypeSystem, n.Type())
	assert.Nil(t, n.RelatedID())
}

// TestCreateActivityNotification verifies the convenience factory for activity notifications.
func TestCreateActivityNotification(t *testing.T) {
	n, err := CreateActivityNotification(1, "Login", "New login from IP", 100)

	require.NoError(t, err)
	require.NotNil(t, n)
	assert.Equal(t, vo.NotificationTypeActivity, n.Type())
	require.NotNil(t, n.RelatedID())
	assert.Equal(t, uint(100), *n.RelatedID())
}

// TestCreateSubscriptionNotification verifies the convenience factory for subscription notifications.
func TestCreateSubscriptionNotification(t *testing.T) {
	n, err := CreateSubscriptionNotification(1, "Expiring", "Plan expires in 3 days", 50)

	require.NoError(t, err)
	require.NotNil(t, n)
	assert.Equal(t, vo.NotificationTypeSubscription, n.Type())
	require.NotNil(t, n.RelatedID())
	assert.Equal(t, uint(50), *n.RelatedID())
}
