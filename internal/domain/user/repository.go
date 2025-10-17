package user

import "context"

// Repository defines the interface for user data operations
type Repository interface {
	// Create creates a new user
	Create(ctx context.Context, user *User) error

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id uint) (*User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *User) error

	// Delete soft deletes a user
	Delete(ctx context.Context, id uint) error

	// List retrieves a paginated list of users
	List(ctx context.Context, filter ListFilter) ([]*User, int64, error)

	// Exists checks if a user exists by ID
	Exists(ctx context.Context, id uint) (bool, error)

	// ExistsByEmail checks if a user exists by email
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// GetByVerificationToken retrieves a user by email verification token
	GetByVerificationToken(ctx context.Context, token string) (*User, error)

	// GetByPasswordResetToken retrieves a user by password reset token
	GetByPasswordResetToken(ctx context.Context, token string) (*User, error)
}

// RepositoryWithSpecifications extends Repository with specification-based queries
type RepositoryWithSpecifications interface {
	Repository
	
	// FindBySpecification finds users matching a specification
	FindBySpecification(ctx context.Context, spec interface{}, limit int) ([]*User, error)
}

// ListFilter represents filtering and pagination options for user list
type ListFilter struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Email    string `json:"email,omitempty"`
	Name     string `json:"name,omitempty"`
	Status   string `json:"status,omitempty"`
	OrderBy  string `json:"order_by,omitempty"` // field to order by
	Order    string `json:"order,omitempty"`    // asc or desc
}