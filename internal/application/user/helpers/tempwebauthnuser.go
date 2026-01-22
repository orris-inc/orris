package helpers

import (
	"crypto/rand"

	"github.com/go-webauthn/webauthn/webauthn"
)

// TempWebAuthnUser adapts temporary user data to the webauthn.User interface
// for passkey registration before the actual user is created in the database
type TempWebAuthnUser struct {
	id          []byte
	email       string
	displayName string
}

// NewTempWebAuthnUser creates a new temporary WebAuthn user adapter
func NewTempWebAuthnUser(id []byte, email, displayName string) *TempWebAuthnUser {
	return &TempWebAuthnUser{
		id:          id,
		email:       email,
		displayName: displayName,
	}
}

// GenerateTempUserID generates a random 8-byte user ID for temporary users
func GenerateTempUserID() ([]byte, error) {
	id := make([]byte, 8)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	return id, nil
}

// WebAuthnID returns the temporary user's ID as bytes for WebAuthn
func (t *TempWebAuthnUser) WebAuthnID() []byte {
	return t.id
}

// WebAuthnName returns the user's name (email for identification)
func (t *TempWebAuthnUser) WebAuthnName() string {
	return t.email
}

// WebAuthnDisplayName returns the user's display name
func (t *TempWebAuthnUser) WebAuthnDisplayName() string {
	return t.displayName
}

// WebAuthnCredentials returns empty credentials for new user registration
func (t *TempWebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return []webauthn.Credential{}
}

// WebAuthnIcon is deprecated but required by the interface
func (t *TempWebAuthnUser) WebAuthnIcon() string {
	return ""
}

// ID returns the temporary user ID bytes
func (t *TempWebAuthnUser) ID() []byte {
	return t.id
}

// Email returns the email
func (t *TempWebAuthnUser) Email() string {
	return t.email
}

// DisplayName returns the display name
func (t *TempWebAuthnUser) DisplayName() string {
	return t.displayName
}
