package mappers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/ticket"
	vo "github.com/orris-inc/orris/internal/domain/ticket/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// TicketMapper handles the conversion between Ticket domain entities and persistence models.
type TicketMapper interface {
	// ToModel converts a ticket domain entity to a persistence model.
	ToModel(t *ticket.Ticket) *models.TicketModel

	// ToDomain converts a ticket persistence model to a domain entity.
	ToDomain(model *models.TicketModel) (*ticket.Ticket, error)

	// CommentToDomain converts a comment persistence model to a domain entity.
	CommentToDomain(model *models.CommentModel) (*ticket.Comment, error)
}

// TicketMapperImpl is the concrete implementation of TicketMapper.
type TicketMapperImpl struct{}

// NewTicketMapper creates a new TicketMapper.
func NewTicketMapper() TicketMapper {
	return &TicketMapperImpl{}
}

// ToModel converts a ticket domain entity to a persistence model.
func (m *TicketMapperImpl) ToModel(t *ticket.Ticket) *models.TicketModel {
	model := &models.TicketModel{
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

// ToDomain converts a ticket persistence model to a domain entity.
// This method only converts the ticket fields. Comments must be loaded separately by the repository.
func (m *TicketMapperImpl) ToDomain(model *models.TicketModel) (*ticket.Ticket, error) {
	category, _ := vo.NewCategory(model.Category)
	priority, _ := vo.NewPriority(model.Priority)
	status, _ := vo.NewTicketStatus(model.Status)

	var tags []string
	if model.Tags != "" {
		if err := json.Unmarshal([]byte(model.Tags), &tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ticket tags (id=%d): %w", model.ID, err)
		}
	}

	var metadata map[string]interface{}
	if model.Metadata != "" {
		if err := json.Unmarshal([]byte(model.Metadata), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ticket metadata (id=%d): %w", model.ID, err)
		}
	}

	createdAt := ticketConvertMillisToTime(model.CreatedAt)
	updatedAt := ticketConvertMillisToTime(model.UpdatedAt)

	var slaDueTime, responseTime, resolvedTime, closedAt *time.Time
	if model.SLADueTime != nil {
		t := ticketConvertMillisToTime(*model.SLADueTime)
		slaDueTime = &t
	}
	if model.ResponseTime != nil {
		t := ticketConvertMillisToTime(*model.ResponseTime)
		responseTime = &t
	}
	if model.ResolvedTime != nil {
		t := ticketConvertMillisToTime(*model.ResolvedTime)
		resolvedTime = &t
	}
	if model.ClosedAt != nil {
		t := ticketConvertMillisToTime(*model.ClosedAt)
		closedAt = &t
	}

	return ticket.ReconstructTicket(
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
}

// CommentToDomain converts a comment persistence model to a domain entity.
func (m *TicketMapperImpl) CommentToDomain(model *models.CommentModel) (*ticket.Comment, error) {
	createdAt := ticketConvertMillisToTime(model.CreatedAt)
	updatedAt := ticketConvertMillisToTime(model.UpdatedAt)

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

func ticketConvertMillisToTime(millis int64) time.Time {
	return time.Unix(0, millis*int64(time.Millisecond))
}
