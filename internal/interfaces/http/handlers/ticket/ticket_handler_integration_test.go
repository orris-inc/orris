package ticket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"orris/internal/application/ticket/usecases"
)

type mockCreateTicketUC struct {
	executeFunc func(ctx context.Context, cmd usecases.CreateTicketCommand) (*usecases.CreateTicketResult, error)
}

func (m *mockCreateTicketUC) Execute(ctx context.Context, cmd usecases.CreateTicketCommand) (*usecases.CreateTicketResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return &usecases.CreateTicketResult{
		TicketID:  1,
		Number:    "TK-001",
		Status:    "new",
		CreatedAt: time.Now(),
	}, nil
}

type mockAssignTicketUC struct {
	executeFunc func(ctx context.Context, cmd usecases.AssignTicketCommand) (*usecases.AssignTicketResult, error)
}

func (m *mockAssignTicketUC) Execute(ctx context.Context, cmd usecases.AssignTicketCommand) (*usecases.AssignTicketResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return &usecases.AssignTicketResult{
		TicketID:   cmd.TicketID,
		AssigneeID: cmd.AssigneeID,
		Status:     "open",
	}, nil
}

type mockUpdateTicketStatusUC struct {
	executeFunc func(ctx context.Context, cmd usecases.ChangeStatusCommand) (*usecases.ChangeStatusResult, error)
}

