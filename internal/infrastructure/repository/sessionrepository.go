package repository

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/errors"
)

type SessionRepository struct {
	db     *gorm.DB
	mapper mappers.SessionMapper
}

func NewSessionRepository(db *gorm.DB) user.SessionRepository {
	return &SessionRepository{
		db:     db,
		mapper: mappers.NewSessionMapper(),
	}
}

func (r *SessionRepository) Create(session *user.Session) error {
	model := r.mapper.ToModel(session)
	if err := r.db.Create(model).Error; err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

func (r *SessionRepository) GetByID(sessionID string) (*user.Session, error) {
	var model models.SessionModel
	err := r.db.Where("id = ?", sessionID).First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("session not found")
		}
		return nil, fmt.Errorf("failed to get session by ID: %w", err)
	}
	return r.mapper.ToDomain(&model), nil
}

func (r *SessionRepository) GetByUserID(userID uint) ([]*user.Session, error) {
	var sessionModels []models.SessionModel
	err := r.db.Where("user_id = ? AND expires_at > ?", userID, biztime.NowUTC()).
		Order("last_activity_at DESC").
		Find(&sessionModels).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by user ID: %w", err)
	}

	sessions := make([]*user.Session, len(sessionModels))
	for i := range sessionModels {
		sessions[i] = r.mapper.ToDomain(&sessionModels[i])
	}
	return sessions, nil
}

func (r *SessionRepository) GetByTokenHash(tokenHash string) (*user.Session, error) {
	var model models.SessionModel
	err := r.db.Where("token_hash = ? AND expires_at > ?", tokenHash, biztime.NowUTC()).
		First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("session not found")
		}
		return nil, fmt.Errorf("failed to get session by token hash: %w", err)
	}
	return r.mapper.ToDomain(&model), nil
}

func (r *SessionRepository) GetByRefreshTokenHash(refreshTokenHash string) (*user.Session, error) {
	var model models.SessionModel
	err := r.db.Where("refresh_token_hash = ? AND expires_at > ?", refreshTokenHash, biztime.NowUTC()).
		First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("session not found")
		}
		return nil, fmt.Errorf("failed to get session by refresh token hash: %w", err)
	}
	return r.mapper.ToDomain(&model), nil
}

func (r *SessionRepository) Update(session *user.Session) error {
	model := r.mapper.ToModel(session)
	result := r.db.Save(model)
	if result.Error != nil {
		return fmt.Errorf("failed to update session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("session not found")
	}
	return nil
}

func (r *SessionRepository) Delete(sessionID string) error {
	result := r.db.Where("id = ?", sessionID).Delete(&models.SessionModel{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("session not found")
	}
	return nil
}

func (r *SessionRepository) DeleteByUserID(userID uint) error {
	if err := r.db.Where("user_id = ?", userID).Delete(&models.SessionModel{}).Error; err != nil {
		return fmt.Errorf("failed to delete sessions by user ID: %w", err)
	}
	return nil
}

func (r *SessionRepository) DeleteExpired() error {
	if err := r.db.Where("expires_at <= ?", biztime.NowUTC()).Delete(&models.SessionModel{}).Error; err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	return nil
}
