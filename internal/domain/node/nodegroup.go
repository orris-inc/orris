package node

import (
	"fmt"
	"time"
)

// NodeGroup represents the node group aggregate root
type NodeGroup struct {
	id                  uint
	name                string
	description         string
	nodeIDs             []uint
	subscriptionPlanIDs []uint
	isPublic            bool
	sortOrder           int
	metadata            map[string]interface{}
	version             int
	createdAt           time.Time
	updatedAt           time.Time
}

// NewNodeGroup creates a new node group
func NewNodeGroup(name, description string, isPublic bool, sortOrder int) (*NodeGroup, error) {
	if name == "" {
		return nil, fmt.Errorf("node group name is required")
	}

	now := time.Now()
	ng := &NodeGroup{
		name:                name,
		description:         description,
		nodeIDs:             []uint{},
		subscriptionPlanIDs: []uint{},
		isPublic:            isPublic,
		sortOrder:           sortOrder,
		metadata:            make(map[string]interface{}),
		version:             1,
		createdAt:           now,
		updatedAt:           now,
	}

	return ng, nil
}

// ReconstructNodeGroup reconstructs a node group from persistence
func ReconstructNodeGroup(
	id uint,
	name, description string,
	nodeIDs, subscriptionPlanIDs []uint,
	isPublic bool,
	sortOrder int,
	metadata map[string]interface{},
	version int,
	createdAt, updatedAt time.Time,
) (*NodeGroup, error) {
	if id == 0 {
		return nil, fmt.Errorf("node group ID cannot be zero")
	}
	if name == "" {
		return nil, fmt.Errorf("node group name is required")
	}

	if nodeIDs == nil {
		nodeIDs = []uint{}
	}
	if subscriptionPlanIDs == nil {
		subscriptionPlanIDs = []uint{}
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &NodeGroup{
		id:                  id,
		name:                name,
		description:         description,
		nodeIDs:             nodeIDs,
		subscriptionPlanIDs: subscriptionPlanIDs,
		isPublic:            isPublic,
		sortOrder:           sortOrder,
		metadata:            metadata,
		version:             version,
		createdAt:           createdAt,
		updatedAt:           updatedAt,
	}, nil
}

// ID returns the node group ID
func (ng *NodeGroup) ID() uint {
	return ng.id
}

// Name returns the node group name
func (ng *NodeGroup) Name() string {
	return ng.name
}

// Description returns the node group description
func (ng *NodeGroup) Description() string {
	return ng.description
}

// NodeIDs returns the list of node IDs in this group
func (ng *NodeGroup) NodeIDs() []uint {
	ids := make([]uint, len(ng.nodeIDs))
	copy(ids, ng.nodeIDs)
	return ids
}

// SubscriptionPlanIDs returns the list of subscription plan IDs associated with this group
func (ng *NodeGroup) SubscriptionPlanIDs() []uint {
	ids := make([]uint, len(ng.subscriptionPlanIDs))
	copy(ids, ng.subscriptionPlanIDs)
	return ids
}

// IsPublic returns whether the node group is public
func (ng *NodeGroup) IsPublic() bool {
	return ng.isPublic
}

// SortOrder returns the sort order
func (ng *NodeGroup) SortOrder() int {
	return ng.sortOrder
}

// Metadata returns the node group metadata
func (ng *NodeGroup) Metadata() map[string]interface{} {
	return ng.metadata
}

// Version returns the aggregate version for optimistic locking
func (ng *NodeGroup) Version() int {
	return ng.version
}

// CreatedAt returns when the node group was created
func (ng *NodeGroup) CreatedAt() time.Time {
	return ng.createdAt
}

// UpdatedAt returns when the node group was last updated
func (ng *NodeGroup) UpdatedAt() time.Time {
	return ng.updatedAt
}

// SetID sets the node group ID (only for persistence layer use)
func (ng *NodeGroup) SetID(id uint) error {
	if ng.id != 0 {
		return fmt.Errorf("node group ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("node group ID cannot be zero")
	}
	ng.id = id
	return nil
}

// AddNode adds a node to the group
func (ng *NodeGroup) AddNode(nodeID uint) error {
	if nodeID == 0 {
		return fmt.Errorf("node ID cannot be zero")
	}

	if ng.containsNodeUnsafe(nodeID) {
		return nil
	}

	ng.nodeIDs = append(ng.nodeIDs, nodeID)
	ng.updatedAt = time.Now()
	ng.version++

	return nil
}

// RemoveNode removes a node from the group
func (ng *NodeGroup) RemoveNode(nodeID uint) error {
	if nodeID == 0 {
		return fmt.Errorf("node ID cannot be zero")
	}

	index := ng.findNodeIndexUnsafe(nodeID)
	if index == -1 {
		return nil
	}

	ng.nodeIDs = append(ng.nodeIDs[:index], ng.nodeIDs[index+1:]...)
	ng.updatedAt = time.Now()
	ng.version++

	return nil
}

// AddNodes adds multiple nodes to the group in batch
func (ng *NodeGroup) AddNodes(nodeIDs []uint) (addedCount int, err error) {
	if len(nodeIDs) == 0 {
		return 0, nil
	}

	addedCount = 0
	for _, nodeID := range nodeIDs {
		if nodeID == 0 {
			continue // Skip invalid IDs
		}

		if ng.containsNodeUnsafe(nodeID) {
			continue // Skip duplicates
		}

		ng.nodeIDs = append(ng.nodeIDs, nodeID)
		addedCount++
	}

	if addedCount > 0 {
		ng.updatedAt = time.Now()
		ng.version++
	}

	return addedCount, nil
}

// RemoveNodes removes multiple nodes from the group in batch
func (ng *NodeGroup) RemoveNodes(nodeIDs []uint) (removedCount int, err error) {
	if len(nodeIDs) == 0 {
		return 0, nil
	}

	// Create a set for faster lookup
	nodeIDSet := make(map[uint]bool)
	for _, nodeID := range nodeIDs {
		if nodeID != 0 {
			nodeIDSet[nodeID] = true
		}
	}

	// Filter out nodes that should be removed
	newNodeIDs := make([]uint, 0, len(ng.nodeIDs))
	removedCount = 0
	for _, nodeID := range ng.nodeIDs {
		if nodeIDSet[nodeID] {
			removedCount++
		} else {
			newNodeIDs = append(newNodeIDs, nodeID)
		}
	}

	if removedCount > 0 {
		ng.nodeIDs = newNodeIDs
		ng.updatedAt = time.Now()
		ng.version++
	}

	return removedCount, nil
}

// ContainsNode checks if the group contains a specific node
func (ng *NodeGroup) ContainsNode(nodeID uint) bool {
	return ng.containsNodeUnsafe(nodeID)
}

// AssociatePlan associates a subscription plan with this group
func (ng *NodeGroup) AssociatePlan(planID uint) error {
	if planID == 0 {
		return fmt.Errorf("plan ID cannot be zero")
	}

	if ng.containsPlanUnsafe(planID) {
		return nil
	}

	ng.subscriptionPlanIDs = append(ng.subscriptionPlanIDs, planID)
	ng.updatedAt = time.Now()
	ng.version++

	return nil
}

// DisassociatePlan removes a subscription plan association from this group
func (ng *NodeGroup) DisassociatePlan(planID uint) error {
	if planID == 0 {
		return fmt.Errorf("plan ID cannot be zero")
	}

	index := ng.findPlanIndexUnsafe(planID)
	if index == -1 {
		return nil
	}

	ng.subscriptionPlanIDs = append(ng.subscriptionPlanIDs[:index], ng.subscriptionPlanIDs[index+1:]...)
	ng.updatedAt = time.Now()
	ng.version++

	return nil
}

// IsAssociatedWithPlan checks if the group is associated with a specific plan
func (ng *NodeGroup) IsAssociatedWithPlan(planID uint) bool {
	return ng.containsPlanUnsafe(planID)
}

// NodeCount returns the number of nodes in the group
func (ng *NodeGroup) NodeCount() int {
	return len(ng.nodeIDs)
}

// UpdateName updates the node group name
func (ng *NodeGroup) UpdateName(name string) error {
	if name == "" {
		return fmt.Errorf("node group name is required")
	}

	if ng.name == name {
		return nil
	}

	ng.name = name
	ng.updatedAt = time.Now()
	ng.version++

	return nil
}

// UpdateDescription updates the node group description
func (ng *NodeGroup) UpdateDescription(description string) error {
	if ng.description == description {
		return nil
	}

	ng.description = description
	ng.updatedAt = time.Now()
	ng.version++

	return nil
}

// SetPublic updates the public visibility setting
func (ng *NodeGroup) SetPublic(isPublic bool) error {
	if ng.isPublic == isPublic {
		return nil
	}

	ng.isPublic = isPublic
	ng.updatedAt = time.Now()
	ng.version++

	return nil
}

// UpdateSortOrder updates the sort order
func (ng *NodeGroup) UpdateSortOrder(sortOrder int) error {
	if ng.sortOrder == sortOrder {
		return nil
	}

	ng.sortOrder = sortOrder
	ng.updatedAt = time.Now()
	ng.version++

	return nil
}

// containsNodeUnsafe checks if the group contains a node (without lock)
func (ng *NodeGroup) containsNodeUnsafe(nodeID uint) bool {
	return ng.findNodeIndexUnsafe(nodeID) != -1
}

// findNodeIndexUnsafe finds the index of a node in the group (without lock)
func (ng *NodeGroup) findNodeIndexUnsafe(nodeID uint) int {
	for i, id := range ng.nodeIDs {
		if id == nodeID {
			return i
		}
	}
	return -1
}

// containsPlanUnsafe checks if the group contains a plan (without lock)
func (ng *NodeGroup) containsPlanUnsafe(planID uint) bool {
	return ng.findPlanIndexUnsafe(planID) != -1
}

// findPlanIndexUnsafe finds the index of a plan in the group (without lock)
func (ng *NodeGroup) findPlanIndexUnsafe(planID uint) int {
	for i, id := range ng.subscriptionPlanIDs {
		if id == planID {
			return i
		}
	}
	return -1
}

// Validate performs domain-level validation
func (ng *NodeGroup) Validate() error {
	if ng.name == "" {
		return fmt.Errorf("node group name is required")
	}
	if ng.sortOrder < 0 {
		return fmt.Errorf("sort order cannot be negative")
	}
	return nil
}
