// Package testutil provides mock implementations for testing the forward application layer.
package testutil

import (
	"context"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// MockForwardRuleRepository is a mock implementation of forward.Repository for testing.
type MockForwardRuleRepository struct {
	mu             sync.RWMutex
	rules          map[uint]*forward.ForwardRule
	rulesByShortID map[string]*forward.ForwardRule
	rulesByPort    map[uint16]*forward.ForwardRule
	nextID         uint

	// Error injection for testing
	createError  error
	getError     error
	updateError  error
	deleteError  error
	listError    error
	existsError  error
	trafficError error
}

// NewMockForwardRuleRepository creates a new mock forward rule repository.
func NewMockForwardRuleRepository() *MockForwardRuleRepository {
	return &MockForwardRuleRepository{
		rules:          make(map[uint]*forward.ForwardRule),
		rulesByShortID: make(map[string]*forward.ForwardRule),
		rulesByPort:    make(map[uint16]*forward.ForwardRule),
		nextID:         0,
	}
}

// Create creates a new forward rule in the mock repository.
func (m *MockForwardRuleRepository) Create(ctx context.Context, rule *forward.ForwardRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.createError != nil {
		return m.createError
	}

	// Generate ID if not set
	if rule.ID() == 0 {
		m.nextID++
		if err := rule.SetID(m.nextID); err != nil {
			return err
		}
	}

	m.rules[rule.ID()] = rule
	m.rulesByShortID[rule.SID()] = rule
	m.rulesByPort[rule.ListenPort()] = rule

	return nil
}

// GetByID retrieves a forward rule by ID.
func (m *MockForwardRuleRepository) GetByID(ctx context.Context, id uint) (*forward.ForwardRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getError != nil {
		return nil, m.getError
	}

	rule, exists := m.rules[id]
	if !exists {
		return nil, nil
	}

	return rule, nil
}

// GetBySID retrieves a forward rule by short ID.
func (m *MockForwardRuleRepository) GetBySID(ctx context.Context, shortID string) (*forward.ForwardRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getError != nil {
		return nil, m.getError
	}

	rule, exists := m.rulesByShortID[shortID]
	if !exists {
		return nil, nil
	}

	return rule, nil
}

// GetByListenPort retrieves a forward rule by listen port.
func (m *MockForwardRuleRepository) GetByListenPort(ctx context.Context, port uint16) (*forward.ForwardRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getError != nil {
		return nil, m.getError
	}

	rule, exists := m.rulesByPort[port]
	if !exists {
		return nil, nil
	}

	return rule, nil
}

// Update updates an existing forward rule.
func (m *MockForwardRuleRepository) Update(ctx context.Context, rule *forward.ForwardRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.updateError != nil {
		return m.updateError
	}

	// Check if rule exists
	if _, exists := m.rules[rule.ID()]; !exists {
		return nil // Rule not found, no-op
	}

	// Remove old port mapping if port changed
	if oldRule, exists := m.rules[rule.ID()]; exists {
		if oldRule.ListenPort() != rule.ListenPort() {
			delete(m.rulesByPort, oldRule.ListenPort())
		}
	}

	m.rules[rule.ID()] = rule
	m.rulesByShortID[rule.SID()] = rule
	m.rulesByPort[rule.ListenPort()] = rule

	return nil
}

// Delete removes a forward rule.
func (m *MockForwardRuleRepository) Delete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.deleteError != nil {
		return m.deleteError
	}

	rule, exists := m.rules[id]
	if !exists {
		return nil // Rule not found, no-op
	}

	delete(m.rules, id)
	delete(m.rulesByShortID, rule.SID())
	delete(m.rulesByPort, rule.ListenPort())

	return nil
}

