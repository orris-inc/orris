package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// SubscriptionURLSigner handles subscription URL signing and verification
type SubscriptionURLSigner struct {
	secretKey []byte
	logger    logger.Interface
}

// NewSubscriptionURLSigner creates a new subscription URL signer
func NewSubscriptionURLSigner(secretKey string, logger logger.Interface) *SubscriptionURLSigner {
	return &SubscriptionURLSigner{
		secretKey: []byte(secretKey),
		logger:    logger,
	}
}

// GenerateURL generates a signed subscription URL
// Parameters:
//   - subscriptionID: the subscription ID
//   - expiresAt: expiration time of the URL
//   - baseURL: base URL for the subscription endpoint
//
// Returns the complete signed URL
func (s *SubscriptionURLSigner) GenerateURL(subscriptionID uint, expiresAt time.Time, baseURL string) string {
	timestamp := expiresAt.Unix()
	message := fmt.Sprintf("%d:%d", subscriptionID, timestamp)

	signature := s.sign(message)

	return fmt.Sprintf("%s?subscription_id=%d&expires=%d&signature=%s",
		baseURL,
		subscriptionID,
		timestamp,
		signature,
	)
}

// Verify verifies the signature of a subscription URL
// Parameters:
//   - subscriptionID: the subscription ID from URL
//   - expiresAt: expiration timestamp from URL
//   - signature: signature from URL
//
// Returns true if signature is valid and not expired
func (s *SubscriptionURLSigner) Verify(subscriptionID uint, expiresAt int64, signature string) bool {
	// Check if expired
	if time.Now().Unix() > expiresAt {
		s.logger.Warnw("subscription URL has expired",
			"subscription_id", subscriptionID,
			"expires_at", expiresAt,
		)
		return false
	}

	// Verify signature
	message := fmt.Sprintf("%d:%d", subscriptionID, expiresAt)
	expectedSignature := s.sign(message)

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) == 1
}

// sign generates HMAC-SHA256 signature for the given message
func (s *SubscriptionURLSigner) sign(message string) string {
	mac := hmac.New(sha256.New, s.secretKey)
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// SubscriptionSecurityMiddleware is a Gin middleware that verifies subscription URL signatures
type SubscriptionSecurityMiddleware struct {
	signer *SubscriptionURLSigner
	logger logger.Interface
}

// NewSubscriptionSecurityMiddleware creates a new subscription security middleware
func NewSubscriptionSecurityMiddleware(
	signer *SubscriptionURLSigner,
	logger logger.Interface,
) *SubscriptionSecurityMiddleware {
	return &SubscriptionSecurityMiddleware{
		signer: signer,
		logger: logger,
	}
}

// VerifySignature is a Gin middleware that verifies subscription URL signatures
func (m *SubscriptionSecurityMiddleware) VerifySignature() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract parameters from query string
		subscriptionIDStr := c.Query("subscription_id")
		expiresStr := c.Query("expires")
		signature := c.Query("signature")

		// Validate parameters presence
		if subscriptionIDStr == "" || expiresStr == "" || signature == "" {
			m.logger.Warnw("missing required parameters for subscription URL verification",
				"ip", c.ClientIP(),
				"path", c.Request.URL.Path,
			)
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription URL")
			c.Abort()
			return
		}

		// Parse subscription ID
		subscriptionID, err := strconv.ParseUint(subscriptionIDStr, 10, 32)
		if err != nil {
			m.logger.Warnw("invalid subscription ID format",
				"subscription_id", subscriptionIDStr,
				"error", err,
			)
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
			c.Abort()
			return
		}

		// Parse expiration timestamp
		expiresAt, err := strconv.ParseInt(expiresStr, 10, 64)
		if err != nil {
			m.logger.Warnw("invalid expires format",
				"expires", expiresStr,
				"error", err,
			)
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid expiration time")
			c.Abort()
			return
		}

		// Verify signature and expiration
		if !m.signer.Verify(uint(subscriptionID), expiresAt, signature) {
			m.logger.Warnw("subscription URL verification failed",
				"subscription_id", subscriptionID,
				"expires_at", expiresAt,
				"ip", c.ClientIP(),
			)
			utils.ErrorResponse(c, http.StatusUnauthorized, "invalid or expired subscription URL")
			c.Abort()
			return
		}

		// Store subscription ID in context for downstream handlers
		c.Set("subscription_id", uint(subscriptionID))

		c.Next()
	}
}

// PreventReplay is a Gin middleware that prevents replay attacks using timestamp validation
// maxAge defines the maximum age of a request in seconds (e.g., 300 for 5 minutes)
func (m *SubscriptionSecurityMiddleware) PreventReplay(maxAge int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		expiresStr := c.Query("expires")
		if expiresStr == "" {
			m.logger.Warnw("missing expires parameter for replay prevention",
				"ip", c.ClientIP(),
			)
			utils.ErrorResponse(c, http.StatusBadRequest, "missing expiration time")
			c.Abort()
			return
		}

		expiresAt, err := strconv.ParseInt(expiresStr, 10, 64)
		if err != nil {
			m.logger.Warnw("invalid expires format for replay prevention",
				"expires", expiresStr,
				"error", err,
			)
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid expiration time")
			c.Abort()
			return
		}

		now := time.Now().Unix()

		// Check if URL has expired
		if now > expiresAt {
			m.logger.Warnw("subscription URL has expired",
				"expires_at", expiresAt,
				"current_time", now,
				"ip", c.ClientIP(),
			)
			utils.ErrorResponse(c, http.StatusUnauthorized, "subscription URL has expired")
			c.Abort()
			return
		}

		// Check if URL is too old (potential replay attack)
		age := expiresAt - now
		if age > maxAge {
			m.logger.Warnw("subscription URL timestamp is too far in the future",
				"expires_at", expiresAt,
				"current_time", now,
				"age", age,
				"max_age", maxAge,
				"ip", c.ClientIP(),
			)
			utils.ErrorResponse(c, http.StatusUnauthorized, "invalid timestamp")
			c.Abort()
			return
		}

		c.Next()
	}
}
