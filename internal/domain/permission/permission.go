package permission

import (
	"fmt"
	"time"

	vo "orris/internal/domain/permission/value_objects"
)

type Permission struct {
	id          uint
	resource    vo.Resource
	action      vo.Action
	description string
	createdAt   time.Time
	updatedAt   time.Time
}

func NewPermission(resource vo.Resource, action vo.Action, description string) (*Permission, error) {
	if resource == "" {
		return nil, fmt.Errorf("resource is required")
	}
	if action == "" {
		return nil, fmt.Errorf("action is required")
	}

	now := time.Now()
	return &Permission{
		resource:    resource,
		action:      action,
		description: description,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

func ReconstructPermission(id uint, resource vo.Resource, action vo.Action, description string, createdAt, updatedAt time.Time) (*Permission, error) {
	if id == 0 {
		return nil, fmt.Errorf("permission ID cannot be zero")
	}

	return &Permission{
		id:          id,
		resource:    resource,
		action:      action,
		description: description,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}, nil
}

func (p *Permission) ID() uint {
	return p.id
}

func (p *Permission) SetID(id uint) error {
	if p.id != 0 {
		return fmt.Errorf("permission ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("permission ID cannot be zero")
	}
	p.id = id
	return nil
}

func (p *Permission) Resource() vo.Resource {
	return p.resource
}

func (p *Permission) Action() vo.Action {
	return p.action
}

func (p *Permission) Description() string {
	return p.description
}

func (p *Permission) CreatedAt() time.Time {
	return p.createdAt
}

func (p *Permission) UpdatedAt() time.Time {
	return p.updatedAt
}

func (p *Permission) UpdateDescription(description string) {
	p.description = description
	p.updatedAt = time.Now()
}

func (p *Permission) Code() string {
	return fmt.Sprintf("%s:%s", p.resource, p.action)
}
