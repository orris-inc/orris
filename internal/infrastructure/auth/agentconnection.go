package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AgentConnectionClaims represents JWT claims for agent-to-agent connection authentication.
// Used for short-term connection tokens between entry and exit agents.
type AgentConnectionClaims struct {
	EntryAgentID string `json:"entry_agent_id"` // Stripe-style ID (fa_xxx)
	ExitAgentID  string `json:"exit_agent_id"`  // Stripe-style ID (fa_xxx)
	jwt.RegisteredClaims
}

// AgentConnectionTokenService handles generation and verification of
// short-term connection tokens for agent-to-agent tunnel establishment.
type AgentConnectionTokenService struct {
	secret     []byte
	expMinutes int
}

// NewAgentConnectionTokenService creates a new AgentConnectionTokenService instance.
// secret: the secret key used for signing tokens (should match JWTService secret).
// expMinutes: token expiration time in minutes (recommended: 5).
func NewAgentConnectionTokenService(secret string, expMinutes int) *AgentConnectionTokenService {
	return &AgentConnectionTokenService{
		secret:     []byte(secret),
		expMinutes: expMinutes,
	}
}

// Generate creates a new connection token for agent-to-agent authentication.
// entryAgentID: the ID of the entry agent initiating the connection.
// exitAgentID: the ID of the exit agent accepting the connection.
// Returns the signed JWT token string or an error.
func (s *AgentConnectionTokenService) Generate(entryAgentID, exitAgentID string) (string, error) {
	now := time.Now()
	exp := now.Add(time.Duration(s.expMinutes) * time.Minute)

	claims := &AgentConnectionClaims{
		EntryAgentID: entryAgentID,
		ExitAgentID:  exitAgentID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign connection token: %w", err)
	}

	return tokenString, nil
}

// Verify validates a connection token and returns its claims.
// tokenString: the JWT token string to verify.
// Returns the claims if valid, or an error if invalid/expired.
func (s *AgentConnectionTokenService) Verify(tokenString string) (*AgentConnectionClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AgentConnectionClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse connection token: %w", err)
	}

	if claims, ok := token.Claims.(*AgentConnectionClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid connection token")
}
