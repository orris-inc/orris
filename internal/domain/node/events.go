package node

import (
	"fmt"
	"time"

	"orris/internal/domain/shared/events"
)

const (
	EventTypeNodeCreated           = "node.created"
	EventTypeNodeUpdated           = "node.updated"
	EventTypeNodeDeleted           = "node.deleted"
	EventTypeNodeStatusChanged     = "node.status.changed"
	EventTypeNodeGroupCreated      = "node_group.created"
	EventTypeNodeGroupUpdated      = "node_group.updated"
	EventTypeSubscriptionGenerated = "subscription.generated"
	EventTypeNodeTrafficReported   = "node.traffic.reported"
	EventTypeNodeTrafficExceeded   = "node.traffic.exceeded"
)

type NodeCreatedEvent struct {
	events.BaseEvent
	NodeID        uint      `json:"node_id"`
	Name          string    `json:"name"`
	ServerAddress string    `json:"server_address"`
	ServerPort    uint16    `json:"server_port"`
	Status        string    `json:"status"`
	CreatedBy     uint      `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

func NewNodeCreatedEvent(nodeID uint, name, serverAddress string, serverPort uint16, status string, createdBy uint) NodeCreatedEvent {
	return NodeCreatedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("node:%d", nodeID),
			EventType:   EventTypeNodeCreated,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		NodeID:        nodeID,
		Name:          name,
		ServerAddress: serverAddress,
		ServerPort:    serverPort,
		Status:        status,
		CreatedBy:     createdBy,
		CreatedAt:     time.Now(),
	}
}

type NodeUpdatedEvent struct {
	events.BaseEvent
	NodeID        uint                   `json:"node_id"`
	UpdatedFields []string               `json:"updated_fields"`
	OldValues     map[string]interface{} `json:"old_values"`
	NewValues     map[string]interface{} `json:"new_values"`
	UpdatedBy     uint                   `json:"updated_by"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

func NewNodeUpdatedEvent(nodeID uint, updatedFields []string, oldValues, newValues map[string]interface{}, updatedBy uint) NodeUpdatedEvent {
	return NodeUpdatedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("node:%d", nodeID),
			EventType:   EventTypeNodeUpdated,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		NodeID:        nodeID,
		UpdatedFields: updatedFields,
		OldValues:     oldValues,
		NewValues:     newValues,
		UpdatedBy:     updatedBy,
		UpdatedAt:     time.Now(),
	}
}

type NodeDeletedEvent struct {
	events.BaseEvent
	NodeID    uint      `json:"node_id"`
	Name      string    `json:"name"`
	DeletedBy uint      `json:"deleted_by"`
	DeletedAt time.Time `json:"deleted_at"`
}

func NewNodeDeletedEvent(nodeID uint, name string, deletedBy uint) NodeDeletedEvent {
	return NodeDeletedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("node:%d", nodeID),
			EventType:   EventTypeNodeDeleted,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		NodeID:    nodeID,
		Name:      name,
		DeletedBy: deletedBy,
		DeletedAt: time.Now(),
	}
}

type NodeStatusChangedEvent struct {
	events.BaseEvent
	NodeID    uint      `json:"node_id"`
	OldStatus string    `json:"old_status"`
	NewStatus string    `json:"new_status"`
	Reason    string    `json:"reason"`
	ChangedAt time.Time `json:"changed_at"`
}

func NewNodeStatusChangedEvent(nodeID uint, oldStatus, newStatus, reason string) NodeStatusChangedEvent {
	return NodeStatusChangedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("node:%d", nodeID),
			EventType:   EventTypeNodeStatusChanged,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		NodeID:    nodeID,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Reason:    reason,
		ChangedAt: time.Now(),
	}
}

type NodeGroupCreatedEvent struct {
	events.BaseEvent
	GroupID     uint      `json:"group_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsPublic    bool      `json:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewNodeGroupCreatedEvent(groupID uint, name, description string, isPublic bool) NodeGroupCreatedEvent {
	return NodeGroupCreatedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("node_group:%d", groupID),
			EventType:   EventTypeNodeGroupCreated,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		GroupID:     groupID,
		Name:        name,
		Description: description,
		IsPublic:    isPublic,
		CreatedAt:   time.Now(),
	}
}

