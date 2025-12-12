package forward

import (
	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
)

// parseRuleShortID extracts the short ID from a prefixed rule ID (e.g., "fr_xK9mP2vL3nQ" -> "xK9mP2vL3nQ").
func parseRuleShortID(c *gin.Context) (string, error) {
	prefixedID := c.Param("id")
	if prefixedID == "" {
		return "", errors.NewValidationError("forward rule ID is required")
	}

	shortID, err := id.ParseForwardRuleID(prefixedID)
	if err != nil {
		return "", errors.NewValidationError("invalid forward rule ID format, expected fr_xxxxx")
	}

	return shortID, nil
}
