// Package forward provides domain models and business logic for forward agent management.
package forward

import (
	"crypto/subtle"
	"fmt"
	"net"
	"regexp"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/domain/shared/services"
	"github.com/orris-inc/orris/internal/shared/biztime"
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

// domainNameRegex is a pre-compiled regex for validating RFC 1123 hostnames
var domainNameRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

// ForwardAgent represents the forward agent aggregate root.
// An agent can participate in multiple rules with different roles simultaneously:
// - As entry in rule A, exit in rule B, relay in rule C, etc.
// The role is determined by the rule configuration, not the agent itself.
type ForwardAgent struct {
	id               uint
	sid              string // Stripe-style prefixed ID (fa_xxx)
	name             string
	tokenHash        string
	apiToken         string // stored token for retrieval
	status           AgentStatus
	publicAddress    string // optional public address for Entry to obtain Exit connection information
	tunnelAddress    string // IP or hostname only (no port), configure if agent may serve as relay/exit in any rule
	remark           string
	groupID          *uint         // resource group ID
	agentVersion     string        // agent software version (e.g., "1.2.3")
	platform         string        // OS platform (linux, darwin, windows)
	arch             string        // CPU architecture (amd64, arm64, arm, 386)
	allowedPortRange *vo.PortRange // allowed listen port range (nil means all ports allowed)
	createdAt        time.Time
	updatedAt        time.Time
	tokenGenerator   services.TokenGenerator
}

// TokenGenerator is a function that generates a token for a given short ID.
// Returns (plainToken, tokenHash).
type TokenGenerator func(shortID string) (string, string)

// NewForwardAgent creates a new forward agent with the given token generator.
func NewForwardAgent(name string, publicAddress string, tunnelAddress string, remark string, sidGenerator func() (string, error), tokenGenerator TokenGenerator) (*ForwardAgent, error) {
	if name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	// Validate public address if provided
	if publicAddress != "" {
		if err := validatePublicAddress(publicAddress); err != nil {
			return nil, err
		}
	}

	// Validate tunnel address if provided
	if tunnelAddress != "" {
		if err := validateTunnelAddress(tunnelAddress); err != nil {
			return nil, err
		}
	}

	// Generate SID for external API use
	sid, err := sidGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SID: %w", err)
	}

	// Generate token using the provided generator
	plainToken, tokenHash := tokenGenerator(sid)

	now := biztime.NowUTC()
	agent := &ForwardAgent{
		sid:            sid,
		name:           name,
		tokenHash:      tokenHash,
		apiToken:       plainToken,
		status:         AgentStatusEnabled,
		publicAddress:  publicAddress,
		tunnelAddress:  tunnelAddress,
		remark:         remark,
		createdAt:      now,
		updatedAt:      now,
		tokenGenerator: services.NewTokenGenerator(),
	}

	return agent, nil
}

