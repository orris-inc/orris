package user

import (
	"fmt"

	vo "orris/internal/domain/user/value_objects"
)

// Factory is responsible for creating User aggregates
type Factory interface {
	// CreateUser creates a new user with validation
	CreateUser(email, name string) (*User, error)
	
	// CreateUserWithStatus creates a new user with a specific status
	CreateUserWithStatus(email, name, status string) (*User, error)
	
	// CreateBusinessUser creates a new user with business email validation
	CreateBusinessUser(email, name string) (*User, error)
}

// UserFactory is the concrete implementation of Factory
type UserFactory struct {
	// Can inject dependencies like ID generator, validators, etc.
}

// NewUserFactory creates a new user factory
func NewUserFactory() *UserFactory {
	return &UserFactory{}
}

// CreateUser creates a new user with default status (pending)
func (f *UserFactory) CreateUser(email, name string) (*User, error) {
	// Create and validate email value object
	emailVO, err := vo.NewEmail(email)
	if err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}
	
	// Create and validate name value object
	nameVO, err := vo.NewName(name)
	if err != nil {
		return nil, fmt.Errorf("invalid name: %w", err)
	}
	
	// Create the user aggregate
	user, err := NewUser(emailVO, nameVO)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	return user, nil
}

// CreateUserWithStatus creates a new user with a specific status
func (f *UserFactory) CreateUserWithStatus(email, name, status string) (*User, error) {
	// Create and validate email value object
	emailVO, err := vo.NewEmail(email)
	if err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}
	
	// Create and validate name value object
	nameVO, err := vo.NewName(name)
	if err != nil {
		return nil, fmt.Errorf("invalid name: %w", err)
	}
	
	// Create and validate status value object
	statusVO, err := vo.NewStatus(status)
	if err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}
	
	// Create the user aggregate
	user, err := NewUser(emailVO, nameVO)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	// Set the specific status if not pending
	if *statusVO != vo.StatusPending {
		user.status = *statusVO
	}
	
	return user, nil
}

// CreateBusinessUser creates a new user with business email validation
func (f *UserFactory) CreateBusinessUser(email, name string) (*User, error) {
	// Create and validate email value object
	emailVO, err := vo.NewEmail(email)
	if err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}
	
	// Ensure it's a business email
	if !emailVO.IsBusinessEmail() {
		return nil, fmt.Errorf("business email required, got: %s", emailVO.Domain())
	}
	
	// Create and validate name value object
	nameVO, err := vo.NewName(name)
	if err != nil {
		return nil, fmt.Errorf("invalid name: %w", err)
	}
	
	// Create the user aggregate
	user, err := NewUser(emailVO, nameVO)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	// Business users can be auto-activated based on business rules
	// This is an example of encapsulating business logic in the factory
	if emailVO.IsBusinessEmail() {
		// Could auto-activate based on trusted domains
		trustedDomains := map[string]bool{
			"company.com": true,
			"partner.org": true,
		}
		
		if trustedDomains[emailVO.Domain()] {
			user.status = vo.StatusActive
		}
	}
	
	return user, nil
}

// UserBuilder provides a fluent interface for building users
type UserBuilder struct {
	email  string
	name   string
	status string
	errors []error
}

// NewUserBuilder creates a new user builder
func NewUserBuilder() *UserBuilder {
	return &UserBuilder{}
}

// WithEmail sets the email
func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.email = email
	return b
}

// WithName sets the name
func (b *UserBuilder) WithName(name string) *UserBuilder {
	b.name = name
	return b
}

// WithStatus sets the status
func (b *UserBuilder) WithStatus(status string) *UserBuilder {
	b.status = status
	return b
}

// Build creates the user aggregate
func (b *UserBuilder) Build() (*User, error) {
	// Check for accumulated errors
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("builder has errors: %v", b.errors)
	}
	
	// Create value objects
	emailVO, err := vo.NewEmail(b.email)
	if err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}
	
	nameVO, err := vo.NewName(b.name)
	if err != nil {
		return nil, fmt.Errorf("invalid name: %w", err)
	}
	
	// Create user
	user, err := NewUser(emailVO, nameVO)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	// Set status if provided
	if b.status != "" {
		statusVO, err := vo.NewStatus(b.status)
		if err != nil {
			return nil, fmt.Errorf("invalid status: %w", err)
		}
		user.status = *statusVO
	}
	
	return user, nil
}

// Validate checks if the builder state is valid
func (b *UserBuilder) Validate() []error {
	var errors []error
	
	if b.email == "" {
		errors = append(errors, fmt.Errorf("email is required"))
	}
	
	if b.name == "" {
		errors = append(errors, fmt.Errorf("name is required"))
	}
	
	return errors
}