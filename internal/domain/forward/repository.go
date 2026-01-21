package forward

import (
	"context"

	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
)

// Repository defines the interface for forward rule persistence.
type Repository interface {
	// Create persists a new forward rule.
	Create(ctx context.Context, rule *ForwardRule) error

	// GetByID retrieves a forward rule by ID.
	GetByID(ctx context.Context, id uint) (*ForwardRule, error)

	// GetBySID retrieves a forward rule by SID.
	GetBySID(ctx context.Context, sid string) (*ForwardRule, error)

	// GetBySIDs retrieves multiple forward rules by their SIDs.
	// Returns a map from SID to ForwardRule.
	GetBySIDs(ctx context.Context, sids []string) (map[string]*ForwardRule, error)

	// GetByIDs retrieves multiple forward rules by their internal IDs.
	// Returns a map from internal ID to ForwardRule.
	GetByIDs(ctx context.Context, ids []uint) (map[uint]*ForwardRule, error)

	// GetByListenPort retrieves a forward rule by listen port.
	GetByListenPort(ctx context.Context, port uint16) (*ForwardRule, error)

	// Update updates an existing forward rule.
	Update(ctx context.Context, rule *ForwardRule) error

	// Delete removes a forward rule.
	Delete(ctx context.Context, id uint) error

	// List returns all forward rules with optional filtering.
	List(ctx context.Context, filter ListFilter) ([]*ForwardRule, int64, error)

	// ListEnabled returns all enabled forward rules.
	ListEnabled(ctx context.Context) ([]*ForwardRule, error)

	// ListByAgentID returns all forward rules for a specific agent.
	ListByAgentID(ctx context.Context, agentID uint) ([]*ForwardRule, error)

	// ListEnabledByAgentID returns all enabled forward rules for a specific agent.
	ListEnabledByAgentID(ctx context.Context, agentID uint) ([]*ForwardRule, error)

	// ExistsByListenPort checks if a rule with the given listen port exists.
	ExistsByListenPort(ctx context.Context, port uint16) (bool, error)

	// ExistsByAgentIDAndListenPort checks if a rule with the given agent ID and listen port exists.
	// This is used for auto-assigning ports within an agent's scope.
	ExistsByAgentIDAndListenPort(ctx context.Context, agentID uint, port uint16) (bool, error)

	// IsPortInUseByAgent checks if a port is in use by the specified agent across all rules.
	// This includes both:
	// - Rules where agent_id matches and listen_port matches (main rule ports)
	// - Rules where chain_port_config contains the agent with the specified port
	// The excludeRuleID parameter can be used to exclude a specific rule from the check (useful for updates).
	IsPortInUseByAgent(ctx context.Context, agentID uint, port uint16, excludeRuleID uint) (bool, error)

	// UpdateTraffic updates the traffic counters for a rule.
	UpdateTraffic(ctx context.Context, id uint, upload, download int64) error

	// ListByExitAgentID returns all entrance rules for a specific exit agent.
	ListByExitAgentID(ctx context.Context, exitAgentID uint) ([]*ForwardRule, error)

	// ListEnabledByExitAgentID returns all enabled entry rules for a specific exit agent.
	ListEnabledByExitAgentID(ctx context.Context, exitAgentID uint) ([]*ForwardRule, error)

	// ListEnabledByChainAgentID returns all enabled chain rules where the agent participates.
	// This includes rules where the agent is in the chain_agent_ids array.
	ListEnabledByChainAgentID(ctx context.Context, agentID uint) ([]*ForwardRule, error)

	// ListByUserID returns forward rules for a specific user with filtering and pagination.
	ListByUserID(ctx context.Context, userID uint, filter ListFilter) ([]*ForwardRule, int64, error)

	// CountByUserID returns the total count of forward rules for a specific user.
	CountByUserID(ctx context.Context, userID uint) (int64, error)

	// ListBySubscriptionID returns all forward rules for a specific subscription.
	ListBySubscriptionID(ctx context.Context, subscriptionID uint) ([]*ForwardRule, error)

	// CountBySubscriptionID returns the total count of forward rules for a specific subscription.
	CountBySubscriptionID(ctx context.Context, subscriptionID uint) (int64, error)

	// GetTotalTrafficByUserID returns the total traffic (upload + download) for all rules owned by a user.
	GetTotalTrafficByUserID(ctx context.Context, userID uint) (int64, error)

	// UpdateSortOrders batch updates sort_order for multiple rules.
	UpdateSortOrders(ctx context.Context, ruleOrders map[uint]int) error

	// ListSystemRulesByTargetNodes returns enabled system rules targeting the specified nodes.
	// Only includes rules with system scope (user_id IS NULL or 0).
	// If groupIDs is not empty, only returns rules that belong to at least one of the specified resource groups.
	// This is used for Node Plan subscription delivery where user rules should be excluded.
	ListSystemRulesByTargetNodes(ctx context.Context, nodeIDs []uint, groupIDs []uint) ([]*ForwardRule, error)

	// ListUserRulesForDelivery returns enabled user rules for subscription delivery.
	// Only includes rules with user scope (user_id = userID) and target_node_id set.
	// This is used for Forward Plan subscription delivery.
	ListUserRulesForDelivery(ctx context.Context, userID uint) ([]*ForwardRule, error)

	// ListEnabledByTargetNodeID returns all enabled forward rules targeting a specific node.
	// This is used for notifying agents when a node's address changes.
	ListEnabledByTargetNodeID(ctx context.Context, nodeID uint) ([]*ForwardRule, error)

	// ListByGroupID returns all forward rules that belong to the specified resource group.
	// Uses JSON_CONTAINS to check if group_ids array contains the given group ID.
	// Supports pagination when page > 0 and pageSize > 0.
	ListByGroupID(ctx context.Context, groupID uint, page, pageSize int) ([]*ForwardRule, int64, error)

	// AddGroupIDAtomically adds a group ID to a rule's group_ids array atomically using JSON_ARRAY_APPEND.
	// Returns true if the group ID was added, false if it already exists.
	// This avoids read-modify-write race conditions.
	AddGroupIDAtomically(ctx context.Context, ruleID uint, groupID uint) (bool, error)

	// RemoveGroupIDAtomically removes a group ID from a rule's group_ids array atomically using JSON_REMOVE.
	// Returns true if the group ID was removed, false if it was not found.
	// This avoids read-modify-write race conditions.
	RemoveGroupIDAtomically(ctx context.Context, ruleID uint, groupID uint) (bool, error)

	// RemoveGroupIDFromAllRules removes a group ID from all rules that contain it.
	// This is used when deleting a resource group to clean up orphaned references.
	RemoveGroupIDFromAllRules(ctx context.Context, groupID uint) (int64, error)

	// ListByExternalSource returns all forward rules with the given external source.
	// Used for querying rules imported from a specific external system.
	ListByExternalSource(ctx context.Context, source string) ([]*ForwardRule, error)
}

