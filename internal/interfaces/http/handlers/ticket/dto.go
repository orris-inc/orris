package ticket

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"orris/internal/application/ticket/usecases"
	"orris/internal/shared/errors"
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	req := &ListTicketsRequest{
		Page:     page,
		PageSize: pageSize,
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
