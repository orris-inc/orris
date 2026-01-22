package auth

import (
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/orris-inc/orris/internal/shared/config"
)

// WebAuthnService provides WebAuthn authentication functionality
type WebAuthnService struct {
	webAuthn *webauthn.WebAuthn
}

// NewWebAuthnService creates a new WebAuthn service
func NewWebAuthnService(cfg config.WebAuthnConfig) (*WebAuthnService, error) {
	if !cfg.IsConfigured() {
		return nil, fmt.Errorf("WebAuthn is not configured: rp_id, rp_name, and rp_origins are required")
	}

	timeout := time.Duration(cfg.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 60 * time.Second // default 60 seconds
	}

	wconfig := &webauthn.Config{
		RPDisplayName: cfg.RPName,
		RPID:          cfg.RPID,
		RPOrigins:     cfg.RPOrigins,
		// Accept "none" attestation format (common for Touch ID, Face ID, Windows Hello)
		// This is the recommended setting for consumer applications
		AttestationPreference: protocol.PreferNoAttestation,
		Timeouts: webauthn.TimeoutsConfig{
			Login: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    timeout,
				TimeoutUVD: timeout,
			},
			Registration: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    timeout,
				TimeoutUVD: timeout,
			},
		},
	}

	w, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebAuthn instance: %w", err)
	}

	return &WebAuthnService{
		webAuthn: w,
	}, nil
}

// WebAuthn returns the underlying webauthn instance
func (s *WebAuthnService) WebAuthn() *webauthn.WebAuthn {
	return s.webAuthn
}

// BeginRegistration starts the registration ceremony for a user
func (s *WebAuthnService) BeginRegistration(user webauthn.User, opts ...webauthn.RegistrationOption) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	return s.webAuthn.BeginRegistration(user, opts...)
}

// FinishRegistration completes the registration ceremony
func (s *WebAuthnService) FinishRegistration(user webauthn.User, sessionData webauthn.SessionData, response *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error) {
	return s.webAuthn.CreateCredential(user, sessionData, response)
}

// BeginLogin starts the login ceremony for a user
func (s *WebAuthnService) BeginLogin(user webauthn.User, opts ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	return s.webAuthn.BeginLogin(user, opts...)
}

// BeginDiscoverableLogin starts a discoverable login ceremony (passwordless)
func (s *WebAuthnService) BeginDiscoverableLogin(opts ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	return s.webAuthn.BeginDiscoverableLogin(opts...)
}

// FinishLogin completes the login ceremony
func (s *WebAuthnService) FinishLogin(user webauthn.User, sessionData webauthn.SessionData, response *protocol.ParsedCredentialAssertionData) (*webauthn.Credential, error) {
	return s.webAuthn.ValidateLogin(user, sessionData, response)
}

// FinishDiscoverableLogin completes a discoverable login ceremony
func (s *WebAuthnService) FinishDiscoverableLogin(
	userHandler webauthn.DiscoverableUserHandler,
	sessionData webauthn.SessionData,
	response *protocol.ParsedCredentialAssertionData,
) (*webauthn.Credential, error) {
	return s.webAuthn.ValidateDiscoverableLogin(userHandler, sessionData, response)
}
