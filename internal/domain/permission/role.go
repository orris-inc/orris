package permission

import (
	"fmt"
	"time"
)

type RoleStatus string

const (
	RoleStatusActive   RoleStatus = "active"
	RoleStatusInactive RoleStatus = "inactive"
)

type Role struct {
	id          uint
	name        string
	slug        string
	description string
	status      RoleStatus
	isSystem    bool
	createdAt   time.Time
	updatedAt   time.Time
}

func NewRole(name, slug, description string) (*Role, error) {
	if name == "" {
		return nil, fmt.Errorf("role name is required")
	}
	if slug == "" {
		return nil, fmt.Errorf("role slug is required")
	}
	if len(name) > 50 {
		return nil, fmt.Errorf("role name too long (max 50 characters)")
	}
	if len(slug) > 50 {
		return nil, fmt.Errorf("role slug too long (max 50 characters)")
	}

	now := time.Now()
	return &Role{
		name:        name,
		slug:        slug,
		description: description,
		status:      RoleStatusActive,
		isSystem:    false,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

func ReconstructRole(id uint, name, slug, description string, status RoleStatus, isSystem bool, createdAt, updatedAt time.Time) (*Role, error) {
	if id == 0 {
		return nil, fmt.Errorf("role ID cannot be zero")
	}

	return &Role{
		id:          id,
		name:        name,
		slug:        slug,
		description: description,
		status:      status,
		isSystem:    isSystem,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}, nil
}

func (r *Role) ID() uint {
	return r.id
}

func (r *Role) SetID(id uint) error {
	if r.id != 0 {
		return fmt.Errorf("role ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("role ID cannot be zero")
	}
	r.id = id
	return nil
}

func (r *Role) Name() string {
	return r.name
}

func (r *Role) Slug() string {
	return r.slug
}

func (r *Role) Description() string {
	return r.description
}

func (r *Role) Status() RoleStatus {
	return r.status
}

func (r *Role) IsSystem() bool {
	return r.isSystem
}

func (r *Role) CreatedAt() time.Time {
	return r.createdAt
}

func (r *Role) UpdatedAt() time.Time {
	return r.updatedAt
}

func (r *Role) UpdateName(name string) error {
	if name == "" {
		return fmt.Errorf("role name cannot be empty")
	}
	if len(name) > 50 {
		return fmt.Errorf("role name too long (max 50 characters)")
	}
	r.name = name
	r.updatedAt = time.Now()
	return nil
}

func (r *Role) UpdateDescription(description string) {
	r.description = description
	r.updatedAt = time.Now()
}

func (r *Role) Activate() error {
	if r.status == RoleStatusActive {
		return nil
	}
	r.status = RoleStatusActive
	r.updatedAt = time.Now()
	return nil
}

func (r *Role) Deactivate() error {
	if r.isSystem {
		return fmt.Errorf("cannot deactivate system role")
	}
	if r.status == RoleStatusInactive {
		return nil
	}
	r.status = RoleStatusInactive
	r.updatedAt = time.Now()
	return nil
}

func (r *Role) IsActive() bool {
	return r.status == RoleStatusActive
}