// List returns all forward rules with optional filtering.
func (m *MockForwardRuleRepository) List(ctx context.Context, filter forward.ListFilter) ([]*forward.ForwardRule, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listError != nil {
		return nil, 0, m.listError
	}

	// Collect all rules
	var rules []*forward.ForwardRule
	for _, rule := range m.rules {
		// Apply filters
		if filter.AgentID != 0 && rule.AgentID() != filter.AgentID {
			continue
		}
		if filter.Name != "" && rule.Name() != filter.Name {
			continue
		}
		if filter.Protocol != "" && string(rule.Protocol()) != filter.Protocol {
			continue
		}
		if filter.Status != "" && string(rule.Status()) != filter.Status {
			continue
		}
		rules = append(rules, rule)
	}

	total := int64(len(rules))

	// Apply pagination
	if filter.PageSize > 0 {
		start := (filter.Page - 1) * filter.PageSize
		end := start + filter.PageSize
		if start >= len(rules) {
			return []*forward.ForwardRule{}, total, nil
		}
		if end > len(rules) {
			end = len(rules)
		}
		rules = rules[start:end]
	}

	return rules, total, nil
}

// ListEnabled returns all enabled forward rules.
func (m *MockForwardRuleRepository) ListEnabled(ctx context.Context) ([]*forward.ForwardRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listError != nil {
		return nil, m.listError
	}

	var rules []*forward.ForwardRule
	for _, rule := range m.rules {
		if rule.IsEnabled() {
			rules = append(rules, rule)
		}
	}

	return rules, nil
}

// ListByAgentID returns all forward rules for a specific agent.
func (m *MockForwardRuleRepository) ListByAgentID(ctx context.Context, agentID uint) ([]*forward.ForwardRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listError != nil {
		return nil, m.listError
	}

	var rules []*forward.ForwardRule
	for _, rule := range m.rules {
		if rule.AgentID() == agentID {
			rules = append(rules, rule)
		}
	}

	return rules, nil
}

// ListEnabledByAgentID returns all enabled forward rules for a specific agent.
func (m *MockForwardRuleRepository) ListEnabledByAgentID(ctx context.Context, agentID uint) ([]*forward.ForwardRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listError != nil {
		return nil, m.listError
	}

	var rules []*forward.ForwardRule
	for _, rule := range m.rules {
		if rule.AgentID() == agentID && rule.IsEnabled() {
			rules = append(rules, rule)
		}
	}

	return rules, nil
}

// ExistsByListenPort checks if a rule with the given listen port exists.
func (m *MockForwardRuleRepository) ExistsByListenPort(ctx context.Context, port uint16) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.existsError != nil {
		return false, m.existsError
	}

	_, exists := m.rulesByPort[port]
	return exists, nil
}

// ExistsByAgentIDAndListenPort checks if a rule with the given agent ID and listen port exists.
func (m *MockForwardRuleRepository) ExistsByAgentIDAndListenPort(ctx context.Context, agentID uint, port uint16) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.existsError != nil {
		return false, m.existsError
	}

	for _, rule := range m.rules {
		if rule.AgentID() == agentID && rule.ListenPort() == port {
			return true, nil
		}
	}
	return false, nil
}

// IsPortInUseByAgent checks if a port is in use by the specified agent across all rules.
// This includes both main rule ports and chain_port_config entries.
func (m *MockForwardRuleRepository) IsPortInUseByAgent(ctx context.Context, agentID uint, port uint16, excludeRuleID uint) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.existsError != nil {
		return false, m.existsError
	}

	for _, rule := range m.rules {
		// Skip excluded rule
		if excludeRuleID > 0 && rule.ID() == excludeRuleID {
			continue
		}

		// Check main rule port
		if rule.AgentID() == agentID && rule.ListenPort() == port {
			return true, nil
		}

		// Check chain_port_config
		for chainAgentID, chainPort := range rule.ChainPortConfig() {
			if chainAgentID == agentID && chainPort == port {
				return true, nil
			}
		}
	}
	return false, nil
}

// UpdateTraffic updates the traffic counters for a rule.
func (m *MockForwardRuleRepository) UpdateTraffic(ctx context.Context, id uint, upload, download int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.trafficError != nil {
		return m.trafficError
	}

	rule, exists := m.rules[id]
	if !exists {
		return nil // Rule not found, no-op
	}

	rule.RecordTraffic(upload, download)
	return nil
}

// ListByExitAgentID returns all entrance rules for a specific exit agent.
func (m *MockForwardRuleRepository) ListByExitAgentID(ctx context.Context, exitAgentID uint) ([]*forward.ForwardRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listError != nil {
		return nil, m.listError
	}

	var rules []*forward.ForwardRule
	for _, rule := range m.rules {
		if rule.ExitAgentID() == exitAgentID {
			rules = append(rules, rule)
		}
	}

	return rules, nil
}

