package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/orris-inc/orris/internal/shared/authorization"
	"github.com/orris-inc/orris/internal/shared/biztime"
)

const (
	// defaultInsecureSecret is the placeholder secret shipped in default config.
	// It MUST be replaced before running in production.
	defaultInsecureSecret = "change-me-in-production"

	// minSigningKeyLength is the minimum acceptable length for a signing key.
	minSigningKeyLength = 32
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type Claims struct {
	UserUUID  string                 `json:"user_uuid"`
	SessionID string                 `json:"session_id"`
	Role      authorization.UserRole `json:"role"`
	TokenType TokenType              `json:"token_type"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type JWTService struct {
	secret           []byte
	accessExpMinutes int
	refreshExpDays   int
}

// NewJWTService creates a new JWTService with the given signing parameters.
// serverMode should be the value of server.mode from config (e.g. "debug", "release").
// In non-debug modes, the signing key is validated for strength.
func NewJWTService(secret string, accessExpMinutes, refreshExpDays int, serverMode string) (*JWTService, error) {
	if err := ValidateSigningKey(secret, serverMode, "auth.jwt.secret"); err != nil {
		return nil, err
	}

	return &JWTService{
		secret:           []byte(secret),
		accessExpMinutes: accessExpMinutes,
		refreshExpDays:   refreshExpDays,
	}, nil
}

// ValidateSigningKey checks that a signing key is safe for the current server mode.
// In debug mode, weak keys are silently allowed to simplify local development.
// In any other mode (release, test, etc.), weak keys cause a fatal startup error.
func ValidateSigningKey(key, serverMode, configField string) error {
	isDebug := serverMode == "debug" || serverMode == ""

	if key == defaultInsecureSecret {
		if isDebug {
			return nil
		}
		return fmt.Errorf(
			"%s is set to the default insecure value %q; "+
				"set a strong secret (>= %d chars) via config or ORRIS_%s env var before running in production",
			configField, defaultInsecureSecret, minSigningKeyLength,
			configFieldToEnvKey(configField),
		)
	}

	if len(key) < minSigningKeyLength {
		if isDebug {
			return nil
		}
		return fmt.Errorf(
			"%s is too short (%d chars); minimum length is %d for non-debug modes",
			configField, len(key), minSigningKeyLength,
		)
	}

	return nil
}

// configFieldToEnvKey converts a dot-separated config field to the ORRIS env-var suffix.
// e.g. "auth.jwt.secret" -> "AUTH_JWT_SECRET"
func configFieldToEnvKey(field string) string {
	return strings.ToUpper(strings.ReplaceAll(field, ".", "_"))
}

func (s *JWTService) Generate(userUUID string, sessionID string, role authorization.UserRole) (*TokenPair, error) {
	now := biztime.NowUTC()

	accessExp := now.Add(time.Duration(s.accessExpMinutes) * time.Minute)
	accessClaims := &Claims{
		UserUUID:  userUUID,
		SessionID: sessionID,
		Role:      role,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshExp := now.Add(time.Duration(s.refreshExpDays) * 24 * time.Hour)
	refreshClaims := &Claims{
		UserUUID:  userUUID,
		SessionID: sessionID,
		Role:      role,
		TokenType: TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(s.accessExpMinutes * 60),
	}, nil
}

func (s *JWTService) Verify(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ShouldRefresh checks if the access token should be refreshed
// Returns true if the token will expire within the threshold (default: 5 minutes)
func (s *JWTService) ShouldRefresh(claims *Claims) bool {
	if claims == nil || claims.ExpiresAt == nil {
		return false
	}
	// Refresh if token expires within 5 minutes
	threshold := 5 * time.Minute
	return biztime.NowUTC().Add(threshold).After(claims.ExpiresAt.Time)
}

// RefreshAccessToken generates a new access token using the same claims but with a fresh role
// from the database. This ensures role changes (e.g. demotion) are reflected immediately.
func (s *JWTService) RefreshAccessToken(claims *Claims, freshRole authorization.UserRole) (string, error) {
	now := biztime.NowUTC()
	accessExp := now.Add(time.Duration(s.accessExpMinutes) * time.Minute)

	newClaims := &Claims{
		UserUUID:  claims.UserUUID,
		SessionID: claims.SessionID,
		Role:      freshRole,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	return accessToken.SignedString(s.secret)
}

// AccessExpMinutes returns the access token expiration time in minutes
func (s *JWTService) AccessExpMinutes() int {
	return s.accessExpMinutes
}

// Refresh generates a new access token AND a new refresh token from the given refresh token.
// The freshRole parameter ensures the new tokens use the user's current role from the database,
// not the stale role from the old token. The old refresh token is effectively invalidated
// because the session's refresh token hash will be updated to match the new refresh token
// (refresh token rotation).
func (s *JWTService) Refresh(refreshTokenString string, freshRole authorization.UserRole) (*TokenPair, error) {
	claims, err := s.Verify(refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.TokenType != TokenTypeRefresh {
		return nil, fmt.Errorf("token is not a refresh token")
	}

	now := biztime.NowUTC()

	// Generate new access token with the fresh role from database
	accessExp := now.Add(time.Duration(s.accessExpMinutes) * time.Minute)
	accessClaims := &Claims{
		UserUUID:  claims.UserUUID,
		SessionID: claims.SessionID,
		Role:      freshRole,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign new access token: %w", err)
	}

	// Generate new refresh token (rotation) with the fresh role from database
	refreshExp := now.Add(time.Duration(s.refreshExpDays) * 24 * time.Hour)
	refreshClaims := &Claims{
		UserUUID:  claims.UserUUID,
		SessionID: claims.SessionID,
		Role:      freshRole,
		TokenType: TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign new refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int64(s.accessExpMinutes * 60),
	}, nil
}
