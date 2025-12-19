// Package entitlement provides domain models and business logic for entitlement management.
// It handles access control and resource authorization for users across different resources.
package entitlement

// SubjectType represents the type of entity that receives entitlement
type SubjectType string

const (
	// SubjectTypeUser represents a user subject (phase 1 only supports user)
	SubjectTypeUser SubjectType = "user"
	// Future: SubjectTypeUserGroup for group-based entitlements
)

// IsValid checks if the subject type is valid
func (st SubjectType) IsValid() bool {
	switch st {
	case SubjectTypeUser:
		return true
	default:
		return false
	}
}

// String returns the string representation of the subject type
func (st SubjectType) String() string {
	return string(st)
}

// ResourceType represents the type of resource being entitled
type ResourceType string

const (
	// ResourceTypeNode represents a node resource
	ResourceTypeNode ResourceType = "node"
	// ResourceTypeForwardAgent represents a forward agent resource
	ResourceTypeForwardAgent ResourceType = "forward_agent"
	// ResourceTypeFeature represents a feature resource
	ResourceTypeFeature ResourceType = "feature"
)

// IsValid checks if the resource type is valid
func (rt ResourceType) IsValid() bool {
	switch rt {
	case ResourceTypeNode, ResourceTypeForwardAgent, ResourceTypeFeature:
		return true
	default:
		return false
	}
}

// String returns the string representation of the resource type
func (rt ResourceType) String() string {
	return string(rt)
}

// SourceType represents the source of the entitlement
type SourceType string

const (
	// SourceTypeSubscription indicates entitlement from a subscription
	SourceTypeSubscription SourceType = "subscription"
	// SourceTypeDirect indicates direct entitlement grant
	SourceTypeDirect SourceType = "direct"
	// SourceTypePromotion indicates entitlement from a promotion
	SourceTypePromotion SourceType = "promotion"
	// SourceTypeTrial indicates entitlement from a trial
	SourceTypeTrial SourceType = "trial"
)

// IsValid checks if the source type is valid
func (st SourceType) IsValid() bool {
	switch st {
	case SourceTypeSubscription, SourceTypeDirect, SourceTypePromotion, SourceTypeTrial:
		return true
	default:
		return false
	}
}

// String returns the string representation of the source type
func (st SourceType) String() string {
	return string(st)
}

// EntitlementStatus represents the status of an entitlement
type EntitlementStatus string

const (
	// EntitlementStatusActive indicates the entitlement is active and valid
	EntitlementStatusActive EntitlementStatus = "active"
	// EntitlementStatusExpired indicates the entitlement has expired
	EntitlementStatusExpired EntitlementStatus = "expired"
	// EntitlementStatusRevoked indicates the entitlement has been revoked
	EntitlementStatusRevoked EntitlementStatus = "revoked"
)

// IsValid checks if the entitlement status is valid
func (es EntitlementStatus) IsValid() bool {
	switch es {
	case EntitlementStatusActive, EntitlementStatusExpired, EntitlementStatusRevoked:
		return true
	default:
		return false
	}
}

// String returns the string representation of the entitlement status
func (es EntitlementStatus) String() string {
	return string(es)
}

// IsActive checks if the status indicates an active entitlement
func (es EntitlementStatus) IsActive() bool {
	return es == EntitlementStatusActive
}

// IsExpired checks if the status indicates an expired entitlement
func (es EntitlementStatus) IsExpired() bool {
	return es == EntitlementStatusExpired
}

// IsRevoked checks if the status indicates a revoked entitlement
func (es EntitlementStatus) IsRevoked() bool {
	return es == EntitlementStatusRevoked
}