// ListEnabledByExitAgentID returns all enabled entry rules for a specific exit agent.
func (m *MockForwardRuleRepository) ListEnabledByExitAgentID(ctx context.Context, exitAgentID uint) ([]*forward.ForwardRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listError != nil {
		return nil, m.listError
	}

	var rules []*forward.ForwardRule
	for _, rule := range m.rules {
		if rule.ExitAgentID() == exitAgentID && rule.IsEnabled() {
			rules = append(rules, rule)
		}
	}

	return rules, nil
}

// ListEnabledByChainAgentID returns all enabled chain rules where the agent participates.
func (m *MockForwardRuleRepository) ListEnabledByChainAgentID(ctx context.Context, agentID uint) ([]*forward.ForwardRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listError != nil {
		return nil, m.listError
	}

	var rules []*forward.ForwardRule
	for _, rule := range m.rules {
		if !rule.IsEnabled() {
			continue
		}
		// Check if agent is in chain
		for _, chainAgentID := range rule.ChainAgentIDs() {
			if chainAgentID == agentID {
				rules = append(rules, rule)
				break
			}
		}
	}

	return rules, nil
}

// ListByUserID returns forward rules for a specific user with filtering and pagination.
func (m *MockForwardRuleRepository) ListByUserID(ctx context.Context, userID uint, filter forward.ListFilter) ([]*forward.ForwardRule, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listError != nil {
		return nil, 0, m.listError
	}

	// Collect rules for the user
	var rules []*forward.ForwardRule
	for _, rule := range m.rules {
		if rule.UserID() != nil && *rule.UserID() == userID {
			// Apply additional filters
			if filter.Name != "" && rule.Name() != filter.Name {
				continue
			}
			if filter.Protocol != "" && string(rule.Protocol()) != filter.Protocol {
				continue
			}
			if filter.Status != "" && string(rule.Status()) != filter.Status {
				continue
			}
			rules = append(rules, rule)
		}
	}

	total := int64(len(rules))

	// Apply pagination
	if filter.PageSize > 0 {
		start := (filter.Page - 1) * filter.PageSize
		end := start + filter.PageSize
		if start >= len(rules) {
			return []*forward.ForwardRule{}, total, nil
		}
		if end > len(rules) {
			end = len(rules)
		}
		rules = rules[start:end]
	}

	return rules, total, nil
}

// CountByUserID returns the total count of forward rules for a specific user.
func (m *MockForwardRuleRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listError != nil {
		return 0, m.listError
	}

	count := int64(0)
	for _, rule := range m.rules {
		if rule.UserID() != nil && *rule.UserID() == userID {
			count++
		}
	}

	return count, nil
}

// ListBySubscriptionID returns all forward rules for a specific subscription.
func (m *MockForwardRuleRepository) ListBySubscriptionID(ctx context.Context, subscriptionID uint) ([]*forward.ForwardRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listError != nil {
		return nil, m.listError
	}

	var rules []*forward.ForwardRule
	for _, rule := range m.rules {
		if rule.SubscriptionID() != nil && *rule.SubscriptionID() == subscriptionID {
			rules = append(rules, rule)
		}
	}

	return rules, nil
}

// CountBySubscriptionID returns the total count of forward rules for a specific subscription.
func (m *MockForwardRuleRepository) CountBySubscriptionID(ctx context.Context, subscriptionID uint) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listError != nil {
		return 0, m.listError
	}

	count := int64(0)
	for _, rule := range m.rules {
		if rule.SubscriptionID() != nil && *rule.SubscriptionID() == subscriptionID {
			count++
		}
	}

	return count, nil
}

// GetTotalTrafficByUserID returns the total traffic (upload + download) for all rules owned by a user.
func (m *MockForwardRuleRepository) GetTotalTrafficByUserID(ctx context.Context, userID uint) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listError != nil {
		return 0, m.listError
	}

	total := int64(0)
	for _, rule := range m.rules {
		if rule.UserID() != nil && *rule.UserID() == userID {
			total += rule.GetRawTotalBytes()
		}
	}

	return total, nil
}

