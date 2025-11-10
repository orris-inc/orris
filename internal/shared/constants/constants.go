package constants

const (
	// Environment constants
	EnvDevelopment = "development"
	EnvTest        = "test"
	EnvProduction  = "production"

	// Default pagination
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100

	// HTTP Headers
	HeaderContentType   = "Content-Type"
	HeaderAuthorization = "Authorization"
	HeaderXRequestID    = "X-Request-ID"
	HeaderXForwardedFor = "X-Forwarded-For"
	HeaderUserAgent     = "User-Agent"

	// Content Types
	ContentTypeJSON = "application/json"
	ContentTypeXML  = "application/xml"
	ContentTypeForm = "application/x-www-form-urlencoded"

	// API version prefix (removed for small project)
	// APIVersionPrefix = "/api/v1"

	// Context keys
	ContextKeyUserID    = "user_id"
	ContextKeyRequestID = "request_id"
	ContextKeyTraceID   = "trace_id"

	// User status
	UserStatusActive   = "active"
	UserStatusInactive = "inactive"
	UserStatusPending  = "pending"

	// Database table names
	TableUsers        = "users"
	TableRoles        = "roles"
	TableUserRoles    = "user_roles"
	TableNodes        = "nodes"
	TableNodeGroups   = "node_groups"
	TableNodeGroupNodes = "node_group_nodes"
	TableNodeGroupPlans = "node_group_plans"
	TableNodeTraffic  = "node_traffic"
	TableUserTraffic  = "user_traffic"

	// Default values
	DefaultUserStatus = UserStatusActive

	// Error messages
	ErrMsgInternalServerError = "Internal server error occurred"
	ErrMsgResourceNotFound    = "Resource not found"
	ErrMsgUnauthorized        = "Unauthorized access"
	ErrMsgForbidden           = "Access forbidden"
	ErrMsgValidationFailed    = "Validation failed"
	ErrMsgConflict            = "Resource already exists"
)