// ListFilter defines the filtering options for listing forward rules.
type ListFilter struct {
	Page             int
	PageSize         int
	AgentID          uint
	UserID           *uint
	Scope            *vo.RuleScope // Filter by rule scope (system or user). Takes precedence over IncludeUserRules if set.
	IncludeUserRules bool          // When false (default), excludes rules with user_id set; when true, includes all rules
	Name             string
	Protocol         string
	Status           string
	RuleType         string // Filter by rule type (direct, entry, chain, direct_chain, external)
	ExternalSource   string // Filter by external source (only for external rules)
	OrderBy          string
	Order            string
	GroupIDs         []uint // Filter by resource group IDs (uses JSON_OVERLAPS on group_ids column)
}

// AgentMetadata holds lightweight agent metadata for SSE broadcasting.
// This avoids loading full agent entities.
type AgentMetadata struct {
	ID   uint
	SID  string
	Name string
}

// AgentRepository defines the interface for forward agent persistence.
type AgentRepository interface {
	// Create persists a new forward agent.
	Create(ctx context.Context, agent *ForwardAgent) error

	// GetByID retrieves a forward agent by ID.
	GetByID(ctx context.Context, id uint) (*ForwardAgent, error)

	// GetBySID retrieves a forward agent by SID.
	GetBySID(ctx context.Context, sid string) (*ForwardAgent, error)

	// GetBySIDs retrieves multiple forward agents by their SIDs.
	// Returns a slice of forward agents found.
	GetBySIDs(ctx context.Context, sids []string) ([]*ForwardAgent, error)

	// GetByTokenHash retrieves a forward agent by token hash.
	GetByTokenHash(ctx context.Context, tokenHash string) (*ForwardAgent, error)

	// GetAllEnabledMetadata returns lightweight metadata for all enabled agents.
	// Used for SSE broadcasting where full entity is not needed.
	GetAllEnabledMetadata(ctx context.Context) ([]*AgentMetadata, error)

	// GetMetadataBySIDs returns lightweight metadata for agents by SIDs.
	// Used for SSE broadcasting where full entity is not needed.
	GetMetadataBySIDs(ctx context.Context, sids []string) ([]*AgentMetadata, error)

	// GetSIDsByIDs retrieves SIDs for multiple agents by their internal IDs.
	// Returns a map from internal ID to SID.
	GetSIDsByIDs(ctx context.Context, ids []uint) (map[uint]string, error)

	// GetByIDs retrieves multiple forward agents by their internal IDs.
	// Returns a map from internal ID to ForwardAgent.
	GetByIDs(ctx context.Context, ids []uint) (map[uint]*ForwardAgent, error)

	// Update updates an existing forward agent.
	Update(ctx context.Context, agent *ForwardAgent) error

	// Delete removes a forward agent.
	Delete(ctx context.Context, id uint) error

	// List returns all forward agents with optional filtering.
	List(ctx context.Context, filter AgentListFilter) ([]*ForwardAgent, int64, error)

	// ListEnabled returns all enabled forward agents.
	ListEnabled(ctx context.Context) ([]*ForwardAgent, error)

	// ExistsByName checks if an agent with the given name exists.
	ExistsByName(ctx context.Context, name string) (bool, error)

	// UpdateLastSeen updates the last_seen_at timestamp for an agent.
	UpdateLastSeen(ctx context.Context, id uint) error

	// UpdateAgentInfo updates the agent info (version, platform, arch) for an agent.
	UpdateAgentInfo(ctx context.Context, id uint, agentVersion, platform, arch string) error
}

// AgentListFilter defines the filtering options for listing forward agents.
type AgentListFilter struct {
	Page     int
	PageSize int
	Name     string
	Status   string
	OrderBy  string
	Order    string
	GroupIDs []uint // Filter by resource group IDs
}
