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

	// GetTotalTrafficByUserID returns the total traffic (upload + download) for all rules owned by a user.
	GetTotalTrafficByUserID(ctx context.Context, userID uint) (int64, error)

	// UpdateSortOrders batch updates sort_order for multiple rules.
	UpdateSortOrders(ctx context.Context, ruleOrders map[uint]int) error

	// ListSystemRulesByTargetNodes returns enabled system rules targeting the specified nodes.
	// Only includes rules with system scope (user_id IS NULL or 0).
	// This is used for Node Plan subscription delivery where user rules should be excluded.
	ListSystemRulesByTargetNodes(ctx context.Context, nodeIDs []uint) ([]*ForwardRule, error)

	// ListUserRulesForDelivery returns enabled user rules for subscription delivery.
	// Only includes rules with user scope (user_id = userID) and target_node_id set.
	// This is used for Forward Plan subscription delivery.
	ListUserRulesForDelivery(ctx context.Context, userID uint) ([]*ForwardRule, error)

	// ListEnabledByTargetNodeID returns all enabled forward rules targeting a specific node.
	// This is used for notifying agents when a node's address changes.
	ListEnabledByTargetNodeID(ctx context.Context, nodeID uint) ([]*ForwardRule, error)
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
	OrderBy          string
	Order            string
}

// AgentRepository defines the interface for forward agent persistence.
type AgentRepository interface {
	// Create persists a new forward agent.
	Create(ctx context.Context, agent *ForwardAgent) error

	// GetByID retrieves a forward agent by ID.
	GetByID(ctx context.Context, id uint) (*ForwardAgent, error)

	// GetBySID retrieves a forward agent by SID.
	GetBySID(ctx context.Context, sid string) (*ForwardAgent, error)

	// GetByTokenHash retrieves a forward agent by token hash.
	GetByTokenHash(ctx context.Context, tokenHash string) (*ForwardAgent, error)

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
