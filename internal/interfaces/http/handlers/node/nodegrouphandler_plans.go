package node

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// AssociatePlan handles POST /node-groups/:id/plans
func (h *NodeGroupHandler) AssociatePlan(c *gin.Context) {
	groupID, err := parseNodeGroupID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req AssociatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for associate plan",
			"group_id", groupID,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.AssociateGroupWithPlanCommand{
		GroupID: groupID,
		PlanID:  req.PlanID,
	}

	result, err := h.associateGroupWithPlanUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Plan associated successfully", result)
}

// DisassociatePlan handles DELETE /node-groups/:id/plans/:planId
func (h *NodeGroupHandler) DisassociatePlan(c *gin.Context) {
	h.logger.Infow("received disassociate plan request",
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
	)

	groupID, err := parseNodeGroupID(c)
	if err != nil {
		h.logger.Warnw("failed to parse group ID",
			"error", err,
			"param", c.Param("id"),
		)
		utils.ErrorResponseWithError(c, err)
		return
	}

	planID, err := parsePlanIDFromParam(c, "plan_id")
	if err != nil {
		h.logger.Warnw("failed to parse plan ID",
			"error", err,
			"param", c.Param("plan_id"),
			"group_id", groupID,
		)
		utils.ErrorResponseWithError(c, err)
		return
	}

	h.logger.Infow("parsed disassociate plan request parameters",
		"group_id", groupID,
		"plan_id", planID,
	)

	cmd := usecases.DisassociateGroupFromPlanCommand{
		GroupID: groupID,
		PlanID:  planID,
	}

	_, err = h.disassociateGroupFromPlanUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("failed to disassociate plan from group",
			"error", err,
			"group_id", groupID,
			"plan_id", planID,
		)
		utils.ErrorResponseWithError(c, err)
		return
	}

	h.logger.Infow("successfully disassociated plan from group",
		"group_id", groupID,
		"plan_id", planID,
	)

	utils.NoContentResponse(c)
}
