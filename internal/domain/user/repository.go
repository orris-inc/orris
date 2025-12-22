package user

import "context"

// Repository defines the interface for user data operations
type Repository interface {
	// Create creates a new user
	Create(ctx context.Context, user *User) error

	// GetByID retrieves a user by internal ID
	GetByID(ctx context.Context, id uint) (*User, error)

	// GetByIDs retrieves multiple users by internal IDs
	GetByIDs(ctx context.Context, ids []uint) ([]*User, error)

	// GetBySID retrieves a user by external SID (Stripe-style ID)
	GetBySID(ctx context.Context, sid string) (*User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *User) error

	// Delete soft deletes a user by internal ID
	Delete(ctx context.Context, id uint) error

	// DeleteBySID soft deletes a user by external SID
	DeleteBySID(ctx context.Context, sid string) error

	// List retrieves a paginated list of users
	List(ctx context.Context, filter ListFilter) ([]*User, int64, error)

	// Exists checks if a user exists by internal ID
	Exists(ctx context.Context, id uint) (bool, error)

	// ExistsByEmail checks if a user exists by email
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// GetByVerificationToken retrieves a user by email verification token
	GetByVerificationToken(ctx context.Context, token string) (*User, error)

	// GetByPasswordResetToken retrieves a user by password reset token
	GetByPasswordResetToken(ctx context.Context, token string) (*User, error)
}

// ListFilter represents filtering and pagination options for user list
type ListFilter struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Email    string `json:"email,omitempty"`
	Name     string `json:"name,omitempty"`
	Status   string `json:"status,omitempty"`
	Role     string `json:"role,omitempty"`
	OrderBy  string `json:"order_by,omitempty"` // field to order by
	Order    string `json:"order,omitempty"`    // asc or desc
}