// GetBySIDs retrieves multiple forward rules by their SIDs.
func (m *MockForwardRuleRepository) GetBySIDs(ctx context.Context, sids []string) (map[string]*forward.ForwardRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getError != nil {
		return nil, m.getError
	}

	result := make(map[string]*forward.ForwardRule)
	for _, sid := range sids {
		if rule, exists := m.rulesByShortID[sid]; exists {
			result[sid] = rule
		}
	}

	return result, nil
}

// UpdateSortOrders batch updates sort_order for multiple rules.
func (m *MockForwardRuleRepository) UpdateSortOrders(ctx context.Context, ruleOrders map[uint]int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.updateError != nil {
		return m.updateError
	}

	for id, sortOrder := range ruleOrders {
		if rule, exists := m.rules[id]; exists {
			_ = rule.UpdateSortOrder(sortOrder)
		}
	}

	return nil
}

// Helper methods for testing

// AddRule adds a rule to the mock repository (for test setup).
func (m *MockForwardRuleRepository) AddRule(rule *forward.ForwardRule) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rule.ID() == 0 {
		m.nextID++
		_ = rule.SetID(m.nextID)
	}

	m.rules[rule.ID()] = rule
	m.rulesByShortID[rule.SID()] = rule
	m.rulesByPort[rule.ListenPort()] = rule
}

// SetCreateError sets the error to return on Create calls.
func (m *MockForwardRuleRepository) SetCreateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createError = err
}

// SetGetError sets the error to return on Get calls.
func (m *MockForwardRuleRepository) SetGetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getError = err
}

// SetUpdateError sets the error to return on Update calls.
func (m *MockForwardRuleRepository) SetUpdateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateError = err
}

// SetDeleteError sets the error to return on Delete calls.
func (m *MockForwardRuleRepository) SetDeleteError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteError = err
}

// SetListError sets the error to return on List calls.
func (m *MockForwardRuleRepository) SetListError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listError = err
}

// SetExistsError sets the error to return on Exists calls.
func (m *MockForwardRuleRepository) SetExistsError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.existsError = err
}

// SetTrafficError sets the error to return on UpdateTraffic calls.
func (m *MockForwardRuleRepository) SetTrafficError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trafficError = err
}

// Reset resets the mock repository to its initial state.
func (m *MockForwardRuleRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = make(map[uint]*forward.ForwardRule)
	m.rulesByShortID = make(map[string]*forward.ForwardRule)
	m.rulesByPort = make(map[uint16]*forward.ForwardRule)
	m.nextID = 0
	m.createError = nil
	m.getError = nil
	m.updateError = nil
	m.deleteError = nil
	m.listError = nil
	m.existsError = nil
	m.trafficError = nil
}

// MockForwardAgentRepository is a mock implementation of forward.AgentRepository for testing.
type MockForwardAgentRepository struct {
	mu              sync.RWMutex
	agents          map[uint]*forward.ForwardAgent
	agentsByShortID map[string]*forward.ForwardAgent
	agentsByToken   map[string]*forward.ForwardAgent
	nextID          uint

	// Error injection for testing
	createError   error
	getError      error
	updateError   error
	deleteError   error
	listError     error
	existsError   error
	lastSeenError error
	shortIDsError error
}

// NewMockForwardAgentRepository creates a new mock forward agent repository.
func NewMockForwardAgentRepository() *MockForwardAgentRepository {
	return &MockForwardAgentRepository{
		agents:          make(map[uint]*forward.ForwardAgent),
		agentsByShortID: make(map[string]*forward.ForwardAgent),
		agentsByToken:   make(map[string]*forward.ForwardAgent),
		nextID:          0,
	}
}

// Create persists a new forward agent.
func (m *MockForwardAgentRepository) Create(ctx context.Context, agent *forward.ForwardAgent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.createError != nil {
		return m.createError
	}

	// Generate ID if not set
	if agent.ID() == 0 {
		m.nextID++
		if err := agent.SetID(m.nextID); err != nil {
			return err
		}
	}

	m.agents[agent.ID()] = agent
	m.agentsByShortID[agent.SID()] = agent
	m.agentsByToken[agent.TokenHash()] = agent

	return nil
}

