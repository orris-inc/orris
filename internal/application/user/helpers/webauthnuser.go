package helpers

import (
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/orris-inc/orris/internal/domain/user"
)

// WebAuthnUser adapts the domain User to the webauthn.User interface
type WebAuthnUser struct {
	user        *user.User
	credentials []*user.PasskeyCredential
}

// NewWebAuthnUser creates a new WebAuthn user adapter
func NewWebAuthnUser(u *user.User, credentials []*user.PasskeyCredential) *WebAuthnUser {
	return &WebAuthnUser{
		user:        u,
		credentials: credentials,
	}
}

// WebAuthnID returns the user's ID as bytes for WebAuthn
// We use the internal user ID as bytes
func (w *WebAuthnUser) WebAuthnID() []byte {
	// Convert uint to bytes (big-endian)
	id := w.user.ID()
	return []byte{
		byte(id >> 56),
		byte(id >> 48),
		byte(id >> 40),
		byte(id >> 32),
		byte(id >> 24),
		byte(id >> 16),
		byte(id >> 8),
		byte(id),
	}
}

// WebAuthnName returns the user's name (email for identification)
func (w *WebAuthnUser) WebAuthnName() string {
	return w.user.Email().String()
}

// WebAuthnDisplayName returns the user's display name
func (w *WebAuthnUser) WebAuthnDisplayName() string {
	return w.user.Name().DisplayName()
}

// WebAuthnCredentials returns the user's WebAuthn credentials
func (w *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	creds := make([]webauthn.Credential, len(w.credentials))
	for i, c := range w.credentials {
		creds[i] = c.ToWebAuthnCredential()
	}
	return creds
}

// WebAuthnIcon is deprecated but required by the interface
func (w *WebAuthnUser) WebAuthnIcon() string {
	return ""
}

// User returns the underlying domain user
func (w *WebAuthnUser) User() *user.User {
	return w.user
}

// Credentials returns the underlying passkey credentials
func (w *WebAuthnUser) Credentials() []*user.PasskeyCredential {
	return w.credentials
}

// ParseUserIDFromBytes converts WebAuthn user ID bytes back to uint
func ParseUserIDFromBytes(b []byte) uint {
	if len(b) != 8 {
		return 0
	}
	return uint(b[0])<<56 | uint(b[1])<<48 | uint(b[2])<<40 | uint(b[3])<<32 |
		uint(b[4])<<24 | uint(b[5])<<16 | uint(b[6])<<8 | uint(b[7])
}
