package notification

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	vo "github.com/orris-inc/orris/internal/domain/notification/valueobjects"
)

// mockSIDGenerator returns a predictable SID generator for testing.
func mockSIDGenerator() SIDGenerator {
	counter := 0
	return func() (string, error) {
		counter++
		return fmt.Sprintf("ann_test_%d", counter), nil
	}
}

// failingSIDGenerator returns a SID generator that always fails.
func failingSIDGenerator() SIDGenerator {
	return func() (string, error) {
		return "", fmt.Errorf("sid generation failed")
	}
}

// =============================================================================
// NewAnnouncement - Valid Input Tests
// =============================================================================

// TestNewAnnouncement_ValidMinimal verifies creating an announcement with
// minimum required fields (no scheduledAt, no expiresAt).
func TestNewAnnouncement_ValidMinimal(t *testing.T) {
	gen := mockSIDGenerator()
	a, err := NewAnnouncement(
		"System Update", "We are upgrading servers",
		vo.AnnouncementTypeSystem, 1, 3, nil, nil, gen,
	)

	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Equal(t, uint(0), a.ID(), "new announcement should have zero ID")
	assert.NotEmpty(t, a.SID())
	assert.Equal(t, "System Update", a.Title())
	assert.Equal(t, "We are upgrading servers", a.Content())
	assert.Equal(t, vo.AnnouncementTypeSystem, a.Type())
	assert.Equal(t, vo.AnnouncementStatusDraft, a.Status())
	assert.Equal(t, uint(1), a.CreatorID())
	assert.Equal(t, 3, a.Priority())
	assert.Nil(t, a.ScheduledAt())
	assert.Nil(t, a.ExpiresAt())
	assert.Equal(t, 0, a.ViewCount())
	assert.WithinDuration(t, time.Now().UTC(), a.CreatedAt(), 2*time.Second)
	assert.WithinDuration(t, time.Now().UTC(), a.UpdatedAt(), 2*time.Second)
}

// TestNewAnnouncement_ValidWithScheduleAndExpiry verifies creating an announcement
// with scheduled and expiry times.
func TestNewAnnouncement_ValidWithScheduleAndExpiry(t *testing.T) {
	gen := mockSIDGenerator()
	scheduled := time.Now().UTC().Add(time.Hour)
	expires := time.Now().UTC().Add(24 * time.Hour)

	a, err := NewAnnouncement(
		"Maintenance", "Planned maintenance window",
		vo.AnnouncementTypeMaintenance, 1, 5, &scheduled, &expires, gen,
	)

	require.NoError(t, err)
	require.NotNil(t, a)
	assert.NotNil(t, a.ScheduledAt())
	assert.NotNil(t, a.ExpiresAt())
	assert.True(t, a.ExpiresAt().After(*a.ScheduledAt()))
}

// TestNewAnnouncement_AllValidTypes verifies that each announcement type is accepted.
func TestNewAnnouncement_AllValidTypes(t *testing.T) {
	types := []vo.AnnouncementType{
		vo.AnnouncementTypeSystem,
		vo.AnnouncementTypeMaintenance,
		vo.AnnouncementTypeEvent,
	}

	for _, at := range types {
		t.Run(string(at), func(t *testing.T) {
			gen := mockSIDGenerator()
			a, err := NewAnnouncement("Title", "Content", at, 1, 1, nil, nil, gen)
			require.NoError(t, err)
			require.NotNil(t, a)
			assert.Equal(t, at, a.Type())
		})
	}
}

// TestNewAnnouncement_PriorityBounds verifies that priority 1 and 5 are both accepted.
func TestNewAnnouncement_PriorityBounds(t *testing.T) {
	tests := []struct {
		name     string
		priority int
	}{
		{"MinPriority_1", 1},
		{"MidPriority_3", 3},
		{"MaxPriority_5", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := mockSIDGenerator()
			a, err := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, tt.priority, nil, nil, gen)
			require.NoError(t, err)
			require.NotNil(t, a)
			assert.Equal(t, tt.priority, a.Priority())
		})
	}
}

// TestNewAnnouncement_InitializesEventsSlice verifies that the events slice
// is initialized to empty (not nil).
func TestNewAnnouncement_InitializesEventsSlice(t *testing.T) {
	gen := mockSIDGenerator()
	a, err := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	require.NoError(t, err)
	events := a.GetEvents()
	assert.NotNil(t, events)
	assert.Empty(t, events)
}

