// Package testutil provides testing utilities and fixtures for the forward domain.
package testutil

import (
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
)

// MockShortIDGenerator generates a predictable short ID for testing.
func MockShortIDGenerator() func() (string, error) {
	counter := 0
	return func() (string, error) {
		counter++
		return fmt.Sprintf("test_id_%d", counter), nil
	}
}

// MockTokenGenerator generates a predictable token for testing.
func MockTokenGenerator(shortID string) (string, string) {
	plainToken := fmt.Sprintf("token_%s", shortID)
	tokenHash := fmt.Sprintf("hash_%s", shortID)
	return plainToken, tokenHash
}

// RuleParams holds parameters for creating a test forward rule.
type RuleParams struct {
	AgentID           uint
	UserID            *uint
	RuleType          vo.ForwardRuleType
	ExitAgentID       uint
	ChainAgentIDs     []uint
	ChainPortConfig   map[uint]uint16
	TunnelHops        *int
	TunnelType        vo.TunnelType
	Name              string
	ListenPort        uint16
	TargetAddress     string
	TargetPort        uint16
	TargetNodeID      *uint
	BindIP            string
	IPVersion         vo.IPVersion
	Protocol          vo.ForwardProtocol
	Remark            string
	TrafficMultiplier *float64
}

// RuleOption is a function that modifies RuleParams.
type RuleOption func(*RuleParams)

// WithAgentID sets the agent ID.
func WithAgentID(id uint) RuleOption {
	return func(p *RuleParams) {
		p.AgentID = id
	}
}

// WithListenPort sets the listen port.
func WithListenPort(port uint16) RuleOption {
	return func(p *RuleParams) {
		p.ListenPort = port
	}
}

// WithTargetAddress sets the target address.
func WithTargetAddress(addr string) RuleOption {
	return func(p *RuleParams) {
		p.TargetAddress = addr
	}
}

// WithTargetPort sets the target port.
func WithTargetPort(port uint16) RuleOption {
	return func(p *RuleParams) {
		p.TargetPort = port
	}
}

// WithTargetNodeID sets the target node ID.
func WithTargetNodeID(nodeID uint) RuleOption {
	return func(p *RuleParams) {
		p.TargetNodeID = &nodeID
	}
}

// WithExitAgentID sets the exit agent ID.
func WithExitAgentID(id uint) RuleOption {
	return func(p *RuleParams) {
		p.ExitAgentID = id
	}
}

// WithChainAgents sets the chain agent IDs.
func WithChainAgents(ids []uint) RuleOption {
	return func(p *RuleParams) {
		p.ChainAgentIDs = ids
	}
}

// WithChainPortConfig sets the chain port configuration.
func WithChainPortConfig(config map[uint]uint16) RuleOption {
	return func(p *RuleParams) {
		p.ChainPortConfig = config
	}
}

// WithName sets the rule name.
func WithName(name string) RuleOption {
	return func(p *RuleParams) {
		p.Name = name
	}
}

// WithBindIP sets the bind IP.
func WithBindIP(ip string) RuleOption {
	return func(p *RuleParams) {
		p.BindIP = ip
	}
}

// WithIPVersion sets the IP version.
func WithIPVersion(version vo.IPVersion) RuleOption {
	return func(p *RuleParams) {
		p.IPVersion = version
	}
}

// WithProtocol sets the protocol.
func WithProtocol(protocol vo.ForwardProtocol) RuleOption {
	return func(p *RuleParams) {
		p.Protocol = protocol
	}
}

// WithRemark sets the remark.
func WithRemark(remark string) RuleOption {
	return func(p *RuleParams) {
		p.Remark = remark
	}
}

// ValidDirectRuleParams returns valid parameters for a direct rule.
func ValidDirectRuleParams(opts ...RuleOption) RuleParams {
	params := RuleParams{
		AgentID:       1,
		RuleType:      vo.ForwardRuleTypeDirect,
		Name:          "test-direct-rule",
		ListenPort:    8080,
		TargetAddress: "192.168.1.100",
		TargetPort:    9000,
		IPVersion:     vo.IPVersionAuto,
		Protocol:      vo.ForwardProtocolTCP,
	}
	for _, opt := range opts {
		opt(&params)
	}
	return params
}

