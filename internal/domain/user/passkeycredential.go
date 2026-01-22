package user

import (
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/orris-inc/orris/internal/shared/biztime"
)

// PasskeyCredential represents a WebAuthn/Passkey credential for a user
type PasskeyCredential struct {
	id              uint
	sid             string // external API identifier (pk_xxx)
	userID          uint
	credentialID    []byte
	publicKey       []byte
	attestationType string
	aaguid          []byte
	signCount       uint32
	backupEligible  bool // WebAuthn BE flag: credential can be backed up
	backupState     bool // WebAuthn BS flag: credential is currently backed up
	transports      []string
	deviceName      string
	lastUsedAt      *time.Time
	createdAt       time.Time
	updatedAt       time.Time
}

// NewPasskeyCredential creates a new passkey credential
func NewPasskeyCredential(
	userID uint,
	credentialID []byte,
	publicKey []byte,
	attestationType string,
	aaguid []byte,
	signCount uint32,
	backupEligible bool,
	backupState bool,
	transports []string,
	deviceName string,
	sidGenerator func() (string, error),
) (*PasskeyCredential, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}
	if len(credentialID) == 0 {
		return nil, fmt.Errorf("credential ID is required")
	}
	if len(publicKey) == 0 {
		return nil, fmt.Errorf("public key is required")
	}

	sid, err := sidGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SID: %w", err)
	}

	now := biztime.NowUTC()
	return &PasskeyCredential{
		sid:             sid,
		userID:          userID,
		credentialID:    credentialID,
		publicKey:       publicKey,
		attestationType: attestationType,
		aaguid:          aaguid,
		signCount:       signCount,
		backupEligible:  backupEligible,
		backupState:     backupState,
		transports:      transports,
		deviceName:      deviceName,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// ReconstructPasskeyCredential reconstructs a passkey credential from persistence
func ReconstructPasskeyCredential(
	id uint,
	sid string,
	userID uint,
	credentialID []byte,
	publicKey []byte,
	attestationType string,
	aaguid []byte,
	signCount uint32,
	backupEligible bool,
	backupState bool,
	transports []string,
	deviceName string,
	lastUsedAt *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) (*PasskeyCredential, error) {
	if id == 0 {
		return nil, fmt.Errorf("passkey credential ID cannot be zero")
	}
	if sid == "" {
		return nil, fmt.Errorf("passkey credential SID is required")
	}

	return &PasskeyCredential{
		id:              id,
		sid:             sid,
		userID:          userID,
		credentialID:    credentialID,
		publicKey:       publicKey,
		attestationType: attestationType,
		aaguid:          aaguid,
		signCount:       signCount,
		backupEligible:  backupEligible,
		backupState:     backupState,
		transports:      transports,
		deviceName:      deviceName,
		lastUsedAt:      lastUsedAt,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}, nil
}

// Getters

// ID returns the internal ID
func (p *PasskeyCredential) ID() uint {
	return p.id
}

// SID returns the external SID (pk_xxx)
func (p *PasskeyCredential) SID() string {
	return p.sid
}

// UserID returns the user ID
func (p *PasskeyCredential) UserID() uint {
	return p.userID
}

// CredentialID returns the WebAuthn credential ID
func (p *PasskeyCredential) CredentialID() []byte {
	return p.credentialID
}

// PublicKey returns the COSE public key
func (p *PasskeyCredential) PublicKey() []byte {
	return p.publicKey
}

// AttestationType returns the attestation type
func (p *PasskeyCredential) AttestationType() string {
	return p.attestationType
}

// AAGUID returns the authenticator attestation GUID
func (p *PasskeyCredential) AAGUID() []byte {
	return p.aaguid
}

// SignCount returns the signature counter
func (p *PasskeyCredential) SignCount() uint32 {
	return p.signCount
}

// BackupEligible returns the WebAuthn BE flag
func (p *PasskeyCredential) BackupEligible() bool {
	return p.backupEligible
}

// BackupState returns the WebAuthn BS flag
func (p *PasskeyCredential) BackupState() bool {
	return p.backupState
}

// Transports returns the transport hints
func (p *PasskeyCredential) Transports() []string {
	return p.transports
}

// DeviceName returns the device name
func (p *PasskeyCredential) DeviceName() string {
	return p.deviceName
}

// LastUsedAt returns when the credential was last used
func (p *PasskeyCredential) LastUsedAt() *time.Time {
	return p.lastUsedAt
}

// CreatedAt returns when the credential was created
func (p *PasskeyCredential) CreatedAt() time.Time {
	return p.createdAt
}

// UpdatedAt returns when the credential was last updated
func (p *PasskeyCredential) UpdatedAt() time.Time {
	return p.updatedAt
}

// SetID sets the internal ID (only for persistence layer use)
func (p *PasskeyCredential) SetID(id uint) error {
	if p.id != 0 {
		return fmt.Errorf("passkey credential ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("passkey credential ID cannot be zero")
	}
	p.id = id
	return nil
}

// UpdateSignCount updates the sign count after successful authentication.
// Returns an error if the new count is not greater than the current count,
// which may indicate a cloned credential attack.
//
// WebAuthn sign count behavior:
// - If both old and new are 0: authenticator doesn't support sign count (valid)
// - If old is 0 and new > 0: authenticator started reporting sign count (valid)
// - If old > 0 and new > old: normal increment (valid)
// - If old > 0 and new <= old: possible credential cloning (invalid)
func (p *PasskeyCredential) UpdateSignCount(newCount uint32) error {
	// If current sign count is non-zero, new count must be greater
	// This detects potential credential cloning attacks
	if p.signCount > 0 && newCount <= p.signCount {
		return fmt.Errorf("sign count not increasing: got %d, expected > %d (possible cloned credential)", newCount, p.signCount)
	}
	p.signCount = newCount
	p.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateLastUsed updates the last used timestamp
func (p *PasskeyCredential) UpdateLastUsed() {
	now := biztime.NowUTC()
	p.lastUsedAt = &now
	p.updatedAt = now
}

// UpdateDeviceName updates the device name
func (p *PasskeyCredential) UpdateDeviceName(name string) {
	p.deviceName = name
	p.updatedAt = biztime.NowUTC()
}

// ToWebAuthnCredential converts to webauthn.Credential for authentication
func (p *PasskeyCredential) ToWebAuthnCredential() webauthn.Credential {
	transports := make([]protocol.AuthenticatorTransport, len(p.transports))
	for i, t := range p.transports {
		transports[i] = protocol.AuthenticatorTransport(t)
	}

	var aaguid [16]byte
	if len(p.aaguid) == 16 {
		copy(aaguid[:], p.aaguid)
	}

	return webauthn.Credential{
		ID:              p.credentialID,
		PublicKey:       p.publicKey,
		AttestationType: p.attestationType,
		Flags: webauthn.CredentialFlags{
			BackupEligible: p.backupEligible,
			BackupState:    p.backupState,
		},
		Authenticator: webauthn.Authenticator{
			AAGUID:    aaguid[:],
			SignCount: p.signCount,
		},
		Transport: transports,
	}
}

// NewPasskeyCredentialFromWebAuthn creates a PasskeyCredential from webauthn.Credential
func NewPasskeyCredentialFromWebAuthn(
	userID uint,
	cred *webauthn.Credential,
	deviceName string,
	sidGenerator func() (string, error),
) (*PasskeyCredential, error) {
	transports := make([]string, len(cred.Transport))
	for i, t := range cred.Transport {
		transports[i] = string(t)
	}

	return NewPasskeyCredential(
		userID,
		cred.ID,
		cred.PublicKey,
		cred.AttestationType,
		cred.Authenticator.AAGUID,
		cred.Authenticator.SignCount,
		cred.Flags.BackupEligible,
		cred.Flags.BackupState,
		transports,
		deviceName,
		sidGenerator,
	)
}

// PasskeyCredentialDisplayInfo represents credential information for display
type PasskeyCredentialDisplayInfo struct {
	ID         string     `json:"id"`
	DeviceName string     `json:"device_name"`
	Transports []string   `json:"transports"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// GetDisplayInfo returns credential information for display
func (p *PasskeyCredential) GetDisplayInfo() PasskeyCredentialDisplayInfo {
	return PasskeyCredentialDisplayInfo{
		ID:         p.sid,
		DeviceName: p.deviceName,
		Transports: p.transports,
		LastUsedAt: p.lastUsedAt,
		CreatedAt:  p.createdAt,
	}
}
