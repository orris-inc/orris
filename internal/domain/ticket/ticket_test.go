package ticket

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	vo "github.com/orris-inc/orris/internal/domain/ticket/valueobjects"
	"github.com/orris-inc/orris/internal/shared/constants"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newValidTicket creates a ticket with sensible defaults for testing.
func newValidTicket(t *testing.T) *Ticket {
	t.Helper()
	tk, err := NewTicket("Test ticket", "Detailed description", vo.CategoryTechnical, vo.PriorityMedium, 1)
	require.NoError(t, err)
	return tk
}

// reconstructedTicket builds a persisted-style ticket via ReconstructTicket.
func reconstructedTicket(t *testing.T, status vo.TicketStatus) *Ticket {
	t.Helper()
	now := time.Now().UTC()
	tk, err := ReconstructTicket(
		1, "TKT-0001",
		"Persisted ticket", "desc",
		vo.CategoryBilling, vo.PriorityHigh,
		status,
		10,   // creatorID
		nil,  // assigneeID
		nil,  // tags
		nil,  // metadata
		nil,  // slaDueTime
		nil,  // responseTime
		nil,  // resolvedTime
		1,    // version
		now, now,
		nil, // closedAt
	)
	require.NoError(t, err)
	return tk
}

// ---------------------------------------------------------------------------
// Constructor Tests
// ---------------------------------------------------------------------------

func TestNewTicket_ValidInput(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		desc     string
		cat      vo.Category
		pri      vo.Priority
		creator  uint
	}{
		{
			name: "all valid fields - technical/low",
			title: "Login page broken", desc: "Cannot log in after update",
			cat: vo.CategoryTechnical, pri: vo.PriorityLow, creator: 1,
		},
		{
			name: "all valid fields - billing/urgent",
			title: "Overcharged", desc: "Billed twice this month",
			cat: vo.CategoryBilling, pri: vo.PriorityUrgent, creator: 42,
		},
		{
			name: "boundary title length 200",
			title: strings.Repeat("a", 200), desc: "desc",
			cat: vo.CategoryOther, pri: vo.PriorityMedium, creator: 5,
		},
		{
			name: "boundary description length 5000",
			title: "Title", desc: strings.Repeat("d", 5000),
			cat: vo.CategoryFeature, pri: vo.PriorityHigh, creator: 7,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tk, err := NewTicket(tc.title, tc.desc, tc.cat, tc.pri, tc.creator)
			require.NoError(t, err)
			require.NotNil(t, tk)

			assert.Equal(t, tc.title, tk.Title())
			assert.Equal(t, tc.desc, tk.Description())
			assert.Equal(t, tc.cat, tk.Category())
			assert.Equal(t, tc.pri, tk.Priority())
			assert.Equal(t, tc.creator, tk.CreatorID())
			assert.Equal(t, vo.StatusNew, tk.Status(), "new ticket must have status 'new'")
			assert.Equal(t, 1, tk.Version())
			assert.NotNil(t, tk.SLADueTime(), "SLA due time should be set")
			assert.Nil(t, tk.AssigneeID())
			assert.Nil(t, tk.ResponseTime())
			assert.Nil(t, tk.ResolvedTime())
			assert.Nil(t, tk.ClosedAt())
			assert.Empty(t, tk.Tags())
			assert.Empty(t, tk.Metadata())
			assert.Empty(t, tk.Comments())
			assert.False(t, tk.CreatedAt().IsZero())
			assert.False(t, tk.UpdatedAt().IsZero())
		})
	}
}

func TestNewTicket_EmptyTitle(t *testing.T) {
	tk, err := NewTicket("", "description", vo.CategoryTechnical, vo.PriorityMedium, 1)
	require.Error(t, err)
	assert.Nil(t, tk)
	assert.Contains(t, err.Error(), "title is required")
}

func TestNewTicket_TitleTooLong(t *testing.T) {
	longTitle := strings.Repeat("x", 201)
	tk, err := NewTicket(longTitle, "description", vo.CategoryTechnical, vo.PriorityMedium, 1)
	require.Error(t, err)
	assert.Nil(t, tk)
	assert.Contains(t, err.Error(), "title exceeds maximum length")
}

func TestNewTicket_EmptyDescription(t *testing.T) {
	tk, err := NewTicket("Title", "", vo.CategoryTechnical, vo.PriorityMedium, 1)
	require.Error(t, err)
	assert.Nil(t, tk)
	assert.Contains(t, err.Error(), "description is required")
}

func TestNewTicket_DescriptionTooLong(t *testing.T) {
	longDesc := strings.Repeat("d", 5001)
	tk, err := NewTicket("Title", longDesc, vo.CategoryTechnical, vo.PriorityMedium, 1)
	require.Error(t, err)
	assert.Nil(t, tk)
	assert.Contains(t, err.Error(), "description exceeds maximum length")
}

