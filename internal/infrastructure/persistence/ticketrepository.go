package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"orris/internal/domain/ticket"
	vo "orris/internal/domain/ticket/value_objects"
)

type TicketModel struct {
	ID           uint   `gorm:"primaryKey"`
	Number       string `gorm:"uniqueIndex;size:50;not null"`
	Title        string `gorm:"size:200;not null"`
	Description  string `gorm:"type:text;not null"`
	Category     string `gorm:"size:50;not null;index"`
	Priority     string `gorm:"size:20;not null;index"`
	Status       string `gorm:"size:20;not null;index"`
	CreatorID    uint   `gorm:"not null;index"`
	AssigneeID   *uint  `gorm:"index"`
	Tags         string `gorm:"type:json"`
	Metadata     string `gorm:"type:json"`
	SLADueTime   *int64 `gorm:"index"`
	ResponseTime *int64
	ResolvedTime *int64
	Version      int   `gorm:"not null;default:1"`
	CreatedAt    int64 `gorm:"autoCreateTime:milli;not null"`
	UpdatedAt    int64 `gorm:"autoUpdateTime:milli;not null"`
	ClosedAt     *int64
	Comments     []CommentModel `gorm:"foreignKey:TicketID;constraint:OnDelete:CASCADE"`
}

func (TicketModel) TableName() string {
	return "tickets"
}

type CommentModel struct {
	ID         uint   `gorm:"primaryKey"`
	TicketID   uint   `gorm:"not null;index"`
	UserID     uint   `gorm:"not null;index"`
	Content    string `gorm:"type:text;not null"`
	IsInternal bool   `gorm:"not null;default:false"`
	CreatedAt  int64  `gorm:"autoCreateTime:milli;not null;index"`
	UpdatedAt  int64  `gorm:"autoUpdateTime:milli;not null"`
}

func (CommentModel) TableName() string {
	return "ticket_comments"
}

type TicketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) *TicketRepository {
	return &TicketRepository{db: db}
}

func (r *TicketRepository) Save(ctx context.Context, t *ticket.Ticket) error {
	model := r.toModel(t)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to save ticket: %w", err)
	}

	if err := t.SetID(model.ID); err != nil {
		return err
	}

	return nil
}

func (r *TicketRepository) Update(ctx context.Context, t *ticket.Ticket) error {
	model := r.toModel(t)

	result := r.db.WithContext(ctx).
		Model(&TicketModel{}).
		Where("id = ? AND version = ?", model.ID, model.Version-1).
		Updates(model)

	if result.Error != nil {
		return fmt.Errorf("failed to update ticket: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("ticket not found or version mismatch (optimistic locking)")
	}

	return nil
}

func (r *TicketRepository) FindByID(ctx context.Context, id uint) (*ticket.Ticket, error) {
	var model TicketModel

	if err := r.db.WithContext(ctx).
		Preload("Comments").
		First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ticket not found")
		}
		return nil, fmt.Errorf("failed to find ticket: %w", err)
	}

	return r.toDomain(&model)
}

func (r *TicketRepository) FindByNumber(ctx context.Context, number string) (*ticket.Ticket, error) {
	var model TicketModel

	if err := r.db.WithContext(ctx).
		Preload("Comments").
		Where("number = ?", number).
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ticket not found")
		}
		return nil, fmt.Errorf("failed to find ticket: %w", err)
	}

	return r.toDomain(&model)
}

func (r *TicketRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&TicketModel{}, id)
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
	query := r.db.WithContext(ctx).Model(&TicketModel{})

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

	if filter.SortBy != "" {
		order := filter.SortBy
		if filter.SortOrder == "desc" {
			order += " DESC"
		} else {
			order += " ASC"
		}
		query = query.Order(order)
	} else {
		query = query.Order("created_at DESC")
	}

	if filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Limit(filter.PageSize).Offset(offset)
	}

	var models []TicketModel
	if err := query.Preload("Comments").Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list tickets: %w", err)
	}

	tickets := make([]*ticket.Ticket, len(models))
	for i, model := range models {
		t, err := r.toDomain(&model)
		if err != nil {
			return nil, 0, err
		}
		tickets[i] = t
	}

	return tickets, total, nil
}

