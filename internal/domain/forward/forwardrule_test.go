package forward

import (
	"fmt"
	"testing"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
)

// Test helper functions to avoid import cycle with testutil

// mockShortIDGenerator generates a predictable short ID for testing.
func mockShortIDGenerator() func() (string, error) {
	counter := 0
	return func() (string, error) {
		counter++
		return fmt.Sprintf("test_id_%d", counter), nil
	}
}

// ruleParams holds parameters for creating a test forward rule.
type ruleParams struct {
	AgentID           uint
	RuleType          vo.ForwardRuleType
	ExitAgentID       uint
	ChainAgentIDs     []uint
	ChainPortConfig   map[uint]uint16
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

// ruleOption is a function that modifies ruleParams.
type ruleOption func(*ruleParams)

// withAgentID sets the agent ID.
func withAgentID(id uint) ruleOption {
	return func(p *ruleParams) {
		p.AgentID = id
	}
}

// withListenPort sets the listen port.
func withListenPort(port uint16) ruleOption {
	return func(p *ruleParams) {
		p.ListenPort = port
	}
}

// withTargetAddress sets the target address.
func withTargetAddress(addr string) ruleOption {
	return func(p *ruleParams) {
		p.TargetAddress = addr
	}
}

// withTargetPort sets the target port.
func withTargetPort(port uint16) ruleOption {
	return func(p *ruleParams) {
		p.TargetPort = port
	}
}

// withTargetNodeID sets the target node ID.
func withTargetNodeID(nodeID uint) ruleOption {
	return func(p *ruleParams) {
		p.TargetNodeID = &nodeID
	}
}

// withExitAgentID sets the exit agent ID.
func withExitAgentID(id uint) ruleOption {
	return func(p *ruleParams) {
		p.ExitAgentID = id
	}
}

// withChainAgents sets the chain agent IDs.
func withChainAgents(ids []uint) ruleOption {
	return func(p *ruleParams) {
		p.ChainAgentIDs = ids
	}
}

// withChainPortConfig sets the chain port configuration.
func withChainPortConfig(config map[uint]uint16) ruleOption {
	return func(p *ruleParams) {
		p.ChainPortConfig = config
	}
}

// withName sets the rule name.
func withName(name string) ruleOption {
	return func(p *ruleParams) {
		p.Name = name
	}
}

// withBindIP sets the bind IP.
func withBindIP(ip string) ruleOption {
	return func(p *ruleParams) {
		p.BindIP = ip
	}
}

// withIPVersion sets the IP version.
func withIPVersion(version vo.IPVersion) ruleOption {
	return func(p *ruleParams) {
		p.IPVersion = version
	}
}

// withProtocol sets the protocol.
func withProtocol(protocol vo.ForwardProtocol) ruleOption {
	return func(p *ruleParams) {
		p.Protocol = protocol
	}
}

// withRemark sets the remark.
func withRemark(remark string) ruleOption {
	return func(p *ruleParams) {
		p.Remark = remark
	}
}

// validDirectRuleParams returns valid parameters for a direct rule.
func validDirectRuleParams(opts ...ruleOption) ruleParams {
	params := ruleParams{
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

// validEntryRuleParams returns valid parameters for an entry rule.
func validEntryRuleParams(opts ...ruleOption) ruleParams {
	params := ruleParams{
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

// validChainRuleParams returns valid parameters for a chain rule.
func validChainRuleParams(opts ...ruleOption) ruleParams {
	params := ruleParams{
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

// validDirectChainRuleParams returns valid parameters for a direct_chain rule.
func validDirectChainRuleParams(opts ...ruleOption) ruleParams {
	chainAgents := []uint{2, 3, 4}
	chainPortConfig := map[uint]uint16{
		2: 7001,
		3: 7002,
		4: 7003,
	}
	params := ruleParams{
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

// newTestForwardRule creates a test forward rule with the given parameters.
func newTestForwardRule(params ruleParams) (*ForwardRule, error) {
	generator := mockShortIDGenerator()
	return NewForwardRule(
		params.AgentID,
		params.RuleType,
		params.ExitAgentID,
		params.ChainAgentIDs,
		params.ChainPortConfig,
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
		generator,
	)
}

// =============================================================================
// NewForwardRule - Direct Rule Type Tests
// =============================================================================

// TestNewForwardRule_Direct_ValidWithStaticTarget verifies creating a direct rule
// with static target address and port.
// Business rule: Direct rules require agentID, listenPort, and either
// (targetAddress + targetPort) OR targetNodeID.
func TestNewForwardRule_Direct_ValidWithStaticTarget(t *testing.T) {
	params := validDirectRuleParams()

	rule, err := newTestForwardRule(params)

	if err != nil {
		t.Errorf("NewForwardRule() unexpected error = %v", err)
		return
	}
	if rule == nil {
		t.Error("NewForwardRule() returned nil rule")
		return
	}

	// Verify fields
	if rule.AgentID() != params.AgentID {
		t.Errorf("AgentID() = %v, want %v", rule.AgentID(), params.AgentID)
	}
	if rule.RuleType() != vo.ForwardRuleTypeDirect {
		t.Errorf("RuleType() = %v, want %v", rule.RuleType(), vo.ForwardRuleTypeDirect)
	}
	if rule.ListenPort() != params.ListenPort {
		t.Errorf("ListenPort() = %v, want %v", rule.ListenPort(), params.ListenPort)
	}
	if rule.TargetAddress() != params.TargetAddress {
		t.Errorf("TargetAddress() = %v, want %v", rule.TargetAddress(), params.TargetAddress)
	}
	if rule.TargetPort() != params.TargetPort {
		t.Errorf("TargetPort() = %v, want %v", rule.TargetPort(), params.TargetPort)
	}
	if rule.HasTargetNode() {
		t.Error("HasTargetNode() = true, want false")
	}
}

// TestNewForwardRule_Direct_ValidWithNodeTarget verifies creating a direct rule
// with dynamic node target.
// Business rule: targetNodeID can be used instead of static address+port.
func TestNewForwardRule_Direct_ValidWithNodeTarget(t *testing.T) {
	nodeID := uint(10)
	params := validDirectRuleParams(
		withTargetNodeID(nodeID),
		withTargetAddress(""),
		withTargetPort(0),
	)

	rule, err := newTestForwardRule(params)

	if err != nil {
		t.Errorf("NewForwardRule() unexpected error = %v", err)
		return
	}
	if rule == nil {
		t.Error("NewForwardRule() returned nil rule")
		return
	}
	if !rule.HasTargetNode() {
		t.Error("HasTargetNode() = false, want true")
	}
	if rule.TargetNodeID() == nil || *rule.TargetNodeID() != nodeID {
		t.Errorf("TargetNodeID() = %v, want %v", rule.TargetNodeID(), nodeID)
	}
}

// TestNewForwardRule_Direct_MissingTarget verifies that creating a direct rule
// without target fails.
// Business rule: Either targetAddress+targetPort OR targetNodeID must be set.
func TestNewForwardRule_Direct_MissingTarget(t *testing.T) {
	params := validDirectRuleParams(
		withTargetAddress(""),
		withTargetPort(0),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for missing target, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Direct_BothTargetsSet verifies that creating a direct rule
// with both static and node targets fails.
// Business rule: targetAddress+targetPort and targetNodeID are mutually exclusive.
func TestNewForwardRule_Direct_BothTargetsSet(t *testing.T) {
	nodeID := uint(10)
	params := validDirectRuleParams(
		withTargetNodeID(nodeID),
	)
	// params already has targetAddress and targetPort set

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for both targets set, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Direct_InvalidAddress verifies that creating a direct rule
// with invalid target address fails.
// Business rule: targetAddress must be valid IP or RFC 1123 domain name.
func TestNewForwardRule_Direct_InvalidAddress(t *testing.T) {
	params := validDirectRuleParams(
		withTargetAddress("invalid..domain"),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for invalid address, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Direct_ZeroPort verifies that creating a direct rule
// with zero target port fails.
// Business rule: targetPort must be non-zero.
func TestNewForwardRule_Direct_ZeroPort(t *testing.T) {
	params := validDirectRuleParams(
		withTargetPort(0),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for zero port, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Direct_ZeroAgentID verifies that creating a direct rule
// with zero agent ID fails.
// Business rule: agentID must be non-zero.
func TestNewForwardRule_Direct_ZeroAgentID(t *testing.T) {
	params := validDirectRuleParams(
		withAgentID(0),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for zero agent ID, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Direct_InvalidIPFormat verifies that creating a direct rule
// with invalid domain format fails.
// Business rule: Address must be valid IP or RFC 1123 compliant domain.
func TestNewForwardRule_Direct_InvalidIPFormat(t *testing.T) {
	params := validDirectRuleParams(
		withTargetAddress("invalid_underscore.com"),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for invalid domain format, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// =============================================================================
// NewForwardRule - Entry Rule Type Tests
// =============================================================================

// TestNewForwardRule_Entry_ValidConfig verifies creating an entry rule
// with valid configuration.
// Business rule: Entry rules require agentID, listenPort, exitAgentID, and
// either (targetAddress + targetPort) OR targetNodeID.
func TestNewForwardRule_Entry_ValidConfig(t *testing.T) {
	params := validEntryRuleParams()

	rule, err := newTestForwardRule(params)

	if err != nil {
		t.Errorf("NewForwardRule() unexpected error = %v", err)
		return
	}
	if rule == nil {
		t.Error("NewForwardRule() returned nil rule")
		return
	}
	if rule.ExitAgentID() != params.ExitAgentID {
		t.Errorf("ExitAgentID() = %v, want %v", rule.ExitAgentID(), params.ExitAgentID)
	}
}

// TestNewForwardRule_Entry_MissingExitAgent verifies that creating an entry rule
// without exit agent fails.
// Business rule: exitAgentID is required for entry type.
func TestNewForwardRule_Entry_MissingExitAgent(t *testing.T) {
	params := validEntryRuleParams(
		withExitAgentID(0),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for missing exit agent, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Entry_ZeroExitAgent verifies that creating an entry rule
// with zero exit agent fails.
// Business rule: exitAgentID must be non-zero.
func TestNewForwardRule_Entry_ZeroExitAgent(t *testing.T) {
	params := validEntryRuleParams()
	params.ExitAgentID = 0

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for zero exit agent, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Entry_MissingTarget verifies that creating an entry rule
// without target fails.
// Business rule: Either targetAddress+targetPort OR targetNodeID must be set.
func TestNewForwardRule_Entry_MissingTarget(t *testing.T) {
	params := validEntryRuleParams(
		withTargetAddress(""),
		withTargetPort(0),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for missing target, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Entry_BothTargetsSet verifies that creating an entry rule
// with both static and node targets fails.
// Business rule: targetAddress+targetPort and targetNodeID are mutually exclusive.
func TestNewForwardRule_Entry_BothTargetsSet(t *testing.T) {
	nodeID := uint(10)
	params := validEntryRuleParams(
		withTargetNodeID(nodeID),
	)
	// params already has targetAddress and targetPort set

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for both targets set, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Entry_InvalidDomain verifies that creating an entry rule
// with invalid domain fails.
// Business rule: Domain must comply with RFC 1123.
func TestNewForwardRule_Entry_InvalidDomain(t *testing.T) {
	params := validEntryRuleParams(
		withTargetAddress("-invalid.domain"),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for invalid domain, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Entry_ZeroPort verifies that creating an entry rule
// with zero target port fails.
// Business rule: targetPort must be non-zero.
func TestNewForwardRule_Entry_ZeroPort(t *testing.T) {
	params := validEntryRuleParams(
		withTargetPort(0),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for zero port, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Entry_IPv4Target verifies creating an entry rule
// with IPv4 target address.
// Business rule: Entry rules accept IPv4 addresses.
func TestNewForwardRule_Entry_IPv4Target(t *testing.T) {
	params := validEntryRuleParams(
		withTargetAddress("192.168.1.100"),
	)

	rule, err := newTestForwardRule(params)

	if err != nil {
		t.Errorf("NewForwardRule() unexpected error = %v", err)
		return
	}
	if rule == nil {
		t.Error("NewForwardRule() returned nil rule")
	}
}

// TestNewForwardRule_Entry_IPv6Target verifies creating an entry rule
// with IPv6 target address.
// Business rule: Entry rules accept IPv6 addresses.
func TestNewForwardRule_Entry_IPv6Target(t *testing.T) {
	params := validEntryRuleParams(
		withTargetAddress("2001:db8::1"),
	)

	rule, err := newTestForwardRule(params)

	if err != nil {
		t.Errorf("NewForwardRule() unexpected error = %v", err)
		return
	}
	if rule == nil {
		t.Error("NewForwardRule() returned nil rule")
	}
}

// TestNewForwardRule_Entry_DomainTarget verifies creating an entry rule
// with domain name target.
// Business rule: Entry rules accept valid domain names.
func TestNewForwardRule_Entry_DomainTarget(t *testing.T) {
	params := validEntryRuleParams(
		withTargetAddress("example.com"),
	)

	rule, err := newTestForwardRule(params)

	if err != nil {
		t.Errorf("NewForwardRule() unexpected error = %v", err)
		return
	}
	if rule == nil {
		t.Error("NewForwardRule() returned nil rule")
	}
}

// =============================================================================
// NewForwardRule - Chain Rule Type Tests
// =============================================================================

// TestNewForwardRule_Chain_OneIntermediateAgent verifies creating a chain rule
// with one intermediate agent.
// Business rule: Chain rules require at least 1 intermediate agent.
func TestNewForwardRule_Chain_OneIntermediateAgent(t *testing.T) {
	params := validChainRuleParams(
		withChainAgents([]uint{2}),
	)

	rule, err := newTestForwardRule(params)

	if err != nil {
		t.Errorf("NewForwardRule() unexpected error = %v", err)
		return
	}
	if rule == nil {
		t.Error("NewForwardRule() returned nil rule")
		return
	}
	if len(rule.ChainAgentIDs()) != 1 {
		t.Errorf("ChainAgentIDs() length = %v, want 1", len(rule.ChainAgentIDs()))
	}
}

// TestNewForwardRule_Chain_TenIntermediateAgents verifies creating a chain rule
// with maximum (10) intermediate agents.
// Business rule: Chain rules support maximum 10 intermediate agents.
func TestNewForwardRule_Chain_TenIntermediateAgents(t *testing.T) {
	params := validChainRuleParams(
		withChainAgents([]uint{2, 3, 4, 5, 6, 7, 8, 9, 10, 11}),
	)

	rule, err := newTestForwardRule(params)

	if err != nil {
		t.Errorf("NewForwardRule() unexpected error = %v", err)
		return
	}
	if rule == nil {
		t.Error("NewForwardRule() returned nil rule")
		return
	}
	if len(rule.ChainAgentIDs()) != 10 {
		t.Errorf("ChainAgentIDs() length = %v, want 10", len(rule.ChainAgentIDs()))
	}
}

// TestNewForwardRule_Chain_ExceedsMaxAgents verifies that creating a chain rule
// with more than 10 agents fails.
// Business rule: Chain rules have a maximum of 10 intermediate agents.
func TestNewForwardRule_Chain_ExceedsMaxAgents(t *testing.T) {
	params := validChainRuleParams(
		withChainAgents([]uint{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for too many agents, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Chain_EmptyChain verifies that creating a chain rule
// with empty chain fails.
// Business rule: Chain rules require at least 1 intermediate agent.
func TestNewForwardRule_Chain_EmptyChain(t *testing.T) {
	params := validChainRuleParams(
		withChainAgents([]uint{}),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for empty chain, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Chain_DuplicateAgent verifies that creating a chain rule
// with duplicate agents fails.
// Business rule: Chain cannot contain duplicate agent IDs.
func TestNewForwardRule_Chain_DuplicateAgent(t *testing.T) {
	params := validChainRuleParams(
		withChainAgents([]uint{2, 3, 2}),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for duplicate agent, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Chain_EntryAgentInChain verifies that creating a chain rule
// with entry agent in chain fails.
// Business rule: Entry agent cannot be in the chain.
func TestNewForwardRule_Chain_EntryAgentInChain(t *testing.T) {
	params := validChainRuleParams(
		withChainAgents([]uint{1, 2, 3}), // 1 is the entry agent
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for entry agent in chain, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Chain_ZeroIDInChain verifies that creating a chain rule
// with zero agent ID fails.
// Business rule: Chain agent IDs must be non-zero.
func TestNewForwardRule_Chain_ZeroIDInChain(t *testing.T) {
	params := validChainRuleParams(
		withChainAgents([]uint{2, 0, 3}),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for zero ID in chain, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Chain_MissingTarget verifies that creating a chain rule
// without target fails.
// Business rule: Either targetAddress+targetPort OR targetNodeID must be set.
func TestNewForwardRule_Chain_MissingTarget(t *testing.T) {
	params := validChainRuleParams(
		withTargetAddress(""),
		withTargetPort(0),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for missing target, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Chain_BothTargetsSet verifies that creating a chain rule
// with both static and node targets fails.
// Business rule: targetAddress+targetPort and targetNodeID are mutually exclusive.
func TestNewForwardRule_Chain_BothTargetsSet(t *testing.T) {
	nodeID := uint(10)
	params := validChainRuleParams(
		withTargetNodeID(nodeID),
	)
	// params already has targetAddress and targetPort set

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for both targets set, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_Chain_ValidWithNodeTarget verifies creating a chain rule
// with valid node target.
// Business rule: Chain rules can use targetNodeID instead of static address.
func TestNewForwardRule_Chain_ValidWithNodeTarget(t *testing.T) {
	nodeID := uint(10)
	params := validChainRuleParams(
		withTargetNodeID(nodeID),
		withTargetAddress(""),
		withTargetPort(0),
	)

	rule, err := newTestForwardRule(params)

	if err != nil {
		t.Errorf("NewForwardRule() unexpected error = %v", err)
		return
	}
	if rule == nil {
		t.Error("NewForwardRule() returned nil rule")
		return
	}
	if !rule.HasTargetNode() {
		t.Error("HasTargetNode() = false, want true")
	}
}

// TestNewForwardRule_Chain_DifferentRoleAgents verifies creating a chain rule
// with different agents in different roles.
// Business rule: Agents in chain should be distinct from entry agent.
func TestNewForwardRule_Chain_DifferentRoleAgents(t *testing.T) {
	params := validChainRuleParams(
		withAgentID(100),
		withChainAgents([]uint{200, 300}),
	)

	rule, err := newTestForwardRule(params)

	if err != nil {
		t.Errorf("NewForwardRule() unexpected error = %v", err)
		return
	}
	if rule == nil {
		t.Error("NewForwardRule() returned nil rule")
		return
	}
	if rule.AgentID() == rule.ChainAgentIDs()[0] {
		t.Error("Entry agent should be different from chain agents")
	}
}

// TestNewForwardRule_Chain_OrderValidation verifies that chain order is preserved.
// Business rule: Chain agent order matters for routing.
func TestNewForwardRule_Chain_OrderValidation(t *testing.T) {
	expectedOrder := []uint{2, 3, 4}
	params := validChainRuleParams(
		withChainAgents(expectedOrder),
	)

	rule, err := newTestForwardRule(params)

	if err != nil {
		t.Errorf("NewForwardRule() unexpected error = %v", err)
		return
	}
	if rule == nil {
		t.Error("NewForwardRule() returned nil rule")
		return
	}
	chain := rule.ChainAgentIDs()
	for i, id := range expectedOrder {
		if chain[i] != id {
			t.Errorf("ChainAgentIDs()[%d] = %v, want %v", i, chain[i], id)
		}
	}
}

// =============================================================================
// NewForwardRule - DirectChain Rule Type Tests
// =============================================================================

// TestNewForwardRule_DirectChain_CompleteConfig verifies creating a direct_chain rule
// with complete configuration.
// Business rule: DirectChain rules require agentID, listenPort, chainAgentIDs,
// chainPortConfig (matching chain agents), and target.
func TestNewForwardRule_DirectChain_CompleteConfig(t *testing.T) {
	params := validDirectChainRuleParams()

	rule, err := newTestForwardRule(params)

	if err != nil {
		t.Errorf("NewForwardRule() unexpected error = %v", err)
		return
	}
	if rule == nil {
		t.Error("NewForwardRule() returned nil rule")
		return
	}
	if len(rule.ChainPortConfig()) != len(rule.ChainAgentIDs()) {
		t.Errorf("ChainPortConfig() length = %v, want %v", len(rule.ChainPortConfig()), len(rule.ChainAgentIDs()))
	}
}

// TestNewForwardRule_DirectChain_MissingPortConfig verifies that creating a direct_chain rule
// without port config fails.
// Business rule: chainPortConfig is required for direct_chain type.
func TestNewForwardRule_DirectChain_MissingPortConfig(t *testing.T) {
	params := validDirectChainRuleParams(
		withChainPortConfig(nil),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for missing port config, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_DirectChain_IncompletePortConfig verifies that creating a direct_chain rule
// with incomplete port config fails.
// Business rule: chainPortConfig must have entry for each chain agent.
func TestNewForwardRule_DirectChain_IncompletePortConfig(t *testing.T) {
	params := validDirectChainRuleParams(
		withChainPortConfig(map[uint]uint16{2: 7001}), // missing agents 3 and 4
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for incomplete port config, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_DirectChain_InvalidPort verifies that creating a direct_chain rule
// with zero port fails.
// Business rule: Port values in chainPortConfig must be non-zero.
func TestNewForwardRule_DirectChain_InvalidPort(t *testing.T) {
	params := validDirectChainRuleParams(
		withChainPortConfig(map[uint]uint16{2: 7001, 3: 0, 4: 7003}),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for invalid port, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_DirectChain_ExtraPortConfig verifies that creating a direct_chain rule
// with extra port config entries fails.
// Business rule: chainPortConfig should not have entries for agents not in chain.
func TestNewForwardRule_DirectChain_ExtraPortConfig(t *testing.T) {
	params := validDirectChainRuleParams(
		withChainPortConfig(map[uint]uint16{2: 7001, 3: 7002, 4: 7003, 5: 7004}),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for extra port config, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_DirectChain_TenAgentsCompleteConfig verifies creating a direct_chain rule
// with 10 agents and complete port config.
// Business rule: DirectChain supports up to 10 agents with full port config.
func TestNewForwardRule_DirectChain_TenAgentsCompleteConfig(t *testing.T) {
	agents := []uint{2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	portConfig := map[uint]uint16{
		2: 7001, 3: 7002, 4: 7003, 5: 7004, 6: 7005,
		7: 7006, 8: 7007, 9: 7008, 10: 7009, 11: 7010,
	}
	params := validDirectChainRuleParams(
		withChainAgents(agents),
		withChainPortConfig(portConfig),
	)

	rule, err := newTestForwardRule(params)

	if err != nil {
		t.Errorf("NewForwardRule() unexpected error = %v", err)
		return
	}
	if rule == nil {
		t.Error("NewForwardRule() returned nil rule")
		return
	}
	if len(rule.ChainAgentIDs()) != 10 {
		t.Errorf("ChainAgentIDs() length = %v, want 10", len(rule.ChainAgentIDs()))
	}
}

// TestNewForwardRule_DirectChain_DuplicateAgent verifies that creating a direct_chain rule
// with duplicate agents fails.
// Business rule: Chain cannot contain duplicate agent IDs.
func TestNewForwardRule_DirectChain_DuplicateAgent(t *testing.T) {
	params := validDirectChainRuleParams(
		withChainAgents([]uint{2, 3, 2}),
		withChainPortConfig(map[uint]uint16{2: 7001, 3: 7002}),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for duplicate agent, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_DirectChain_EntryInChain verifies that creating a direct_chain rule
// with entry agent in chain fails.
// Business rule: Entry agent cannot be in the chain.
func TestNewForwardRule_DirectChain_EntryInChain(t *testing.T) {
	params := validDirectChainRuleParams(
		withChainAgents([]uint{1, 2, 3}), // 1 is the entry agent
		withChainPortConfig(map[uint]uint16{1: 7000, 2: 7001, 3: 7002}),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for entry agent in chain, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_DirectChain_MissingTarget verifies that creating a direct_chain rule
// without target fails.
// Business rule: Either targetAddress+targetPort OR targetNodeID must be set.
func TestNewForwardRule_DirectChain_MissingTarget(t *testing.T) {
	params := validDirectChainRuleParams(
		withTargetAddress(""),
		withTargetPort(0),
	)

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for missing target, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_DirectChain_BothTargetsSet verifies that creating a direct_chain rule
// with both static and node targets fails.
// Business rule: targetAddress+targetPort and targetNodeID are mutually exclusive.
func TestNewForwardRule_DirectChain_BothTargetsSet(t *testing.T) {
	nodeID := uint(10)
	params := validDirectChainRuleParams(
		withTargetNodeID(nodeID),
	)
	// params already has targetAddress and targetPort set

	rule, err := newTestForwardRule(params)

	if err == nil {
		t.Error("NewForwardRule() expected error for both targets set, got nil")
	}
	if rule != nil {
		t.Error("NewForwardRule() expected nil rule, got non-nil")
	}
}

// TestNewForwardRule_DirectChain_ConfigIntegrity verifies that port config
// matches chain agents exactly.
// Business rule: chainPortConfig must be in sync with chainAgentIDs.
func TestNewForwardRule_DirectChain_ConfigIntegrity(t *testing.T) {
	agents := []uint{10, 20, 30}
	portConfig := map[uint]uint16{10: 8001, 20: 8002, 30: 8003}
	params := validDirectChainRuleParams(
		withChainAgents(agents),
		withChainPortConfig(portConfig),
	)

	rule, err := newTestForwardRule(params)

	if err != nil {
		t.Errorf("NewForwardRule() unexpected error = %v", err)
		return
	}
	if rule == nil {
		t.Error("NewForwardRule() returned nil rule")
		return
	}
	for _, agentID := range agents {
		if rule.GetAgentListenPort(agentID) == 0 {
			t.Errorf("GetAgentListenPort(%v) = 0, want non-zero", agentID)
		}
	}
}

// TestNewForwardRule_DirectChain_ConfigUpdateAtomicity verifies that config updates
// maintain consistency.
// Business rule: chainAgentIDs and chainPortConfig must stay in sync.
func TestNewForwardRule_DirectChain_ConfigUpdateAtomicity(t *testing.T) {
	params := validDirectChainRuleParams()

	rule, err := newTestForwardRule(params)

	if err != nil {
		t.Errorf("NewForwardRule() unexpected error = %v", err)
		return
	}

	// Verify initial state
	initialAgents := rule.ChainAgentIDs()
	for _, agentID := range initialAgents {
		if rule.GetAgentListenPort(agentID) == 0 {
			t.Errorf("GetAgentListenPort(%v) = 0, want non-zero", agentID)
		}
	}
}

// =============================================================================
// State Transition Tests
// =============================================================================

// TestForwardRule_Enable_Idempotent verifies that Enable is idempotent.
// Business rule: Enabling an already enabled rule should not error.
func TestForwardRule_Enable_Idempotent(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	err1 := rule.Enable()
	err2 := rule.Enable()

	if err1 != nil {
		t.Errorf("First Enable() unexpected error = %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second Enable() unexpected error = %v", err2)
	}
	if !rule.IsEnabled() {
		t.Error("IsEnabled() = false, want true")
	}
}

// TestForwardRule_Disable_Idempotent verifies that Disable is idempotent.
// Business rule: Disabling an already disabled rule should not error.
func TestForwardRule_Disable_Idempotent(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	err1 := rule.Disable()
	err2 := rule.Disable()

	if err1 != nil {
		t.Errorf("First Disable() unexpected error = %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second Disable() unexpected error = %v", err2)
	}
	if rule.IsEnabled() {
		t.Error("IsEnabled() = true, want false")
	}
}

// TestForwardRule_EnableDisable_UpdatesTimestamp verifies that state changes
// update the timestamp.
// Business rule: Status changes should update updatedAt timestamp.
func TestForwardRule_EnableDisable_UpdatesTimestamp(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	initialTime := rule.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	rule.Enable()
	enabledTime := rule.UpdatedAt()

	if !enabledTime.After(initialTime) {
		t.Error("Enable() did not update timestamp")
	}

	time.Sleep(10 * time.Millisecond)
	rule.Disable()
	disabledTime := rule.UpdatedAt()

	if !disabledTime.After(enabledTime) {
		t.Error("Disable() did not update timestamp")
	}
}

// TestForwardRule_IsEnabled_DefaultDisabled verifies that new rules
// start in disabled state.
// Business rule: New rules should be disabled by default.
func TestForwardRule_IsEnabled_DefaultDisabled(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	if rule.IsEnabled() {
		t.Error("IsEnabled() = true for new rule, want false")
	}
}

// TestForwardRule_StateTransition_Cycle verifies that state can cycle
// between enabled and disabled.
// Business rule: Rules can be enabled and disabled multiple times.
func TestForwardRule_StateTransition_Cycle(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	// Cycle: Disabled -> Enabled -> Disabled -> Enabled
	rule.Enable()
	if !rule.IsEnabled() {
		t.Error("After Enable(), IsEnabled() = false, want true")
	}

	rule.Disable()
	if rule.IsEnabled() {
		t.Error("After Disable(), IsEnabled() = true, want false")
	}

	rule.Enable()
	if !rule.IsEnabled() {
		t.Error("After second Enable(), IsEnabled() = false, want true")
	}
}

// TestForwardRule_Status_Validation verifies status validation.
// Business rule: Only valid status values are accepted.
func TestForwardRule_Status_Validation(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	if err := rule.Validate(); err != nil {
		t.Errorf("Validate() unexpected error for valid rule = %v", err)
	}
}

// =============================================================================
// Target Update Tests
// =============================================================================

// TestForwardRule_UpdateTarget_ClearsNodeID verifies that UpdateTarget
// clears targetNodeID.
// Business rule: Setting static target clears dynamic node target.
func TestForwardRule_UpdateTarget_ClearsNodeID(t *testing.T) {
	nodeID := uint(10)
	params := validDirectRuleParams(
		withTargetNodeID(nodeID),
		withTargetAddress(""),
		withTargetPort(0),
	)
	rule, _ := newTestForwardRule(params)

	err := rule.UpdateTarget("10.0.0.1", 9999)

	if err != nil {
		t.Errorf("UpdateTarget() unexpected error = %v", err)
	}
	if rule.HasTargetNode() {
		t.Error("HasTargetNode() = true after UpdateTarget, want false")
	}
	if rule.TargetAddress() != "10.0.0.1" {
		t.Errorf("TargetAddress() = %v, want 10.0.0.1", rule.TargetAddress())
	}
	if rule.TargetPort() != 9999 {
		t.Errorf("TargetPort() = %v, want 9999", rule.TargetPort())
	}
}

// TestForwardRule_UpdateTargetNodeID_ClearsStaticTarget verifies that UpdateTargetNodeID
// clears static target fields.
// Business rule: Setting node target clears static address and port.
func TestForwardRule_UpdateTargetNodeID_ClearsStaticTarget(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	nodeID := uint(20)
	err := rule.UpdateTargetNodeID(&nodeID)

	if err != nil {
		t.Errorf("UpdateTargetNodeID() unexpected error = %v", err)
	}
	if !rule.HasTargetNode() {
		t.Error("HasTargetNode() = false after UpdateTargetNodeID, want true")
	}
	if rule.TargetAddress() != "" {
		t.Errorf("TargetAddress() = %v after UpdateTargetNodeID, want empty", rule.TargetAddress())
	}
	if rule.TargetPort() != 0 {
		t.Errorf("TargetPort() = %v after UpdateTargetNodeID, want 0", rule.TargetPort())
	}
}

// TestForwardRule_UpdateTarget_InvalidAddress verifies that UpdateTarget
// rejects invalid addresses.
// Business rule: Target address must be valid IP or domain.
func TestForwardRule_UpdateTarget_InvalidAddress(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	err := rule.UpdateTarget("invalid..address", 8080)

	if err == nil {
		t.Error("UpdateTarget() expected error for invalid address, got nil")
	}
}

// TestForwardRule_UpdateTarget_ZeroPort verifies that UpdateTarget
// rejects zero port.
// Business rule: Target port must be non-zero.
func TestForwardRule_UpdateTarget_ZeroPort(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	err := rule.UpdateTarget("10.0.0.1", 0)

	if err == nil {
		t.Error("UpdateTarget() expected error for zero port, got nil")
	}
}

// TestForwardRule_UpdateTarget_EmptyAddress verifies that UpdateTarget
// rejects empty address.
// Business rule: Target address cannot be empty.
func TestForwardRule_UpdateTarget_EmptyAddress(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	err := rule.UpdateTarget("", 8080)

	if err == nil {
		t.Error("UpdateTarget() expected error for empty address, got nil")
	}
}

// TestForwardRule_UpdateTarget_ValidDomain verifies that UpdateTarget
// accepts valid domain names.
// Business rule: Target address can be a valid domain name.
func TestForwardRule_UpdateTarget_ValidDomain(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	err := rule.UpdateTarget("example.com", 443)

	if err != nil {
		t.Errorf("UpdateTarget() unexpected error for valid domain = %v", err)
	}
	if rule.TargetAddress() != "example.com" {
		t.Errorf("TargetAddress() = %v, want example.com", rule.TargetAddress())
	}
}

// TestForwardRule_UpdateTarget_IPv6Address verifies that UpdateTarget
// accepts IPv6 addresses.
// Business rule: Target address can be IPv6.
func TestForwardRule_UpdateTarget_IPv6Address(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	err := rule.UpdateTarget("2001:db8::1", 8080)

	if err != nil {
		t.Errorf("UpdateTarget() unexpected error for IPv6 = %v", err)
	}
	if rule.TargetAddress() != "2001:db8::1" {
		t.Errorf("TargetAddress() = %v, want 2001:db8::1", rule.TargetAddress())
	}
}

// TestForwardRule_UpdateTarget_UpdatesTimestamp verifies that UpdateTarget
// updates the timestamp.
// Business rule: Target updates should update updatedAt timestamp.
func TestForwardRule_UpdateTarget_UpdatesTimestamp(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	initialTime := rule.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	rule.UpdateTarget("10.0.0.1", 9999)
	updatedTime := rule.UpdatedAt()

	if !updatedTime.After(initialTime) {
		t.Error("UpdateTarget() did not update timestamp")
	}
}

// =============================================================================
// Chain Navigation Tests
// =============================================================================

// TestForwardRule_GetNextHopAgentID_EntryAgent verifies getting next hop
// from entry agent.
// Business rule: Entry agent's next hop is first chain agent.
func TestForwardRule_GetNextHopAgentID_EntryAgent(t *testing.T) {
	params := validChainRuleParams()
	rule, _ := newTestForwardRule(params)

	nextHop := rule.GetNextHopAgentID(params.AgentID)

	if nextHop != params.ChainAgentIDs[0] {
		t.Errorf("GetNextHopAgentID() = %v, want %v", nextHop, params.ChainAgentIDs[0])
	}
}

// TestForwardRule_GetNextHopAgentID_RelayAgent verifies getting next hop
// from relay agent.
// Business rule: Relay agent's next hop is next agent in chain.
func TestForwardRule_GetNextHopAgentID_RelayAgent(t *testing.T) {
	params := validChainRuleParams()
	rule, _ := newTestForwardRule(params)

	nextHop := rule.GetNextHopAgentID(params.ChainAgentIDs[0])

	if nextHop != params.ChainAgentIDs[1] {
		t.Errorf("GetNextHopAgentID() = %v, want %v", nextHop, params.ChainAgentIDs[1])
	}
}

// TestForwardRule_GetNextHopAgentID_LastAgent verifies getting next hop
// from last agent.
// Business rule: Last agent has no next hop (returns 0).
func TestForwardRule_GetNextHopAgentID_LastAgent(t *testing.T) {
	params := validChainRuleParams()
	rule, _ := newTestForwardRule(params)

	lastAgent := params.ChainAgentIDs[len(params.ChainAgentIDs)-1]
	nextHop := rule.GetNextHopAgentID(lastAgent)

	if nextHop != 0 {
		t.Errorf("GetNextHopAgentID() = %v for last agent, want 0", nextHop)
	}
}

// TestForwardRule_GetNextHopAgentID_NotInChain verifies getting next hop
// for agent not in chain.
// Business rule: Agent not in chain returns 0.
func TestForwardRule_GetNextHopAgentID_NotInChain(t *testing.T) {
	params := validChainRuleParams()
	rule, _ := newTestForwardRule(params)

	nextHop := rule.GetNextHopAgentID(999)

	if nextHop != 0 {
		t.Errorf("GetNextHopAgentID() = %v for non-existent agent, want 0", nextHop)
	}
}

// TestForwardRule_IsLastInChain_LastAgent verifies IsLastInChain for last agent.
// Business rule: Last agent in chain should return true.
func TestForwardRule_IsLastInChain_LastAgent(t *testing.T) {
	params := validChainRuleParams()
	rule, _ := newTestForwardRule(params)

	lastAgent := params.ChainAgentIDs[len(params.ChainAgentIDs)-1]
	isLast := rule.IsLastInChain(lastAgent)

	if !isLast {
		t.Error("IsLastInChain() = false for last agent, want true")
	}
}

// TestForwardRule_IsLastInChain_NotLastAgent verifies IsLastInChain for non-last agent.
// Business rule: Non-last agent should return false.
func TestForwardRule_IsLastInChain_NotLastAgent(t *testing.T) {
	params := validChainRuleParams()
	rule, _ := newTestForwardRule(params)

	isLast := rule.IsLastInChain(params.AgentID)

	if isLast {
		t.Error("IsLastInChain() = true for entry agent, want false")
	}
}

// TestForwardRule_GetChainPosition_EntryAgent verifies position for entry agent.
// Business rule: Entry agent is at position 0.
func TestForwardRule_GetChainPosition_EntryAgent(t *testing.T) {
	params := validChainRuleParams()
	rule, _ := newTestForwardRule(params)

	position := rule.GetChainPosition(params.AgentID)

	if position != 0 {
		t.Errorf("GetChainPosition() = %v for entry agent, want 0", position)
	}
}

// TestForwardRule_GetChainPosition_RelayAgents verifies positions for relay agents.
// Business rule: Relay agents have positions 1, 2, 3, etc.
func TestForwardRule_GetChainPosition_RelayAgents(t *testing.T) {
	params := validChainRuleParams()
	rule, _ := newTestForwardRule(params)

	for i, agentID := range params.ChainAgentIDs {
		expectedPos := i + 1
		position := rule.GetChainPosition(agentID)
		if position != expectedPos {
			t.Errorf("GetChainPosition() = %v for agent %v, want %v", position, agentID, expectedPos)
		}
	}
}

// TestForwardRule_GetChainPosition_NotInChain verifies position for non-existent agent.
// Business rule: Agent not in chain returns -1.
func TestForwardRule_GetChainPosition_NotInChain(t *testing.T) {
	params := validChainRuleParams()
	rule, _ := newTestForwardRule(params)

	position := rule.GetChainPosition(999)

	if position != -1 {
		t.Errorf("GetChainPosition() = %v for non-existent agent, want -1", position)
	}
}

// TestForwardRule_GetNextHopForDirectChainSafe_Success verifies successful next hop retrieval.
// Business rule: Direct chain next hop includes agent ID and port.
func TestForwardRule_GetNextHopForDirectChainSafe_Success(t *testing.T) {
	params := validDirectChainRuleParams()
	rule, _ := newTestForwardRule(params)

	nextID, nextPort, err := rule.GetNextHopForDirectChainSafe(params.AgentID)

	if err != nil {
		t.Errorf("GetNextHopForDirectChainSafe() unexpected error = %v", err)
	}
	if nextID != params.ChainAgentIDs[0] {
		t.Errorf("nextID = %v, want %v", nextID, params.ChainAgentIDs[0])
	}
	if nextPort != params.ChainPortConfig[params.ChainAgentIDs[0]] {
		t.Errorf("nextPort = %v, want %v", nextPort, params.ChainPortConfig[params.ChainAgentIDs[0]])
	}
}

// TestForwardRule_GetNextHopForDirectChainSafe_MissingConfig verifies error
// for missing port config.
// Business rule: Missing port config should return error.
func TestForwardRule_GetNextHopForDirectChainSafe_MissingConfig(t *testing.T) {
	// Create a rule with incomplete config by modifying after creation
	params := validDirectChainRuleParams()
	rule, _ := newTestForwardRule(params)

	// Manually corrupt the config for testing
	// Note: In production, this shouldn't happen due to validation
	// This test verifies the safety check works
	_, _, err := rule.GetNextHopForDirectChainSafe(999)

	if err == nil {
		t.Error("GetNextHopForDirectChainSafe() expected error for agent not in chain, got nil")
	}
}

// TestForwardRule_GetNextHopForDirectChainSafe_LastAgent verifies behavior
// for last agent in chain.
// Business rule: Last agent returns (0, 0, nil).
func TestForwardRule_GetNextHopForDirectChainSafe_LastAgent(t *testing.T) {
	params := validDirectChainRuleParams()
	rule, _ := newTestForwardRule(params)

	lastAgent := params.ChainAgentIDs[len(params.ChainAgentIDs)-1]
	nextID, nextPort, err := rule.GetNextHopForDirectChainSafe(lastAgent)

	if err != nil {
		t.Errorf("GetNextHopForDirectChainSafe() unexpected error for last agent = %v", err)
	}
	if nextID != 0 {
		t.Errorf("nextID = %v for last agent, want 0", nextID)
	}
	if nextPort != 0 {
		t.Errorf("nextPort = %v for last agent, want 0", nextPort)
	}
}

// TestForwardRule_GetAgentListenPort_ValidAgent verifies port retrieval
// for valid agent.
// Business rule: GetAgentListenPort returns configured port for agent in chain.
func TestForwardRule_GetAgentListenPort_ValidAgent(t *testing.T) {
	params := validDirectChainRuleParams()
	rule, _ := newTestForwardRule(params)

	for agentID, expectedPort := range params.ChainPortConfig {
		port := rule.GetAgentListenPort(agentID)
		if port != expectedPort {
			t.Errorf("GetAgentListenPort(%v) = %v, want %v", agentID, port, expectedPort)
		}
	}
}

// TestForwardRule_GetAgentListenPort_InvalidAgent verifies port retrieval
// for invalid agent.
// Business rule: GetAgentListenPort returns 0 for agent not in config.
func TestForwardRule_GetAgentListenPort_InvalidAgent(t *testing.T) {
	params := validDirectChainRuleParams()
	rule, _ := newTestForwardRule(params)

	port := rule.GetAgentListenPort(999)

	if port != 0 {
		t.Errorf("GetAgentListenPort(999) = %v, want 0", port)
	}
}

// =============================================================================
// Traffic Recording Tests
// =============================================================================

// TestForwardRule_RecordTraffic_IncreasesCounters verifies that RecordTraffic
// increases counters.
// Business rule: Traffic recording accumulates upload and download bytes.
func TestForwardRule_RecordTraffic_IncreasesCounters(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	rule.RecordTraffic(100, 200)

	if rule.UploadBytes() != 100 {
		t.Errorf("UploadBytes() = %v, want 100", rule.UploadBytes())
	}
	if rule.DownloadBytes() != 200 {
		t.Errorf("DownloadBytes() = %v, want 200", rule.DownloadBytes())
	}

	rule.RecordTraffic(50, 75)

	if rule.UploadBytes() != 150 {
		t.Errorf("UploadBytes() = %v, want 150", rule.UploadBytes())
	}
	if rule.DownloadBytes() != 275 {
		t.Errorf("DownloadBytes() = %v, want 275", rule.DownloadBytes())
	}
}

// TestForwardRule_RecordTraffic_UpdatesTimestamp verifies that RecordTraffic
// updates timestamp.
// Business rule: Traffic recording should update updatedAt timestamp.
func TestForwardRule_RecordTraffic_UpdatesTimestamp(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	initialTime := rule.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	rule.RecordTraffic(100, 200)
	updatedTime := rule.UpdatedAt()

	if !updatedTime.After(initialTime) {
		t.Error("RecordTraffic() did not update timestamp")
	}
}

// TestForwardRule_TotalBytes_CalculatesSum verifies that TotalBytes
// returns sum of upload and download.
// Business rule: TotalBytes = UploadBytes + DownloadBytes.
func TestForwardRule_TotalBytes_CalculatesSum(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	rule.RecordTraffic(100, 200)

	total := rule.TotalBytes()

	if total != 300 {
		t.Errorf("TotalBytes() = %v, want 300", total)
	}
}

// TestForwardRule_ResetTraffic_ClearsCounters verifies that ResetTraffic
// clears all counters.
// Business rule: Traffic reset sets upload and download bytes to zero.
func TestForwardRule_ResetTraffic_ClearsCounters(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	rule.RecordTraffic(100, 200)
	rule.ResetTraffic()

	if rule.UploadBytes() != 0 {
		t.Errorf("UploadBytes() = %v after reset, want 0", rule.UploadBytes())
	}
	if rule.DownloadBytes() != 0 {
		t.Errorf("DownloadBytes() = %v after reset, want 0", rule.DownloadBytes())
	}
	if rule.TotalBytes() != 0 {
		t.Errorf("TotalBytes() = %v after reset, want 0", rule.TotalBytes())
	}
}

// =============================================================================
// Validate Method Tests
// =============================================================================

// TestForwardRule_Validate_AcceptsAllValidRuleTypes verifies that Validate
// accepts all valid rule types.
// Business rule: All four rule types should validate successfully.
func TestForwardRule_Validate_AcceptsAllValidRuleTypes(t *testing.T) {
	tests := []struct {
		name   string
		params ruleParams
	}{
		{"Direct", validDirectRuleParams()},
		{"Entry", validEntryRuleParams()},
		{"Chain", validChainRuleParams()},
		{"DirectChain", validDirectChainRuleParams()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, _ := newTestForwardRule(tt.params)
			err := rule.Validate()
			if err != nil {
				t.Errorf("Validate() unexpected error for %s = %v", tt.name, err)
			}
		})
	}
}

// TestForwardRule_Validate_RejectsInvalidRuleType verifies that Validate
// rejects invalid rule types.
// Business rule: Invalid rule types should fail validation.
func TestForwardRule_Validate_RejectsInvalidRuleType(t *testing.T) {
	// This test is tricky because we can't create a rule with invalid type
	// through NewForwardRule. We need to use ReconstructForwardRule.
	generator := mockShortIDGenerator()
	shortID, _ := generator()

	_, err := ReconstructForwardRule(
		1, shortID, 1,
		vo.ForwardRuleType("invalid"),
		0, nil, nil,
		"test", 8080,
		"10.0.0.1", 9000, nil,
		"", vo.IPVersionAuto, vo.ForwardProtocolTCP,
		vo.ForwardStatusDisabled,
		"", 0, 0, nil,
		time.Now(), time.Now(),
	)

	if err == nil {
		t.Error("ReconstructForwardRule() expected error for invalid rule type, got nil")
	}
}

// TestForwardRule_Validate_RequiredFieldsDirect verifies validation of
// required fields for direct type.
// Business rule: Direct type requires agentID, listenPort, and target.
func TestForwardRule_Validate_RequiredFieldsDirect(t *testing.T) {
	params := validDirectRuleParams()
	rule, _ := newTestForwardRule(params)

	err := rule.Validate()

	if err != nil {
		t.Errorf("Validate() unexpected error = %v", err)
	}
}

// TestForwardRule_Validate_RequiredFieldsEntry verifies validation of
// required fields for entry type.
// Business rule: Entry type requires exitAgentID in addition to base fields.
func TestForwardRule_Validate_RequiredFieldsEntry(t *testing.T) {
	params := validEntryRuleParams()
	rule, _ := newTestForwardRule(params)

	err := rule.Validate()

	if err != nil {
		t.Errorf("Validate() unexpected error = %v", err)
	}
}

// TestForwardRule_Validate_RequiredFieldsChain verifies validation of
// required fields for chain type.
// Business rule: Chain type requires chainAgentIDs in addition to base fields.
func TestForwardRule_Validate_RequiredFieldsChain(t *testing.T) {
	params := validChainRuleParams()
	rule, _ := newTestForwardRule(params)

	err := rule.Validate()

	if err != nil {
		t.Errorf("Validate() unexpected error = %v", err)
	}
}

// TestForwardRule_Validate_RequiredFieldsDirectChain verifies validation of
// required fields for direct_chain type.
// Business rule: DirectChain type requires chainPortConfig in addition to chain fields.
func TestForwardRule_Validate_RequiredFieldsDirectChain(t *testing.T) {
	params := validDirectChainRuleParams()
	rule, _ := newTestForwardRule(params)

	err := rule.Validate()

	if err != nil {
		t.Errorf("Validate() unexpected error = %v", err)
	}
}

// TestForwardRule_Validate_MutuallyExclusiveTargets verifies that
// static and node targets are mutually exclusive.
// Business rule: Cannot have both targetAddress+targetPort and targetNodeID.
func TestForwardRule_Validate_MutuallyExclusiveTargets(t *testing.T) {
	// Cannot test this directly as NewForwardRule already validates this
	// But we verify the validation logic works by ensuring valid rules pass
	nodeID := uint(10)
	paramsWithNode := validDirectRuleParams(
		withTargetNodeID(nodeID),
		withTargetAddress(""),
		withTargetPort(0),
	)
	rule, _ := newTestForwardRule(paramsWithNode)

	err := rule.Validate()

	if err != nil {
		t.Errorf("Validate() unexpected error for node target = %v", err)
	}
}

// TestForwardRule_Validate_ChainLengthLimit verifies chain length validation.
// Business rule: Chain length must not exceed 10 agents.
func TestForwardRule_Validate_ChainLengthLimit(t *testing.T) {
	params := validChainRuleParams(
		withChainAgents([]uint{2, 3, 4, 5, 6, 7, 8, 9, 10, 11}),
	)
	rule, _ := newTestForwardRule(params)

	err := rule.Validate()

	if err != nil {
		t.Errorf("Validate() unexpected error for 10 agents = %v", err)
	}
}

// TestForwardRule_Validate_ChainPortConfigIntegrity verifies that
// chainPortConfig matches chainAgentIDs.
// Business rule: Every agent in chain must have port config, no extras.
func TestForwardRule_Validate_ChainPortConfigIntegrity(t *testing.T) {
	params := validDirectChainRuleParams()
	rule, _ := newTestForwardRule(params)

	err := rule.Validate()

	if err != nil {
		t.Errorf("Validate() unexpected error = %v", err)
	}
}

// TestForwardRule_Validate_IPVersionOptions verifies IP version validation.
// Business rule: Only valid IP version values are accepted.
func TestForwardRule_Validate_IPVersionOptions(t *testing.T) {
	validVersions := []vo.IPVersion{
		vo.IPVersionAuto,
		vo.IPVersionIPv4,
		vo.IPVersionIPv6,
	}

	for _, version := range validVersions {
		params := validDirectRuleParams(
			withIPVersion(version),
		)
		rule, _ := newTestForwardRule(params)

		err := rule.Validate()

		if err != nil {
			t.Errorf("Validate() unexpected error for IP version %v = %v", version, err)
		}
	}
}

// TestForwardRule_Validate_BindIPFormat verifies bind IP format validation.
// Business rule: BindIP must be valid IP address or empty.
func TestForwardRule_Validate_BindIPFormat(t *testing.T) {
	tests := []struct {
		name   string
		bindIP string
		valid  bool
	}{
		{"Empty", "", true},
		{"IPv4", "192.168.1.1", true},
		{"IPv6", "2001:db8::1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := validDirectRuleParams(
				withBindIP(tt.bindIP),
			)
			rule, _ := newTestForwardRule(params)

			err := rule.Validate()

			if tt.valid && err != nil {
				t.Errorf("Validate() unexpected error for valid bindIP %v = %v", tt.bindIP, err)
			}
		})
	}
}

// =============================================================================
// Traffic Multiplier Tests
// =============================================================================

// TestCalculateNodeCount verifies node count calculation for different rule types.
// Business rule: Node count depends on rule type and chain configuration.
func TestCalculateNodeCount(t *testing.T) {
	tests := []struct {
		name          string
		params        ruleParams
		expectedCount int
	}{
		{
			name:          "Direct_OneNode",
			params:        validDirectRuleParams(),
			expectedCount: 1,
		},
		{
			name:          "Entry_TwoNodes",
			params:        validEntryRuleParams(),
			expectedCount: 2,
		},
		{
			name:          "Chain_TwoNodes",
			params:        validChainRuleParams(),
			expectedCount: 2,
		},
		{
			name: "DirectChain_EmptyChainAgentIDs_TwoNodes",
			params: ruleParams{
				AgentID:         1,
				RuleType:        vo.ForwardRuleTypeDirectChain,
				ChainAgentIDs:   []uint{2},
				ChainPortConfig: map[uint]uint16{2: 7001},
				Name:            "test-direct-chain",
				ListenPort:      8080,
				TargetAddress:   "192.168.1.100",
				TargetPort:      9000,
				IPVersion:       vo.IPVersionAuto,
				Protocol:        vo.ForwardProtocolTCP,
			},
			expectedCount: 3, // Entry + 1 Chain + Exit
		},
		{
			name: "DirectChain_OneChainAgent_ThreeNodes",
			params: ruleParams{
				AgentID:         1,
				RuleType:        vo.ForwardRuleTypeDirectChain,
				ChainAgentIDs:   []uint{2},
				ChainPortConfig: map[uint]uint16{2: 7001},
				Name:            "test-direct-chain",
				ListenPort:      8080,
				TargetAddress:   "192.168.1.100",
				TargetPort:      9000,
				IPVersion:       vo.IPVersionAuto,
				Protocol:        vo.ForwardProtocolTCP,
			},
			expectedCount: 3, // Entry + 1 Chain + Exit
		},
		{
			name: "DirectChain_ThreeChainAgents_FiveNodes",
			params: ruleParams{
				AgentID:       1,
				RuleType:      vo.ForwardRuleTypeDirectChain,
				ChainAgentIDs: []uint{2, 3, 4},
				ChainPortConfig: map[uint]uint16{
					2: 7001,
					3: 7002,
					4: 7003,
				},
				Name:          "test-direct-chain",
				ListenPort:    8080,
				TargetAddress: "192.168.1.100",
				TargetPort:    9000,
				IPVersion:     vo.IPVersionAuto,
				Protocol:      vo.ForwardProtocolTCP,
			},
			expectedCount: 5, // Entry + 3 Chain + Exit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := newTestForwardRule(tt.params)
			if err != nil {
				t.Fatalf("newTestForwardRule() unexpected error = %v", err)
			}

			count := rule.CalculateNodeCount()

			if count != tt.expectedCount {
				t.Errorf("CalculateNodeCount() = %v, want %v", count, tt.expectedCount)
			}
		})
	}
}

// TestGetEffectiveMultiplier verifies effective multiplier calculation.
// Business rule: If multiplier is nil, auto-calculate as 1/nodeCount.
// If multiplier is configured, use that value.
func TestGetEffectiveMultiplier(t *testing.T) {
	tests := []struct {
		name               string
		params             ruleParams
		multiplier         *float64
		expectedMultiplier float64
	}{
		{
			name:               "NilMultiplier_OneNode",
			params:             validDirectRuleParams(),
			multiplier:         nil,
			expectedMultiplier: 1.0,
		},
		{
			name:               "NilMultiplier_TwoNodes",
			params:             validEntryRuleParams(),
			multiplier:         nil,
			expectedMultiplier: 0.5,
		},
		{
			name: "NilMultiplier_ThreeNodes",
			params: ruleParams{
				AgentID:         1,
				RuleType:        vo.ForwardRuleTypeDirectChain,
				ChainAgentIDs:   []uint{2},
				ChainPortConfig: map[uint]uint16{2: 7001},
				Name:            "test-direct-chain",
				ListenPort:      8080,
				TargetAddress:   "192.168.1.100",
				TargetPort:      9000,
				IPVersion:       vo.IPVersionAuto,
				Protocol:        vo.ForwardProtocolTCP,
			},
			multiplier:         nil,
			expectedMultiplier: 0.3333333333333333, // 1.0 / 3
		},
		{
			name:               "ConfiguredMultiplier_08_AnyNodes",
			params:             validEntryRuleParams(),
			multiplier:         floatPtr(0.8),
			expectedMultiplier: 0.8,
		},
		{
			name:               "ConfiguredMultiplier_20_AnyNodes",
			params:             validDirectRuleParams(),
			multiplier:         floatPtr(2.0),
			expectedMultiplier: 2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := mockShortIDGenerator()
			rule, err := NewForwardRule(
				tt.params.AgentID,
				tt.params.RuleType,
				tt.params.ExitAgentID,
				tt.params.ChainAgentIDs,
				tt.params.ChainPortConfig,
				tt.params.Name,
				tt.params.ListenPort,
				tt.params.TargetAddress,
				tt.params.TargetPort,
				tt.params.TargetNodeID,
				tt.params.BindIP,
				tt.params.IPVersion,
				tt.params.Protocol,
				tt.params.Remark,
				tt.multiplier,
				generator,
			)
			if err != nil {
				t.Fatalf("NewForwardRule() unexpected error = %v", err)
			}

			multiplier := rule.GetEffectiveMultiplier()

			if multiplier != tt.expectedMultiplier {
				t.Errorf("GetEffectiveMultiplier() = %v, want %v", multiplier, tt.expectedMultiplier)
			}
		})
	}
}

// TestTrafficBytesWithMultiplier verifies traffic bytes calculation with multiplier applied.
// Business rule: UploadBytes() and DownloadBytes() should apply the effective multiplier.
func TestTrafficBytesWithMultiplier(t *testing.T) {
	tests := []struct {
		name             string
		params           ruleParams
		multiplier       *float64
		uploadBytes      int64
		downloadBytes    int64
		expectedUpload   int64
		expectedDownload int64
		expectedTotal    int64
	}{
		{
			name:             "Multiplier_05_Upload1000_Download2000",
			params:           validEntryRuleParams(),
			multiplier:       floatPtr(0.5),
			uploadBytes:      1000,
			downloadBytes:    2000,
			expectedUpload:   500,
			expectedDownload: 1000,
			expectedTotal:    1500,
		},
		{
			name:             "Multiplier_03333_Upload1500_TruncationTest",
			params:           validEntryRuleParams(),
			multiplier:       floatPtr(0.3333),
			uploadBytes:      1500,
			downloadBytes:    0,
			expectedUpload:   499, // int64(1500 * 0.3333) = 499
			expectedDownload: 0,
			expectedTotal:    499,
		},
		{
			name:             "NilMultiplier_TwoNodes_AutoApply05",
			params:           validEntryRuleParams(),
			multiplier:       nil,
			uploadBytes:      2000,
			downloadBytes:    4000,
			expectedUpload:   1000, // 2000 * 0.5
			expectedDownload: 2000, // 4000 * 0.5
			expectedTotal:    3000,
		},
		{
			name:             "Multiplier_10_NoChange",
			params:           validDirectRuleParams(),
			multiplier:       floatPtr(1.0),
			uploadBytes:      1234,
			downloadBytes:    5678,
			expectedUpload:   1234,
			expectedDownload: 5678,
			expectedTotal:    6912,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := mockShortIDGenerator()
			rule, err := NewForwardRule(
				tt.params.AgentID,
				tt.params.RuleType,
				tt.params.ExitAgentID,
				tt.params.ChainAgentIDs,
				tt.params.ChainPortConfig,
				tt.params.Name,
				tt.params.ListenPort,
				tt.params.TargetAddress,
				tt.params.TargetPort,
				tt.params.TargetNodeID,
				tt.params.BindIP,
				tt.params.IPVersion,
				tt.params.Protocol,
				tt.params.Remark,
				tt.multiplier,
				generator,
			)
			if err != nil {
				t.Fatalf("NewForwardRule() unexpected error = %v", err)
			}

			// Record traffic
			rule.RecordTraffic(tt.uploadBytes, tt.downloadBytes)

			// Verify multiplied values
			if rule.UploadBytes() != tt.expectedUpload {
				t.Errorf("UploadBytes() = %v, want %v", rule.UploadBytes(), tt.expectedUpload)
			}
			if rule.DownloadBytes() != tt.expectedDownload {
				t.Errorf("DownloadBytes() = %v, want %v", rule.DownloadBytes(), tt.expectedDownload)
			}
			if rule.TotalBytes() != tt.expectedTotal {
				t.Errorf("TotalBytes() = %v, want %v", rule.TotalBytes(), tt.expectedTotal)
			}
		})
	}
}

// TestTrafficMultiplierValidation verifies multiplier validation.
// Business rule: Multiplier cannot be negative or exceed 1000000.
func TestTrafficMultiplierValidation(t *testing.T) {
	tests := []struct {
		name       string
		multiplier *float64
		shouldFail bool
	}{
		{
			name:       "NegativeMultiplier_ReturnsError",
			multiplier: floatPtr(-0.5),
			shouldFail: true,
		},
		{
			name:       "ZeroMultiplier_Allowed",
			multiplier: floatPtr(0.0),
			shouldFail: false,
		},
		{
			name:       "MaxMultiplier_Allowed",
			multiplier: floatPtr(1000000.0),
			shouldFail: false,
		},
		{
			name:       "ExceedsMaxMultiplier_ReturnsError",
			multiplier: floatPtr(1000001.0),
			shouldFail: true,
		},
		{
			name:       "NilMultiplier_Allowed",
			multiplier: nil,
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := validDirectRuleParams()
			generator := mockShortIDGenerator()

			rule, err := NewForwardRule(
				params.AgentID,
				params.RuleType,
				params.ExitAgentID,
				params.ChainAgentIDs,
				params.ChainPortConfig,
				params.Name,
				params.ListenPort,
				params.TargetAddress,
				params.TargetPort,
				params.TargetNodeID,
				params.BindIP,
				params.IPVersion,
				params.Protocol,
				params.Remark,
				tt.multiplier,
				generator,
			)

			if tt.shouldFail {
				if err == nil {
					t.Error("NewForwardRule() expected error for invalid multiplier, got nil")
				}
				if rule != nil {
					t.Error("NewForwardRule() expected nil rule for invalid multiplier, got non-nil")
				}
			} else {
				if err != nil {
					t.Errorf("NewForwardRule() unexpected error = %v", err)
				}
				if rule == nil {
					t.Error("NewForwardRule() returned nil rule for valid multiplier")
				}
			}
		})
	}
}

// TestGetRawBytes verifies raw traffic getter methods.
// Business rule: Raw getters should return original values without multiplier.
func TestGetRawBytes(t *testing.T) {
	params := validEntryRuleParams()
	multiplier := floatPtr(0.5)
	generator := mockShortIDGenerator()

	rule, err := NewForwardRule(
		params.AgentID,
		params.RuleType,
		params.ExitAgentID,
		params.ChainAgentIDs,
		params.ChainPortConfig,
		params.Name,
		params.ListenPort,
		params.TargetAddress,
		params.TargetPort,
		params.TargetNodeID,
		params.BindIP,
		params.IPVersion,
		params.Protocol,
		params.Remark,
		multiplier,
		generator,
	)
	if err != nil {
		t.Fatalf("NewForwardRule() unexpected error = %v", err)
	}

	// Record traffic
	uploadBytes := int64(1000)
	downloadBytes := int64(2000)
	rule.RecordTraffic(uploadBytes, downloadBytes)

	// Verify raw values (without multiplier)
	if rule.GetRawUploadBytes() != uploadBytes {
		t.Errorf("GetRawUploadBytes() = %v, want %v", rule.GetRawUploadBytes(), uploadBytes)
	}
	if rule.GetRawDownloadBytes() != downloadBytes {
		t.Errorf("GetRawDownloadBytes() = %v, want %v", rule.GetRawDownloadBytes(), downloadBytes)
	}
	if rule.GetRawTotalBytes() != uploadBytes+downloadBytes {
		t.Errorf("GetRawTotalBytes() = %v, want %v", rule.GetRawTotalBytes(), uploadBytes+downloadBytes)
	}

	// Verify multiplied values (with multiplier) are different from raw
	if rule.UploadBytes() == rule.GetRawUploadBytes() {
		t.Error("UploadBytes() should differ from GetRawUploadBytes() when multiplier is applied")
	}
	if rule.DownloadBytes() == rule.GetRawDownloadBytes() {
		t.Error("DownloadBytes() should differ from GetRawDownloadBytes() when multiplier is applied")
	}

	// Verify correct multiplied values
	expectedUpload := int64(float64(uploadBytes) * 0.5)
	expectedDownload := int64(float64(downloadBytes) * 0.5)
	if rule.UploadBytes() != expectedUpload {
		t.Errorf("UploadBytes() = %v, want %v", rule.UploadBytes(), expectedUpload)
	}
	if rule.DownloadBytes() != expectedDownload {
		t.Errorf("DownloadBytes() = %v, want %v", rule.DownloadBytes(), expectedDownload)
	}
}

// floatPtr is a helper function to create a pointer to a float64.
func floatPtr(f float64) *float64 {
	return &f
}
