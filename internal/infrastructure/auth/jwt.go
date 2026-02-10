package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/orris-inc/orris/internal/shared/authorization"
	"github.com/orris-inc/orris/internal/shared/biztime"
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

func NewJWTService(secret string, accessExpMinutes, refreshExpDays int) *JWTService {
	return &JWTService{
		secret:           []byte(secret),
		accessExpMinutes: accessExpMinutes,
		refreshExpDays:   refreshExpDays,
	}
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

// RefreshAccessToken generates a new access token using the same claims
func (s *JWTService) RefreshAccessToken(claims *Claims) (string, error) {
	now := biztime.NowUTC()
	accessExp := now.Add(time.Duration(s.accessExpMinutes) * time.Minute)

	newClaims := &Claims{
		UserUUID:  claims.UserUUID,
		SessionID: claims.SessionID,
		Role:      claims.Role,
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
// The old refresh token is effectively invalidated because the session's refresh token hash
// will be updated to match the new refresh token (refresh token rotation).
func (s *JWTService) Refresh(refreshTokenString string) (*TokenPair, error) {
	claims, err := s.Verify(refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.TokenType != TokenTypeRefresh {
		return nil, fmt.Errorf("token is not a refresh token")
	}

	now := biztime.NowUTC()

	// Generate new access token
	accessExp := now.Add(time.Duration(s.accessExpMinutes) * time.Minute)
	accessClaims := &Claims{
		UserUUID:  claims.UserUUID,
		SessionID: claims.SessionID,
		Role:      claims.Role,
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

	// Generate new refresh token (rotation)
	refreshExp := now.Add(time.Duration(s.refreshExpDays) * 24 * time.Hour)
	refreshClaims := &Claims{
		UserUUID:  claims.UserUUID,
		SessionID: claims.SessionID,
		Role:      claims.Role,
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
