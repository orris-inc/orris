package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/ticket"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	db "github.com/orris-inc/orris/internal/shared/db"
)

// allowedTicketOrderByFields defines the whitelist of allowed ORDER BY fields
// to prevent SQL injection attacks.
var allowedTicketOrderByFields = map[string]bool{
	"id":          true,
	"number":      true,
	"title":       true,
	"status":      true,
	"priority":    true,
	"category":    true,
	"creator_id":  true,
	"assignee_id": true,
	"created_at":  true,
	"updated_at":  true,
}

type TicketRepository struct {
	db     *gorm.DB
	mapper mappers.TicketMapper
}

func NewTicketRepository(db *gorm.DB) *TicketRepository {
	return &TicketRepository{
		db:     db,
		mapper: mappers.NewTicketMapper(),
	}
}

func (r *TicketRepository) Save(ctx context.Context, t *ticket.Ticket) error {
	model := r.mapper.ToModel(t)
	tx := db.GetTxFromContext(ctx, r.db)

	if err := tx.Create(model).Error; err != nil {
		return fmt.Errorf("failed to save ticket: %w", err)
	}

	if err := t.SetID(model.ID); err != nil {
		return err
	}

	return nil
}

func (r *TicketRepository) Update(ctx context.Context, t *ticket.Ticket) error {
	model := r.mapper.ToModel(t)
	tx := db.GetTxFromContext(ctx, r.db)

	result := tx.
		Model(&models.TicketModel{}).
		Where("id = ?", model.ID).
		Updates(model)

	if result.Error != nil {
		return fmt.Errorf("failed to update ticket: %w", result.Error)
	}

	// Note: RowsAffected may be 0 when updated values are identical to existing values.

	return nil
}

func (r *TicketRepository) FindByID(ctx context.Context, id uint) (*ticket.Ticket, error) {
	var model models.TicketModel
	tx := db.GetTxFromContext(ctx, r.db)

	if err := tx.
		First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ticket not found")
		}
		return nil, fmt.Errorf("failed to find ticket: %w", err)
	}

	t, err := r.mapper.ToDomain(&model)
	if err != nil {
		return nil, err
	}

	// Load comments in a single query and convert via mapper
	if err := r.loadComments(ctx, t, model.ID); err != nil {
		return nil, err
	}

	return t, nil
}

func (r *TicketRepository) FindByNumber(ctx context.Context, number string) (*ticket.Ticket, error) {
	var model models.TicketModel
	tx := db.GetTxFromContext(ctx, r.db)

	if err := tx.
		Where("number = ?", number).
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ticket not found")
		}
		return nil, fmt.Errorf("failed to find ticket: %w", err)
	}

	t, err := r.mapper.ToDomain(&model)
	if err != nil {
		return nil, err
	}

	// Load comments in a single query and convert via mapper
	if err := r.loadComments(ctx, t, model.ID); err != nil {
		return nil, err
	}

	return t, nil
}

func (r *TicketRepository) Delete(ctx context.Context, id uint) error {
	tx := db.GetTxFromContext(ctx, r.db)
	result := tx.Delete(&models.TicketModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete ticket: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("ticket not found")
	}
	return nil
}

func (r *TicketRepository) List(
	ctx context.Context,
	filter ticket.TicketFilter,
) ([]*ticket.Ticket, int64, error) {
	tx := db.GetTxFromContext(ctx, r.db)
	query := tx.Model(&models.TicketModel{})

	if filter.Status != nil {
		query = query.Where("status = ?", filter.Status.String())
	}
	if filter.Priority != nil {
		query = query.Where("priority = ?", filter.Priority.String())
	}
	if filter.Category != nil {
		query = query.Where("category = ?", filter.Category.String())
	}
	if filter.CreatorID != nil {
		query = query.Where("creator_id = ?", *filter.CreatorID)
	}
	if filter.AssigneeID != nil {
		query = query.Where("assignee_id = ?", *filter.AssigneeID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tickets: %w", err)
	}

	// Apply sorting with whitelist validation to prevent SQL injection
	sortBy := strings.ToLower(filter.SortBy)
	if sortBy != "" && allowedTicketOrderByFields[sortBy] {
		order := strings.ToUpper(filter.SortOrder)
		if order != "ASC" && order != "DESC" {
			order = "DESC"
		}
		query = query.Order(sortBy + " " + order)
	} else {
		query = query.Order("created_at DESC")
	}

	if filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Limit(filter.PageSize).Offset(offset)
	}

	var ticketModels []models.TicketModel
	if err := query.Find(&ticketModels).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list tickets: %w", err)
	}

	tickets := make([]*ticket.Ticket, len(ticketModels))
	for i, model := range ticketModels {
		t, err := r.mapper.ToDomain(&model)
		if err != nil {
			return nil, 0, err
		}
		tickets[i] = t
	}

	return tickets, total, nil
}

func (r *TicketRepository) SaveComment(ctx context.Context, c *ticket.Comment) error {
	model := &models.CommentModel{
		TicketID:   c.TicketID(),
		UserID:     c.UserID(),
		Content:    c.Content(),
		IsInternal: c.IsInternal(),
		CreatedAt:  c.CreatedAt().UnixMilli(),
		UpdatedAt:  c.UpdatedAt().UnixMilli(),
	}

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.Create(model).Error; err != nil {
		return fmt.Errorf("failed to save comment: %w", err)
	}

	if err := c.SetID(model.ID); err != nil {
		return err
	}

	return nil
}

func (r *TicketRepository) FindCommentsByTicketID(
	ctx context.Context,
	ticketID uint,
) ([]*ticket.Comment, error) {
	var commentModels []models.CommentModel

	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.
		Where("ticket_id = ?", ticketID).
		Order("created_at ASC").
		Find(&commentModels).Error; err != nil {
		return nil, fmt.Errorf("failed to find comments: %w", err)
	}

	comments := make([]*ticket.Comment, len(commentModels))
	for i, model := range commentModels {
		c, err := r.mapper.CommentToDomain(&model)
		if err != nil {
			return nil, err
		}
		comments[i] = c
	}

	return comments, nil
}

// loadComments queries comments for a ticket and adds them to the domain entity.
func (r *TicketRepository) loadComments(ctx context.Context, t *ticket.Ticket, ticketID uint) error {
	var commentModels []models.CommentModel
	tx := db.GetTxFromContext(ctx, r.db)
	if err := tx.
		Where("ticket_id = ?", ticketID).
		Order("created_at ASC").
		Find(&commentModels).Error; err != nil {
		return fmt.Errorf("failed to load comments: %w", err)
	}

	for _, cm := range commentModels {
		comment, err := r.mapper.CommentToDomain(&cm)
		if err != nil {
			return err
		}
		t.AddComment(comment)
	}

	return nil
}
