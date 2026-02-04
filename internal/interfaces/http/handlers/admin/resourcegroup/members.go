package resourcegroup

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// AddNodes adds nodes to a resource group by SID (Stripe-style ID: rg_xxx)
func (h *Handler) AddNodes(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	var req dto.AddNodesToGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for add nodes", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.manageNodesUseCase.AddNodesBySID(c.Request.Context(), sid, req.NodeSIDs)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		if err == resource.ErrGroupPlanTypeMismatchNode {
			utils.ErrorResponse(c, http.StatusBadRequest, "resource group's plan type is not node, cannot add node resources")
			return
		}
		h.logger.Errorw("failed to add nodes to resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Nodes added to resource group", result)
}

// RemoveNodes removes nodes from a resource group by SID (Stripe-style ID: rg_xxx)
func (h *Handler) RemoveNodes(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	var req dto.RemoveNodesFromGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for remove nodes", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.manageNodesUseCase.RemoveNodesBySID(c.Request.Context(), sid, req.NodeSIDs)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to remove nodes from resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Nodes removed from resource group", result)
}

// ListNodes lists all nodes in a resource group by SID (Stripe-style ID: rg_xxx)
func (h *Handler) ListNodes(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	var req dto.ListGroupMembersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warnw("invalid request query for list nodes", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.manageNodesUseCase.ListNodesBySID(c.Request.Context(), sid, req.Page, req.PageSize)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to list nodes in resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Items, result.Total, result.Page, result.PageSize)
}

// AddForwardAgents adds forward agents to a resource group by SID (Stripe-style ID: rg_xxx)
func (h *Handler) AddForwardAgents(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	var req dto.AddForwardAgentsToGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for add forward agents", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.manageAgentsUseCase.AddAgentsBySID(c.Request.Context(), sid, req.AgentSIDs)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to add forward agents to resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward agents added to resource group", result)
}

// RemoveForwardAgents removes forward agents from a resource group by SID (Stripe-style ID: rg_xxx)
func (h *Handler) RemoveForwardAgents(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	var req dto.RemoveForwardAgentsFromGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for remove forward agents", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.manageAgentsUseCase.RemoveAgentsBySID(c.Request.Context(), sid, req.AgentSIDs)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to remove forward agents from resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward agents removed from resource group", result)
}

// ListForwardAgents lists all forward agents in a resource group by SID (Stripe-style ID: rg_xxx)
func (h *Handler) ListForwardAgents(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	var req dto.ListGroupMembersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warnw("invalid request query for list forward agents", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.manageAgentsUseCase.ListAgentsBySID(c.Request.Context(), sid, req.Page, req.PageSize)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to list forward agents in resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Items, result.Total, result.Page, result.PageSize)
}

// AddForwardRules adds forward rules to a resource group by SID (Stripe-style ID: rg_xxx)
func (h *Handler) AddForwardRules(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	var req dto.AddForwardRulesToGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for add forward rules", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.manageRulesUseCase.AddRulesBySID(c.Request.Context(), sid, req.RuleSIDs)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		if err == resource.ErrGroupPlanTypeMismatchForward {
			utils.ErrorResponse(c, http.StatusBadRequest, "resource group's plan type is not forward, cannot add forward rule resources")
			return
		}
		h.logger.Errorw("failed to add forward rules to resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward rules added to resource group", result)
}

// RemoveForwardRules removes forward rules from a resource group by SID (Stripe-style ID: rg_xxx)
func (h *Handler) RemoveForwardRules(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	var req dto.RemoveForwardRulesFromGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for remove forward rules", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.manageRulesUseCase.RemoveRulesBySID(c.Request.Context(), sid, req.RuleSIDs)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to remove forward rules from resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward rules removed from resource group", result)
}

// ListForwardRules lists all forward rules in a resource group by SID (Stripe-style ID: rg_xxx)
func (h *Handler) ListForwardRules(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	var req dto.ListGroupRulesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warnw("invalid request query for list forward rules", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.manageRulesUseCase.ListRulesBySID(c.Request.Context(), sid, req.Page, req.PageSize, req.OrderBy, req.Order)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to list forward rules in resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Items, result.Total, result.Page, result.PageSize)
}
