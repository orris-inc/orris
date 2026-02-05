package utils

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
)

// ParseSIDParam parses and validates a Stripe-style prefixed ID from a URL path parameter.
// paramName is the Gin route parameter name (e.g., "id", "rule_id").
// prefix is the expected SID prefix (e.g., id.PrefixNode).
// entityName is used in error messages (e.g., "node", "forward rule").
func ParseSIDParam(c *gin.Context, paramName, prefix, entityName string) (string, error) {
	sid := c.Param(paramName)
	if sid == "" {
		return "", errors.NewValidationError(entityName + " ID is required")
	}

	if err := id.ValidatePrefix(sid, prefix); err != nil {
		return "", errors.NewValidationError(
			fmt.Sprintf("invalid %s ID format, expected %s_xxxxx", entityName, prefix),
		)
	}

	return sid, nil
}