// =============================================================================
// NewAnnouncement - Validation Error Tests
// =============================================================================

// TestNewAnnouncement_EmptyTitle verifies that empty title is rejected.
func TestNewAnnouncement_EmptyTitle(t *testing.T) {
	gen := mockSIDGenerator()
	a, err := NewAnnouncement("", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "title is required")
}

// TestNewAnnouncement_TitleExceedsMaxLength verifies that title > 200 chars is rejected.
func TestNewAnnouncement_TitleExceedsMaxLength(t *testing.T) {
	gen := mockSIDGenerator()
	longTitle := strings.Repeat("x", 201)
	a, err := NewAnnouncement(longTitle, "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "title exceeds maximum length of 200 characters")
}

// TestNewAnnouncement_TitleAtMaxLength verifies that title at exactly 200 chars is accepted.
func TestNewAnnouncement_TitleAtMaxLength(t *testing.T) {
	gen := mockSIDGenerator()
	title := strings.Repeat("x", 200)
	a, err := NewAnnouncement(title, "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Len(t, a.Title(), 200)
}

// TestNewAnnouncement_EmptyContent verifies that empty content is rejected.
func TestNewAnnouncement_EmptyContent(t *testing.T) {
	gen := mockSIDGenerator()
	a, err := NewAnnouncement("Title", "", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "content is required")
}

// TestNewAnnouncement_ContentExceedsMaxLength verifies that content > 10000 chars is rejected.
func TestNewAnnouncement_ContentExceedsMaxLength(t *testing.T) {
	gen := mockSIDGenerator()
	longContent := strings.Repeat("y", 10001)
	a, err := NewAnnouncement("Title", longContent, vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "content exceeds maximum length of 10000 characters")
}

// TestNewAnnouncement_ContentAtMaxLength verifies that content at exactly 10000 chars is accepted.
func TestNewAnnouncement_ContentAtMaxLength(t *testing.T) {
	gen := mockSIDGenerator()
	content := strings.Repeat("y", 10000)
	a, err := NewAnnouncement("Title", content, vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Len(t, a.Content(), 10000)
}

// TestNewAnnouncement_InvalidType verifies that an invalid announcement type is rejected.
func TestNewAnnouncement_InvalidType(t *testing.T) {
	gen := mockSIDGenerator()
	a, err := NewAnnouncement("Title", "Content", vo.AnnouncementType("invalid"), 1, 1, nil, nil, gen)

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "invalid announcement type")
}

// TestNewAnnouncement_ZeroCreatorID verifies that zero creatorID is rejected.
func TestNewAnnouncement_ZeroCreatorID(t *testing.T) {
	gen := mockSIDGenerator()
	a, err := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 0, 1, nil, nil, gen)

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "creator ID is required")
}

// TestNewAnnouncement_PriorityOutOfRange verifies that priority outside 1-5 is rejected.
func TestNewAnnouncement_PriorityOutOfRange(t *testing.T) {
	tests := []struct {
		name     string
		priority int
	}{
		{"Zero", 0},
		{"Negative", -1},
		{"TooHigh_6", 6},
		{"TooHigh_100", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := mockSIDGenerator()
			a, err := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, tt.priority, nil, nil, gen)

			assert.Error(t, err)
			assert.Nil(t, a)
			assert.Contains(t, err.Error(), "priority must be between 1 and 5")
		})
	}
}

// TestNewAnnouncement_ExpiresBeforeScheduled verifies that expiresAt before
// scheduledAt is rejected.
func TestNewAnnouncement_ExpiresBeforeScheduled(t *testing.T) {
	gen := mockSIDGenerator()
	scheduled := time.Now().UTC().Add(24 * time.Hour)
	expires := time.Now().UTC().Add(time.Hour) // before scheduled

	a, err := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, &scheduled, &expires, gen)

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "expires at must be after scheduled at")
}

// TestNewAnnouncement_NilSIDGenerator verifies that nil SID generator is rejected.
func TestNewAnnouncement_NilSIDGenerator(t *testing.T) {
	a, err := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "SID generator is required")
}

// TestNewAnnouncement_SIDGeneratorFails verifies that SID generation error is propagated.
func TestNewAnnouncement_SIDGeneratorFails(t *testing.T) {
	gen := failingSIDGenerator()
	a, err := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "failed to generate SID")
}

