package ticket

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewComment(t *testing.T) {
	tests := []struct {
		name       string
		ticketID   uint
		userID     uint
		content    string
		isInternal bool
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid comment",
			ticketID:   1,
			userID:     2,
			content:    "This is a test comment",
			isInternal: false,
			wantErr:    false,
		},
		{
			name:       "valid internal comment",
			ticketID:   1,
			userID:     2,
			content:    "Internal note for agents",
			isInternal: true,
			wantErr:    false,
		},
		{
			name:       "zero ticket ID",
			ticketID:   0,
			userID:     2,
			content:    "Test",
			isInternal: false,
			wantErr:    true,
			errMsg:     "ticket ID is required",
		},
		{
			name:       "zero user ID",
			ticketID:   1,
			userID:     0,
			content:    "Test",
			isInternal: false,
			wantErr:    true,
			errMsg:     "user ID is required",
		},
		{
			name:       "empty content",
			ticketID:   1,
			userID:     2,
			content:    "",
			isInternal: false,
			wantErr:    true,
			errMsg:     "content cannot be empty",
		},
		{
			name:       "content too long",
			ticketID:   1,
			userID:     2,
			content:    strings.Repeat("a", 5001),
			isInternal: false,
			wantErr:    true,
			errMsg:     "content exceeds maximum length of 5000 characters",
		},
		{
			name:       "content max length",
			ticketID:   1,
			userID:     2,
			content:    strings.Repeat("a", 5000),
			isInternal: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment, err := NewComment(
				tt.ticketID,
				tt.userID,
				tt.content,
				tt.isInternal,
			)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, comment)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, comment)
				assert.Equal(t, tt.ticketID, comment.TicketID())
				assert.Equal(t, tt.userID, comment.UserID())
				assert.Equal(t, tt.content, comment.Content())
				assert.Equal(t, tt.isInternal, comment.IsInternal())
				assert.NotZero(t, comment.CreatedAt())
				assert.NotZero(t, comment.UpdatedAt())
			}
		})
	}
}

func TestComment_SetID(t *testing.T) {
	tests := []struct {
		name    string
		id      uint
		twice   bool
		wantErr bool
		errMsg  string
	}{
		{
			name:    "set valid ID",
			id:      1,
			wantErr: false,
		},
		{
			name:    "set zero ID",
			id:      0,
			wantErr: true,
			errMsg:  "comment ID cannot be zero",
		},
		{
			name:    "set ID twice",
			id:      1,
			twice:   true,
			wantErr: true,
			errMsg:  "comment ID is already set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment, err := NewComment(1, 2, "Test comment", false)
			require.NoError(t, err)

			if tt.twice {
				err = comment.SetID(1)
				require.NoError(t, err)
			}

			err = comment.SetID(tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.id, comment.ID())
			}
		})
	}
}

func TestComment_UpdateContent(t *testing.T) {
	tests := []struct {
		name       string
		newContent string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "update to valid content",
			newContent: "Updated comment content",
			wantErr:    false,
		},
		{
			name:       "update to empty content",
			newContent: "",
			wantErr:    true,
			errMsg:     "content cannot be empty",
		},
		{
			name:       "update to content too long",
			newContent: strings.Repeat("a", 5001),
			wantErr:    true,
			errMsg:     "content exceeds maximum length of 5000 characters",
		},
		{
			name:       "update to max length content",
			newContent: strings.Repeat("a", 5000),
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment, err := NewComment(1, 2, "Original content", false)
			require.NoError(t, err)

			oldUpdatedAt := comment.UpdatedAt()
			time.Sleep(1 * time.Millisecond)

			err = comment.UpdateContent(tt.newContent)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.newContent, comment.Content())
				assert.True(t, comment.UpdatedAt().After(oldUpdatedAt))
			}
		})
	}
}

func TestReconstructComment(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		id       uint
		ticketID uint
		userID   uint
		content  string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid reconstruction",
			id:       1,
			ticketID: 1,
			userID:   2,
			content:  "Test comment",
			wantErr:  false,
		},
		{
			name:     "zero comment ID",
			id:       0,
			ticketID: 1,
			userID:   2,
			content:  "Test",
			wantErr:  true,
			errMsg:   "comment ID cannot be zero",
		},
		{
			name:     "zero ticket ID",
			id:       1,
			ticketID: 0,
			userID:   2,
			content:  "Test",
			wantErr:  true,
			errMsg:   "ticket ID is required",
		},
		{
			name:     "zero user ID",
			id:       1,
			ticketID: 1,
			userID:   0,
			content:  "Test",
			wantErr:  true,
			errMsg:   "user ID is required",
		},
		{
			name:     "empty content allowed in reconstruction",
			id:       1,
			ticketID: 1,
			userID:   2,
			content:  "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment, err := ReconstructComment(
				tt.id,
				tt.ticketID,
				tt.userID,
				tt.content,
				false,
				now,
				now,
			)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, comment)
				assert.Equal(t, tt.id, comment.ID())
				assert.Equal(t, tt.ticketID, comment.TicketID())
				assert.Equal(t, tt.userID, comment.UserID())
				assert.Equal(t, tt.content, comment.Content())
			}
		})
	}
}

func TestComment_Getters(t *testing.T) {
	ticketID := uint(1)
	userID := uint(2)
	content := "Test comment"
	isInternal := true

	comment, err := NewComment(ticketID, userID, content, isInternal)
	require.NoError(t, err)

	err = comment.SetID(10)
	require.NoError(t, err)

	assert.Equal(t, uint(10), comment.ID())
	assert.Equal(t, ticketID, comment.TicketID())
	assert.Equal(t, userID, comment.UserID())
	assert.Equal(t, content, comment.Content())
	assert.Equal(t, isInternal, comment.IsInternal())
	assert.NotZero(t, comment.CreatedAt())
	assert.NotZero(t, comment.UpdatedAt())
}

func TestComment_ConcurrentAccess(t *testing.T) {
	comment, err := NewComment(1, 2, "Test", false)
	require.NoError(t, err)

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			_ = comment.Content()
			_ = comment.IsInternal()
			_ = comment.TicketID()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
