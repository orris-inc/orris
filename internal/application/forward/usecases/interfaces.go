package usecases

// AgentTokenGenerator generates and verifies HMAC-based agent tokens.
type AgentTokenGenerator interface {
	// Generate creates a token for the given agent short ID.
	// Returns the plain token and its hash (for storage).
	Generate(shortID string) (plainToken string, tokenHash string)

	// Verify validates a token and returns the agent short ID if valid.
	Verify(token string) (shortID string, err error)

	// HashToken computes SHA256 hash of a token (for storage compatibility).
	HashToken(token string) string
}