func TestNewTicket_InvalidCategory(t *testing.T) {
	tk, err := NewTicket("Title", "desc", vo.Category("invalid"), vo.PriorityMedium, 1)
	require.Error(t, err)
	assert.Nil(t, tk)
	assert.Contains(t, err.Error(), "invalid category")
}

func TestNewTicket_InvalidPriority(t *testing.T) {
	tk, err := NewTicket("Title", "desc", vo.CategoryTechnical, vo.Priority("invalid"), 1)
	require.Error(t, err)
	assert.Nil(t, tk)
	assert.Contains(t, err.Error(), "invalid priority")
}

func TestNewTicket_ZeroCreatorID(t *testing.T) {
	tk, err := NewTicket("Title", "desc", vo.CategoryTechnical, vo.PriorityMedium, 0)
	require.Error(t, err)
	assert.Nil(t, tk)
	assert.Contains(t, err.Error(), "creator ID is required")
}

// ---------------------------------------------------------------------------
// ReconstructTicket Tests
// ---------------------------------------------------------------------------

func TestReconstructTicket_Valid(t *testing.T) {
	now := time.Now().UTC()
	assignee := uint(99)
	sla := now.Add(24 * time.Hour)
	resp := now.Add(1 * time.Hour)
	resolved := now.Add(10 * time.Hour)
	closed := now.Add(12 * time.Hour)

	tk, err := ReconstructTicket(
		1, "TKT-0001",
		"Title", "Description",
		vo.CategoryTechnical, vo.PriorityHigh,
		vo.StatusClosed,
		10,
		&assignee,
		[]string{"tag1", "tag2"},
		map[string]interface{}{"key": "val"},
		&sla,
		&resp,
		&resolved,
		5,
		now, now,
		&closed,
	)
	require.NoError(t, err)
	assert.Equal(t, uint(1), tk.ID())
	assert.Equal(t, "TKT-0001", tk.Number())
	assert.Equal(t, vo.StatusClosed, tk.Status())
	assert.Equal(t, &assignee, tk.AssigneeID())
	assert.Equal(t, []string{"tag1", "tag2"}, tk.Tags())
	assert.Equal(t, "val", tk.Metadata()["key"])
	assert.NotNil(t, tk.ClosedAt())
}

