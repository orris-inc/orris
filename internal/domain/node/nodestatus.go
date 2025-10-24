package node

import "fmt"

type NodeStatus string

const (
	NodeStatusActive      NodeStatus = "active"
	NodeStatusInactive    NodeStatus = "inactive"
	NodeStatusMaintenance NodeStatus = "maintenance"
)

var nodeStatusTransitions = map[NodeStatus][]NodeStatus{
	NodeStatusInactive: {
		NodeStatusActive,
	},
	NodeStatusActive: {
		NodeStatusInactive,
		NodeStatusMaintenance,
	},
	NodeStatusMaintenance: {
		NodeStatusActive,
		NodeStatusInactive,
	},
}

func NewNodeStatus(status string) (NodeStatus, error) {
	ns := NodeStatus(status)

	switch ns {
	case NodeStatusActive, NodeStatusInactive, NodeStatusMaintenance:
		return ns, nil
	default:
		return "", fmt.Errorf("invalid node status: %s", status)
	}
}

func (ns NodeStatus) String() string {
	return string(ns)
}

func (ns NodeStatus) IsActive() bool {
	return ns == NodeStatusActive
}

func (ns NodeStatus) IsInactive() bool {
	return ns == NodeStatusInactive
}

func (ns NodeStatus) IsMaintenance() bool {
	return ns == NodeStatusMaintenance
}

func (ns NodeStatus) CanTransitionTo(target NodeStatus) bool {
	allowedTransitions, ok := nodeStatusTransitions[ns]
	if !ok {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == target {
			return true
		}
	}
	return false
}

func (ns NodeStatus) ValidateTransition(target NodeStatus) error {
	if !ns.CanTransitionTo(target) {
		return fmt.Errorf("cannot transition from %s to %s", ns, target)
	}
	return nil
}

func (ns NodeStatus) Equals(other NodeStatus) bool {
	return ns == other
}

func GetAllNodeStatuses() []NodeStatus {
	return []NodeStatus{
		NodeStatusActive,
		NodeStatusInactive,
		NodeStatusMaintenance,
	}
}
