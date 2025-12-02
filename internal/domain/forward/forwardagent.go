// Package forward provides domain models and business logic for forward agent management.
package forward

import (
	"crypto/subtle"
	"fmt"
	"time"

	"orris/internal/domain/shared/services"
)

// AgentStatus represents the status of a forward agent
type AgentStatus string

const (
	// AgentStatusEnabled indicates the agent is enabled
	AgentStatusEnabled AgentStatus = "enabled"
	// AgentStatusDisabled indicates the agent is disabled
	AgentStatusDisabled AgentStatus = "disabled"
)

// IsValid checks if the agent status is valid
func (s AgentStatus) IsValid() bool {
	return s == AgentStatusEnabled || s == AgentStatusDisabled
}

// ForwardAgent represents the forward agent aggregate root
type ForwardAgent struct {
	id             uint
	name           string
	tokenHash      string
	apiToken       string // transient field, only available after creation or regeneration
	status         AgentStatus
	remark         string
	createdAt      time.Time
	updatedAt      time.Time
	tokenGenerator services.TokenGenerator
}

// NewForwardAgent creates a new forward agent with auto-generated token
func NewForwardAgent(name string, remark string) (*ForwardAgent, error) {
	if name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	tokenGen := services.NewTokenGenerator()
	plainToken, tokenHash, err := tokenGen.GenerateAPIToken("fwd")
	if err != nil {
		return nil, fmt.Errorf("failed to generate API token: %w", err)
	}

	now := time.Now()
	agent := &ForwardAgent{
		name:           name,
		tokenHash:      tokenHash,
		apiToken:       plainToken,
		status:         AgentStatusEnabled,
		remark:         remark,
		createdAt:      now,
		updatedAt:      now,
		tokenGenerator: tokenGen,
	}

	return agent, nil
}

// ReconstructForwardAgent reconstructs a forward agent from persistence
func ReconstructForwardAgent(
	id uint,
	name string,
	tokenHash string,
	status AgentStatus,
	remark string,
	createdAt, updatedAt time.Time,
) (*ForwardAgent, error) {
	if id == 0 {
		return nil, fmt.Errorf("agent ID cannot be zero")
	}
	if name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if tokenHash == "" {
		return nil, fmt.Errorf("token hash is required")
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid agent status: %s", status)
	}

	return &ForwardAgent{
		id:             id,
		name:           name,
		tokenHash:      tokenHash,
		status:         status,
		remark:         remark,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
		tokenGenerator: services.NewTokenGenerator(),
	}, nil
}

// ID returns the agent ID
func (a *ForwardAgent) ID() uint {
	return a.id
}

// Name returns the agent name
func (a *ForwardAgent) Name() string {
	return a.name
}

// TokenHash returns the API token hash
func (a *ForwardAgent) TokenHash() string {
	return a.tokenHash
}

// Status returns the agent status
func (a *ForwardAgent) Status() AgentStatus {
	return a.status
}

// Remark returns the agent remark
func (a *ForwardAgent) Remark() string {
	return a.remark
}

// CreatedAt returns when the agent was created
func (a *ForwardAgent) CreatedAt() time.Time {
	return a.createdAt
}

// UpdatedAt returns when the agent was last updated
func (a *ForwardAgent) UpdatedAt() time.Time {
	return a.updatedAt
}

// SetID sets the agent ID (only for persistence layer use)
func (a *ForwardAgent) SetID(id uint) error {
	if a.id != 0 {
		return fmt.Errorf("agent ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("agent ID cannot be zero")
	}
	a.id = id
	return nil
}

// GenerateAPIToken generates a new API token and updates the token hash
func (a *ForwardAgent) GenerateAPIToken() (string, error) {
	if a.tokenGenerator == nil {
		a.tokenGenerator = services.NewTokenGenerator()
	}

	plainToken, tokenHash, err := a.tokenGenerator.GenerateAPIToken("fwd")
	if err != nil {
		return "", fmt.Errorf("failed to generate API token: %w", err)
	}

	a.apiToken = plainToken
	a.tokenHash = tokenHash
	a.updatedAt = time.Now()

	return plainToken, nil
}

// VerifyAPIToken verifies if the provided plain token matches the stored hash
func (a *ForwardAgent) VerifyAPIToken(plainToken string) bool {
	if a.tokenGenerator == nil {
		a.tokenGenerator = services.NewTokenGenerator()
	}
	computedHash := a.tokenGenerator.HashToken(plainToken)
	return subtle.ConstantTimeCompare([]byte(a.tokenHash), []byte(computedHash)) == 1
}

// GetAPIToken returns the plain API token (only available after creation or regeneration)
func (a *ForwardAgent) GetAPIToken() string {
	return a.apiToken
}

// ClearAPIToken clears the plain API token from memory
func (a *ForwardAgent) ClearAPIToken() {
	a.apiToken = ""
}

// Enable enables the forward agent
func (a *ForwardAgent) Enable() error {
	if a.status == AgentStatusEnabled {
		return nil
	}

	a.status = AgentStatusEnabled
	a.updatedAt = time.Now()

	return nil
}

// Disable disables the forward agent
func (a *ForwardAgent) Disable() error {
	if a.status == AgentStatusDisabled {
		return nil
	}

	a.status = AgentStatusDisabled
	a.updatedAt = time.Now()

	return nil
}

// UpdateName updates the agent name
func (a *ForwardAgent) UpdateName(name string) error {
	if name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}

	if a.name == name {
		return nil
	}

	a.name = name
	a.updatedAt = time.Now()

	return nil
}

// UpdateRemark updates the agent remark
func (a *ForwardAgent) UpdateRemark(remark string) error {
	if a.remark == remark {
		return nil
	}

	a.remark = remark
	a.updatedAt = time.Now()

	return nil
}

// IsEnabled checks if the agent is enabled
func (a *ForwardAgent) IsEnabled() bool {
	return a.status == AgentStatusEnabled
}

// Validate performs domain-level validation
func (a *ForwardAgent) Validate() error {
	if a.name == "" {
		return fmt.Errorf("agent name is required")
	}
	if a.tokenHash == "" {
		return fmt.Errorf("token hash is required")
	}
	if !a.status.IsValid() {
		return fmt.Errorf("invalid agent status: %s", a.status)
	}
	return nil
}
