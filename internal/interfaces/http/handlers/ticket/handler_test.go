package ticket

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ticketdto "github.com/orris-inc/orris/internal/application/ticket/dto"
	"github.com/orris-inc/orris/internal/application/ticket/usecases"
	"github.com/orris-inc/orris/internal/interfaces/http/handlers/testutil"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
)

// =====================================================================
// Mock use cases
// =====================================================================

type mockCreateTicketUC struct {
	result *usecases.CreateTicketResult
	err    error
}

func (m *mockCreateTicketUC) Execute(_ context.Context, _ usecases.CreateTicketCommand) (*usecases.CreateTicketResult, error) {
	return m.result, m.err
}

type mockAssignTicketUC struct {
	result *usecases.AssignTicketResult
	err    error
}

func (m *mockAssignTicketUC) Execute(_ context.Context, _ usecases.AssignTicketCommand) (*usecases.AssignTicketResult, error) {
	return m.result, m.err
}

type mockChangeStatusUC struct {
	result *usecases.ChangeStatusResult
	err    error
}

func (m *mockChangeStatusUC) Execute(_ context.Context, _ usecases.ChangeStatusCommand) (*usecases.ChangeStatusResult, error) {
	return m.result, m.err
}

type mockAddCommentUC struct {
	result *usecases.AddCommentResult
	err    error
}

func (m *mockAddCommentUC) Execute(_ context.Context, _ usecases.AddCommentCommand) (*usecases.AddCommentResult, error) {
	return m.result, m.err
}

type mockGetTicketUC struct {
	result *ticketdto.TicketDTO
	err    error
}

func (m *mockGetTicketUC) Execute(_ context.Context, _ usecases.GetTicketQuery) (*ticketdto.TicketDTO, error) {
	return m.result, m.err
}

type mockListTicketsUC struct {
	result *usecases.ListTicketsResult
	err    error
}

func (m *mockListTicketsUC) Execute(_ context.Context, _ usecases.ListTicketsQuery) (*usecases.ListTicketsResult, error) {
	return m.result, m.err
}

type mockDeleteTicketUC struct {
	result *usecases.DeleteTicketResult
	err    error
}

func (m *mockDeleteTicketUC) Execute(_ context.Context, _ usecases.DeleteTicketCommand) (*usecases.DeleteTicketResult, error) {
	return m.result, m.err
}

type mockChangePriorityUC struct {
	result *usecases.ChangePriorityResult
	err    error
}

func (m *mockChangePriorityUC) Execute(_ context.Context, _ usecases.ChangePriorityCommand) (*usecases.ChangePriorityResult, error) {
	return m.result, m.err
}

// =====================================================================
// Test helper
// =====================================================================

type testDeps struct {
	createTicketUC   usecases.CreateTicketExecutor
	assignTicketUC   usecases.AssignTicketExecutor
	updateStatusUC   usecases.UpdateTicketStatusExecutor
	addCommentUC     usecases.AddCommentExecutor
	changeStatusUC   usecases.ChangeStatusExecutor
	getTicketUC      usecases.GetTicketExecutor
	listTicketsUC    usecases.ListTicketsExecutor
	deleteTicketUC   usecases.DeleteTicketExecutor
	updatePriorityUC usecases.UpdateTicketPriorityExecutor
}

func newTestTicketHandler(deps testDeps) *TicketHandler {
	return NewTicketHandler(
		deps.createTicketUC,
		deps.assignTicketUC,
		deps.updateStatusUC,
		deps.addCommentUC,
		deps.changeStatusUC,
		deps.getTicketUC,
		deps.listTicketsUC,
		deps.deleteTicketUC,
		deps.updatePriorityUC,
		testutil.NewMockLogger(),
	)
}

// =====================================================================
// TestTicketHandler_CreateTicket
// =====================================================================

