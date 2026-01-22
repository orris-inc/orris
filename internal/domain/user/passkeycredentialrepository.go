package user

import "context"

// PasskeyCredentialRepository defines the interface for passkey credential data operations
type PasskeyCredentialRepository interface {
	// Create creates a new passkey credential
	Create(ctx context.Context, credential *PasskeyCredential) error

	// GetByID retrieves a passkey credential by internal ID
	GetByID(ctx context.Context, id uint) (*PasskeyCredential, error)

	// GetBySID retrieves a passkey credential by external SID (pk_xxx)
	GetBySID(ctx context.Context, sid string) (*PasskeyCredential, error)

	// GetByCredentialID retrieves a passkey credential by WebAuthn credential ID
	GetByCredentialID(ctx context.Context, credentialID []byte) (*PasskeyCredential, error)

	// GetByUserID retrieves all passkey credentials for a user
	GetByUserID(ctx context.Context, userID uint) ([]*PasskeyCredential, error)

	// Update updates an existing passkey credential
	Update(ctx context.Context, credential *PasskeyCredential) error

	// Delete deletes a passkey credential by internal ID
	Delete(ctx context.Context, id uint) error

	// DeleteBySID deletes a passkey credential by external SID
	DeleteBySID(ctx context.Context, sid string) error

	// CountByUserID returns the count of passkey credentials for a user
	CountByUserID(ctx context.Context, userID uint) (int64, error)

	// ExistsByCredentialID checks if a credential with the given WebAuthn credential ID exists
	ExistsByCredentialID(ctx context.Context, credentialID []byte) (bool, error)
}