func (m *mockUpdateTicketStatusUC) Execute(ctx context.Context, cmd usecases.ChangeStatusCommand) (*usecases.ChangeStatusResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return &usecases.ChangeStatusResult{
		TicketID:  cmd.TicketID,
		NewStatus: cmd.NewStatus.String(),
		OldStatus: "open",
		UpdatedAt: time.Now().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

type mockAddCommentUC struct {
	executeFunc func(ctx context.Context, cmd usecases.AddCommentCommand) (*usecases.AddCommentResult, error)
}

func (m *mockAddCommentUC) Execute(ctx context.Context, cmd usecases.AddCommentCommand) (*usecases.AddCommentResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return &usecases.AddCommentResult{
		CommentID: 1,
		CreatedAt: time.Now(),
	}, nil
}

type mockCloseTicketUC struct {
	executeFunc func(ctx context.Context, cmd usecases.CloseTicketCommand) (*usecases.CloseTicketResult, error)
}

func (m *mockCloseTicketUC) Execute(ctx context.Context, cmd usecases.CloseTicketCommand) (*usecases.CloseTicketResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return &usecases.CloseTicketResult{
		TicketID: cmd.TicketID,
		Status:   "closed",
		Reason:   cmd.Reason,
		ClosedAt: time.Now().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

type mockReopenTicketUC struct {
	executeFunc func(ctx context.Context, cmd usecases.ReopenTicketCommand) (*usecases.ReopenTicketResult, error)
}

func (m *mockReopenTicketUC) Execute(ctx context.Context, cmd usecases.ReopenTicketCommand) (*usecases.ReopenTicketResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return &usecases.ReopenTicketResult{
		TicketID:   cmd.TicketID,
		Status:     "reopened",
		Reason:     cmd.Reason,
		ReopenedAt: time.Now().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

type mockGetTicketUC struct {
	executeFunc func(ctx context.Context, query usecases.GetTicketQuery) (*usecases.GetTicketResult, error)
}

func (m *mockGetTicketUC) Execute(ctx context.Context, query usecases.GetTicketQuery) (*usecases.GetTicketResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, query)
	}
	return &usecases.GetTicketResult{
		ID:          query.TicketID,
		Number:      "TK-001",
		Title:       "Test Ticket",
		Description: "Test Description",
		Category:    "technical",
		Priority:    "high",
		Status:      "new",
		CreatorID:   1,
		Tags:        []string{},
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Comments:    []usecases.CommentDTO{},
	}, nil
}

type mockListTicketsUC struct {
	executeFunc func(ctx context.Context, query usecases.ListTicketsQuery) (*usecases.ListTicketsResult, error)
}

func (m *mockListTicketsUC) Execute(ctx context.Context, query usecases.ListTicketsQuery) (*usecases.ListTicketsResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, query)
	}
	now := time.Now().Format("2006-01-02T15:04:05Z07:00")
	return &usecases.ListTicketsResult{
		Tickets: []usecases.TicketListItem{
			{
				ID:         1,
				Number:     "TK-001",
				Title:      "Test Ticket 1",
				Category:   "technical",
				Priority:   "high",
				Status:     "new",
				CreatorID:  1,
				IsOverdue:  false,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
			{
				ID:         2,
				Number:     "TK-002",
				Title:      "Test Ticket 2",
				Category:   "billing",
				Priority:   "medium",
				Status:     "open",
				CreatorID:  1,
				IsOverdue:  false,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
		TotalCount: 2,
		Page:       query.Page,
		PageSize:   query.PageSize,
	}, nil
}

type mockDeleteTicketUC struct {
	executeFunc func(ctx context.Context, cmd usecases.DeleteTicketCommand) (*usecases.DeleteTicketResult, error)
}

func (m *mockDeleteTicketUC) Execute(ctx context.Context, cmd usecases.DeleteTicketCommand) (*usecases.DeleteTicketResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return &usecases.DeleteTicketResult{}, nil
}

type mockUpdatePriorityUC struct {
	executeFunc func(ctx context.Context, cmd usecases.ChangePriorityCommand) (*usecases.ChangePriorityResult, error)
}

func (m *mockUpdatePriorityUC) Execute(ctx context.Context, cmd usecases.ChangePriorityCommand) (*usecases.ChangePriorityResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return &usecases.ChangePriorityResult{
		TicketID:   cmd.TicketID,
		Priority:   cmd.NewPriority,
		UpdatedAt:  time.Now().Format("2006-01-02T15:04:05Z07:00"),
		SLADueTime: time.Now().Add(24 * time.Hour).Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func setupTestRouter() (*gin.Engine, *TicketHandler) {
	gin.SetMode(gin.TestMode)

	createUC := &mockCreateTicketUC{}
	assignUC := &mockAssignTicketUC{}
	updateStatusUC := &mockUpdateTicketStatusUC{}
	addCommentUC := &mockAddCommentUC{}
	closeUC := &mockCloseTicketUC{}
	reopenUC := &mockReopenTicketUC{}
	getUC := &mockGetTicketUC{}
	listUC := &mockListTicketsUC{}
	deleteUC := &mockDeleteTicketUC{}
	updatePriorityUC := &mockUpdatePriorityUC{}

	handler := NewTicketHandler(
		createUC,
		assignUC,
		updateStatusUC,
		addCommentUC,
		closeUC,
		reopenUC,
		getUC,
		listUC,
		deleteUC,
		updatePriorityUC,
	)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	})

	router.POST("/tickets", handler.CreateTicket)
	router.GET("/tickets", handler.ListTickets)
	router.GET("/tickets/:id", handler.GetTicket)
	router.DELETE("/tickets/:id", handler.DeleteTicket)
	router.POST("/tickets/:id/assign", handler.AssignTicket)
	router.POST("/tickets/:id/comments", handler.AddComment)
	router.POST("/tickets/:id/close", handler.CloseTicket)

	return router, handler
}

func TestCreateTicket_Success(t *testing.T) {
	router, _ := setupTestRouter()

	reqBody := CreateTicketRequest{
		Title:       "New Support Ticket",
		Description: "Need help with billing issue",
		Category:    "billing",
		Priority:    "high",
		Tags:        []string{"urgent", "billing"},
		Metadata:    map[string]interface{}{"source": "web"},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/tickets", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Ticket created successfully", response["message"])
	assert.NotNil(t, response["data"])
}

func TestCreateTicket_ValidationError(t *testing.T) {
	router, _ := setupTestRouter()

	tests := []struct {
		name           string
		reqBody        interface{}
		expectedStatus int
	}{
		{
			name: "missing title",
			reqBody: map[string]interface{}{
				"description": "Test description",
				"category":    "technical",
				"priority":    "high",
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "missing description",
			reqBody: map[string]interface{}{
				"title":    "Test Title",
				"category": "technical",
				"priority": "high",
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "missing category",
			reqBody: map[string]interface{}{
				"title":       "Test Title",
				"description": "Test description",
				"priority":    "high",
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "missing priority",
			reqBody: map[string]interface{}{
				"title":       "Test Title",
				"description": "Test description",
				"category":    "technical",
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "invalid json",
			reqBody:        "invalid-json",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if str, ok := tt.reqBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.reqBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/tickets", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response["success"].(bool))
		})
	}
}

func TestGetTicket_Success(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/tickets/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "TK-001", data["number"])
	assert.Equal(t, "Test Ticket", data["title"])
}

func TestGetTicket_InvalidID(t *testing.T) {
	router, _ := setupTestRouter()

	tests := []struct {
		name     string
		ticketID string
		wantCode int
	}{
		{
			name:     "non-numeric id",
			ticketID: "invalid",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "zero id",
			ticketID: "0",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "negative id",
			ticketID: "-1",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tickets/%s", tt.ticketID), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response["success"].(bool))
		})
	}
}

func TestListTickets_Success(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/tickets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["items"])
	assert.Equal(t, float64(2), data["total"])
	assert.Equal(t, float64(1), data["page"])
}

func TestListTickets_WithPagination(t *testing.T) {
	router, _ := setupTestRouter()

	tests := []struct {
		name         string
		queryParams  string
		expectedPage int
		expectedSize int
	}{
		{
			name:         "default pagination",
			queryParams:  "",
			expectedPage: 1,
			expectedSize: 20,
		},
		{
			name:         "custom page and size",
			queryParams:  "?page=2&page_size=10",
			expectedPage: 2,
			expectedSize: 10,
		},
		{
			name:         "invalid page defaults to 1",
			queryParams:  "?page=-1",
			expectedPage: 1,
			expectedSize: 20,
		},
		{
			name:         "page_size too large defaults to 20",
			queryParams:  "?page_size=200",
			expectedPage: 1,
			expectedSize: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/tickets"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.True(t, response["success"].(bool))

			data := response["data"].(map[string]interface{})
			assert.Equal(t, float64(tt.expectedPage), data["page"])
			assert.Equal(t, float64(tt.expectedSize), data["page_size"])
		})
	}
}

func TestListTickets_WithFilters(t *testing.T) {
	router, _ := setupTestRouter()

	tests := []struct {
		name        string
		queryParams string
		wantCode    int
	}{
		{
			name:        "filter by status",
			queryParams: "?status=open",
			wantCode:    http.StatusOK,
		},
		{
			name:        "filter by priority",
			queryParams: "?priority=high",
			wantCode:    http.StatusOK,
		},
		{
			name:        "filter by category",
			queryParams: "?category=technical",
			wantCode:    http.StatusOK,
		},
		{
			name:        "multiple filters",
			queryParams: "?status=open&priority=high&category=technical",
			wantCode:    http.StatusOK,
		},
		{
			name:        "filter by assignee_id",
			queryParams: "?assignee_id=5",
			wantCode:    http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/tickets"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.True(t, response["success"].(bool))
		})
	}
}

func TestAssignTicket_Success(t *testing.T) {
	router, _ := setupTestRouter()

	reqBody := AssignTicketRequest{
		AssigneeID: 5,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/tickets/1/assign", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Ticket assigned successfully", response["message"])
}

func TestAssignTicket_InvalidRequest(t *testing.T) {
	router, _ := setupTestRouter()

	tests := []struct {
		name           string
		ticketID       string
		reqBody        interface{}
		expectedStatus int
	}{
		{
			name:           "invalid ticket id",
			ticketID:       "invalid",
			reqBody:        AssignTicketRequest{AssigneeID: 5},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing assignee_id",
			ticketID:       "1",
			reqBody:        map[string]interface{}{},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "invalid json",
			ticketID:       "1",
			reqBody:        "invalid-json",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if str, ok := tt.reqBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.reqBody)
			}

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/tickets/%s/assign", tt.ticketID), bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestAddComment_Success(t *testing.T) {
	router, _ := setupTestRouter()

	reqBody := AddCommentRequest{
		Content:    "This is a test comment",
		IsInternal: false,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/tickets/1/comments", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Comment added successfully", response["message"])
}

func TestAddComment_ValidationError(t *testing.T) {
	router, _ := setupTestRouter()

	tests := []struct {
		name           string
		ticketID       string
		reqBody        interface{}
		expectedStatus int
	}{
		{
			name:           "invalid ticket id",
			ticketID:       "invalid",
			reqBody:        AddCommentRequest{Content: "Test comment"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing content",
			ticketID:       "1",
			reqBody:        map[string]interface{}{"is_internal": false},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "invalid json",
			ticketID:       "1",
			reqBody:        "invalid-json",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if str, ok := tt.reqBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.reqBody)
			}

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/tickets/%s/comments", tt.ticketID), bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCloseTicket_Success(t *testing.T) {
	router, _ := setupTestRouter()

	reqBody := CloseTicketRequest{
		Reason: "Issue resolved",
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/tickets/1/close", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Ticket closed successfully", response["message"])
}

func TestCloseTicket_ValidationError(t *testing.T) {
	router, _ := setupTestRouter()

	tests := []struct {
		name           string
		ticketID       string
		reqBody        interface{}
		expectedStatus int
	}{
		{
			name:           "invalid ticket id",
			ticketID:       "invalid",
			reqBody:        CloseTicketRequest{Reason: "Resolved"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing reason",
			ticketID:       "1",
			reqBody:        map[string]interface{}{},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if str, ok := tt.reqBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.reqBody)
			}

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/tickets/%s/close", tt.ticketID), bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestDeleteTicket_Success(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodDelete, "/tickets/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestDeleteTicket_InvalidID(t *testing.T) {
	router, _ := setupTestRouter()

	tests := []struct {
		name     string
		ticketID string
		wantCode int
	}{
		{
			name:     "non-numeric id",
			ticketID: "invalid",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "zero id",
			ticketID: "0",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "negative id",
			ticketID: "-1",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/tickets/%s", tt.ticketID), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestParseTicketID(t *testing.T) {
	tests := []struct {
		name      string
		urlParam  string
		wantID    uint
		wantError bool
	}{
		{
			name:      "valid id",
			urlParam:  "123",
			wantID:    123,
			wantError: false,
		},
		{
			name:      "zero id",
			urlParam:  "0",
			wantID:    0,
			wantError: true,
		},
		{
			name:      "negative id",
			urlParam:  "-1",
			wantID:    0,
			wantError: true,
		},
		{
			name:      "non-numeric",
			urlParam:  "abc",
			wantID:    0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Params = gin.Params{
				{Key: "id", Value: tt.urlParam},
			}

			id, err := parseTicketID(c)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantID, id)
			}
		})
	}
}