func TestTicketHandler_CreateTicket_Success(t *testing.T) {
	now := time.Now().UTC()
	mockUC := &mockCreateTicketUC{
		result: &usecases.CreateTicketResult{
			TicketID:  1,
			Number:    "TKT-00001",
			Status:    "open",
			CreatedAt: now,
		},
	}
	handler := newTestTicketHandler(testDeps{createTicketUC: mockUC})

	reqBody := CreateTicketRequest{
		Title:       "Test ticket",
		Description: "Something went wrong",
		Category:    "bug",
		Priority:    "high",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/tickets", reqBody)
	testutil.SetAuthContext(c, 1)

	handler.CreateTicket(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestTicketHandler_CreateTicket_BindError(t *testing.T) {
	handler := newTestTicketHandler(testDeps{})

	// Missing required fields
	reqBody := map[string]string{"title": "only title"}
	c, w := testutil.NewTestContext(http.MethodPost, "/tickets", reqBody)
	testutil.SetAuthContext(c, 1)

	handler.CreateTicket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTicketHandler_CreateTicket_NotAuthenticated(t *testing.T) {
	handler := newTestTicketHandler(testDeps{})

	reqBody := CreateTicketRequest{
		Title:       "Test ticket",
		Description: "Something went wrong",
		Category:    "bug",
		Priority:    "high",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/tickets", reqBody)
	// No auth context set

	handler.CreateTicket(c)

	assert.NotEqual(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTicketHandler_CreateTicket_UseCaseError(t *testing.T) {
	mockUC := &mockCreateTicketUC{
		err: errors.NewValidationError("invalid category"),
	}
	handler := newTestTicketHandler(testDeps{createTicketUC: mockUC})

	reqBody := CreateTicketRequest{
		Title:       "Test ticket",
		Description: "Something went wrong",
		Category:    "invalid_cat",
		Priority:    "high",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/tickets", reqBody)
	testutil.SetAuthContext(c, 1)

	handler.CreateTicket(c)

	assert.NotEqual(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestTicketHandler_GetTicket
// =====================================================================

func TestTicketHandler_GetTicket_Success(t *testing.T) {
	now := time.Now().UTC()
	mockUC := &mockGetTicketUC{
		result: &ticketdto.TicketDTO{
			ID:          1,
			Number:      "TKT-00001",
			Title:       "Test ticket",
			Description: "Something went wrong",
			Category:    "bug",
			Priority:    "high",
			Status:      "open",
			CreatorID:   1,
			Tags:        []string{},
			CreatedAt:   now,
			UpdatedAt:   now,
			Comments:    []ticketdto.CommentDTO{},
		},
	}
	handler := newTestTicketHandler(testDeps{getTicketUC: mockUC})

	c, w := testutil.NewTestContext(http.MethodGet, "/tickets/1", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "1")

	handler.GetTicket(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestTicketHandler_GetTicket_InvalidID(t *testing.T) {
	handler := newTestTicketHandler(testDeps{})

	c, w := testutil.NewTestContext(http.MethodGet, "/tickets/abc", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "abc")

	handler.GetTicket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTicketHandler_GetTicket_ZeroID(t *testing.T) {
	handler := newTestTicketHandler(testDeps{})

	c, w := testutil.NewTestContext(http.MethodGet, "/tickets/0", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "0")

	handler.GetTicket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTicketHandler_GetTicket_NotFound(t *testing.T) {
	mockUC := &mockGetTicketUC{
		err: errors.NewNotFoundError("ticket not found"),
	}
	handler := newTestTicketHandler(testDeps{getTicketUC: mockUC})

	c, w := testutil.NewTestContext(http.MethodGet, "/tickets/999", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "999")

	handler.GetTicket(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestTicketHandler_ListTickets
// =====================================================================

func TestTicketHandler_ListTickets_Success(t *testing.T) {
	mockUC := &mockListTicketsUC{
		result: &usecases.ListTicketsResult{
			Tickets: []ticketdto.TicketListItemDTO{
				{
					ID:       1,
					Number:   "TKT-00001",
					Title:    "First ticket",
					Status:   "open",
					Priority: "high",
					Category: "bug",
				},
			},
			TotalCount: 1,
			Page:       1,
			PageSize:   20,
		},
	}
	handler := newTestTicketHandler(testDeps{listTicketsUC: mockUC})

	c, w := testutil.NewTestContext(http.MethodGet, "/tickets", nil)
	testutil.SetAuthContext(c, 1)

	handler.ListTickets(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestTicketHandler_ListTickets_WithFilters(t *testing.T) {
	mockUC := &mockListTicketsUC{
		result: &usecases.ListTicketsResult{
			Tickets:    []ticketdto.TicketListItemDTO{},
			TotalCount: 0,
			Page:       1,
			PageSize:   20,
		},
	}
	handler := newTestTicketHandler(testDeps{listTicketsUC: mockUC})

	c, w := testutil.NewTestContext(http.MethodGet, "/tickets", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetQueryParams(c, map[string]string{
		"status":   "open",
		"priority": "high",
		"page":     "1",
	})

	handler.ListTickets(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestTicketHandler_ListTickets_InvalidAssigneeID(t *testing.T) {
	handler := newTestTicketHandler(testDeps{})

	c, w := testutil.NewTestContext(http.MethodGet, "/tickets", nil)
	testutil.SetAuthContext(c, 1)
	testutil.SetQueryParams(c, map[string]string{
		"assignee_id": "not_a_number",
	})

	handler.ListTickets(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTicketHandler_ListTickets_UseCaseError(t *testing.T) {
	mockUC := &mockListTicketsUC{
		err: errors.NewInternalError("database error"),
	}
	handler := newTestTicketHandler(testDeps{listTicketsUC: mockUC})

	c, w := testutil.NewTestContext(http.MethodGet, "/tickets", nil)
	testutil.SetAuthContext(c, 1)

	handler.ListTickets(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestTicketHandler_AddComment
// =====================================================================

func TestTicketHandler_AddComment_Success(t *testing.T) {
	now := time.Now().UTC()
	mockUC := &mockAddCommentUC{
		result: &usecases.AddCommentResult{
			CommentID: 10,
			CreatedAt: now,
		},
	}
	handler := newTestTicketHandler(testDeps{addCommentUC: mockUC})

	reqBody := AddCommentRequest{
		Content:    "This is a comment",
		IsInternal: false,
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/tickets/1/comments", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "1")
	c.Set(constants.ContextKeyUserRole, "user")

	handler.AddComment(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestTicketHandler_AddComment_BindError(t *testing.T) {
	handler := newTestTicketHandler(testDeps{})

	// Missing required "content" field
	reqBody := map[string]interface{}{"is_internal": true}
	c, w := testutil.NewTestContext(http.MethodPost, "/tickets/1/comments", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "1")
	c.Set(constants.ContextKeyUserRole, "user")

	handler.AddComment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTicketHandler_AddComment_InvalidTicketID(t *testing.T) {
	handler := newTestTicketHandler(testDeps{})

	reqBody := AddCommentRequest{
		Content: "This is a comment",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/tickets/abc/comments", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "abc")

	handler.AddComment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTicketHandler_AddComment_UseCaseError(t *testing.T) {
	mockUC := &mockAddCommentUC{
		err: errors.NewNotFoundError("ticket not found"),
	}
	handler := newTestTicketHandler(testDeps{addCommentUC: mockUC})

	reqBody := AddCommentRequest{
		Content: "This is a comment",
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/tickets/1/comments", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "1")
	c.Set(constants.ContextKeyUserRole, "user")

	handler.AddComment(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestTicketHandler_UpdateTicketStatus
// =====================================================================

func TestTicketHandler_UpdateTicketStatus_Success(t *testing.T) {
	mockUC := &mockChangeStatusUC{
		result: &usecases.ChangeStatusResult{
			TicketID:  1,
			OldStatus: "open",
			NewStatus: "in_progress",
			UpdatedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		},
	}
	handler := newTestTicketHandler(testDeps{changeStatusUC: mockUC})

	reqBody := UpdateTicketStatusRequest{
		Status: "in_progress",
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/tickets/1/status", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "1")
	c.Set(constants.ContextKeyUserRole, "admin")

	handler.UpdateTicketStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestTicketHandler_UpdateTicketStatus_BindError(t *testing.T) {
	handler := newTestTicketHandler(testDeps{})

	// Missing "status" field
	reqBody := map[string]string{}
	c, w := testutil.NewTestContext(http.MethodPatch, "/tickets/1/status", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "1")
	c.Set(constants.ContextKeyUserRole, "admin")

	handler.UpdateTicketStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTicketHandler_UpdateTicketStatus_InvalidStatus(t *testing.T) {
	handler := newTestTicketHandler(testDeps{})

	// "invalid_status" is not in the binding oneof
	reqBody := map[string]string{"status": "invalid_status"}
	c, w := testutil.NewTestContext(http.MethodPatch, "/tickets/1/status", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "1")
	c.Set(constants.ContextKeyUserRole, "admin")

	handler.UpdateTicketStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTicketHandler_UpdateTicketStatus_InvalidTicketID(t *testing.T) {
	handler := newTestTicketHandler(testDeps{})

	reqBody := UpdateTicketStatusRequest{
		Status: "resolved",
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/tickets/abc/status", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "abc")
	c.Set(constants.ContextKeyUserRole, "admin")

	handler.UpdateTicketStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTicketHandler_UpdateTicketStatus_UseCaseError(t *testing.T) {
	mockUC := &mockChangeStatusUC{
		err: errors.NewValidationError("cannot transition from closed to in_progress"),
	}
	handler := newTestTicketHandler(testDeps{changeStatusUC: mockUC})

	reqBody := UpdateTicketStatusRequest{
		Status: "in_progress",
	}
	c, w := testutil.NewTestContext(http.MethodPatch, "/tickets/1/status", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "1")
	c.Set(constants.ContextKeyUserRole, "admin")

	handler.UpdateTicketStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTicketHandler_UpdateTicketStatus_AllValidStatuses(t *testing.T) {
	validStatuses := []string{
		"open",
		"in_progress",
		"resolved",
		"closed",
		"reopened",
	}

	for _, status := range validStatuses {
		t.Run(status, func(t *testing.T) {
			mockUC := &mockChangeStatusUC{
				result: &usecases.ChangeStatusResult{
					TicketID:  1,
					OldStatus: "open",
					NewStatus: status,
					UpdatedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
				},
			}
			handler := newTestTicketHandler(testDeps{changeStatusUC: mockUC})

			reqBody := UpdateTicketStatusRequest{Status: status}
			c, w := testutil.NewTestContext(http.MethodPatch, "/tickets/1/status", reqBody)
			testutil.SetAuthContext(c, 1)
			testutil.SetURLParam(c, "id", "1")
			c.Set(constants.ContextKeyUserRole, "admin")

			handler.UpdateTicketStatus(c)

			assert.Equal(t, http.StatusOK, w.Code, "status %q should succeed", status)
		})
	}
}

// =====================================================================
// TestTicketHandler_AssignTicket
// =====================================================================

func TestTicketHandler_AssignTicket_Success(t *testing.T) {
	mockUC := &mockAssignTicketUC{
		result: &usecases.AssignTicketResult{
			TicketID:   1,
			AssigneeID: 2,
			Status:     "in_progress",
			UpdatedAt:  time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		},
	}
	handler := newTestTicketHandler(testDeps{assignTicketUC: mockUC})

	reqBody := AssignTicketRequest{
		AssigneeID: 2,
	}
	c, w := testutil.NewTestContext(http.MethodPost, "/tickets/1/assign", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "1")

	handler.AssignTicket(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestTicketHandler_AssignTicket_BindError(t *testing.T) {
	handler := newTestTicketHandler(testDeps{})

	// Missing required "assignee_id"
	reqBody := map[string]interface{}{}
	c, w := testutil.NewTestContext(http.MethodPost, "/tickets/1/assign", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "1")

	handler.AssignTicket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTicketHandler_AssignTicket_InvalidTicketID(t *testing.T) {
	handler := newTestTicketHandler(testDeps{})

	reqBody := AssignTicketRequest{AssigneeID: 2}
	c, w := testutil.NewTestContext(http.MethodPost, "/tickets/xyz/assign", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "xyz")

	handler.AssignTicket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTicketHandler_AssignTicket_UseCaseError(t *testing.T) {
	mockUC := &mockAssignTicketUC{
		err: errors.NewNotFoundError("assignee not found"),
	}
	handler := newTestTicketHandler(testDeps{assignTicketUC: mockUC})

	reqBody := AssignTicketRequest{AssigneeID: 999}
	c, w := testutil.NewTestContext(http.MethodPost, "/tickets/1/assign", reqBody)
	testutil.SetAuthContext(c, 1)
	testutil.SetURLParam(c, "id", "1")

	handler.AssignTicket(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// =====================================================================
// TestTicketHandler_DeleteTicket
// =====================================================================

func TestTicketHandler_DeleteTicket_Success(t *testing.T) {
	mockUC := &mockDeleteTicketUC{
		result: &usecases.DeleteTicketResult{TicketID: 1},
	}
	handler := newTestTicketHandler(testDeps{deleteTicketUC: mockUC})

	c, _ := testutil.NewTestContext(http.MethodDelete, "/tickets/1", nil)
	testutil.SetURLParam(c, "id", "1")

	handler.DeleteTicket(c)

	// gin's c.Status() sets the status on the writer; use Writer.Status() for reliable check.
	assert.Equal(t, http.StatusNoContent, c.Writer.Status())
}

func TestTicketHandler_DeleteTicket_InvalidID(t *testing.T) {
	handler := newTestTicketHandler(testDeps{})

	c, w := testutil.NewTestContext(http.MethodDelete, "/tickets/abc", nil)
	testutil.SetURLParam(c, "id", "abc")

	handler.DeleteTicket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestTicketHandler_DeleteTicket_UseCaseError(t *testing.T) {
	mockUC := &mockDeleteTicketUC{
		err: errors.NewInternalError("failed to delete ticket"),
	}
	handler := newTestTicketHandler(testDeps{deleteTicketUC: mockUC})

	c, w := testutil.NewTestContext(http.MethodDelete, "/tickets/1", nil)
	testutil.SetURLParam(c, "id", "1")

	handler.DeleteTicket(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp testutil.APIResponse
	err := testutil.ParseResponse(w, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}
