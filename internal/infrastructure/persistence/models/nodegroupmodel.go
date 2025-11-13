package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"orris/internal/shared/constants"
)

// NodeGroupModel represents the database persistence model for node groups
// This is the anti-corruption layer between domain and database
type NodeGroupModel struct {
	ID          uint   `gorm:"primarykey"`
	Name        string `gorm:"uniqueIndex;not null;size:100"`
	Description *string `gorm:"size:500"`
	IsPublic    bool   `gorm:"not null;default:false;index:idx_is_public"`
	SortOrder   int    `gorm:"not null;default:0"`
	Metadata    datatypes.JSON
	Version     int `gorm:"not null;default:1"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for GORM
func (NodeGroupModel) TableName() string {
	return constants.TableNodeGroups
}

// BeforeCreate hook for GORM
func (g *NodeGroupModel) BeforeCreate(tx *gorm.DB) error {
	if g.Version == 0 {
		g.Version = 1
	}
	return nil
}

// BeforeUpdate implements optimistic locking
func (g *NodeGroupModel) BeforeUpdate(tx *gorm.DB) error {
	// Increment version for optimistic locking
	tx.Statement.SetColumn("version", g.Version+1)
	return nil
}

// NodeGroupNodeModel represents the many-to-many relationship between node groups and nodes
type NodeGroupNodeModel struct {
	ID          uint      `gorm:"primarykey"`
	NodeGroupID uint      `gorm:"not null;index:idx_node_group_node"`
	NodeID      uint      `gorm:"not null;index:idx_node_group_node"`
	CreatedAt   time.Time

	// Note: No foreign key constraints or associations.
	// All relationships are managed by application business logic.
}

// TableName specifies the table name for GORM
func (NodeGroupNodeModel) TableName() string {
	return constants.TableNodeGroupNodes
}

// NodeGroupPlanModel represents the many-to-many relationship between node groups and subscription plans
type NodeGroupPlanModel struct {
	ID                 uint      `gorm:"primarykey"`
	NodeGroupID        uint      `gorm:"not null;index:idx_node_group_plan"`
	SubscriptionPlanID uint      `gorm:"not null;index:idx_node_group_plan"`
	CreatedAt          time.Time

	// Note: No foreign key constraints or associations.
	// All relationships are managed by application business logic.
}

// TableName specifies the table name for GORM
func (NodeGroupPlanModel) TableName() string {
	return constants.TableNodeGroupPlans
}
