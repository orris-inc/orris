// Package forward provides HTTP handlers for forward chain management.
package forward

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// ForwardChainHandler handles HTTP requests for forward chain management.
type ForwardChainHandler struct {
	createChainUC  *usecases.CreateForwardChainUseCase
	getChainUC     *usecases.GetForwardChainUseCase
	listChainsUC   *usecases.ListForwardChainsUseCase
	enableChainUC  *usecases.EnableForwardChainUseCase
	disableChainUC *usecases.DisableForwardChainUseCase
	deleteChainUC  *usecases.DeleteForwardChainUseCase
	logger         logger.Interface
}

// NewForwardChainHandler creates a new ForwardChainHandler.
func NewForwardChainHandler(
	createChainUC *usecases.CreateForwardChainUseCase,
	getChainUC *usecases.GetForwardChainUseCase,
	listChainsUC *usecases.ListForwardChainsUseCase,
	enableChainUC *usecases.EnableForwardChainUseCase,
	disableChainUC *usecases.DisableForwardChainUseCase,
	deleteChainUC *usecases.DeleteForwardChainUseCase,
) *ForwardChainHandler {
	return &ForwardChainHandler{
		createChainUC:  createChainUC,
		getChainUC:     getChainUC,
		listChainsUC:   listChainsUC,
		enableChainUC:  enableChainUC,
		disableChainUC: disableChainUC,
		deleteChainUC:  deleteChainUC,
		logger:         logger.NewLogger(),
	}
}

// CreateForwardChainNodeRequest represents a node in the chain creation request.
type CreateForwardChainNodeRequest struct {
	AgentID    uint   `json:"agent_id" binding:"required" example:"1"`
	ListenPort uint16 `json:"listen_port" binding:"required" example:"8080"`
}

// CreateForwardChainRequest represents a request to create a forward chain.
type CreateForwardChainRequest struct {
	Name          string                          `json:"name" binding:"required" example:"Production Chain"`
	Protocol      string                          `json:"protocol" binding:"required,oneof=tcp udp both" example:"tcp"`
	Nodes         []CreateForwardChainNodeRequest `json:"nodes" binding:"required,min=1"`
	TargetAddress string                          `json:"target_address" binding:"required" example:"192.168.1.100"`
	TargetPort    uint16                          `json:"target_port" binding:"required" example:"3306"`
	Remark        string                          `json:"remark" example:"Database forwarding chain"`
}

// UpdateChainStatusRequest represents a request to update forward chain status.
type UpdateChainStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=enabled disabled" example:"enabled"`
}

// CreateChain handles POST /forward-chains
func (h *ForwardChainHandler) CreateChain(c *gin.Context) {
	var req CreateForwardChainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create forward chain", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Convert request nodes to command nodes
	nodes := make([]usecases.CreateForwardChainNodeInput, len(req.Nodes))
	for i, n := range req.Nodes {
		nodes[i] = usecases.CreateForwardChainNodeInput{
			AgentID:    n.AgentID,
			ListenPort: n.ListenPort,
		}
	}

	cmd := usecases.CreateForwardChainCommand{
		Name:          req.Name,
		Protocol:      req.Protocol,
		Nodes:         nodes,
		TargetAddress: req.TargetAddress,
		TargetPort:    req.TargetPort,
		Remark:        req.Remark,
	}

	result, err := h.createChainUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Forward chain created successfully")
}

// GetChain handles GET /forward-chains/:id
func (h *ForwardChainHandler) GetChain(c *gin.Context) {
	chainID, err := parseChainID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.getChainUC.Execute(c.Request.Context(), chainID)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// ListChains handles GET /forward-chains
func (h *ForwardChainHandler) ListChains(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := usecases.ListForwardChainsQuery{
		Page:     page,
		PageSize: pageSize,
		Name:     c.Query("name"),
		Status:   c.Query("status"),
		OrderBy:  c.Query("order_by"),
		Order:    c.Query("order"),
	}

	result, err := h.listChainsUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Chains, result.Total, page, pageSize)
}

// DeleteChain handles DELETE /forward-chains/:id
func (h *ForwardChainHandler) DeleteChain(c *gin.Context) {
	chainID, err := parseChainID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := h.deleteChainUC.Execute(c.Request.Context(), chainID); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}

// EnableChain handles POST /forward-chains/:id/enable
func (h *ForwardChainHandler) EnableChain(c *gin.Context) {
	chainID, err := parseChainID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := h.enableChainUC.Execute(c.Request.Context(), chainID); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward chain enabled successfully", nil)
}

// DisableChain handles POST /forward-chains/:id/disable
func (h *ForwardChainHandler) DisableChain(c *gin.Context) {
	chainID, err := parseChainID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := h.disableChainUC.Execute(c.Request.Context(), chainID); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward chain disabled successfully", nil)
}

// UpdateStatus handles PATCH /forward-chains/:id/status
func (h *ForwardChainHandler) UpdateStatus(c *gin.Context) {
	var req UpdateChainStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update chain status", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	if req.Status == "enabled" {
		h.EnableChain(c)
	} else {
		h.DisableChain(c)
	}
}

func parseChainID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid forward chain ID")
	}
	if id == 0 {
		return 0, errors.NewValidationError("Forward chain ID must be greater than 0")
	}
	return uint(id), nil
}