// =============================================================================
// ReconstructAnnouncement Tests
// =============================================================================

// TestReconstructAnnouncement_ValidInput verifies reconstructing an announcement
// from persisted data.
func TestReconstructAnnouncement_ValidInput(t *testing.T) {
	now := time.Now().UTC()
	scheduled := now.Add(-time.Hour)
	expires := now.Add(24 * time.Hour)

	a, err := ReconstructAnnouncement(
		1, "ann_abc123", "Title", "Content",
		vo.AnnouncementTypeSystem, vo.AnnouncementStatusPublished,
		5, 3, &scheduled, &expires, 42, now, now,
	)

	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Equal(t, uint(1), a.ID())
	assert.Equal(t, "ann_abc123", a.SID())
	assert.Equal(t, vo.AnnouncementStatusPublished, a.Status())
	assert.Equal(t, uint(5), a.CreatorID())
	assert.Equal(t, 42, a.ViewCount())
}

// TestReconstructAnnouncement_ZeroID verifies that zero ID is rejected.
func TestReconstructAnnouncement_ZeroID(t *testing.T) {
	now := time.Now().UTC()
	a, err := ReconstructAnnouncement(
		0, "ann_abc123", "Title", "Content",
		vo.AnnouncementTypeSystem, vo.AnnouncementStatusDraft,
		1, 1, nil, nil, 0, now, now,
	)

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "announcement ID cannot be zero")
}

// TestReconstructAnnouncement_EmptySID verifies that empty SID is rejected.
func TestReconstructAnnouncement_EmptySID(t *testing.T) {
	now := time.Now().UTC()
	a, err := ReconstructAnnouncement(
		1, "", "Title", "Content",
		vo.AnnouncementTypeSystem, vo.AnnouncementStatusDraft,
		1, 1, nil, nil, 0, now, now,
	)

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "announcement SID cannot be empty")
}

// TestReconstructAnnouncement_EmptyTitle verifies that empty title is rejected.
func TestReconstructAnnouncement_EmptyTitle(t *testing.T) {
	now := time.Now().UTC()
	a, err := ReconstructAnnouncement(
		1, "ann_abc123", "", "Content",
		vo.AnnouncementTypeSystem, vo.AnnouncementStatusDraft,
		1, 1, nil, nil, 0, now, now,
	)

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "title is required")
}

// TestReconstructAnnouncement_InvalidType verifies that invalid type is rejected.
func TestReconstructAnnouncement_InvalidType(t *testing.T) {
	now := time.Now().UTC()
	a, err := ReconstructAnnouncement(
		1, "ann_abc123", "Title", "Content",
		vo.AnnouncementType("bogus"), vo.AnnouncementStatusDraft,
		1, 1, nil, nil, 0, now, now,
	)

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "invalid announcement type")
}

// TestReconstructAnnouncement_InvalidStatus verifies that invalid status is rejected.
func TestReconstructAnnouncement_InvalidStatus(t *testing.T) {
	now := time.Now().UTC()
	a, err := ReconstructAnnouncement(
		1, "ann_abc123", "Title", "Content",
		vo.AnnouncementTypeSystem, vo.AnnouncementStatus("bogus"),
		1, 1, nil, nil, 0, now, now,
	)

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "invalid status")
}

// TestReconstructAnnouncement_InvalidPriority verifies that invalid priority is rejected.
func TestReconstructAnnouncement_InvalidPriority(t *testing.T) {
	now := time.Now().UTC()
	a, err := ReconstructAnnouncement(
		1, "ann_abc123", "Title", "Content",
		vo.AnnouncementTypeSystem, vo.AnnouncementStatusDraft,
		1, 0, nil, nil, 0, now, now,
	)

	assert.Error(t, err)
	assert.Nil(t, a)
	assert.Contains(t, err.Error(), "priority must be between 1 and 5")
}

// =============================================================================
// SetID Tests
// =============================================================================

// TestAnnouncement_SetID_Success verifies setting ID on a new announcement.
func TestAnnouncement_SetID_Success(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	err := a.SetID(5)

	require.NoError(t, err)
	assert.Equal(t, uint(5), a.ID())
}

// TestAnnouncement_SetID_AlreadySet verifies that setting ID twice fails.
func TestAnnouncement_SetID_AlreadySet(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)
	_ = a.SetID(5)

	err := a.SetID(10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "announcement ID is already set")
	assert.Equal(t, uint(5), a.ID())
}

