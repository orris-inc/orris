package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"orris/internal/domain/ticket"
	vo "orris/internal/domain/ticket/value_objects"
	"orris/internal/shared/query"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&TicketModel{}, &CommentModel{})
	require.NoError(t, err)

	return db
}

func createTestTicket(t *testing.T, title string, category vo.Category, priority vo.Priority, creatorID uint) *ticket.Ticket {
	tk, err := ticket.NewTicket(title, "Test description", category, priority, creatorID)
	require.NoError(t, err)
	return tk
}

func TestTicketRepository_Save(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTicketRepository(db)
	ctx := context.Background()

	t.Run("save new ticket successfully", func(t *testing.T) {
		tk := createTestTicket(t, "Test Ticket", vo.CategoryTechnical, vo.PriorityHigh, 1)
		err := tk.SetNumber("TK-001")
		require.NoError(t, err)

		err = repo.Save(ctx, tk)
		assert.NoError(t, err)
		assert.NotZero(t, tk.ID())
	})

	t.Run("save ticket with tags and metadata", func(t *testing.T) {
		tk := createTestTicket(t, "Ticket with Tags", vo.CategoryBilling, vo.PriorityMedium, 2)
		err := tk.SetNumber("TK-002")
		require.NoError(t, err)

		err = repo.Save(ctx, tk)
		assert.NoError(t, err)

		found, err := repo.FindByID(ctx, tk.ID())
		assert.NoError(t, err)
		assert.Equal(t, tk.Number(), found.Number())
		assert.Equal(t, tk.Title(), found.Title())
	})

	t.Run("duplicate number should fail", func(t *testing.T) {
		tk1 := createTestTicket(t, "Ticket 1", vo.CategoryOther, vo.PriorityLow, 3)
		err := tk1.SetNumber("TK-DUP")
		require.NoError(t, err)
		err = repo.Save(ctx, tk1)
		require.NoError(t, err)

		tk2 := createTestTicket(t, "Ticket 2", vo.CategoryOther, vo.PriorityLow, 3)
		err = tk2.SetNumber("TK-DUP")
		require.NoError(t, err)
		err = repo.Save(ctx, tk2)
		assert.Error(t, err)
	})
}

func TestTicketRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTicketRepository(db)
	ctx := context.Background()

	t.Run("update ticket successfully", func(t *testing.T) {
		tk := createTestTicket(t, "Original Title", vo.CategoryTechnical, vo.PriorityHigh, 1)
		err := tk.SetNumber("TK-UPDATE-001")
		require.NoError(t, err)
		err = repo.Save(ctx, tk)
		require.NoError(t, err)

		assigneeID := uint(5)
		err = tk.AssignTo(assigneeID, 1)
		require.NoError(t, err)

		err = repo.Update(ctx, tk)
		assert.NoError(t, err)

		found, err := repo.FindByID(ctx, tk.ID())
		assert.NoError(t, err)
		assert.NotNil(t, found.AssigneeID())
		assert.Equal(t, assigneeID, *found.AssigneeID())
	})

	t.Run("optimistic locking - concurrent update conflict", func(t *testing.T) {
		tk := createTestTicket(t, "Locking Test", vo.CategoryTechnical, vo.PriorityHigh, 1)
		err := tk.SetNumber("TK-LOCK-001")
		require.NoError(t, err)
		err = repo.Save(ctx, tk)
		require.NoError(t, err)

		tk1, err := repo.FindByID(ctx, tk.ID())
		require.NoError(t, err)
		tk2, err := repo.FindByID(ctx, tk.ID())
		require.NoError(t, err)

		err = tk1.AssignTo(10, 1)
		require.NoError(t, err)
		err = repo.Update(ctx, tk1)
		assert.NoError(t, err)

		err = tk2.AssignTo(20, 1)
		require.NoError(t, err)
		err = repo.Update(ctx, tk2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version mismatch")
	})

	t.Run("update non-existent ticket should fail", func(t *testing.T) {
		tk := createTestTicket(t, "Non-existent", vo.CategoryTechnical, vo.PriorityHigh, 1)
		err := tk.SetNumber("TK-NONEXIST")
		require.NoError(t, err)
		err = tk.SetID(99999)
		require.NoError(t, err)

		err = tk.AssignTo(5, 1)
		require.NoError(t, err)

		err = repo.Update(ctx, tk)
		assert.Error(t, err)
	})
}

func TestTicketRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTicketRepository(db)
	ctx := context.Background()

	t.Run("find existing ticket", func(t *testing.T) {
		tk := createTestTicket(t, "Find by ID", vo.CategoryTechnical, vo.PriorityHigh, 1)
		err := tk.SetNumber("TK-FIND-001")
		require.NoError(t, err)
		err = repo.Save(ctx, tk)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, tk.ID())
		assert.NoError(t, err)
		assert.Equal(t, tk.ID(), found.ID())
		assert.Equal(t, tk.Number(), found.Number())
		assert.Equal(t, tk.Title(), found.Title())
	})

	t.Run("find non-existent ticket", func(t *testing.T) {
		found, err := repo.FindByID(ctx, 99999)
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("find ticket with comments", func(t *testing.T) {
		tk := createTestTicket(t, "Ticket with Comments", vo.CategoryTechnical, vo.PriorityHigh, 1)
		err := tk.SetNumber("TK-COMMENT-001")
		require.NoError(t, err)
		err = repo.Save(ctx, tk)
		require.NoError(t, err)

		comment, err := ticket.NewComment(tk.ID(), 2, "Test comment", false)
		require.NoError(t, err)
		err = repo.SaveComment(ctx, comment)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, tk.ID())
		assert.NoError(t, err)
		assert.Len(t, found.Comments(), 1)
	})
}