// GetByID retrieves a forward agent by ID.
func (m *MockForwardAgentRepository) GetByID(ctx context.Context, id uint) (*forward.ForwardAgent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getError != nil {
		return nil, m.getError
	}

	agent, exists := m.agents[id]
	if !exists {
		return nil, nil
	}

	return agent, nil
}

// GetBySID retrieves a forward agent by short ID.
func (m *MockForwardAgentRepository) GetBySID(ctx context.Context, shortID string) (*forward.ForwardAgent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getError != nil {
		return nil, m.getError
	}

	agent, exists := m.agentsByShortID[shortID]
	if !exists {
		return nil, nil
	}

	return agent, nil
}

// GetByTokenHash retrieves a forward agent by token hash.
func (m *MockForwardAgentRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*forward.ForwardAgent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getError != nil {
		return nil, m.getError
	}

	agent, exists := m.agentsByToken[tokenHash]
	if !exists {
		return nil, nil
	}

	return agent, nil
}

// GetSIDsByIDs retrieves short IDs for multiple agents by their internal IDs.
func (m *MockForwardAgentRepository) GetSIDsByIDs(ctx context.Context, ids []uint) (map[uint]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shortIDsError != nil {
		return nil, m.shortIDsError
	}

	result := make(map[uint]string)
	for _, id := range ids {
		if agent, exists := m.agents[id]; exists {
			result[id] = agent.SID()
		}
	}

	return result, nil
}

// Update updates an existing forward agent.
func (m *MockForwardAgentRepository) Update(ctx context.Context, agent *forward.ForwardAgent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.updateError != nil {
		return m.updateError
	}

	// Check if agent exists
	if _, exists := m.agents[agent.ID()]; !exists {
		return nil // Agent not found, no-op
	}

	// Remove old token mapping if token changed
	if oldAgent, exists := m.agents[agent.ID()]; exists {
		if oldAgent.TokenHash() != agent.TokenHash() {
			delete(m.agentsByToken, oldAgent.TokenHash())
		}
	}

	m.agents[agent.ID()] = agent
	m.agentsByShortID[agent.SID()] = agent
	m.agentsByToken[agent.TokenHash()] = agent

	return nil
}

// Delete removes a forward agent.
func (m *MockForwardAgentRepository) Delete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.deleteError != nil {
		return m.deleteError
	}

	agent, exists := m.agents[id]
	if !exists {
		return nil // Agent not found, no-op
	}

	delete(m.agents, id)
	delete(m.agentsByShortID, agent.SID())
	delete(m.agentsByToken, agent.TokenHash())

	return nil
}

// List returns all forward agents with optional filtering.
func (m *MockForwardAgentRepository) List(ctx context.Context, filter forward.AgentListFilter) ([]*forward.ForwardAgent, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listError != nil {
		return nil, 0, m.listError
	}

	// Collect all agents
	var agents []*forward.ForwardAgent
	for _, agent := range m.agents {
		// Apply filters
		if filter.Name != "" && agent.Name() != filter.Name {
			continue
		}
		if filter.Status != "" && string(agent.Status()) != filter.Status {
			continue
		}
		agents = append(agents, agent)
	}

	total := int64(len(agents))

	// Apply pagination
	if filter.PageSize > 0 {
		start := (filter.Page - 1) * filter.PageSize
		end := start + filter.PageSize
		if start >= len(agents) {
			return []*forward.ForwardAgent{}, total, nil
		}
		if end > len(agents) {
			end = len(agents)
		}
		agents = agents[start:end]
	}

	return agents, total, nil
}

// ListEnabled returns all enabled forward agents.
func (m *MockForwardAgentRepository) ListEnabled(ctx context.Context) ([]*forward.ForwardAgent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listError != nil {
		return nil, m.listError
	}

	var agents []*forward.ForwardAgent
	for _, agent := range m.agents {
		if agent.IsEnabled() {
			agents = append(agents, agent)
		}
	}

	return agents, nil
}

// ExistsByName checks if an agent with the given name exists.
func (m *MockForwardAgentRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.existsError != nil {
		return false, m.existsError
	}

	for _, agent := range m.agents {
		if agent.Name() == name {
			return true, nil
		}
	}

	return false, nil
}

