package ticket

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"orris/internal/application/ticket/usecases"
	vo "orris/internal/domain/ticket/value_objects"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
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
		logger:           logger.NewLogger(),
	}
}

// CreateTicket handles POST /tickets
//
//	@Summary		Create a new ticket
//	@Description	Create a new support ticket
//	@Tags			tickets
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			ticket	body		CreateTicketRequest	true	"Ticket data"
//	@Success		201		{object}	utils.APIResponse	"Ticket created successfully"
//	@Failure		400		{object}	utils.APIResponse
//	@Failure		401		{object}	utils.APIResponse
//	@Failure		500		{object}	utils.APIResponse	"Internal server error"
//	@Router			/tickets [post]
func (h *TicketHandler) CreateTicket(c *gin.Context) {
	var req CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create ticket", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, _ := c.Get("user_id")
	cmd := req.ToCommand(userID.(uint))

	result, err := h.createTicketUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Ticket created successfully")
}

// GetTicket handles GET /tickets/:id
//
//	@Summary		Get ticket by ID
//	@Description	Get details of a ticket
//	@Tags			tickets
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int					true	"Ticket ID"
//	@Success		200	{object}	utils.APIResponse	"Ticket details"
//	@Failure		400	{object}	utils.APIResponse
//	@Failure		404	{object}	utils.APIResponse
//	@Failure		500	{object}	utils.APIResponse	"Internal server error"
//	@Router			/tickets/{id} [get]
func (h *TicketHandler) GetTicket(c *gin.Context) {
	ticketID, err := parseTicketID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, _ := c.Get("user_id")
	cmd := usecases.GetTicketQuery{
		TicketID: ticketID,
		UserID:   userID.(uint),
	}

	result, err := h.getTicketUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// ListTickets handles GET /tickets
//
//	@Summary		List tickets
//	@Description	Get a paginated list of tickets
//	@Tags			tickets
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			page		query		int					false	"Page number"		default(1)
//	@Param			page_size	query		int					false	"Page size"			default(20)
//	@Param			status		query		string				false	"Status filter"		Enums(open,in_progress,resolved,closed,reopened)
//	@Param			priority	query		string				false	"Priority filter"	Enums(low,normal,high,urgent,critical)
//	@Param			category	query		string				false	"Category filter"	Enums(technical,billing,account,general,feature_request,bug_report)
//	@Success		200			{object}	utils.APIResponse	"Tickets list"
//	@Failure		400			{object}	utils.APIResponse
//	@Failure		500			{object}	utils.APIResponse	"Internal server error"
//	@Router			/tickets [get]
func (h *TicketHandler) ListTickets(c *gin.Context) {
	req, err := parseListTicketsRequest(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, _ := c.Get("user_id")
	cmd := req.ToQuery(userID.(uint))

	result, err := h.listTicketsUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Tickets, result.TotalCount, req.Page, req.PageSize)
}

// AssignTicket handles POST /tickets/:id/assign
//
//	@Summary		Assign ticket
//	@Description	Assign a ticket to an agent
//	@Tags			tickets
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id		path		int					true	"Ticket ID"
//	@Param			body	body		AssignTicketRequest	true	"Assignment data"
//	@Success		200		{object}	utils.APIResponse	"Ticket assigned successfully"
//	@Failure		400		{object}	utils.APIResponse	"Bad request"
//	@Failure		401		{object}	utils.APIResponse	"Unauthorized"
//	@Failure		403		{object}	utils.APIResponse	"Forbidden - Requires admin role"
//	@Failure		404		{object}	utils.APIResponse	"Ticket not found"
//	@Failure		500		{object}	utils.APIResponse	"Internal server error"
//	@Router			/tickets/{id}/assign [post]
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

	userID, _ := c.Get("user_id")
	cmd := usecases.AssignTicketCommand{
		TicketID:   ticketID,
		AssigneeID: req.AssigneeID,
		AssignedBy: userID.(uint),
	}

	result, err := h.assignTicketUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Ticket assigned successfully", result)
}

// AddComment handles POST /tickets/:id/comments
//
//	@Summary		Add comment
//	@Description	Add a comment to a ticket
//	@Tags			tickets
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id		path		int					true	"Ticket ID"
//	@Param			body	body		AddCommentRequest	true	"Comment data"
//	@Success		201		{object}	utils.APIResponse	"Comment added successfully"
//	@Failure		400		{object}	utils.APIResponse
//	@Failure		500		{object}	utils.APIResponse	"Internal server error"
//	@Router			/tickets/{id}/comments [post]
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

	userID, _ := c.Get("user_id")
	cmd := usecases.AddCommentCommand{
		TicketID:   ticketID,
		UserID:     userID.(uint),
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
//
//	@Summary		Update ticket status
//	@Description	Update ticket status (open, in_progress, resolved, closed, or reopened)
//	@Tags			tickets
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id		path		int							true	"Ticket ID"
//	@Param			status	body		UpdateTicketStatusRequest	true	"Status update details"
//	@Success		200		{object}	utils.APIResponse			"Ticket status updated successfully"
//	@Failure		400		{object}	utils.APIResponse			"Bad request"
//	@Failure		401		{object}	utils.APIResponse			"Unauthorized"
//	@Failure		404		{object}	utils.APIResponse			"Ticket not found"
//	@Failure		500		{object}	utils.APIResponse			"Internal server error"
//	@Router			/tickets/{id}/status [patch]
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

	userID, _ := c.Get("user_id")

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
		ChangedBy: userID.(uint),
	}

	result, err := h.changeStatusUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Ticket status updated successfully", result)
}

// DeleteTicket handles DELETE /tickets/:id
//
//	@Summary		Delete ticket
//	@Description	Delete a ticket (admin only)
//	@Tags			tickets
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path	int	true	"Ticket ID"
//	@Success		204
//	@Failure		400	{object}	utils.APIResponse	"Bad request"
//	@Failure		401	{object}	utils.APIResponse	"Unauthorized"
//	@Failure		403	{object}	utils.APIResponse	"Forbidden - Requires admin role"
//	@Failure		404	{object}	utils.APIResponse	"Ticket not found"
//	@Failure		500	{object}	utils.APIResponse	"Internal server error"
//	@Router			/tickets/{id} [delete]
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