// TestAnnouncement_SetID_Zero verifies that setting ID to zero fails.
func TestAnnouncement_SetID_Zero(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	err := a.SetID(0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "announcement ID cannot be zero")
}

// =============================================================================
// Publish Tests
// =============================================================================

// TestAnnouncement_Publish_FromDraft verifies publishing a draft announcement.
func TestAnnouncement_Publish_FromDraft(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)
	assert.Equal(t, vo.AnnouncementStatusDraft, a.Status())

	err := a.Publish()

	require.NoError(t, err)
	assert.Equal(t, vo.AnnouncementStatusPublished, a.Status())
}

// TestAnnouncement_Publish_FromExpired verifies publishing from expired status
// (expired -> published is allowed per transition table).
func TestAnnouncement_Publish_FromExpired(t *testing.T) {
	now := time.Now().UTC()
	a, _ := ReconstructAnnouncement(
		1, "ann_abc123", "Title", "Content",
		vo.AnnouncementTypeSystem, vo.AnnouncementStatusExpired,
		1, 1, nil, nil, 0, now, now,
	)

	err := a.Publish()

	require.NoError(t, err)
	assert.Equal(t, vo.AnnouncementStatusPublished, a.Status())
}

// TestAnnouncement_Publish_FromPublished verifies that publishing a published
// announcement fails (published -> published not in transition table).
func TestAnnouncement_Publish_FromPublished(t *testing.T) {
	now := time.Now().UTC()
	a, _ := ReconstructAnnouncement(
		1, "ann_abc123", "Title", "Content",
		vo.AnnouncementTypeSystem, vo.AnnouncementStatusPublished,
		1, 1, nil, nil, 0, now, now,
	)

	err := a.Publish()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot publish announcement with status")
}

// TestAnnouncement_Publish_FromDeleted verifies that publishing a deleted
// announcement fails (deleted has no allowed transitions).
func TestAnnouncement_Publish_FromDeleted(t *testing.T) {
	now := time.Now().UTC()
	a, _ := ReconstructAnnouncement(
		1, "ann_abc123", "Title", "Content",
		vo.AnnouncementTypeSystem, vo.AnnouncementStatusDeleted,
		1, 1, nil, nil, 0, now, now,
	)

	err := a.Publish()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot publish announcement with status")
}

// TestAnnouncement_Publish_BeforeScheduledTime verifies that publishing before
// scheduled time fails.
func TestAnnouncement_Publish_BeforeScheduledTime(t *testing.T) {
	gen := mockSIDGenerator()
	future := time.Now().UTC().Add(24 * time.Hour)
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, &future, nil, gen)

	err := a.Publish()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot publish before scheduled time")
}

// TestAnnouncement_Publish_AfterExpiryTime verifies that publishing an
// already-expired announcement fails.
func TestAnnouncement_Publish_AfterExpiryTime(t *testing.T) {
	gen := mockSIDGenerator()
	past := time.Now().UTC().Add(-time.Hour)
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, &past, gen)

	err := a.Publish()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot publish expired announcement")
}

// TestAnnouncement_Publish_UpdatesTimestamp verifies that publishing updates updatedAt.
func TestAnnouncement_Publish_UpdatesTimestamp(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)
	beforePublish := a.UpdatedAt()

	time.Sleep(5 * time.Millisecond)
	err := a.Publish()

	require.NoError(t, err)
	assert.True(t, a.UpdatedAt().After(beforePublish))
}

// =============================================================================
// MarkAsExpired Tests
// =============================================================================

// TestAnnouncement_MarkAsExpired_FromPublished verifies marking a published
// announcement as expired.
func TestAnnouncement_MarkAsExpired_FromPublished(t *testing.T) {
	now := time.Now().UTC()
	a, _ := ReconstructAnnouncement(
		1, "ann_abc123", "Title", "Content",
		vo.AnnouncementTypeSystem, vo.AnnouncementStatusPublished,
		1, 1, nil, nil, 0, now, now,
	)

	err := a.MarkAsExpired()

	require.NoError(t, err)
	assert.Equal(t, vo.AnnouncementStatusExpired, a.Status())
}