// UpdateLastSeen updates the last_seen_at timestamp for an agent.
func (m *MockForwardAgentRepository) UpdateLastSeen(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.lastSeenError != nil {
		return m.lastSeenError
	}

	// In mock, we just check if agent exists
	_, exists := m.agents[id]
	if !exists {
		return nil // Agent not found, no-op
	}

	return nil
}

// UpdateAgentInfo updates the agent info (version, platform, arch) for an agent.
func (m *MockForwardAgentRepository) UpdateAgentInfo(ctx context.Context, id uint, agentVersion, platform, arch string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// In mock, we just check if agent exists
	_, exists := m.agents[id]
	if !exists {
		return nil // Agent not found, no-op
	}

	return nil
}

// Helper methods for testing

// AddAgent adds an agent to the mock repository (for test setup).
func (m *MockForwardAgentRepository) AddAgent(agent *forward.ForwardAgent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if agent.ID() == 0 {
		m.nextID++
		_ = agent.SetID(m.nextID)
	}

	m.agents[agent.ID()] = agent
	m.agentsByShortID[agent.SID()] = agent
	m.agentsByToken[agent.TokenHash()] = agent
}

// SetCreateError sets the error to return on Create calls.
func (m *MockForwardAgentRepository) SetCreateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createError = err
}

// SetGetError sets the error to return on Get calls.
func (m *MockForwardAgentRepository) SetGetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getError = err
}

// SetUpdateError sets the error to return on Update calls.
func (m *MockForwardAgentRepository) SetUpdateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateError = err
}

// SetDeleteError sets the error to return on Delete calls.
func (m *MockForwardAgentRepository) SetDeleteError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteError = err
}

// SetListError sets the error to return on List calls.
func (m *MockForwardAgentRepository) SetListError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listError = err
}

// SetExistsError sets the error to return on Exists calls.
func (m *MockForwardAgentRepository) SetExistsError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.existsError = err
}

// SetLastSeenError sets the error to return on UpdateLastSeen calls.
func (m *MockForwardAgentRepository) SetLastSeenError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastSeenError = err
}

// SetShortIDsError sets the error to return on GetSIDsByIDs calls.
func (m *MockForwardAgentRepository) SetShortIDsError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shortIDsError = err
}

// Reset resets the mock repository to its initial state.
func (m *MockForwardAgentRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agents = make(map[uint]*forward.ForwardAgent)
	m.agentsByShortID = make(map[string]*forward.ForwardAgent)
	m.agentsByToken = make(map[string]*forward.ForwardAgent)
	m.nextID = 0
	m.createError = nil
	m.getError = nil
	m.updateError = nil
	m.deleteError = nil
	m.listError = nil
	m.existsError = nil
	m.lastSeenError = nil
	m.shortIDsError = nil
}

// MockNodeRepository is a mock implementation of node.Repository for testing.
type MockNodeRepository struct {
	mu         sync.RWMutex
	nodes      map[uint]*node.Node
	nodesBySID map[string]*node.Node
	nextID     uint

	// Error injection for testing
	getError    error
	createError error
}

// NewMockNodeRepository creates a new mock node repository.
func NewMockNodeRepository() *MockNodeRepository {
	return &MockNodeRepository{
		nodes:      make(map[uint]*node.Node),
		nodesBySID: make(map[string]*node.Node),
		nextID:     0,
	}
}

// Create creates a new node in the mock repository.
func (m *MockNodeRepository) Create(ctx context.Context, n *node.Node) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.createError != nil {
		return m.createError
	}

	// Generate ID if not set
	if n.ID() == 0 {
		m.nextID++
		if err := n.SetID(m.nextID); err != nil {
			return err
		}
	}

	m.nodes[n.ID()] = n
	m.nodesBySID[n.SID()] = n

	return nil
}

// GetByID retrieves a node by ID.
func (m *MockNodeRepository) GetByID(ctx context.Context, id uint) (*node.Node, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getError != nil {
		return nil, m.getError
	}

	n, exists := m.nodes[id]
	if !exists {
		return nil, nil
	}

	return n, nil
}