// ReconstructForwardAgent reconstructs a forward agent from persistence
func ReconstructForwardAgent(
	id uint,
	sid string,
	name string,
	tokenHash string,
	apiToken string,
	status AgentStatus,
	publicAddress string,
	tunnelAddress string,
	remark string,
	groupID *uint,
	agentVersion string,
	platform string,
	arch string,
	allowedPortRange *vo.PortRange,
	createdAt, updatedAt time.Time,
) (*ForwardAgent, error) {
	if id == 0 {
		return nil, fmt.Errorf("agent ID cannot be zero")
	}
	if sid == "" {
		return nil, fmt.Errorf("agent SID is required")
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

	// Validate public address if provided
	if publicAddress != "" {
		if err := validatePublicAddress(publicAddress); err != nil {
			return nil, err
		}
	}

	// Validate tunnel address if provided
	if tunnelAddress != "" {
		if err := validateTunnelAddress(tunnelAddress); err != nil {
			return nil, err
		}
	}

	return &ForwardAgent{
		id:               id,
		sid:              sid,
		name:             name,
		tokenHash:        tokenHash,
		apiToken:         apiToken,
		status:           status,
		publicAddress:    publicAddress,
		tunnelAddress:    tunnelAddress,
		remark:           remark,
		groupID:          groupID,
		agentVersion:     agentVersion,
		platform:         platform,
		arch:             arch,
		allowedPortRange: allowedPortRange,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
		tokenGenerator:   services.NewTokenGenerator(),
	}, nil
}

// ID returns the agent ID
func (a *ForwardAgent) ID() uint {
	return a.id
}

// SID returns the Stripe-style prefixed ID (fa_xxx)
func (a *ForwardAgent) SID() string {
	return a.sid
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

// PublicAddress returns the agent's public address
func (a *ForwardAgent) PublicAddress() string {
	return a.publicAddress
}

// TunnelAddress returns the agent's tunnel address
func (a *ForwardAgent) TunnelAddress() string {
	return a.tunnelAddress
}

// GetEffectiveTunnelAddress returns the address that entry agents should use to connect.
// If tunnelAddress is set, it returns tunnelAddress; otherwise returns publicAddress.
func (a *ForwardAgent) GetEffectiveTunnelAddress() string {
	if a.tunnelAddress != "" {
		return a.tunnelAddress
	}
	return a.publicAddress
}

// GroupID returns the resource group ID
func (a *ForwardAgent) GroupID() *uint {
	return a.groupID
}

// SetGroupID sets the resource group ID
func (a *ForwardAgent) SetGroupID(groupID *uint) {
	a.groupID = groupID
	a.updatedAt = biztime.NowUTC()
}

// AgentVersion returns the agent software version
func (a *ForwardAgent) AgentVersion() string {
	return a.agentVersion
}

// Platform returns the OS platform
func (a *ForwardAgent) Platform() string {
	return a.platform
}

// Arch returns the CPU architecture
func (a *ForwardAgent) Arch() string {
	return a.arch
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

// SetAPIToken sets a new API token and updates the token hash.
// This should be called by use cases with a token generated by AgentTokenService.
func (a *ForwardAgent) SetAPIToken(plainToken, tokenHash string) {
	a.apiToken = plainToken
	a.tokenHash = tokenHash
	a.updatedAt = biztime.NowUTC()
}

// VerifyAPIToken verifies if the provided plain token matches the stored hash
func (a *ForwardAgent) VerifyAPIToken(plainToken string) bool {
	if a.tokenGenerator == nil {
		a.tokenGenerator = services.NewTokenGenerator()
	}
	computedHash := a.tokenGenerator.HashToken(plainToken)
	return subtle.ConstantTimeCompare([]byte(a.tokenHash), []byte(computedHash)) == 1
}

// GetAPIToken returns the plain API token
func (a *ForwardAgent) GetAPIToken() string {
	return a.apiToken
}

// HasToken returns true if the agent has a stored token
func (a *ForwardAgent) HasToken() bool {
	return a.apiToken != ""
}

// Enable enables the forward agent
func (a *ForwardAgent) Enable() error {
	if a.status == AgentStatusEnabled {
		return nil
	}

	a.status = AgentStatusEnabled
	a.updatedAt = biztime.NowUTC()

	return nil
}

// Disable disables the forward agent
func (a *ForwardAgent) Disable() error {
	if a.status == AgentStatusDisabled {
		return nil
	}

	a.status = AgentStatusDisabled
	a.updatedAt = biztime.NowUTC()

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
	a.updatedAt = biztime.NowUTC()

	return nil
}

// UpdateRemark updates the agent remark
func (a *ForwardAgent) UpdateRemark(remark string) error {
	if a.remark == remark {
		return nil
	}

	a.remark = remark
	a.updatedAt = biztime.NowUTC()

	return nil
}

// UpdatePublicAddress updates the agent's public address
func (a *ForwardAgent) UpdatePublicAddress(address string) error {
	// Validate address if not empty
	if address != "" {
		if err := validatePublicAddress(address); err != nil {
			return err
		}
	}

	if a.publicAddress == address {
		return nil
	}

	a.publicAddress = address
	a.updatedAt = biztime.NowUTC()

	return nil
}

// UpdateTunnelAddress updates the agent's tunnel address
func (a *ForwardAgent) UpdateTunnelAddress(address string) error {
	// Validate address if not empty
	if address != "" {
		if err := validateTunnelAddress(address); err != nil {
			return err
		}
	}

	if a.tunnelAddress == address {
		return nil
	}

	a.tunnelAddress = address
	a.updatedAt = biztime.NowUTC()

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
	if a.publicAddress != "" {
		if err := validatePublicAddress(a.publicAddress); err != nil {
			return err
		}
	}
	return nil
}

// validatePublicAddress validates if the address is a valid IP or domain name
func validatePublicAddress(address string) error {
	if address == "" {
		return nil
	}

	// Try parsing as IP address
	if ip := net.ParseIP(address); ip != nil {
		return nil
	}

	// Validate as domain name (basic RFC 1123 hostname validation)
	if domainNameRegex.MatchString(address) {
		return nil
	}

	return fmt.Errorf("invalid public address format: must be a valid IP address or domain name")
}

// validateTunnelAddress validates if the address is a valid IP or domain name.
// Loopback addresses (127.0.0.1, localhost) are allowed to support tunnel scenarios like frp.
func validateTunnelAddress(address string) error {
	if address == "" {
		return nil
	}

	// Try parsing as IP address (including loopback)
	if ip := net.ParseIP(address); ip != nil {
		return nil
	}

	// Validate as domain name (basic RFC 1123 hostname validation)
	// localhost is allowed for tunnel scenarios
	if address == "localhost" {
		return nil
	}
	if domainNameRegex.MatchString(address) {
		return nil
	}

	return fmt.Errorf("invalid tunnel address format: must be a valid IP address or domain name")
}

// AllowedPortRange returns the allowed port range configuration
func (a *ForwardAgent) AllowedPortRange() *vo.PortRange {
	return a.allowedPortRange
}

// SetAllowedPortRange sets the allowed port range.
// Pass nil to allow all ports.
func (a *ForwardAgent) SetAllowedPortRange(portRange *vo.PortRange) error {
	if portRange != nil {
		if err := portRange.Validate(); err != nil {
			return err
		}
	}
	a.allowedPortRange = portRange
	a.updatedAt = biztime.NowUTC()
	return nil
}

// IsPortAllowed checks if a port is allowed for this agent.
// Returns true if no port range is configured (all ports allowed).
func (a *ForwardAgent) IsPortAllowed(port uint16) bool {
	if a.allowedPortRange == nil {
		return true
	}
	return a.allowedPortRange.Contains(port)
}
