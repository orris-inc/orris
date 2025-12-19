package resource

import "errors"

var (
	// ErrGroupNotFound indicates the resource group was not found
	ErrGroupNotFound = errors.New("resource group not found")

	// ErrGroupNameExists indicates a resource group with the name already exists
	ErrGroupNameExists = errors.New("resource group name already exists")

	// ErrGroupHasResources indicates the group cannot be deleted because it has resources
	ErrGroupHasResources = errors.New("resource group has associated resources")

	// ErrInvalidGroupStatus indicates an invalid group status
	ErrInvalidGroupStatus = errors.New("invalid resource group status")

	// ErrVersionConflict indicates an optimistic locking conflict
	ErrVersionConflict = errors.New("version conflict: resource group was modified")

	// ErrGroupPlanTypeMismatchNode indicates the resource group's plan is not node type
	ErrGroupPlanTypeMismatchNode = errors.New("resource group's plan type is not node, cannot add node resources")

	// ErrGroupPlanTypeMismatchForward indicates the resource group's plan is not forward type
	ErrGroupPlanTypeMismatchForward = errors.New("resource group's plan type is not forward, cannot add forward agent resources")
)
