package auth

import (
	"crypto/rand"
	"encoding/base64"
)

// AgentConnectionTokenService handles generation of
// short-term connection tokens for agent-to-agent tunnel establishment.
// Uses simple random tokens - security relies on the fact that
// only authenticated agents can obtain tokens from the API.
type AgentConnectionTokenService struct{}

// NewAgentConnectionTokenService creates a new AgentConnectionTokenService instance.
func NewAgentConnectionTokenService() *AgentConnectionTokenService {
	return &AgentConnectionTokenService{}
}

// Generate creates a new random connection token for agent-to-agent authentication.
// entryAgentID: the ID of the entry agent initiating the connection (unused, for interface compatibility).
// exitAgentID: the ID of the exit agent accepting the connection (unused, for interface compatibility).
// Returns a random token string or an error.
func (s *AgentConnectionTokenService) Generate(entryAgentID, exitAgentID string) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
