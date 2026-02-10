package middleware

import (
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	// HeaderAPIVersion is the custom header for API version negotiation.
	HeaderAPIVersion = "X-API-Version"

	// ContextKeyAPIVersion is the Gin context key for the resolved API version.
	ContextKeyAPIVersion = "api_version"

	// CurrentAPIVersion is the latest supported API version.
	CurrentAPIVersion = 1

	// MinAPIVersion is the minimum supported API version.
	MinAPIVersion = 1
)

// acceptVersionRegex matches Accept header like "application/vnd.orris.v1+json".
var acceptVersionRegex = regexp.MustCompile(`application/vnd\.orris\.v(\d+)\+json`)

// APIVersion returns a middleware that extracts the API version from request headers
// and sets it in the Gin context. It checks two sources in order:
//  1. X-API-Version header (e.g., "1")
//  2. Accept header (e.g., "application/vnd.orris.v1+json")
//
// If neither is present, the current version is used as default.
// The resolved version is echoed back via the X-API-Version response header.
func APIVersion() gin.HandlerFunc {
	return func(c *gin.Context) {
		version := resolveAPIVersion(c)
		c.Set(ContextKeyAPIVersion, version)
		c.Header(HeaderAPIVersion, strconv.Itoa(version))
		c.Next()
	}
}

// GetAPIVersion returns the API version from the Gin context.
// Returns CurrentAPIVersion if not set.
func GetAPIVersion(c *gin.Context) int {
	if v, exists := c.Get(ContextKeyAPIVersion); exists {
		if ver, ok := v.(int); ok {
			return ver
		}
	}
	return CurrentAPIVersion
}

// resolveAPIVersion extracts the API version from request headers.
func resolveAPIVersion(c *gin.Context) int {
	// Priority 1: X-API-Version header
	if h := c.GetHeader(HeaderAPIVersion); h != "" {
		if v, err := strconv.Atoi(h); err == nil && v >= MinAPIVersion && v <= CurrentAPIVersion {
			return v
		}
	}

	// Priority 2: Accept header with vendor media type
	if accept := c.GetHeader("Accept"); accept != "" {
		if matches := acceptVersionRegex.FindStringSubmatch(accept); len(matches) == 2 {
			if v, err := strconv.Atoi(matches[1]); err == nil && v >= MinAPIVersion && v <= CurrentAPIVersion {
				return v
			}
		}
	}

	// Default to current version
	return CurrentAPIVersion
}