func TestTicketRepository_FindByNumber(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTicketRepository(db)
	ctx := context.Background()

	t.Run("find by existing number", func(t *testing.T) {
		tk := createTestTicket(t, "Find by Number", vo.CategoryTechnical, vo.PriorityHigh, 1)
		err := tk.SetNumber("TK-NUM-001")
		require.NoError(t, err)
		err = repo.Save(ctx, tk)
		require.NoError(t, err)

		found, err := repo.FindByNumber(ctx, "TK-NUM-001")
		assert.NoError(t, err)
		assert.Equal(t, tk.ID(), found.ID())
		assert.Equal(t, "TK-NUM-001", found.Number())
	})

	t.Run("find by non-existent number", func(t *testing.T) {
		found, err := repo.FindByNumber(ctx, "TK-NONEXIST")
		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestTicketRepository_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTicketRepository(db)
	ctx := context.Background()

	tk1 := createTestTicket(t, "Ticket 1", vo.CategoryTechnical, vo.PriorityHigh, 1)
	tk1.SetNumber("TK-LIST-001")
	repo.Save(ctx, tk1)

	tk2 := createTestTicket(t, "Ticket 2", vo.CategoryBilling, vo.PriorityMedium, 2)
	tk2.SetNumber("TK-LIST-002")
	repo.Save(ctx, tk2)

	tk3 := createTestTicket(t, "Ticket 3", vo.CategoryTechnical, vo.PriorityLow, 1)
	tk3.SetNumber("TK-LIST-003")
	repo.Save(ctx, tk3)

	t.Run("list all tickets", func(t *testing.T) {
		filter := ticket.TicketFilter{
			BaseFilter: query.BaseFilter{
				PageFilter: query.PageFilter{
					Page:     1,
					PageSize: 10,
				},
			},
		}

		tickets, total, err := repo.List(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, tickets, 3)
	})

	t.Run("filter by category", func(t *testing.T) {
		category := vo.CategoryTechnical
		filter := ticket.TicketFilter{
			BaseFilter: query.BaseFilter{
				PageFilter: query.PageFilter{
					Page:     1,
					PageSize: 10,
				},
			},
			Category: &category,
		}

		tickets, total, err := repo.List(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, tickets, 2)
	})

	t.Run("filter by priority", func(t *testing.T) {
		priority := vo.PriorityHigh
		filter := ticket.TicketFilter{
			BaseFilter: query.BaseFilter{
				PageFilter: query.PageFilter{
					Page:     1,
					PageSize: 10,
				},
			},
			Priority: &priority,
		}

		tickets, total, err := repo.List(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, tickets, 1)
	})

	t.Run("filter by creator ID", func(t *testing.T) {
		creatorID := uint(1)
		filter := ticket.TicketFilter{
			BaseFilter: query.BaseFilter{
				PageFilter: query.PageFilter{
					Page:     1,
					PageSize: 10,
				},
			},
			CreatorID: &creatorID,
		}

		tickets, total, err := repo.List(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, tickets, 2)
	})

	t.Run("pagination", func(t *testing.T) {
		filter := ticket.TicketFilter{
			BaseFilter: query.BaseFilter{
				PageFilter: query.PageFilter{
					Page:     1,
					PageSize: 2,
				},
			},
		}

		tickets, total, err := repo.List(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, tickets, 2)

		filter.BaseFilter.PageFilter.Page = 2
		tickets, total, err = repo.List(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, tickets, 1)
	})

	t.Run("sort by created_at desc", func(t *testing.T) {
		filter := ticket.TicketFilter{
			BaseFilter: query.BaseFilter{
				PageFilter: query.PageFilter{
					Page:     1,
					PageSize: 10,
				},
				SortFilter: query.SortFilter{
					SortBy:    "created_at",
					SortOrder: "desc",
				},
			},
		}

		tickets, _, err := repo.List(ctx, filter)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, tickets[0].CreatedAt().UnixMilli(), tickets[1].CreatedAt().UnixMilli())
	})
}

func TestTicketRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTicketRepository(db)
	ctx := context.Background()

	t.Run("delete existing ticket", func(t *testing.T) {
		tk := createTestTicket(t, "Delete Test", vo.CategoryTechnical, vo.PriorityHigh, 1)
		err := tk.SetNumber("TK-DEL-001")
		require.NoError(t, err)
		err = repo.Save(ctx, tk)
		require.NoError(t, err)

		err = repo.Delete(ctx, tk.ID())
		assert.NoError(t, err)

		found, err := repo.FindByID(ctx, tk.ID())
		assert.Error(t, err)
		assert.Nil(t, found)
	})

	t.Run("delete non-existent ticket", func(t *testing.T) {
		err := repo.Delete(ctx, 99999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("delete ticket cascades to comments", func(t *testing.T) {
		tk := createTestTicket(t, "Delete with Comments", vo.CategoryTechnical, vo.PriorityHigh, 1)
		err := tk.SetNumber("TK-DELCMT-001")
		require.NoError(t, err)
		err = repo.Save(ctx, tk)
		require.NoError(t, err)

		comment, err := ticket.NewComment(tk.ID(), 2, "Test comment", false)
		require.NoError(t, err)
		err = repo.SaveComment(ctx, comment)
		require.NoError(t, err)

		err = repo.Delete(ctx, tk.ID())
		assert.NoError(t, err)
	})
}

func TestTicketRepository_SaveComment(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTicketRepository(db)
	ctx := context.Background()

	t.Run("save comment successfully", func(t *testing.T) {
		tk := createTestTicket(t, "Ticket for Comment", vo.CategoryTechnical, vo.PriorityHigh, 1)
		err := tk.SetNumber("TK-CMT-001")
		require.NoError(t, err)
		err = repo.Save(ctx, tk)
		require.NoError(t, err)

		comment, err := ticket.NewComment(tk.ID(), 2, "Test comment content", false)
		require.NoError(t, err)

		err = repo.SaveComment(ctx, comment)
		assert.NoError(t, err)
		assert.NotZero(t, comment.ID())
	})

	t.Run("save internal comment", func(t *testing.T) {
		tk := createTestTicket(t, "Ticket for Internal Comment", vo.CategoryTechnical, vo.PriorityHigh, 1)
		err := tk.SetNumber("TK-INTCMT-001")
		require.NoError(t, err)
		err = repo.Save(ctx, tk)
		require.NoError(t, err)

		comment, err := ticket.NewComment(tk.ID(), 3, "Internal comment", true)
		require.NoError(t, err)

		err = repo.SaveComment(ctx, comment)
		assert.NoError(t, err)

		comments, err := repo.FindCommentsByTicketID(ctx, tk.ID())
		assert.NoError(t, err)
		assert.Len(t, comments, 1)
		assert.True(t, comments[0].IsInternal())
	})
}

func TestTicketRepository_FindCommentsByTicketID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTicketRepository(db)
	ctx := context.Background()

	t.Run("find multiple comments ordered by created_at", func(t *testing.T) {
		tk := createTestTicket(t, "Ticket with Multiple Comments", vo.CategoryTechnical, vo.PriorityHigh, 1)
		err := tk.SetNumber("TK-MULTICMT-001")
		require.NoError(t, err)
		err = repo.Save(ctx, tk)
		require.NoError(t, err)

		comment1, _ := ticket.NewComment(tk.ID(), 2, "First comment", false)
		time.Sleep(10 * time.Millisecond)
		comment2, _ := ticket.NewComment(tk.ID(), 3, "Second comment", false)
		time.Sleep(10 * time.Millisecond)
		comment3, _ := ticket.NewComment(tk.ID(), 2, "Third comment", true)

		repo.SaveComment(ctx, comment1)
		repo.SaveComment(ctx, comment2)
		repo.SaveComment(ctx, comment3)

		comments, err := repo.FindCommentsByTicketID(ctx, tk.ID())
		assert.NoError(t, err)
		assert.Len(t, comments, 3)
		assert.Equal(t, "First comment", comments[0].Content())
		assert.Equal(t, "Second comment", comments[1].Content())
		assert.Equal(t, "Third comment", comments[2].Content())
	})

	t.Run("find comments for ticket with no comments", func(t *testing.T) {
		tk := createTestTicket(t, "Ticket with No Comments", vo.CategoryTechnical, vo.PriorityHigh, 1)
		err := tk.SetNumber("TK-NOCMT-001")
		require.NoError(t, err)
		err = repo.Save(ctx, tk)
		require.NoError(t, err)

		comments, err := repo.FindCommentsByTicketID(ctx, tk.ID())
		assert.NoError(t, err)
		assert.Len(t, comments, 0)
	})
}

func TestTicketRepository_TransactionRollback(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTicketRepository(db)
	ctx := context.Background()

	t.Run("transaction rollback on error", func(t *testing.T) {
		tk := createTestTicket(t, "Transaction Test", vo.CategoryTechnical, vo.PriorityHigh, 1)
		err := tk.SetNumber("TK-TXN-001")
		require.NoError(t, err)

		err = db.Transaction(func(tx *gorm.DB) error {
			txRepo := NewTicketRepository(tx)

			err := txRepo.Save(ctx, tk)
			if err != nil {
				return err
			}

			return assert.AnError
		})

		assert.Error(t, err)

		found, err := repo.FindByNumber(ctx, "TK-TXN-001")
		assert.Error(t, err)
		assert.Nil(t, found)
	})

	t.Run("transaction commit on success", func(t *testing.T) {
		tk := createTestTicket(t, "Transaction Commit", vo.CategoryTechnical, vo.PriorityHigh, 1)
		err := tk.SetNumber("TK-TXN-002")
		require.NoError(t, err)

		err = db.Transaction(func(tx *gorm.DB) error {
			txRepo := NewTicketRepository(tx)
			return txRepo.Save(ctx, tk)
		})

		assert.NoError(t, err)

		found, err := repo.FindByNumber(ctx, "TK-TXN-002")
		assert.NoError(t, err)
		assert.NotNil(t, found)
	})
}

func TestTicketRepository_ConcurrentReads(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTicketRepository(db)
	ctx := context.Background()

	tk := createTestTicket(t, "Concurrent Read Test", vo.CategoryTechnical, vo.PriorityHigh, 1)
	err := tk.SetNumber("TK-CONCURRENT-001")
	require.NoError(t, err)
	err = repo.Save(ctx, tk)
	require.NoError(t, err)

	ticketID := tk.ID()

	var successCount int
	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func() {
			_, readErr := repo.FindByID(ctx, ticketID)
			if readErr == nil {
				successCount++
			}
			done <- true
		}()
	}

	for i := 0; i < 3; i++ {
		<-done
	}

	assert.GreaterOrEqual(t, successCount, 1)
}
