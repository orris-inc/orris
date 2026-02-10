package mappers

import (
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// SessionMapper handles the conversion between Session domain entities and persistence models.
type SessionMapper interface {
	// ToModel converts a domain entity to a persistence model.
	ToModel(entity *user.Session) *models.SessionModel

	// ToDomain converts a persistence model to a domain entity.
	ToDomain(model *models.SessionModel) *user.Session
}

// SessionMapperImpl is the concrete implementation of SessionMapper.
type SessionMapperImpl struct{}

// NewSessionMapper creates a new SessionMapper.
func NewSessionMapper() SessionMapper {
	return &SessionMapperImpl{}
}

// ToModel converts a domain entity to a persistence model.
func (m *SessionMapperImpl) ToModel(entity *user.Session) *models.SessionModel {
	if entity == nil {
		return nil
	}
	return &models.SessionModel{
		ID:               entity.ID,
		UserID:           entity.UserID,
		DeviceName:       entity.DeviceName,
		DeviceType:       entity.DeviceType,
		IPAddress:        entity.IPAddress,
		UserAgent:        entity.UserAgent,
		TokenHash:        entity.TokenHash,
		RefreshTokenHash: entity.RefreshTokenHash,
		RememberMe:       entity.RememberMe,
		ExpiresAt:        entity.ExpiresAt,
		LastActivityAt:   entity.LastActivityAt,
		CreatedAt:        entity.CreatedAt,
	}
}

// ToDomain converts a persistence model to a domain entity.
func (m *SessionMapperImpl) ToDomain(model *models.SessionModel) *user.Session {
	if model == nil {
		return nil
	}
	return &user.Session{
		ID:               model.ID,
		UserID:           model.UserID,
		DeviceName:       model.DeviceName,
		DeviceType:       model.DeviceType,
		IPAddress:        model.IPAddress,
		UserAgent:        model.UserAgent,
		TokenHash:        model.TokenHash,
		RefreshTokenHash: model.RefreshTokenHash,
		RememberMe:       model.RememberMe,
		ExpiresAt:        model.ExpiresAt,
		LastActivityAt:   model.LastActivityAt,
		CreatedAt:        model.CreatedAt,
	}
}