// ValidEntryRuleParams returns valid parameters for an entry rule.
func ValidEntryRuleParams(opts ...RuleOption) RuleParams {
	params := RuleParams{
		AgentID:       1,
		RuleType:      vo.ForwardRuleTypeEntry,
		ExitAgentID:   2,
		Name:          "test-entry-rule",
		ListenPort:    8080,
		TargetAddress: "192.168.1.100",
		TargetPort:    9000,
		IPVersion:     vo.IPVersionAuto,
		Protocol:      vo.ForwardProtocolTCP,
	}
	for _, opt := range opts {
		opt(&params)
	}
	return params
}

// ValidChainRuleParams returns valid parameters for a chain rule.
func ValidChainRuleParams(opts ...RuleOption) RuleParams {
	params := RuleParams{
		AgentID:       1,
		RuleType:      vo.ForwardRuleTypeChain,
		ChainAgentIDs: []uint{2, 3, 4},
		Name:          "test-chain-rule",
		ListenPort:    8080,
		TargetAddress: "192.168.1.100",
		TargetPort:    9000,
		IPVersion:     vo.IPVersionAuto,
		Protocol:      vo.ForwardProtocolTCP,
	}
	for _, opt := range opts {
		opt(&params)
	}
	return params
}

// ValidDirectChainRuleParams returns valid parameters for a direct_chain rule.
func ValidDirectChainRuleParams(opts ...RuleOption) RuleParams {
	chainAgents := []uint{2, 3, 4}
	chainPortConfig := map[uint]uint16{
		2: 7001,
		3: 7002,
		4: 7003,
	}
	params := RuleParams{
		AgentID:         1,
		RuleType:        vo.ForwardRuleTypeDirectChain,
		ChainAgentIDs:   chainAgents,
		ChainPortConfig: chainPortConfig,
		Name:            "test-direct-chain-rule",
		ListenPort:      8080,
		TargetAddress:   "192.168.1.100",
		TargetPort:      9000,
		IPVersion:       vo.IPVersionAuto,
		Protocol:        vo.ForwardProtocolTCP,
	}
	for _, opt := range opts {
		opt(&params)
	}
	return params
}

// NewTestForwardRule creates a test forward rule with the given parameters.
func NewTestForwardRule(params RuleParams) (*forward.ForwardRule, error) {
	generator := MockShortIDGenerator()
	return forward.NewForwardRule(
		params.AgentID,
		params.UserID,
		params.RuleType,
		params.ExitAgentID,
		params.ChainAgentIDs,
		params.ChainPortConfig,
		params.TunnelHops,
		params.TunnelType,
		params.Name,
		params.ListenPort,
		params.TargetAddress,
		params.TargetPort,
		params.TargetNodeID,
		params.BindIP,
		params.IPVersion,
		params.Protocol,
		params.Remark,
		params.TrafficMultiplier,
		0, // sortOrder: default to 0 for test rules
		generator,
	)
}

// AgentParams holds parameters for creating a test forward agent.
type AgentParams struct {
	Name          string
	PublicAddress string
	TunnelAddress string
	Remark        string
}

// AgentOption is a function that modifies AgentParams.
type AgentOption func(*AgentParams)

// WithAgentName sets the agent name.
func WithAgentName(name string) AgentOption {
	return func(p *AgentParams) {
		p.Name = name
	}
}

// WithPublicAddress sets the public address.
func WithPublicAddress(addr string) AgentOption {
	return func(p *AgentParams) {
		p.PublicAddress = addr
	}
}

// WithTunnelAddress sets the tunnel address.
func WithTunnelAddress(addr string) AgentOption {
	return func(p *AgentParams) {
		p.TunnelAddress = addr
	}
}

// WithAgentRemark sets the agent remark.
func WithAgentRemark(remark string) AgentOption {
	return func(p *AgentParams) {
		p.Remark = remark
	}
}

// ValidAgentParams returns valid parameters for an agent.
func ValidAgentParams(opts ...AgentOption) AgentParams {
	params := AgentParams{
		Name:          "test-agent",
		PublicAddress: "203.0.113.1",
		TunnelAddress: "198.51.100.1",
		Remark:        "",
	}
	for _, opt := range opts {
		opt(&params)
	}
	return params
}

// NewTestForwardAgent creates a test forward agent with the given parameters.
func NewTestForwardAgent(params AgentParams) (*forward.ForwardAgent, error) {
	generator := MockShortIDGenerator()
	return forward.NewForwardAgent(
		params.Name,
		params.PublicAddress,
		params.TunnelAddress,
		params.Remark,
		generator,
		MockTokenGenerator,
	)
}