// GetBySID retrieves a node by SID.
func (m *MockNodeRepository) GetBySID(ctx context.Context, sid string) (*node.Node, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getError != nil {
		return nil, m.getError
	}

	n, exists := m.nodesBySID[sid]
	if !exists {
		return nil, nil
	}

	return n, nil
}

// Stub implementations for interface compliance

func (m *MockNodeRepository) GetByIDs(ctx context.Context, ids []uint) ([]*node.Node, error) {
	return nil, nil
}

func (m *MockNodeRepository) GetByToken(ctx context.Context, tokenHash string) (*node.Node, error) {
	return nil, nil
}

func (m *MockNodeRepository) Update(ctx context.Context, n *node.Node) error {
	return nil
}

func (m *MockNodeRepository) Delete(ctx context.Context, id uint) error {
	return nil
}

func (m *MockNodeRepository) List(ctx context.Context, filter node.NodeFilter) ([]*node.Node, int64, error) {
	return nil, 0, nil
}

func (m *MockNodeRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	return false, nil
}

func (m *MockNodeRepository) ExistsByAddress(ctx context.Context, address string, port int) (bool, error) {
	return false, nil
}

func (m *MockNodeRepository) IncrementTraffic(ctx context.Context, nodeID uint, amount uint64) error {
	return nil
}

func (m *MockNodeRepository) UpdateLastSeenAt(ctx context.Context, nodeID uint, publicIPv4, publicIPv6 string) error {
	return nil
}

func (m *MockNodeRepository) GetLastSeenAt(ctx context.Context, nodeID uint) (*time.Time, error) {
	return nil, nil
}

// Helper methods for testing

// AddNode adds a node to the mock repository (for test setup).
func (m *MockNodeRepository) AddNode(n *node.Node) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if n.ID() == 0 {
		m.nextID++
		_ = n.SetID(m.nextID)
	}

	m.nodes[n.ID()] = n
	m.nodesBySID[n.SID()] = n
}

// SetGetError sets the error to return on Get calls.
func (m *MockNodeRepository) SetGetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getError = err
}

// SetCreateError sets the error to return on Create calls.
func (m *MockNodeRepository) SetCreateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createError = err
}

// Reset resets the mock repository to its initial state.
func (m *MockNodeRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodes = make(map[uint]*node.Node)
	m.nodesBySID = make(map[string]*node.Node)
	m.nextID = 0
	m.getError = nil
	m.createError = nil
}

// MockConfigSyncNotifier is a mock implementation of ConfigSyncNotifier for testing.
type MockConfigSyncNotifier struct {
	mu    sync.RWMutex
	calls []NotifyCall
	err   error
}

// NotifyCall records a call to NotifyRuleChange.
type NotifyCall struct {
	AgentID     uint
	RuleShortID string
	ChangeType  string
}

// NewMockConfigSyncNotifier creates a new mock config sync notifier.
func NewMockConfigSyncNotifier() *MockConfigSyncNotifier {
	return &MockConfigSyncNotifier{
		calls: make([]NotifyCall, 0),
	}
}

// NotifyRuleChange notifies about a rule change.
func (m *MockConfigSyncNotifier) NotifyRuleChange(ctx context.Context, agentID uint, ruleShortID string, changeType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return m.err
	}

	m.calls = append(m.calls, NotifyCall{
		AgentID:     agentID,
		RuleShortID: ruleShortID,
		ChangeType:  changeType,
	})

	return nil
}

// GetCalls returns all recorded calls.
func (m *MockConfigSyncNotifier) GetCalls() []NotifyCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]NotifyCall(nil), m.calls...)
}

// SetError sets the error to return on NotifyRuleChange calls.
func (m *MockConfigSyncNotifier) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// Reset resets the mock to its initial state.
func (m *MockConfigSyncNotifier) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = make([]NotifyCall, 0)
	m.err = nil
}

// MockAgentStatusQuerier is a mock implementation of AgentStatusQuerier for testing.
type MockAgentStatusQuerier struct {
	mu       sync.RWMutex
	statuses map[uint]*dto.AgentStatusDTO
	err      error
}

