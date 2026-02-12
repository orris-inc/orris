package node

import (
	"fmt"

	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
)

// Activate activates the node
func (n *Node) Activate() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status == vo.NodeStatusActive {
		return nil
	}

	if !n.status.CanTransitionTo(vo.NodeStatusActive) {
		return fmt.Errorf("cannot activate node with status %s", n.status)
	}

	n.status = vo.NodeStatusActive
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// Deactivate deactivates the node
func (n *Node) Deactivate() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status == vo.NodeStatusInactive {
		return nil
	}

	if !n.status.CanTransitionTo(vo.NodeStatusInactive) {
		return fmt.Errorf("cannot deactivate node with status %s", n.status)
	}

	n.status = vo.NodeStatusInactive
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// EnterMaintenance puts the node into maintenance mode
func (n *Node) EnterMaintenance(reason string) error {
	if reason == "" {
		return fmt.Errorf("maintenance reason is required")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status == vo.NodeStatusMaintenance {
		return nil
	}

	if !n.status.CanTransitionTo(vo.NodeStatusMaintenance) {
		return fmt.Errorf("cannot enter maintenance mode from status %s", n.status)
	}

	n.status = vo.NodeStatusMaintenance
	n.maintenanceReason = &reason
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// ExitMaintenance exits maintenance mode and returns to active status
func (n *Node) ExitMaintenance() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status != vo.NodeStatusMaintenance {
		return fmt.Errorf("node is not in maintenance mode")
	}

	n.status = vo.NodeStatusActive
	n.maintenanceReason = nil
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}
