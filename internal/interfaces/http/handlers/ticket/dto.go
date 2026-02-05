package ticket

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/ticket/usecases"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/utils"
)

type CreateTicketRequest struct {
	Title       string                 `json:"title" binding:"required,max=200"`
	Description string                 `json:"description" binding:"required,max=5000"`
	Category    string                 `json:"category" binding:"required"`
	Priority    string                 `json:"priority" binding:"required"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (r *CreateTicketRequest) ToCommand(creatorID uint) usecases.CreateTicketCommand {
	return usecases.CreateTicketCommand{
		Title:       r.Title,
		Description: r.Description,
		Category:    r.Category,
		Priority:    r.Priority,
		CreatorID:   creatorID,
		Tags:        r.Tags,
		Metadata:    r.Metadata,
	}
}

type AssignTicketRequest struct {
	AssigneeID uint `json:"assignee_id" binding:"required"`
}

type AddCommentRequest struct {
	Content    string `json:"content" binding:"required,max=10000"`
	IsInternal bool   `json:"is_internal"`
}

type CloseTicketRequest struct {
	Reason string `json:"reason" binding:"required,max=500"`
}

type ReopenTicketRequest struct {
	Reason string `json:"reason" binding:"required,max=500"`
}

type ListTicketsRequest struct {
	Page       int
	PageSize   int
	Status     *string
	Priority   *string
	Category   *string
	AssigneeID *uint
}

func (r *ListTicketsRequest) ToQuery(userID uint) usecases.ListTicketsQuery {
	return usecases.ListTicketsQuery{
		UserID:     userID,
		Page:       r.Page,
		PageSize:   r.PageSize,
		Status:     r.Status,
		Priority:   r.Priority,
		Category:   r.Category,
		AssigneeID: r.AssigneeID,
	}
}

func parseListTicketsRequest(c *gin.Context) (*ListTicketsRequest, error) {
	pagination := utils.ParsePagination(c)

	req := &ListTicketsRequest{
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}

	if status := c.Query("status"); status != "" {
		req.Status = &status
	}

	if priority := c.Query("priority"); priority != "" {
		req.Priority = &priority
	}

	if category := c.Query("category"); category != "" {
		req.Category = &category
	}

	if assigneeIDStr := c.Query("assignee_id"); assigneeIDStr != "" {
		assigneeID, err := strconv.ParseUint(assigneeIDStr, 10, 32)
		if err != nil {
			return nil, errors.NewValidationError("Invalid assignee_id")
		}
		id := uint(assigneeID)
		req.AssigneeID = &id
	}

	return req, nil
}