// TestAnnouncement_MarkAsExpired_Idempotent verifies that marking an already
// expired announcement as expired is idempotent.
func TestAnnouncement_MarkAsExpired_Idempotent(t *testing.T) {
	now := time.Now().UTC()
	a, _ := ReconstructAnnouncement(
		1, "ann_abc123", "Title", "Content",
		vo.AnnouncementTypeSystem, vo.AnnouncementStatusExpired,
		1, 1, nil, nil, 0, now, now,
	)

	err := a.MarkAsExpired()

	require.NoError(t, err)
	assert.Equal(t, vo.AnnouncementStatusExpired, a.Status())
}

// TestAnnouncement_MarkAsExpired_FromDeleted verifies that marking a deleted
// announcement as expired fails.
func TestAnnouncement_MarkAsExpired_FromDeleted(t *testing.T) {
	now := time.Now().UTC()
	a, _ := ReconstructAnnouncement(
		1, "ann_abc123", "Title", "Content",
		vo.AnnouncementTypeSystem, vo.AnnouncementStatusDeleted,
		1, 1, nil, nil, 0, now, now,
	)

	err := a.MarkAsExpired()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot mark announcement with status")
}

// =============================================================================
// Archive Tests
// =============================================================================

// TestAnnouncement_Archive_DraftBecomesDeleted verifies that archiving a draft
// announcement marks it as deleted.
func TestAnnouncement_Archive_DraftBecomesDeleted(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)
	assert.Equal(t, vo.AnnouncementStatusDraft, a.Status())

	err := a.Archive()

	require.NoError(t, err)
	assert.Equal(t, vo.AnnouncementStatusDeleted, a.Status())
}

// TestAnnouncement_Archive_PublishedBecomesExpired verifies that archiving a
// published announcement marks it as expired.
func TestAnnouncement_Archive_PublishedBecomesExpired(t *testing.T) {
	now := time.Now().UTC()
	a, _ := ReconstructAnnouncement(
		1, "ann_abc123", "Title", "Content",
		vo.AnnouncementTypeSystem, vo.AnnouncementStatusPublished,
		1, 1, nil, nil, 0, now, now,
	)

	err := a.Archive()

	require.NoError(t, err)
	assert.Equal(t, vo.AnnouncementStatusExpired, a.Status())
}

// TestAnnouncement_Archive_ExpiredNoOp verifies that archiving an already
// expired announcement is a no-op.
func TestAnnouncement_Archive_ExpiredNoOp(t *testing.T) {
	now := time.Now().UTC()
	a, _ := ReconstructAnnouncement(
		1, "ann_abc123", "Title", "Content",
		vo.AnnouncementTypeSystem, vo.AnnouncementStatusExpired,
		1, 1, nil, nil, 0, now, now,
	)

	err := a.Archive()

	require.NoError(t, err)
	assert.Equal(t, vo.AnnouncementStatusExpired, a.Status())
}

// TestAnnouncement_Archive_DeletedNoOp verifies that archiving an already
// deleted announcement is a no-op.
func TestAnnouncement_Archive_DeletedNoOp(t *testing.T) {
	now := time.Now().UTC()
	a, _ := ReconstructAnnouncement(
		1, "ann_abc123", "Title", "Content",
		vo.AnnouncementTypeSystem, vo.AnnouncementStatusDeleted,
		1, 1, nil, nil, 0, now, now,
	)

	err := a.Archive()

	require.NoError(t, err)
	assert.Equal(t, vo.AnnouncementStatusDeleted, a.Status())
}

// =============================================================================
// Update Tests
// =============================================================================

// TestAnnouncement_Update_ValidInput verifies updating an announcement with
// valid fields.
func TestAnnouncement_Update_ValidInput(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Old Title", "Old Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)
	beforeUpdate := a.UpdatedAt()

	time.Sleep(5 * time.Millisecond)
	newExpires := time.Now().UTC().Add(48 * time.Hour)
	err := a.Update("New Title", "New Content", 4, &newExpires)

	require.NoError(t, err)
	assert.Equal(t, "New Title", a.Title())
	assert.Equal(t, "New Content", a.Content())
	assert.Equal(t, 4, a.Priority())
	assert.NotNil(t, a.ExpiresAt())
	assert.True(t, a.UpdatedAt().After(beforeUpdate))
}