type NodeGroupUpdatedEvent struct {
	events.BaseEvent
	GroupID    uint      `json:"group_id"`
	UpdateType string    `json:"update_type"`
	NodeIDs    []uint    `json:"node_ids"`
	UpdatedBy  uint      `json:"updated_by"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func NewNodeGroupUpdatedEvent(groupID uint, updateType string, nodeIDs []uint, updatedBy uint) NodeGroupUpdatedEvent {
	return NodeGroupUpdatedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("node_group:%d", groupID),
			EventType:   EventTypeNodeGroupUpdated,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		GroupID:    groupID,
		UpdateType: updateType,
		NodeIDs:    nodeIDs,
		UpdatedBy:  updatedBy,
		UpdatedAt:  time.Now(),
	}
}

type SubscriptionGeneratedEvent struct {
	events.BaseEvent
	SubscriptionID uint      `json:"subscription_id"`
	UserID         uint      `json:"user_id"`
	Format         string    `json:"format"`
	NodeCount      int       `json:"node_count"`
	ClientIP       string    `json:"client_ip"`
	UserAgent      string    `json:"user_agent"`
	GeneratedAt    time.Time `json:"generated_at"`
}

func NewSubscriptionGeneratedEvent(subscriptionID, userID uint, format string, nodeCount int, clientIP, userAgent string) SubscriptionGeneratedEvent {
	return SubscriptionGeneratedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("subscription:%d", subscriptionID),
			EventType:   EventTypeSubscriptionGenerated,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		SubscriptionID: subscriptionID,
		UserID:         userID,
		Format:         format,
		NodeCount:      nodeCount,
		ClientIP:       clientIP,
		UserAgent:      userAgent,
		GeneratedAt:    time.Now(),
	}
}

type NodeTrafficReportedEvent struct {
	events.BaseEvent
	NodeID     uint      `json:"node_id"`
	Upload     uint64    `json:"upload"`
	Download   uint64    `json:"download"`
	ReportedAt time.Time `json:"reported_at"`
}

func NewNodeTrafficReportedEvent(nodeID uint, upload, download uint64) NodeTrafficReportedEvent {
	return NodeTrafficReportedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("node:%d", nodeID),
			EventType:   EventTypeNodeTrafficReported,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		NodeID:     nodeID,
		Upload:     upload,
		Download:   download,
		ReportedAt: time.Now(),
	}
}

type NodeTrafficExceededEvent struct {
	events.BaseEvent
	NodeID       uint      `json:"node_id"`
	TrafficLimit uint64    `json:"traffic_limit"`
	TrafficUsed  uint64    `json:"traffic_used"`
	ExceededAt   time.Time `json:"exceeded_at"`
}

func NewNodeTrafficExceededEvent(nodeID uint, trafficLimit, trafficUsed uint64) NodeTrafficExceededEvent {
	return NodeTrafficExceededEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("node:%d", nodeID),
			EventType:   EventTypeNodeTrafficExceeded,
			OccurredAt:  time.Now(),
			Version:     1,
		},
		NodeID:       nodeID,
		TrafficLimit: trafficLimit,
		TrafficUsed:  trafficUsed,
		ExceededAt:   time.Now(),
	}
}

type NodeAddedToGroupEvent struct {
	events.BaseEvent
	NodeGroupID uint      `json:"node_group_id"`
	NodeID      uint      `json:"node_id"`
	AddedAt     time.Time `json:"added_at"`
}

func NewNodeAddedToGroupEvent(nodeGroupID, nodeID uint, addedAt time.Time) NodeAddedToGroupEvent {
	return NodeAddedToGroupEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("node_group:%d", nodeGroupID),
			EventType:   "node_group.node_added",
			OccurredAt:  addedAt,
			Version:     1,
		},
		NodeGroupID: nodeGroupID,
		NodeID:      nodeID,
		AddedAt:     addedAt,
	}
}

type NodeRemovedFromGroupEvent struct {
	events.BaseEvent
	NodeGroupID uint      `json:"node_group_id"`
	NodeID      uint      `json:"node_id"`
	RemovedAt   time.Time `json:"removed_at"`
}

func NewNodeRemovedFromGroupEvent(nodeGroupID, nodeID uint, removedAt time.Time) NodeRemovedFromGroupEvent {
	return NodeRemovedFromGroupEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("node_group:%d", nodeGroupID),
			EventType:   "node_group.node_removed",
			OccurredAt:  removedAt,
			Version:     1,
		},
		NodeGroupID: nodeGroupID,
		NodeID:      nodeID,
		RemovedAt:   removedAt,
	}
}

type PlanAssociatedWithGroupEvent struct {
	events.BaseEvent
	NodeGroupID  uint      `json:"node_group_id"`
	PlanID       uint      `json:"plan_id"`
	AssociatedAt time.Time `json:"associated_at"`
}

func NewPlanAssociatedWithGroupEvent(nodeGroupID, planID uint, associatedAt time.Time) PlanAssociatedWithGroupEvent {
	return PlanAssociatedWithGroupEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("node_group:%d", nodeGroupID),
			EventType:   "node_group.plan_associated",
			OccurredAt:  associatedAt,
			Version:     1,
		},
		NodeGroupID:  nodeGroupID,
		PlanID:       planID,
		AssociatedAt: associatedAt,
	}
}

type PlanDisassociatedFromGroupEvent struct {
	events.BaseEvent
	NodeGroupID     uint      `json:"node_group_id"`
	PlanID          uint      `json:"plan_id"`
	DisassociatedAt time.Time `json:"disassociated_at"`
}

func NewPlanDisassociatedFromGroupEvent(nodeGroupID, planID uint, disassociatedAt time.Time) PlanDisassociatedFromGroupEvent {
	return PlanDisassociatedFromGroupEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("node_group:%d", nodeGroupID),
			EventType:   "node_group.plan_disassociated",
			OccurredAt:  disassociatedAt,
			Version:     1,
		},
		NodeGroupID:     nodeGroupID,
		PlanID:          planID,
		DisassociatedAt: disassociatedAt,
	}
}

type NodeGroupNameChangedEvent struct {
	events.BaseEvent
	NodeGroupID uint      `json:"node_group_id"`
	OldName     string    `json:"old_name"`
	NewName     string    `json:"new_name"`
	ChangedAt   time.Time `json:"changed_at"`
}

func NewNodeGroupNameChangedEvent(nodeGroupID uint, oldName, newName string, changedAt time.Time) NodeGroupNameChangedEvent {
	return NodeGroupNameChangedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("node_group:%d", nodeGroupID),
			EventType:   "node_group.name_changed",
			OccurredAt:  changedAt,
			Version:     1,
		},
		NodeGroupID: nodeGroupID,
		OldName:     oldName,
		NewName:     newName,
		ChangedAt:   changedAt,
	}
}

type NodeGroupVisibilityChangedEvent struct {
	events.BaseEvent
	NodeGroupID uint      `json:"node_group_id"`
	IsPublic    bool      `json:"is_public"`
	ChangedAt   time.Time `json:"changed_at"`
}

func NewNodeGroupVisibilityChangedEvent(nodeGroupID uint, isPublic bool, changedAt time.Time) NodeGroupVisibilityChangedEvent {
	return NodeGroupVisibilityChangedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("node_group:%d", nodeGroupID),
			EventType:   "node_group.visibility_changed",
			OccurredAt:  changedAt,
			Version:     1,
		},
		NodeGroupID: nodeGroupID,
		IsPublic:    isPublic,
		ChangedAt:   changedAt,
	}
}

type NodeTrafficRecordedEvent struct {
	events.BaseEvent
	TrafficID      uint   `json:"traffic_id"`
	NodeID         uint   `json:"node_id"`
	UserID         *uint  `json:"user_id"`
	SubscriptionID *uint  `json:"subscription_id"`
	Upload         uint64 `json:"upload"`
	Download       uint64 `json:"download"`
	Total          uint64 `json:"total"`
}

func NewNodeTrafficRecordedEvent(trafficID, nodeID uint, userID, subscriptionID *uint, upload, download, total uint64) NodeTrafficRecordedEvent {
	return NodeTrafficRecordedEvent{
		BaseEvent: events.BaseEvent{
			AggregateID: fmt.Sprintf("node_traffic:%d", trafficID),
			EventType:   "node_traffic.recorded",
			OccurredAt:  time.Now(),
			Version:     1,
		},
		TrafficID:      trafficID,
		NodeID:         nodeID,
		UserID:         userID,
		SubscriptionID: subscriptionID,
		Upload:         upload,
		Download:       download,
		Total:          total,
	}
}