// NewMockAgentStatusQuerier creates a new mock agent status querier.
func NewMockAgentStatusQuerier() *MockAgentStatusQuerier {
	return &MockAgentStatusQuerier{
		statuses: make(map[uint]*dto.AgentStatusDTO),
	}
}

// GetStatus returns the status for a specific agent.
func (m *MockAgentStatusQuerier) GetStatus(ctx context.Context, agentID uint) (*dto.AgentStatusDTO, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.err != nil {
		return nil, m.err
	}

	status, exists := m.statuses[agentID]
	if !exists {
		return nil, nil
	}

	return status, nil
}

// GetMultipleStatus returns the status for multiple agents.
func (m *MockAgentStatusQuerier) GetMultipleStatus(ctx context.Context, agentIDs []uint) (map[uint]*dto.AgentStatusDTO, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.err != nil {
		return nil, m.err
	}

	result := make(map[uint]*dto.AgentStatusDTO)
	for _, id := range agentIDs {
		if status, exists := m.statuses[id]; exists {
			result[id] = status
		}
	}

	return result, nil
}

// SetStatus sets the status for a specific agent (for test setup).
func (m *MockAgentStatusQuerier) SetStatus(agentID uint, status *dto.AgentStatusDTO) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statuses[agentID] = status
}

// SetError sets the error to return on Get calls.
func (m *MockAgentStatusQuerier) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// Reset resets the mock to its initial state.
func (m *MockAgentStatusQuerier) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statuses = make(map[uint]*dto.AgentStatusDTO)
	m.err = nil
}

// MockLogger is a mock implementation of Logger for testing.
type MockLogger struct {
	mu      sync.RWMutex
	entries []LogEntry
}

// LogEntry records a log call.
type LogEntry struct {
	Level   string
	Message string
	Fields  map[string]interface{}
}

// NewMockLogger creates a new mock logger.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		entries: make([]LogEntry, 0),
	}
}

// Debug logs a debug message.
func (m *MockLogger) Debug(msg string, args ...any) {
	m.log("DEBUG", msg, args...)
}

// Info logs an info message.
func (m *MockLogger) Info(msg string, args ...any) {
	m.log("INFO", msg, args...)
}

// Warn logs a warning message.
func (m *MockLogger) Warn(msg string, args ...any) {
	m.log("WARN", msg, args...)
}

// Error logs an error message.
func (m *MockLogger) Error(msg string, args ...any) {
	m.log("ERROR", msg, args...)
}

// Fatal logs a fatal message.
func (m *MockLogger) Fatal(msg string, args ...any) {
	m.log("FATAL", msg, args...)
}

// With returns a logger with additional fields.
func (m *MockLogger) With(args ...any) logger.Interface {
	return m
}

// Named returns a named logger.
func (m *MockLogger) Named(name string) logger.Interface {
	return m
}

// Debugw logs a debug message with key-value pairs.
func (m *MockLogger) Debugw(msg string, keysAndValues ...interface{}) {
	m.log("DEBUG", msg, keysAndValues...)
}

// Infow logs an info message with key-value pairs.
func (m *MockLogger) Infow(msg string, keysAndValues ...interface{}) {
	m.log("INFO", msg, keysAndValues...)
}

// Warnw logs a warning message with key-value pairs.
func (m *MockLogger) Warnw(msg string, keysAndValues ...interface{}) {
	m.log("WARN", msg, keysAndValues...)
}

// Errorw logs an error message with key-value pairs.
func (m *MockLogger) Errorw(msg string, keysAndValues ...interface{}) {
	m.log("ERROR", msg, keysAndValues...)
}

// Fatalw logs a fatal message with key-value pairs.
func (m *MockLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	m.log("FATAL", msg, keysAndValues...)
}

func (m *MockLogger) log(level, msg string, fields ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := LogEntry{
		Level:   level,
		Message: msg,
		Fields:  make(map[string]interface{}),
	}

	// Parse fields (key-value pairs)
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			entry.Fields[key] = fields[i+1]
		}
	}

	m.entries = append(m.entries, entry)
}

// GetEntries returns all logged entries.
func (m *MockLogger) GetEntries() []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]LogEntry(nil), m.entries...)
}

// Reset resets the mock logger.
func (m *MockLogger) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make([]LogEntry, 0)
}