func (r *TicketRepository) SaveComment(ctx context.Context, c *ticket.Comment) error {
	model := &CommentModel{
		TicketID:   c.TicketID(),
		UserID:     c.UserID(),
		Content:    c.Content(),
		IsInternal: c.IsInternal(),
		CreatedAt:  c.CreatedAt().UnixMilli(),
		UpdatedAt:  c.UpdatedAt().UnixMilli(),
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
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
	var models []CommentModel

	if err := r.db.WithContext(ctx).
		Where("ticket_id = ?", ticketID).
		Order("created_at ASC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to find comments: %w", err)
	}

	comments := make([]*ticket.Comment, len(models))
	for i, model := range models {
		c, err := r.commentToDomain(&model)
		if err != nil {
			return nil, err
		}
		comments[i] = c
	}

	return comments, nil
}

func (r *TicketRepository) toModel(t *ticket.Ticket) *TicketModel {
	model := &TicketModel{
		ID:          t.ID(),
		Number:      t.Number(),
		Title:       t.Title(),
		Description: t.Description(),
		Category:    t.Category().String(),
		Priority:    t.Priority().String(),
		Status:      t.Status().String(),
		CreatorID:   t.CreatorID(),
		AssigneeID:  t.AssigneeID(),
		Version:     t.Version(),
		CreatedAt:   t.CreatedAt().UnixMilli(),
		UpdatedAt:   t.UpdatedAt().UnixMilli(),
	}

	if len(t.Tags()) > 0 {
		tagsJSON, _ := json.Marshal(t.Tags())
		model.Tags = string(tagsJSON)
	}

	if len(t.Metadata()) > 0 {
		metaJSON, _ := json.Marshal(t.Metadata())
		model.Metadata = string(metaJSON)
	}

	if t.SLADueTime() != nil {
		sla := t.SLADueTime().UnixMilli()
		model.SLADueTime = &sla
	}

	if t.ResponseTime() != nil {
		resp := t.ResponseTime().UnixMilli()
		model.ResponseTime = &resp
	}

	if t.ResolvedTime() != nil {
		resolved := t.ResolvedTime().UnixMilli()
		model.ResolvedTime = &resolved
	}

	if t.ClosedAt() != nil {
		closed := t.ClosedAt().UnixMilli()
		model.ClosedAt = &closed
	}

	return model
}

func (r *TicketRepository) toDomain(model *TicketModel) (*ticket.Ticket, error) {
	category, _ := vo.NewCategory(model.Category)
	priority, _ := vo.NewPriority(model.Priority)
	status, _ := vo.NewTicketStatus(model.Status)

	var tags []string
	if model.Tags != "" {
		json.Unmarshal([]byte(model.Tags), &tags)
	}

	var metadata map[string]interface{}
	if model.Metadata != "" {
		json.Unmarshal([]byte(model.Metadata), &metadata)
	}

	createdAt := convertMillisToTime(model.CreatedAt)
	updatedAt := convertMillisToTime(model.UpdatedAt)

	var slaDueTime, responseTime, resolvedTime, closedAt *time.Time
	if model.SLADueTime != nil {
		t := convertMillisToTime(*model.SLADueTime)
		slaDueTime = &t
	}
	if model.ResponseTime != nil {
		t := convertMillisToTime(*model.ResponseTime)
		responseTime = &t
	}
	if model.ResolvedTime != nil {
		t := convertMillisToTime(*model.ResolvedTime)
		resolvedTime = &t
	}
	if model.ClosedAt != nil {
		t := convertMillisToTime(*model.ClosedAt)
		closedAt = &t
	}

	t, err := ticket.ReconstructTicket(
		model.ID,
		model.Number,
		model.Title,
		model.Description,
		category,
		priority,
		status,
		model.CreatorID,
		model.AssigneeID,
		tags,
		metadata,
		slaDueTime,
		responseTime,
		resolvedTime,
		model.Version,
		createdAt,
		updatedAt,
		closedAt,
	)

	if err != nil {
		return nil, err
	}

	for _, commentModel := range model.Comments {
		comment, err := r.commentToDomain(&commentModel)
		if err != nil {
			return nil, err
		}
		t.AddComment(comment)
	}

	return t, nil
}

func (r *TicketRepository) commentToDomain(model *CommentModel) (*ticket.Comment, error) {
	createdAt := convertMillisToTime(model.CreatedAt)
	updatedAt := convertMillisToTime(model.UpdatedAt)

	return ticket.ReconstructComment(
		model.ID,
		model.TicketID,
		model.UserID,
		model.Content,
		model.IsInternal,
		createdAt,
		updatedAt,
	)
}

func convertMillisToTime(millis int64) time.Time {
	return time.Unix(0, millis*int64(time.Millisecond))
}
