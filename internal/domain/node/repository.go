package node

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/shared/query"
)

// ExpiringNodeInfo holds lightweight node info for expiring notifications.
type ExpiringNodeInfo struct {
	ID        uint
	SID       string
	Name      string
	ExpiresAt time.Time
	CostLabel *string
}

// NodeMetadata holds lightweight node metadata for SSE broadcasting.
// This avoids loading full node entities with protocol configs.
type NodeMetadata struct {
	ID   uint
	SID  string
	Name string
}

type NodeRepository interface {
	Create(ctx context.Context, node *Node) error
	GetByID(ctx context.Context, id uint) (*Node, error)
	GetBySID(ctx context.Context, sid string) (*Node, error)
	GetBySIDs(ctx context.Context, sids []string) ([]*Node, error)
	GetByIDs(ctx context.Context, ids []uint) ([]*Node, error)
	// GetAllMetadata returns lightweight metadata for all nodes.
	// Used for SSE broadcasting where full entity is not needed.
	GetAllMetadata(ctx context.Context) ([]*NodeMetadata, error)
	// GetMetadataBySIDs returns lightweight metadata for nodes by SIDs.
	// Used for SSE broadcasting where full entity is not needed.
	GetMetadataBySIDs(ctx context.Context, sids []string) ([]*NodeMetadata, error)
	GetByToken(ctx context.Context, tokenHash string) (*Node, error)
	Update(ctx context.Context, node *Node) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filter NodeFilter) ([]*Node, int64, error)
	ExistsByName(ctx context.Context, name string) (bool, error)
	ExistsByNameExcluding(ctx context.Context, name string, excludeID uint) (bool, error)
	ExistsByAddress(ctx context.Context, address string, port int) (bool, error)
	ExistsByAddressExcluding(ctx context.Context, address string, port int, excludeID uint) (bool, error)
	IncrementTraffic(ctx context.Context, nodeID uint, amount uint64) error
	// UpdateLastSeenAt updates the last_seen_at timestamp, public IPs, and agent info for a node
	// Public IPs and agent info are optional - pass empty strings to skip updating them
	UpdateLastSeenAt(ctx context.Context, nodeID uint, publicIPv4, publicIPv6, agentVersion, platform, arch string) error
	// GetLastSeenAt retrieves the last_seen_at timestamp for a node
	GetLastSeenAt(ctx context.Context, nodeID uint) (*time.Time, error)
	// ListByUserID returns nodes owned by a specific user
	ListByUserID(ctx context.Context, userID uint, filter NodeFilter) ([]*Node, int64, error)
	// CountByUserID counts nodes owned by a specific user
	CountByUserID(ctx context.Context, userID uint) (int64, error)
	// ExistsByNameForUser checks if a node with the given name exists for a specific user
	ExistsByNameForUser(ctx context.Context, name string, userID uint) (bool, error)
	// ExistsByNameForUserExcluding checks if a node with the given name exists for a user, excluding a specific node
	ExistsByNameForUserExcluding(ctx context.Context, name string, userID uint, excludeID uint) (bool, error)
	// ExistsByAddressForUser checks if a node with the given address and port exists for a specific user
	ExistsByAddressForUser(ctx context.Context, address string, port int, userID uint) (bool, error)
	// ExistsByAddressForUserExcluding checks if a node with the given address and port exists for a user, excluding a specific node
	ExistsByAddressForUserExcluding(ctx context.Context, address string, port int, userID uint, excludeID uint) (bool, error)
	// GetPublicIPs retrieves the current public IPs for a node
	// Returns (ipv4, ipv6, error) - empty string if IP is not set
	GetPublicIPs(ctx context.Context, nodeID uint) (string, string, error)
	// UpdatePublicIP updates the public IP for a node (immediate, no throttling)
	// Pass empty string to skip updating that IP version
	UpdatePublicIP(ctx context.Context, nodeID uint, ipv4, ipv6 string) error
	// ValidateNodeSIDsForUser checks if all given node SIDs exist and belong to the specified user.
	// Returns slice of invalid SIDs (not found or not owned by user).
	ValidateNodeSIDsForUser(ctx context.Context, sids []string, userID uint) ([]string, error)
	// ValidateNodeSIDsExist checks if all given node SIDs exist (for admin nodes).
	// Returns slice of invalid SIDs (not found).
	ValidateNodeSIDsExist(ctx context.Context, sids []string) ([]string, error)

	// FindExpiringNodes returns nodes that will expire within the specified days.
	// Only returns nodes that have expires_at set and are not already expired.
	FindExpiringNodes(ctx context.Context, withinDays int) ([]*ExpiringNodeInfo, error)
}

type NodeFilter struct {
	query.BaseFilter
	Name      *string
	Status    *string
	Tag       *string
	UserID    *uint  // Filter by owner user ID
	GroupIDs  []uint // Filter by resource group IDs
	AdminOnly *bool  // If true, only return admin-created nodes (user_id IS NULL)
}
