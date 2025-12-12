package node

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/errors"
)

// parseNodeGroupID extracts and validates the node group ID from URL parameter
func parseNodeGroupID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid node group ID")
	}
	if id == 0 {
		return 0, errors.NewValidationError("Node group ID must be greater than 0")
	}
	return uint(id), nil
}

// parsePlanIDFromParam extracts and validates a plan ID from URL parameter
func parsePlanIDFromParam(c *gin.Context, paramName string) (uint, error) {
	idStr := c.Param(paramName)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid plan ID")
	}
	if id == 0 {
		return 0, errors.NewValidationError("Plan ID must be greater than 0")
	}
	return uint(id), nil
}

// parseListNodeGroupsRequest extracts and validates query parameters for listing node groups
func parseListNodeGroupsRequest(c *gin.Context) (*ListNodeGroupsRequest, error) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	req := &ListNodeGroupsRequest{
		Page:     page,
		PageSize: pageSize,
	}

	if isPublicStr := c.Query("is_public"); isPublicStr != "" {
		isPublic := isPublicStr == "true"
		req.IsPublic = &isPublic
	}

	return req, nil
}
