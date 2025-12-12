package valueobjects

type NodeStatus string

const (
	NodeStatusActive      NodeStatus = "active"
	NodeStatusInactive    NodeStatus = "inactive"
	NodeStatusMaintenance NodeStatus = "maintenance"
)

var validStatuses = map[NodeStatus]bool{
	NodeStatusActive:      true,
	NodeStatusInactive:    true,
	NodeStatusMaintenance: true,
}

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

func (ns NodeStatus) IsValid() bool {
	return validStatuses[ns]
}
