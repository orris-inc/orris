package mappers

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/user"
	vo "github.com/orris-inc/orris/internal/domain/user/value_objects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/authorization"
)

// UserMapper handles the conversion between domain entities and persistence models
type UserMapper interface {
	// ToEntity converts a persistence model to a domain entity
	ToEntity(model *models.UserModel) (*user.User, error)

	// ToModel converts a domain entity to a persistence model
	ToModel(entity *user.User) (*models.UserModel, error)

	// ToEntities converts multiple persistence models to domain entities
	ToEntities(models []*models.UserModel) ([]*user.User, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*user.User) ([]*models.UserModel, error)
}

// UserMapperImpl is the concrete implementation of UserMapper
type UserMapperImpl struct{}

// NewUserMapper creates a new user mapper
func NewUserMapper() UserMapper {
	return &UserMapperImpl{}
}

// ToEntity converts a persistence model to a domain entity
func (m *UserMapperImpl) ToEntity(model *models.UserModel) (*user.User, error) {
	if model == nil {
		return nil, nil
	}

	email, err := vo.NewEmail(model.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to create email value object: %w", err)
	}

	name, err := vo.NewName(model.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create name value object: %w", err)
	}

	status, err := vo.NewStatus(model.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to create status value object: %w", err)
	}

	role := authorization.ParseUserRole(model.Role)

	authData := &user.UserAuthData{
		PasswordHash:               model.PasswordHash,
		EmailVerified:              model.EmailVerified,
		EmailVerificationToken:     model.EmailVerificationToken,
		EmailVerificationExpiresAt: model.EmailVerificationExpiresAt,
		PasswordResetToken:         model.PasswordResetToken,
		PasswordResetExpiresAt:     model.PasswordResetExpiresAt,
		LastPasswordChangeAt:       model.LastPasswordChangeAt,
		FailedLoginAttempts:        model.FailedLoginAttempts,
		LockedUntil:                model.LockedUntil,
	}

	userEntity, err := user.ReconstructUserWithAuth(
		model.ID,
		email,
		name,
		role,
		*status,
		model.CreatedAt,
		model.UpdatedAt,
		model.Version,
		authData,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct user entity: %w", err)
	}

	return userEntity, nil
}

// ToModel converts a domain entity to a persistence model
func (m *UserMapperImpl) ToModel(entity *user.User) (*models.UserModel, error) {
	if entity == nil {
		return nil, nil
	}

	authData := entity.GetAuthData()

	model := &models.UserModel{
		ID:                         entity.ID(),
		Email:                      entity.Email().String(),
		Name:                       entity.Name().String(),
		Role:                       entity.Role().String(),
		Status:                     entity.Status().String(),
		Version:                    entity.Version(),
		CreatedAt:                  entity.CreatedAt(),
		UpdatedAt:                  entity.UpdatedAt(),
		PasswordHash:               authData.PasswordHash,
		EmailVerified:              authData.EmailVerified,
		EmailVerificationToken:     authData.EmailVerificationToken,
		EmailVerificationExpiresAt: authData.EmailVerificationExpiresAt,
		PasswordResetToken:         authData.PasswordResetToken,
		PasswordResetExpiresAt:     authData.PasswordResetExpiresAt,
		LastPasswordChangeAt:       authData.LastPasswordChangeAt,
		FailedLoginAttempts:        authData.FailedLoginAttempts,
		LockedUntil:                authData.LockedUntil,
	}

	if entity.Status().IsDeleted() {
		now := entity.UpdatedAt()
		model.DeletedAt = gorm.DeletedAt{
			Time:  now,
			Valid: true,
		}
	}

	return model, nil
}

// ToEntities converts multiple persistence models to domain entities
func (m *UserMapperImpl) ToEntities(models []*models.UserModel) ([]*user.User, error) {
	entities := make([]*user.User, 0, len(models))

	for _, model := range models {
		entity, err := m.ToEntity(model)
		if err != nil {
			return nil, fmt.Errorf("failed to map model ID %d: %w", model.ID, err)
		}
		if entity != nil {
			entities = append(entities, entity)
		}
	}

	return entities, nil
}

// ToModels converts multiple domain entities to persistence models
func (m *UserMapperImpl) ToModels(entities []*user.User) ([]*models.UserModel, error) {
	models := make([]*models.UserModel, 0, len(entities))

	for _, entity := range entities {
		model, err := m.ToModel(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to map entity ID %d: %w", entity.ID(), err)
		}
		if model != nil {
			models = append(models, model)
		}
	}

	return models, nil
}