// TestAnnouncement_Update_EmptyTitle verifies that empty title is rejected.
func TestAnnouncement_Update_EmptyTitle(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	err := a.Update("", "Content", 1, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title is required")
}

// TestAnnouncement_Update_TitleTooLong verifies that title > 200 chars is rejected.
func TestAnnouncement_Update_TitleTooLong(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	err := a.Update(strings.Repeat("a", 201), "Content", 1, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title exceeds maximum length of 200 characters")
}

// TestAnnouncement_Update_EmptyContent verifies that empty content is rejected.
func TestAnnouncement_Update_EmptyContent(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	err := a.Update("Title", "", 1, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content is required")
}

// TestAnnouncement_Update_ContentTooLong verifies that content > 10000 chars is rejected.
func TestAnnouncement_Update_ContentTooLong(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	err := a.Update("Title", strings.Repeat("b", 10001), 1, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content exceeds maximum length of 10000 characters")
}

// TestAnnouncement_Update_InvalidPriority verifies that priority outside 1-5 is rejected.
func TestAnnouncement_Update_InvalidPriority(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	err := a.Update("Title", "Content", 0, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "priority must be between 1 and 5")
}

// TestAnnouncement_Update_ClearsExpiresAt verifies that passing nil expiresAt
// clears the field.
func TestAnnouncement_Update_ClearsExpiresAt(t *testing.T) {
	gen := mockSIDGenerator()
	expires := time.Now().UTC().Add(time.Hour)
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, &expires, gen)
	require.NotNil(t, a.ExpiresAt())

	err := a.Update("Title", "Content", 1, nil)

	require.NoError(t, err)
	assert.Nil(t, a.ExpiresAt())
}

// =============================================================================
// ViewCount Tests
// =============================================================================

// TestAnnouncement_IncrementViewCount verifies that view count increments correctly.
func TestAnnouncement_IncrementViewCount(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)
	assert.Equal(t, 0, a.ViewCount())

	a.IncrementViewCount()
	assert.Equal(t, 1, a.ViewCount())

	a.IncrementViewCount()
	a.IncrementViewCount()
	assert.Equal(t, 3, a.ViewCount())
}

// =============================================================================
// IsExpired Tests
// =============================================================================

// TestAnnouncement_IsExpired_NoExpiry verifies that announcement without expiresAt
// is not expired.
func TestAnnouncement_IsExpired_NoExpiry(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	assert.False(t, a.IsExpired())
}

// TestAnnouncement_IsExpired_FutureExpiry verifies that announcement with future
// expiresAt is not expired.
func TestAnnouncement_IsExpired_FutureExpiry(t *testing.T) {
	gen := mockSIDGenerator()
	future := time.Now().UTC().Add(24 * time.Hour)
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, &future, gen)

	assert.False(t, a.IsExpired())
}

// TestAnnouncement_IsExpired_PastExpiry verifies that announcement with past
// expiresAt is expired.
func TestAnnouncement_IsExpired_PastExpiry(t *testing.T) {
	gen := mockSIDGenerator()
	past := time.Now().UTC().Add(-time.Hour)
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, &past, gen)

	assert.True(t, a.IsExpired())
}

// =============================================================================
// Domain Event Tests
// =============================================================================

// TestAnnouncement_GetEvents_ReturnsEmptyAndDrains verifies that GetEvents
// returns a copy and clears internal events.
func TestAnnouncement_GetEvents_ReturnsEmptyAndDrains(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	events := a.GetEvents()
	assert.NotNil(t, events)
	assert.Empty(t, events)

	// Second call should also be empty
	events2 := a.GetEvents()
	assert.Empty(t, events2)
}

// TestAnnouncement_ClearEvents verifies that ClearEvents resets the events slice.
func TestAnnouncement_ClearEvents(t *testing.T) {
	gen := mockSIDGenerator()
	a, _ := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 1, nil, nil, gen)

	a.ClearEvents()
	events := a.GetEvents()
	assert.NotNil(t, events)
	assert.Empty(t, events)
}

// =============================================================================
// Status Transition Table Tests
// =============================================================================

