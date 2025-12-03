package auth

import (
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/shared/authorization"

	"github.com/golang-jwt/jwt/v5"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type Claims struct {
	UserID    uint                   `json:"user_id"`
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

func (s *JWTService) Generate(userID uint, sessionID string, role authorization.UserRole) (*TokenPair, error) {
	now := time.Now()

	accessExp := now.Add(time.Duration(s.accessExpMinutes) * time.Minute)
	accessClaims := &Claims{
		UserID:    userID,
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
		UserID:    userID,
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

func (s *JWTService) Refresh(refreshTokenString string) (string, error) {
	claims, err := s.Verify(refreshTokenString)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.TokenType != TokenTypeRefresh {
		return "", fmt.Errorf("token is not a refresh token")
	}

	now := time.Now()
	accessExp := now.Add(time.Duration(s.accessExpMinutes) * time.Minute)

	newClaims := &Claims{
		UserID:    claims.UserID,
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
	accessTokenString, err := accessToken.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign new access token: %w", err)
	}

	return accessTokenString, nil
}
