package forward

import "context"

// Repository defines the interface for forward rule persistence.
type Repository interface {
	// Create persists a new forward rule.
	Create(ctx context.Context, rule *ForwardRule) error

	// GetByID retrieves a forward rule by ID.
	GetByID(ctx context.Context, id uint) (*ForwardRule, error)

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
}

// ListFilter defines the filtering options for listing forward rules.
type ListFilter struct {
	Page     int
	PageSize int
	AgentID  uint
	Name     string
	Protocol string
	Status   string
	OrderBy  string
	Order    string
}

// AgentRepository defines the interface for forward agent persistence.
type AgentRepository interface {
	// Create persists a new forward agent.
	Create(ctx context.Context, agent *ForwardAgent) error

	// GetByID retrieves a forward agent by ID.
	GetByID(ctx context.Context, id uint) (*ForwardAgent, error)

	// GetByTokenHash retrieves a forward agent by token hash.
	GetByTokenHash(ctx context.Context, tokenHash string) (*ForwardAgent, error)

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
}

// AgentListFilter defines the filtering options for listing forward agents.
type AgentListFilter struct {
	Page     int
	PageSize int
	Name     string
	Status   string
	OrderBy  string
	Order    string
}

// ChainRepository defines the interface for forward chain persistence.
type ChainRepository interface {
	// Create persists a new forward chain.
	Create(ctx context.Context, chain *ForwardChain) error

	// GetByID retrieves a forward chain by ID.
	GetByID(ctx context.Context, id uint) (*ForwardChain, error)

	// Update updates an existing forward chain.
	Update(ctx context.Context, chain *ForwardChain) error

	// Delete removes a forward chain.
	Delete(ctx context.Context, id uint) error

	// List returns all forward chains with optional filtering.
	List(ctx context.Context, filter ChainListFilter) ([]*ForwardChain, int64, error)

	// GetRuleIDsByChainID returns all rule IDs associated with a chain.
	GetRuleIDsByChainID(ctx context.Context, chainID uint) ([]uint, error)

	// AssociateRules associates rules with a chain.
	AssociateRules(ctx context.Context, chainID uint, ruleIDs []uint) error
}

// ChainListFilter defines the filtering options for listing forward chains.
type ChainListFilter struct {
	Page     int
	PageSize int
	Name     string
	Status   string
	OrderBy  string
	Order    string
}