// TestAnnouncement_StatusTransitions_Comprehensive verifies the complete status
// transition matrix.
func TestAnnouncement_StatusTransitions_Comprehensive(t *testing.T) {
	tests := []struct {
		name        string
		from        vo.AnnouncementStatus
		to          vo.AnnouncementStatus
		canTransit  bool
	}{
		// From Draft
		{"Draft_to_Published", vo.AnnouncementStatusDraft, vo.AnnouncementStatusPublished, true},
		{"Draft_to_Deleted", vo.AnnouncementStatusDraft, vo.AnnouncementStatusDeleted, true},
		{"Draft_to_Expired", vo.AnnouncementStatusDraft, vo.AnnouncementStatusExpired, false},

		// From Published
		{"Published_to_Expired", vo.AnnouncementStatusPublished, vo.AnnouncementStatusExpired, true},
		{"Published_to_Deleted", vo.AnnouncementStatusPublished, vo.AnnouncementStatusDeleted, true},
		{"Published_to_Draft", vo.AnnouncementStatusPublished, vo.AnnouncementStatusDraft, false},

		// From Expired
		{"Expired_to_Published", vo.AnnouncementStatusExpired, vo.AnnouncementStatusPublished, true},
		{"Expired_to_Deleted", vo.AnnouncementStatusExpired, vo.AnnouncementStatusDeleted, true},
		{"Expired_to_Draft", vo.AnnouncementStatusExpired, vo.AnnouncementStatusDraft, false},

		// From Deleted (terminal state)
		{"Deleted_to_Draft", vo.AnnouncementStatusDeleted, vo.AnnouncementStatusDraft, false},
		{"Deleted_to_Published", vo.AnnouncementStatusDeleted, vo.AnnouncementStatusPublished, false},
		{"Deleted_to_Expired", vo.AnnouncementStatusDeleted, vo.AnnouncementStatusExpired, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.from.CanTransitionTo(tt.to)
			assert.Equal(t, tt.canTransit, result)
		})
	}
}

// =============================================================================
// Factory Function Tests
// =============================================================================

// TestCreateSystemAnnouncement verifies the convenience factory for system announcements.
func TestCreateSystemAnnouncement(t *testing.T) {
	gen := mockSIDGenerator()
	a, err := CreateSystemAnnouncement("Alert", "System alert content", 1, 3, nil, gen)

	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Equal(t, vo.AnnouncementTypeSystem, a.Type())
	assert.Nil(t, a.ScheduledAt())
}

// TestCreateMaintenanceAnnouncement verifies the convenience factory for maintenance announcements.
func TestCreateMaintenanceAnnouncement(t *testing.T) {
	gen := mockSIDGenerator()
	scheduled := time.Now().UTC().Add(time.Hour)
	expires := time.Now().UTC().Add(24 * time.Hour)

	a, err := CreateMaintenanceAnnouncement("Maintenance", "Planned downtime", 1, 5, &scheduled, &expires, gen)

	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Equal(t, vo.AnnouncementTypeMaintenance, a.Type())
	assert.NotNil(t, a.ScheduledAt())
	assert.NotNil(t, a.ExpiresAt())
}

// TestCreateEventAnnouncement verifies the convenience factory for event announcements.
func TestCreateEventAnnouncement(t *testing.T) {
	gen := mockSIDGenerator()
	scheduled := time.Now().UTC().Add(time.Hour)
	expires := time.Now().UTC().Add(48 * time.Hour)

	a, err := CreateEventAnnouncement("Event", "Special promotion", 1, 2, &scheduled, &expires, gen)

	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Equal(t, vo.AnnouncementTypeEvent, a.Type())
}

// =============================================================================
// Full Lifecycle Test
// =============================================================================

// TestAnnouncement_FullLifecycle verifies the complete lifecycle: create -> publish -> expire.
func TestAnnouncement_FullLifecycle(t *testing.T) {
	gen := mockSIDGenerator()
	a, err := NewAnnouncement("Title", "Content", vo.AnnouncementTypeSystem, 1, 3, nil, nil, gen)
	require.NoError(t, err)
	assert.Equal(t, vo.AnnouncementStatusDraft, a.Status())

	// Step 1: Publish
	err = a.Publish()
	require.NoError(t, err)
	assert.Equal(t, vo.AnnouncementStatusPublished, a.Status())

	// Step 2: Increment views
	a.IncrementViewCount()
	a.IncrementViewCount()
	assert.Equal(t, 2, a.ViewCount())

	// Step 3: Expire
	err = a.MarkAsExpired()
	require.NoError(t, err)
	assert.Equal(t, vo.AnnouncementStatusExpired, a.Status())

	// Step 4: Re-publish from expired
	err = a.Publish()
	require.NoError(t, err)
	assert.Equal(t, vo.AnnouncementStatusPublished, a.Status())

	// Step 5: Archive (published -> expired)
	err = a.Archive()
	require.NoError(t, err)
	assert.Equal(t, vo.AnnouncementStatusExpired, a.Status())
}
