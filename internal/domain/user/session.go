package user

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
)

type Session struct {
	ID               string
	UserID           uint
	DeviceName       string
	DeviceType       string
	IPAddress        string
	UserAgent        string
	TokenHash        string
	RefreshTokenHash string
	ExpiresAt        time.Time
	LastActivityAt   time.Time
	CreatedAt        time.Time
}

func NewSession(userID uint, deviceName, deviceType, ipAddress, userAgent string, expiresAt time.Time) (*Session, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}

	id, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	now := biztime.NowUTC()
	return &Session{
		ID:             id,
		UserID:         userID,
		DeviceName:     deviceName,
		DeviceType:     deviceType,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
		ExpiresAt:      expiresAt,
		LastActivityAt: now,
		CreatedAt:      now,
	}, nil
}

func (s *Session) IsExpired() bool {
	return biztime.NowUTC().After(s.ExpiresAt)
}

func (s *Session) UpdateActivity() {
	s.LastActivityAt = biztime.NowUTC()
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type SessionRepository interface {
	Create(session *Session) error
	GetByID(sessionID string) (*Session, error)
	GetByUserID(userID uint) ([]*Session, error)
	GetByTokenHash(tokenHash string) (*Session, error)
	GetByRefreshTokenHash(refreshTokenHash string) (*Session, error)
	Update(session *Session) error
	Delete(sessionID string) error
	DeleteByUserID(userID uint) error
	DeleteExpired() error
}
