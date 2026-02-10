package ticket

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/ticket/usecases"
	vo "github.com/orris-inc/orris/internal/domain/ticket/valueobjects"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

type TicketHandler struct {
	createTicketUC   usecases.CreateTicketExecutor
	assignTicketUC   usecases.AssignTicketExecutor
	updateStatusUC   usecases.UpdateTicketStatusExecutor
	addCommentUC     usecases.AddCommentExecutor
	changeStatusUC   usecases.ChangeStatusExecutor
	getTicketUC      usecases.GetTicketExecutor
	listTicketsUC    usecases.ListTicketsExecutor
	deleteTicketUC   usecases.DeleteTicketExecutor
	updatePriorityUC usecases.UpdateTicketPriorityExecutor
	logger           logger.Interface
}

func NewTicketHandler(
	createTicketUC usecases.CreateTicketExecutor,
	assignTicketUC usecases.AssignTicketExecutor,
	updateStatusUC usecases.UpdateTicketStatusExecutor,
	addCommentUC usecases.AddCommentExecutor,
	changeStatusUC usecases.ChangeStatusExecutor,
	getTicketUC usecases.GetTicketExecutor,
	listTicketsUC usecases.ListTicketsExecutor,
	deleteTicketUC usecases.DeleteTicketExecutor,
	updatePriorityUC usecases.UpdateTicketPriorityExecutor,
	log logger.Interface,
) *TicketHandler {
	return &TicketHandler{
		createTicketUC:   createTicketUC,
		assignTicketUC:   assignTicketUC,
		updateStatusUC:   updateStatusUC,
		addCommentUC:     addCommentUC,
		changeStatusUC:   changeStatusUC,
		getTicketUC:      getTicketUC,
		listTicketsUC:    listTicketsUC,
		deleteTicketUC:   deleteTicketUC,
		updatePriorityUC: updatePriorityUC,
		logger:           log,
	}
}

// CreateTicket handles POST /tickets
func (h *TicketHandler) CreateTicket(c *gin.Context) {
	var req CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create ticket", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}
	cmd := req.ToCommand(userID)

	result, err := h.createTicketUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Ticket created successfully")
}

// GetTicket handles GET /tickets/:id
func (h *TicketHandler) GetTicket(c *gin.Context) {
	ticketID, err := parseTicketID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}
	cmd := usecases.GetTicketQuery{
		TicketID: ticketID,
		UserID:   userID,
	}

	result, err := h.getTicketUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// ListTickets handles GET /tickets
func (h *TicketHandler) ListTickets(c *gin.Context) {
	req, err := parseListTicketsRequest(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}
	cmd := req.ToQuery(userID)

	result, err := h.listTicketsUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Tickets, result.TotalCount, req.Page, req.PageSize)
}

// AssignTicket handles POST /tickets/:id/assign
func (h *TicketHandler) AssignTicket(c *gin.Context) {
	ticketID, err := parseTicketID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req AssignTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}
	cmd := usecases.AssignTicketCommand{
		TicketID:   ticketID,
		AssigneeID: req.AssigneeID,
		AssignedBy: userID,
	}

	result, err := h.assignTicketUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Ticket assigned successfully", result)
}

// AddComment handles POST /tickets/:id/comments
func (h *TicketHandler) AddComment(c *gin.Context) {
	ticketID, err := parseTicketID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req AddCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}
	userRole := c.GetString(constants.ContextKeyUserRole)
	cmd := usecases.AddCommentCommand{
		TicketID:   ticketID,
		UserID:     userID,
		UserRoles:  []string{userRole},
		Content:    req.Content,
		IsInternal: req.IsInternal,
	}

	result, err := h.addCommentUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Comment added successfully")
}

// UpdateTicketStatusRequest represents a request for ticket status changes
type UpdateTicketStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=open in_progress resolved closed reopened"`
}

// UpdateTicketStatus handles PATCH /tickets/:id/status
func (h *TicketHandler) UpdateTicketStatus(c *gin.Context) {
	ticketID, err := parseTicketID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdateTicketStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update ticket status", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}
	userRole := c.GetString(constants.ContextKeyUserRole)

	// Map string status to vo.TicketStatus
	var newStatus vo.TicketStatus
	switch req.Status {
	case "open":
		newStatus = vo.StatusOpen
	case "in_progress":
		newStatus = vo.StatusInProgress
	case "resolved":
		newStatus = vo.StatusResolved
	case "closed":
		newStatus = vo.StatusClosed
	case "reopened":
		newStatus = vo.StatusReopened
	default:
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid status value")
		return
	}

	cmd := usecases.ChangeStatusCommand{
		TicketID:  ticketID,
		NewStatus: newStatus,
		ChangedBy: userID,
		UserRoles: []string{userRole},
	}

	result, err := h.changeStatusUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Ticket status updated successfully", result)
}

// DeleteTicket handles DELETE /tickets/:id
func (h *TicketHandler) DeleteTicket(c *gin.Context) {
	ticketID, err := parseTicketID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DeleteTicketCommand{
		TicketID: ticketID,
	}

	_, err = h.deleteTicketUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}

func parseTicketID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return 0, errors.NewValidationError("Invalid ticket ID")
	}
	return uint(id), nil
}
