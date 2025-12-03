package repository

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/errors"
)

type SessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) user.SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(session *user.Session) error {
	if err := r.db.Create(session).Error; err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

func (r *SessionRepository) GetByID(sessionID string) (*user.Session, error) {
	var session user.Session
	err := r.db.Where("id = ?", sessionID).First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("session not found")
		}
		return nil, fmt.Errorf("failed to get session by ID: %w", err)
	}
	return &session, nil
}

func (r *SessionRepository) GetByUserID(userID uint) ([]*user.Session, error) {
	var sessions []*user.Session
	err := r.db.Where("user_id = ? AND expires_at > ?", userID, time.Now()).
		Order("last_activity_at DESC").
		Find(&sessions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by user ID: %w", err)
	}
	return sessions, nil
}

func (r *SessionRepository) GetByTokenHash(tokenHash string) (*user.Session, error) {
	var session user.Session
	err := r.db.Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now()).
		First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("session not found")
		}
		return nil, fmt.Errorf("failed to get session by token hash: %w", err)
	}
	return &session, nil
}

func (r *SessionRepository) GetByRefreshTokenHash(refreshTokenHash string) (*user.Session, error) {
	var session user.Session
	err := r.db.Where("refresh_token_hash = ? AND expires_at > ?", refreshTokenHash, time.Now()).
		First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("session not found")
		}
		return nil, fmt.Errorf("failed to get session by refresh token hash: %w", err)
	}
	return &session, nil
}

func (r *SessionRepository) Update(session *user.Session) error {
	result := r.db.Save(session)
	if result.Error != nil {
		return fmt.Errorf("failed to update session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("session not found")
	}
	return nil
}

func (r *SessionRepository) Delete(sessionID string) error {
	result := r.db.Where("id = ?", sessionID).Delete(&user.Session{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("session not found")
	}
	return nil
}

func (r *SessionRepository) DeleteByUserID(userID uint) error {
	if err := r.db.Where("user_id = ?", userID).Delete(&user.Session{}).Error; err != nil {
		return fmt.Errorf("failed to delete sessions by user ID: %w", err)
	}
	return nil
}

func (r *SessionRepository) DeleteExpired() error {
	if err := r.db.Where("expires_at <= ?", time.Now()).Delete(&user.Session{}).Error; err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	return nil
}