func TestReconstructTicket_ZeroID(t *testing.T) {
	now := time.Now().UTC()
	_, err := ReconstructTicket(0, "TKT-0001", "T", "D", vo.CategoryTechnical, vo.PriorityLow, vo.StatusNew, 1, nil, nil, nil, nil, nil, nil, 1, now, now, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ticket ID cannot be zero")
}

func TestReconstructTicket_EmptyNumber(t *testing.T) {
	now := time.Now().UTC()
	_, err := ReconstructTicket(1, "", "T", "D", vo.CategoryTechnical, vo.PriorityLow, vo.StatusNew, 1, nil, nil, nil, nil, nil, nil, 1, now, now, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ticket number is required")
}

func TestReconstructTicket_NilTagsDefaultsToEmpty(t *testing.T) {
	now := time.Now().UTC()
	tk, err := ReconstructTicket(1, "TKT-001", "T", "D", vo.CategoryTechnical, vo.PriorityLow, vo.StatusNew, 1, nil, nil, nil, nil, nil, nil, 1, now, now, nil)
	require.NoError(t, err)
	assert.NotNil(t, tk.Tags())
	assert.Empty(t, tk.Tags())
}

func TestReconstructTicket_NilMetadataDefaultsToEmpty(t *testing.T) {
	now := time.Now().UTC()
	tk, err := ReconstructTicket(1, "TKT-001", "T", "D", vo.CategoryTechnical, vo.PriorityLow, vo.StatusNew, 1, nil, nil, nil, nil, nil, nil, 1, now, now, nil)
	require.NoError(t, err)
	assert.NotNil(t, tk.Metadata())
	assert.Empty(t, tk.Metadata())
}

// ---------------------------------------------------------------------------
// SetID / SetNumber Tests
// ---------------------------------------------------------------------------

func TestTicket_SetID(t *testing.T) {
	tk := newValidTicket(t)
	assert.Equal(t, uint(0), tk.ID())

	err := tk.SetID(42)
	require.NoError(t, err)
	assert.Equal(t, uint(42), tk.ID())
}

func TestTicket_SetID_AlreadySet(t *testing.T) {
	tk := newValidTicket(t)
	require.NoError(t, tk.SetID(1))

	err := tk.SetID(2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already set")
}

func TestTicket_SetID_Zero(t *testing.T) {
	tk := newValidTicket(t)
	err := tk.SetID(0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be zero")
}

func TestTicket_SetNumber(t *testing.T) {
	tk := newValidTicket(t)
	assert.Equal(t, "", tk.Number())

	err := tk.SetNumber("TKT-0001")
	require.NoError(t, err)
	assert.Equal(t, "TKT-0001", tk.Number())
}

func TestTicket_SetNumber_AlreadySet(t *testing.T) {
	tk := newValidTicket(t)
	require.NoError(t, tk.SetNumber("TKT-0001"))

	err := tk.SetNumber("TKT-0002")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already set")
}

func TestTicket_SetNumber_Empty(t *testing.T) {
	tk := newValidTicket(t)
	err := tk.SetNumber("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

// ---------------------------------------------------------------------------
// Status Transition Tests
// ---------------------------------------------------------------------------

func TestTicket_ChangeStatus_AllValidTransitions(t *testing.T) {
	// The full transition map from the value object.
	transitions := map[vo.TicketStatus][]vo.TicketStatus{
		vo.StatusNew:        {vo.StatusOpen, vo.StatusClosed},
		vo.StatusOpen:       {vo.StatusInProgress, vo.StatusPending, vo.StatusClosed},
		vo.StatusInProgress: {vo.StatusPending, vo.StatusResolved, vo.StatusClosed},
		vo.StatusPending:    {vo.StatusInProgress, vo.StatusResolved, vo.StatusClosed},
		vo.StatusResolved:   {vo.StatusClosed, vo.StatusReopened},
		vo.StatusClosed:     {vo.StatusReopened},
		vo.StatusReopened:   {vo.StatusOpen, vo.StatusInProgress, vo.StatusClosed},
	}

	for from, targets := range transitions {
		for _, to := range targets {
			t.Run(string(from)+"->"+string(to), func(t *testing.T) {
				tk := reconstructedTicket(t, from)
				err := tk.ChangeStatus(to, 1)
				require.NoError(t, err)
				assert.Equal(t, to, tk.Status())
			})
		}
	}
}

func TestTicket_ChangeStatus_SameStatusNoop(t *testing.T) {
	tk := reconstructedTicket(t, vo.StatusOpen)
	vBefore := tk.Version()
	err := tk.ChangeStatus(vo.StatusOpen, 1)
	require.NoError(t, err)
	assert.Equal(t, vBefore, tk.Version(), "version should not change for same-status noop")
}

func TestTicket_ChangeStatus_SetsResolvedTime(t *testing.T) {
	tk := reconstructedTicket(t, vo.StatusInProgress)
	assert.Nil(t, tk.ResolvedTime())

	err := tk.ChangeStatus(vo.StatusResolved, 1)
	require.NoError(t, err)
	assert.NotNil(t, tk.ResolvedTime())
}

func TestTicket_ChangeStatus_SetsClosedAt(t *testing.T) {
	tk := reconstructedTicket(t, vo.StatusInProgress)
	assert.Nil(t, tk.ClosedAt())

	err := tk.ChangeStatus(vo.StatusClosed, 1)
	require.NoError(t, err)
	assert.NotNil(t, tk.ClosedAt())
}

func TestTicket_ChangeStatus_ReopenClearsClosedAndResolved(t *testing.T) {
	// Build a closed ticket with closedAt and resolvedTime set.
	now := time.Now().UTC()
	closed := now
	resolved := now
	tk, err := ReconstructTicket(
		1, "TKT-001", "T", "D",
		vo.CategoryTechnical, vo.PriorityLow,
		vo.StatusResolved,
		1, nil, nil, nil, nil, nil, &resolved, 1, now, now, &closed,
	)
	require.NoError(t, err)

	err = tk.ChangeStatus(vo.StatusReopened, 1)
	require.NoError(t, err)
	assert.Nil(t, tk.ClosedAt(), "closedAt should be cleared on reopen")
	assert.Nil(t, tk.ResolvedTime(), "resolvedTime should be cleared on reopen")
}

func TestTicket_InvalidStatusTransition(t *testing.T) {
	invalidTransitions := []struct {
		from vo.TicketStatus
		to   vo.TicketStatus
	}{
		{vo.StatusNew, vo.StatusInProgress},
		{vo.StatusNew, vo.StatusResolved},
		{vo.StatusNew, vo.StatusReopened},
		{vo.StatusOpen, vo.StatusNew},
		{vo.StatusOpen, vo.StatusResolved},
		{vo.StatusOpen, vo.StatusReopened},
		{vo.StatusInProgress, vo.StatusNew},
		{vo.StatusInProgress, vo.StatusOpen},
		{vo.StatusPending, vo.StatusNew},
		{vo.StatusPending, vo.StatusOpen},
		{vo.StatusResolved, vo.StatusNew},
		{vo.StatusResolved, vo.StatusOpen},
		{vo.StatusResolved, vo.StatusInProgress},
		{vo.StatusClosed, vo.StatusNew},
		{vo.StatusClosed, vo.StatusOpen},
		{vo.StatusClosed, vo.StatusInProgress},
		{vo.StatusClosed, vo.StatusResolved},
	}

	for _, tc := range invalidTransitions {
		t.Run(string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			tk := reconstructedTicket(t, tc.from)
			err := tk.ChangeStatus(tc.to, 1)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "cannot transition")
		})
	}
}

func TestTicket_ChangeStatus_InvalidStatus(t *testing.T) {
	tk := reconstructedTicket(t, vo.StatusNew)
	err := tk.ChangeStatus(vo.TicketStatus("bogus"), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
}

func TestTicket_ChangeStatus_IncrementsVersion(t *testing.T) {
	tk := reconstructedTicket(t, vo.StatusNew)
	v := tk.Version()
	err := tk.ChangeStatus(vo.StatusOpen, 1)
	require.NoError(t, err)
	assert.Equal(t, v+1, tk.Version())
}

// ---------------------------------------------------------------------------
// Close Tests
// ---------------------------------------------------------------------------

func TestTicket_Close(t *testing.T) {
	tk := reconstructedTicket(t, vo.StatusInProgress)
	err := tk.Close("issue resolved", 1)
	require.NoError(t, err)
	assert.Equal(t, vo.StatusClosed, tk.Status())
	assert.NotNil(t, tk.ClosedAt())
}

func TestTicket_Close_EmptyReason(t *testing.T) {
	tk := reconstructedTicket(t, vo.StatusOpen)
	err := tk.Close("", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "close reason is required")
}

func TestTicket_Close_AlreadyClosed(t *testing.T) {
	tk := reconstructedTicket(t, vo.StatusClosed)
	// Already closed is a noop - no error.
	err := tk.Close("duplicate", 1)
	require.NoError(t, err)
	assert.Equal(t, vo.StatusClosed, tk.Status())
}

// NOTE: Every status in the transition map allows a direct transition to
// StatusClosed, so there is no "invalid source status" case for Close().
// The only rejection path is an empty reason (covered by TestTicket_Close_EmptyReason).

// ---------------------------------------------------------------------------
// Reopen Tests
// ---------------------------------------------------------------------------

func TestTicket_Reopen(t *testing.T) {
	tk := reconstructedTicket(t, vo.StatusClosed)
	err := tk.Reopen("needs more investigation", 1)
	require.NoError(t, err)
	assert.Equal(t, vo.StatusReopened, tk.Status())
	assert.Nil(t, tk.ClosedAt(), "closedAt should be cleared")
	assert.Nil(t, tk.ResolvedTime(), "resolvedTime should be cleared")
}

func TestTicket_Reopen_FromResolved(t *testing.T) {
	tk := reconstructedTicket(t, vo.StatusResolved)
	err := tk.Reopen("regression found", 1)
	require.NoError(t, err)
	assert.Equal(t, vo.StatusReopened, tk.Status())
}

func TestTicket_Reopen_EmptyReason(t *testing.T) {
	tk := reconstructedTicket(t, vo.StatusClosed)
	err := tk.Reopen("", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reopen reason is required")
}

func TestTicket_Reopen_NotClosedOrResolved(t *testing.T) {
	for _, status := range []vo.TicketStatus{
		vo.StatusNew, vo.StatusOpen, vo.StatusInProgress, vo.StatusPending, vo.StatusReopened,
	} {
		t.Run(string(status), func(t *testing.T) {
			tk := reconstructedTicket(t, status)
			err := tk.Reopen("reason", 1)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "only closed or resolved tickets can be reopened")
		})
	}
}

// ---------------------------------------------------------------------------
// Assignment Tests
// ---------------------------------------------------------------------------

func TestTicket_AssignTo(t *testing.T) {
	tk := newValidTicket(t)
	assert.Nil(t, tk.AssigneeID())

	err := tk.AssignTo(42, 1)
	require.NoError(t, err)
	require.NotNil(t, tk.AssigneeID())
	assert.Equal(t, uint(42), *tk.AssigneeID())
}

func TestTicket_AssignTo_TransitionsNewToOpen(t *testing.T) {
	tk := newValidTicket(t)
	assert.Equal(t, vo.StatusNew, tk.Status())

	err := tk.AssignTo(10, 1)
	require.NoError(t, err)
	assert.Equal(t, vo.StatusOpen, tk.Status(), "assigning a new ticket should move it to open")
}

func TestTicket_AssignTo_DoesNotChangeNonNewStatus(t *testing.T) {
	tk := reconstructedTicket(t, vo.StatusInProgress)
	err := tk.AssignTo(10, 1)
	require.NoError(t, err)
	assert.Equal(t, vo.StatusInProgress, tk.Status(), "non-new status should not change on assignment")
}

func TestTicket_AssignTo_ZeroAssignee(t *testing.T) {
	tk := newValidTicket(t)
	err := tk.AssignTo(0, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "assignee ID cannot be zero")
}

func TestTicket_Reassign(t *testing.T) {
	tk := newValidTicket(t)
	require.NoError(t, tk.AssignTo(10, 1))

	// Reassign to a different agent.
	err := tk.AssignTo(20, 1)
	require.NoError(t, err)
	require.NotNil(t, tk.AssigneeID())
	assert.Equal(t, uint(20), *tk.AssigneeID())
}

func TestTicket_AssignTo_IncrementsVersion(t *testing.T) {
	tk := newValidTicket(t)
	v := tk.Version()
	require.NoError(t, tk.AssignTo(10, 1))
	assert.Equal(t, v+1, tk.Version())
}

// ---------------------------------------------------------------------------
// AddComment Tests
// ---------------------------------------------------------------------------

func TestTicket_AddComment(t *testing.T) {
	tk := newValidTicket(t)
	require.NoError(t, tk.SetID(1))

	comment, err := NewComment(1, 99, "This is a comment", false)
	require.NoError(t, err)

	err = tk.AddComment(comment)
	require.NoError(t, err)
	assert.Len(t, tk.Comments(), 1)
	assert.Equal(t, "This is a comment", tk.Comments()[0].Content())
}

func TestTicket_AddComment_NilComment(t *testing.T) {
	tk := newValidTicket(t)
	err := tk.AddComment(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "comment cannot be nil")
}

func TestTicket_AddComment_TicketIDMismatch(t *testing.T) {
	tk := newValidTicket(t)
	require.NoError(t, tk.SetID(1))

	// Comment belongs to ticket 999, but we try to add to ticket 1.
	comment, err := NewComment(999, 1, "mismatch", false)
	require.NoError(t, err)

	err = tk.AddComment(comment)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ticket ID mismatch")
}

func TestTicket_AddComment_SetsFirstResponseTime(t *testing.T) {
	tk := newValidTicket(t) // creatorID = 1
	require.NoError(t, tk.SetID(5))
	assert.Nil(t, tk.ResponseTime())

	// Non-internal comment from someone other than the creator.
	comment, err := NewComment(5, 99, "Support reply", false)
	require.NoError(t, err)

	err = tk.AddComment(comment)
	require.NoError(t, err)
	assert.NotNil(t, tk.ResponseTime(), "first response time should be set")
}

func TestTicket_AddComment_InternalCommentDoesNotSetFirstResponse(t *testing.T) {
	tk := newValidTicket(t)
	require.NoError(t, tk.SetID(5))

	// Internal comment from a different user.
	comment, err := NewComment(5, 99, "Internal note", true)
	require.NoError(t, err)

	err = tk.AddComment(comment)
	require.NoError(t, err)
	assert.Nil(t, tk.ResponseTime(), "internal comment should not set first response time")
}

func TestTicket_AddComment_CreatorCommentDoesNotSetFirstResponse(t *testing.T) {
	tk := newValidTicket(t) // creatorID = 1
	require.NoError(t, tk.SetID(5))

	// Public comment from the creator themselves.
	comment, err := NewComment(5, 1, "Creator followup", false)
	require.NoError(t, err)

	err = tk.AddComment(comment)
	require.NoError(t, err)
	assert.Nil(t, tk.ResponseTime(), "creator's own comment should not set first response time")
}

func TestTicket_AddComment_SecondCommentDoesNotOverwriteFirstResponse(t *testing.T) {
	tk := newValidTicket(t) // creatorID = 1
	require.NoError(t, tk.SetID(5))

	c1, err := NewComment(5, 99, "First support reply", false)
	require.NoError(t, err)
	require.NoError(t, tk.AddComment(c1))
	firstResponse := tk.ResponseTime()
	require.NotNil(t, firstResponse)

	c2, err := NewComment(5, 88, "Second support reply", false)
	require.NoError(t, err)
	require.NoError(t, tk.AddComment(c2))
	assert.Equal(t, firstResponse, tk.ResponseTime(), "first response time must not be overwritten")
}

func TestTicket_AddComment_Ordering(t *testing.T) {
	tk := newValidTicket(t)
	require.NoError(t, tk.SetID(10))

	for i := 0; i < 5; i++ {
		c, err := NewComment(10, uint(i+1), "Comment "+strings.Repeat("x", i+1), false)
		require.NoError(t, err)
		require.NoError(t, tk.AddComment(c))
	}

	comments := tk.Comments()
	require.Len(t, comments, 5)
	// Comments should maintain insertion order.
	for i, c := range comments {
		assert.Equal(t, uint(i+1), c.UserID())
	}
}

// ---------------------------------------------------------------------------
// ChangePriority Tests
// ---------------------------------------------------------------------------

func TestTicket_ChangePriority(t *testing.T) {
	tk := newValidTicket(t) // starts with PriorityMedium
	assert.Equal(t, vo.PriorityMedium, tk.Priority())

	err := tk.ChangePriority(vo.PriorityUrgent, 1)
	require.NoError(t, err)
	assert.Equal(t, vo.PriorityUrgent, tk.Priority())
}

func TestTicket_ChangePriority_UpdatesSLADueTime(t *testing.T) {
	tk := newValidTicket(t) // PriorityMedium (24h)
	originalSLA := *tk.SLADueTime()

	err := tk.ChangePriority(vo.PriorityUrgent, 1) // 2h
	require.NoError(t, err)

	newSLA := *tk.SLADueTime()
	assert.True(t, newSLA.Before(originalSLA), "urgent SLA should be earlier than medium SLA")

	// SLA should be createdAt + SLAHours of new priority.
	expectedSLA := tk.CreatedAt().Add(time.Duration(vo.PriorityUrgent.GetSLAHours()) * time.Hour)
	assert.WithinDuration(t, expectedSLA, newSLA, time.Second)
}

func TestTicket_ChangePriority_SameNoop(t *testing.T) {
	tk := newValidTicket(t)
	v := tk.Version()
	err := tk.ChangePriority(vo.PriorityMedium, 1)
	require.NoError(t, err)
	assert.Equal(t, v, tk.Version(), "version should not change for same priority")
}

func TestTicket_ChangePriority_Invalid(t *testing.T) {
	tk := newValidTicket(t)
	err := tk.ChangePriority(vo.Priority("critical"), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid priority")
}

func TestTicket_ChangePriority_IncrementsVersion(t *testing.T) {
	tk := newValidTicket(t)
	v := tk.Version()
	err := tk.ChangePriority(vo.PriorityHigh, 1)
	require.NoError(t, err)
	assert.Equal(t, v+1, tk.Version())
}

// ---------------------------------------------------------------------------
// MarkFirstResponse Tests
// ---------------------------------------------------------------------------

func TestTicket_MarkFirstResponse(t *testing.T) {
	tk := newValidTicket(t)
	assert.Nil(t, tk.ResponseTime())

	err := tk.MarkFirstResponse()
	require.NoError(t, err)
	assert.NotNil(t, tk.ResponseTime())
}

func TestTicket_MarkFirstResponse_AlreadyMarked(t *testing.T) {
	tk := newValidTicket(t)
	require.NoError(t, tk.MarkFirstResponse())

	err := tk.MarkFirstResponse()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "first response already marked")
}

// ---------------------------------------------------------------------------
// MarkResolved Tests
// ---------------------------------------------------------------------------

func TestTicket_MarkResolved(t *testing.T) {
	tk := newValidTicket(t)
	assert.Nil(t, tk.ResolvedTime())

	err := tk.MarkResolved()
	require.NoError(t, err)
	assert.NotNil(t, tk.ResolvedTime())
}

func TestTicket_MarkResolved_AlreadyMarked(t *testing.T) {
	tk := newValidTicket(t)
	require.NoError(t, tk.MarkResolved())

	err := tk.MarkResolved()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ticket already marked as resolved")
}

// ---------------------------------------------------------------------------
// IsOverdue Tests
// ---------------------------------------------------------------------------

func TestTicket_IsOverdue_NotOverdue(t *testing.T) {
	tk := newValidTicket(t)
	// Just created with SLA in the future.
	assert.False(t, tk.IsOverdue())
}

func TestTicket_IsOverdue_PastSLA(t *testing.T) {
	now := time.Now().UTC()
	pastSLA := now.Add(-1 * time.Hour)
	tk, err := ReconstructTicket(
		1, "TKT-001", "T", "D",
		vo.CategoryTechnical, vo.PriorityLow,
		vo.StatusOpen,
		1, nil, nil, nil, &pastSLA, nil, nil, 1, now, now, nil,
	)
	require.NoError(t, err)
	assert.True(t, tk.IsOverdue())
}

func TestTicket_IsOverdue_ClosedTicketNotOverdue(t *testing.T) {
	now := time.Now().UTC()
	pastSLA := now.Add(-1 * time.Hour)
	closed := now
	tk, err := ReconstructTicket(
		1, "TKT-001", "T", "D",
		vo.CategoryTechnical, vo.PriorityLow,
		vo.StatusClosed,
		1, nil, nil, nil, &pastSLA, nil, nil, 1, now, now, &closed,
	)
	require.NoError(t, err)
	assert.False(t, tk.IsOverdue(), "closed tickets should not be overdue")
}

func TestTicket_IsOverdue_ResolvedTicketNotOverdue(t *testing.T) {
	now := time.Now().UTC()
	pastSLA := now.Add(-1 * time.Hour)
	resolved := now
	tk, err := ReconstructTicket(
		1, "TKT-001", "T", "D",
		vo.CategoryTechnical, vo.PriorityLow,
		vo.StatusResolved,
		1, nil, nil, nil, &pastSLA, nil, &resolved, 1, now, now, nil,
	)
	require.NoError(t, err)
	assert.False(t, tk.IsOverdue(), "resolved tickets should not be overdue")
}

func TestTicket_IsOverdue_NilSLADueTime(t *testing.T) {
	now := time.Now().UTC()
	tk, err := ReconstructTicket(
		1, "TKT-001", "T", "D",
		vo.CategoryTechnical, vo.PriorityLow,
		vo.StatusOpen,
		1, nil, nil, nil, nil, nil, nil, 1, now, now, nil,
	)
	require.NoError(t, err)
	assert.False(t, tk.IsOverdue(), "ticket without SLA should not be overdue")
}

// ---------------------------------------------------------------------------
// Priority-based SLA Deadline Tests
// ---------------------------------------------------------------------------

func TestTicket_SLADeadline_ByPriority(t *testing.T) {
	tests := []struct {
		priority    vo.Priority
		expectedH   int
	}{
		{vo.PriorityLow, 72},
		{vo.PriorityMedium, 24},
		{vo.PriorityHigh, 8},
		{vo.PriorityUrgent, 2},
	}

	for _, tc := range tests {
		t.Run(tc.priority.String(), func(t *testing.T) {
			tk, err := NewTicket("SLA test", "desc", vo.CategoryTechnical, tc.priority, 1)
			require.NoError(t, err)
			require.NotNil(t, tk.SLADueTime())

			expectedDue := tk.CreatedAt().Add(time.Duration(tc.expectedH) * time.Hour)
			assert.WithinDuration(t, expectedDue, *tk.SLADueTime(), time.Second,
				"SLA due time should be createdAt + %d hours for priority %s", tc.expectedH, tc.priority)
		})
	}
}

// ---------------------------------------------------------------------------
// CanBeViewedBy Tests
// ---------------------------------------------------------------------------

func TestTicket_CanBeViewedBy(t *testing.T) {
	creatorID := uint(10)
	assigneeID := uint(20)
	now := time.Now().UTC()

	tk, err := ReconstructTicket(
		1, "TKT-001", "T", "D",
		vo.CategoryTechnical, vo.PriorityLow,
		vo.StatusOpen,
		creatorID, &assigneeID, nil, nil, nil, nil, nil, 1, now, now, nil,
	)
	require.NoError(t, err)

	tests := []struct {
		name    string
		userID  uint
		roles   []string
		allowed bool
	}{
		{"admin can view", 999, []string{constants.RoleAdmin}, true},
		{"support agent can view", 999, []string{constants.RoleSupportAgent}, true},
		{"creator can view", creatorID, []string{"user"}, true},
		{"assignee can view", assigneeID, []string{"user"}, true},
		{"random user cannot view", 999, []string{"user"}, false},
		{"random user with no roles cannot view", 999, nil, false},
		{"admin among multiple roles", 999, []string{"user", constants.RoleAdmin}, true},
		{"creator with no roles can view", creatorID, nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tk.CanBeViewedBy(tc.userID, tc.roles)
			assert.Equal(t, tc.allowed, result)
		})
	}
}

func TestTicket_CanBeViewedBy_NoAssignee(t *testing.T) {
	now := time.Now().UTC()
	tk, err := ReconstructTicket(
		1, "TKT-001", "T", "D",
		vo.CategoryTechnical, vo.PriorityLow,
		vo.StatusNew,
		10, nil, nil, nil, nil, nil, nil, 1, now, now, nil,
	)
	require.NoError(t, err)

	// Random user should not see it (no assignee to match).
	assert.False(t, tk.CanBeViewedBy(20, []string{"user"}))
	// Creator should still see it.
	assert.True(t, tk.CanBeViewedBy(10, []string{"user"}))
}

// ---------------------------------------------------------------------------
// Validate Tests
// ---------------------------------------------------------------------------

func TestTicket_Validate_Valid(t *testing.T) {
	tk := newValidTicket(t)
	assert.NoError(t, tk.Validate())
}

// ---------------------------------------------------------------------------
// Getter Immutability Tests
// ---------------------------------------------------------------------------

func TestTicket_Tags_ReturnsCopy(t *testing.T) {
	now := time.Now().UTC()
	tk, err := ReconstructTicket(
		1, "TKT-001", "T", "D",
		vo.CategoryTechnical, vo.PriorityLow,
		vo.StatusNew,
		1, nil, []string{"a", "b"}, nil, nil, nil, nil, 1, now, now, nil,
	)
	require.NoError(t, err)

	tags := tk.Tags()
	tags[0] = "modified"
	assert.Equal(t, "a", tk.Tags()[0], "modifying returned slice should not affect internal state")
}

func TestTicket_Metadata_ReturnsCopy(t *testing.T) {
	now := time.Now().UTC()
	tk, err := ReconstructTicket(
		1, "TKT-001", "T", "D",
		vo.CategoryTechnical, vo.PriorityLow,
		vo.StatusNew,
		1, nil, nil, map[string]interface{}{"key": "val"}, nil, nil, nil, 1, now, now, nil,
	)
	require.NoError(t, err)

	meta := tk.Metadata()
	meta["key"] = "hacked"
	assert.Equal(t, "val", tk.Metadata()["key"], "modifying returned map should not affect internal state")
}

func TestTicket_Comments_ReturnsCopy(t *testing.T) {
	tk := newValidTicket(t)
	require.NoError(t, tk.SetID(1))

	c, err := NewComment(1, 2, "hello", false)
	require.NoError(t, err)
	require.NoError(t, tk.AddComment(c))

	comments := tk.Comments()
	comments[0] = nil
	assert.NotNil(t, tk.Comments()[0], "modifying returned slice should not affect internal state")
}

// ---------------------------------------------------------------------------
// Status Machine Completeness Test
// ---------------------------------------------------------------------------

func TestStatusMachine_Completeness(t *testing.T) {
	// Every valid status should appear as a key in the transition map,
	// i.e. it should have at least one allowed transition.
	allStatuses := []vo.TicketStatus{
		vo.StatusNew,
		vo.StatusOpen,
		vo.StatusInProgress,
		vo.StatusPending,
		vo.StatusResolved,
		vo.StatusClosed,
		vo.StatusReopened,
	}

	for _, s := range allStatuses {
		t.Run(string(s)+"_has_transitions", func(t *testing.T) {
			assert.True(t, s.IsValid(), "status %s should be valid", s)
			// Verify at least one outgoing transition exists.
			hasOutgoing := false
			for _, target := range allStatuses {
				if s.CanTransitionTo(target) {
					hasOutgoing = true
					break
				}
			}
			assert.True(t, hasOutgoing, "status %s should have at least one outgoing transition", s)
		})
	}
}

// ---------------------------------------------------------------------------
// Comment Tests
// ---------------------------------------------------------------------------

func TestNewComment_Valid(t *testing.T) {
	c, err := NewComment(1, 2, "Hello world", false)
	require.NoError(t, err)
	assert.Equal(t, uint(1), c.TicketID())
	assert.Equal(t, uint(2), c.UserID())
	assert.Equal(t, "Hello world", c.Content())
	assert.False(t, c.IsInternal())
	assert.False(t, c.CreatedAt().IsZero())
	assert.False(t, c.UpdatedAt().IsZero())
}

func TestNewComment_Internal(t *testing.T) {
	c, err := NewComment(1, 2, "Internal note", true)
	require.NoError(t, err)
	assert.True(t, c.IsInternal())
}

func TestNewComment_ZeroTicketID(t *testing.T) {
	_, err := NewComment(0, 1, "content", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ticket ID is required")
}

func TestNewComment_ZeroUserID(t *testing.T) {
	_, err := NewComment(1, 0, "content", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user ID is required")
}

func TestNewComment_EmptyContent(t *testing.T) {
	_, err := NewComment(1, 1, "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content cannot be empty")
}

func TestNewComment_ContentTooLong(t *testing.T) {
	_, err := NewComment(1, 1, strings.Repeat("x", 5001), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content exceeds maximum length")
}

func TestComment_SetID(t *testing.T) {
	c, err := NewComment(1, 1, "content", false)
	require.NoError(t, err)
	assert.Equal(t, uint(0), c.ID())

	require.NoError(t, c.SetID(42))
	assert.Equal(t, uint(42), c.ID())
}

func TestComment_SetID_AlreadySet(t *testing.T) {
	c, err := NewComment(1, 1, "content", false)
	require.NoError(t, err)
	require.NoError(t, c.SetID(1))

	err = c.SetID(2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already set")
}

func TestComment_SetID_Zero(t *testing.T) {
	c, err := NewComment(1, 1, "content", false)
	require.NoError(t, err)
	err = c.SetID(0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be zero")
}

func TestComment_UpdateContent(t *testing.T) {
	c, err := NewComment(1, 1, "original", false)
	require.NoError(t, err)

	err = c.UpdateContent("updated content")
	require.NoError(t, err)
	assert.Equal(t, "updated content", c.Content())
}

func TestComment_UpdateContent_Empty(t *testing.T) {
	c, err := NewComment(1, 1, "original", false)
	require.NoError(t, err)

	err = c.UpdateContent("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content cannot be empty")
}

func TestComment_UpdateContent_TooLong(t *testing.T) {
	c, err := NewComment(1, 1, "original", false)
	require.NoError(t, err)

	err = c.UpdateContent(strings.Repeat("x", 5001))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content exceeds maximum length")
}

func TestReconstructComment_Valid(t *testing.T) {
	now := time.Now().UTC()
	c, err := ReconstructComment(1, 2, 3, "content", true, now, now)
	require.NoError(t, err)
	assert.Equal(t, uint(1), c.ID())
	assert.Equal(t, uint(2), c.TicketID())
	assert.Equal(t, uint(3), c.UserID())
	assert.Equal(t, "content", c.Content())
	assert.True(t, c.IsInternal())
}

func TestReconstructComment_ZeroID(t *testing.T) {
	now := time.Now().UTC()
	_, err := ReconstructComment(0, 1, 1, "content", false, now, now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "comment ID cannot be zero")
}

func TestReconstructComment_ZeroTicketID(t *testing.T) {
	now := time.Now().UTC()
	_, err := ReconstructComment(1, 0, 1, "content", false, now, now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ticket ID is required")
}

func TestReconstructComment_ZeroUserID(t *testing.T) {
	now := time.Now().UTC()
	_, err := ReconstructComment(1, 1, 0, "content", false, now, now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user ID is required")
}
