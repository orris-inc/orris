package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/redis/go-redis/v9"
)

const (
	// PasskeyChallengePrefix is the Redis key prefix for passkey challenges
	PasskeyChallengePrefix = "passkey:challenge:"
	// PasskeyChallengeTTL is the default TTL for passkey challenges (3 minutes)
	PasskeyChallengeTTL = 3 * time.Minute
)

// PasskeyChallengeStore stores WebAuthn session data for registration/login ceremonies
type PasskeyChallengeStore struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// NewPasskeyChallengeStore creates a new passkey challenge store
func NewPasskeyChallengeStore(client *redis.Client) *PasskeyChallengeStore {
	return &PasskeyChallengeStore{
		client: client,
		prefix: PasskeyChallengePrefix,
		ttl:    PasskeyChallengeTTL,
	}
}

// NewPasskeyChallengeStoreWithConfig creates a new passkey challenge store with custom config
func NewPasskeyChallengeStoreWithConfig(client *redis.Client, prefix string, ttl time.Duration) *PasskeyChallengeStore {
	if prefix == "" {
		prefix = PasskeyChallengePrefix
	}
	if ttl == 0 {
		ttl = PasskeyChallengeTTL
	}
	return &PasskeyChallengeStore{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}
}

// credentialParameterWrapper wraps protocol.CredentialParameter for JSON serialization
type credentialParameterWrapper struct {
	Type      string `json:"type"`
	Algorithm int64  `json:"alg"`
}

// sessionDataWrapper wraps SessionData for JSON serialization
type sessionDataWrapper struct {
	Challenge            string                       `json:"challenge"`
	RelyingPartyID       string                       `json:"rp_id"`
	UserID               []byte                       `json:"user_id"`
	AllowedCredentialIDs [][]byte                     `json:"allowed_credential_ids,omitempty"`
	UserVerification     string                       `json:"user_verification"`
	Expires              int64                        `json:"expires"` // Unix timestamp in milliseconds
	CredParams           []credentialParameterWrapper `json:"cred_params,omitempty"`
	Mediation            string                       `json:"mediation,omitempty"`
}

// Store saves WebAuthn session data with the challenge as key
func (s *PasskeyChallengeStore) Store(ctx context.Context, sessionData *webauthn.SessionData) error {
	if sessionData == nil {
		return errors.New("session data cannot be nil")
	}

	challenge := sessionData.Challenge
	if challenge == "" {
		return errors.New("challenge cannot be empty")
	}

	// Convert credential parameters for JSON serialization
	var credParams []credentialParameterWrapper
	for _, cp := range sessionData.CredParams {
		credParams = append(credParams, credentialParameterWrapper{
			Type:      string(cp.Type),
			Algorithm: int64(cp.Algorithm),
		})
	}

	wrapper := sessionDataWrapper{
		Challenge:            challenge,
		RelyingPartyID:       sessionData.RelyingPartyID,
		UserID:               sessionData.UserID,
		AllowedCredentialIDs: sessionData.AllowedCredentialIDs,
		UserVerification:     string(sessionData.UserVerification),
		Expires:              sessionData.Expires.UnixMilli(),
		CredParams:           credParams,
		Mediation:            string(sessionData.Mediation),
	}

	data, err := json.Marshal(wrapper)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	key := s.buildKey(challenge)
	if err := s.client.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to store session data in Redis: %w", err)
	}

	return nil
}

// Get retrieves WebAuthn session data by challenge (one-time use)
func (s *PasskeyChallengeStore) Get(ctx context.Context, challenge string) (*webauthn.SessionData, error) {
	if challenge == "" {
		return nil, errors.New("challenge cannot be empty")
	}

	key := s.buildKey(challenge)

	// Use GETDEL for one-time use semantics
	data, err := s.client.GetDel(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errors.New("challenge not found or expired")
		}
		return nil, fmt.Errorf("failed to retrieve session data from Redis: %w", err)
	}

	var wrapper sessionDataWrapper
	if err := json.Unmarshal([]byte(data), &wrapper); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	// Restore credential parameters from JSON
	var credParams []protocol.CredentialParameter
	for _, cp := range wrapper.CredParams {
		credParams = append(credParams, protocol.CredentialParameter{
			Type:      protocol.CredentialType(cp.Type),
			Algorithm: webauthncose.COSEAlgorithmIdentifier(cp.Algorithm),
		})
	}

	sessionData := &webauthn.SessionData{
		Challenge:            wrapper.Challenge,
		RelyingPartyID:       wrapper.RelyingPartyID,
		UserID:               wrapper.UserID,
		AllowedCredentialIDs: wrapper.AllowedCredentialIDs,
		UserVerification:     protocol.UserVerificationRequirement(wrapper.UserVerification),
		Expires:              time.UnixMilli(wrapper.Expires),
		CredParams:           credParams,
		Mediation:            protocol.CredentialMediationRequirement(wrapper.Mediation),
	}

	return sessionData, nil
}

// Delete removes a challenge from the store
func (s *PasskeyChallengeStore) Delete(ctx context.Context, challenge string) error {
	if challenge == "" {
		return errors.New("challenge cannot be empty")
	}

	key := s.buildKey(challenge)
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete challenge from Redis: %w", err)
	}

	return nil
}

// buildKey constructs the full Redis key
func (s *PasskeyChallengeStore) buildKey(challenge string) string {
	return s.prefix + challenge
}
