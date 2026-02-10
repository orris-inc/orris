package mappers

import (
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/mapper"
)

// OAuthAccountMapper handles the conversion between domain entities and persistence models.
type OAuthAccountMapper interface {
	// ToModel converts a domain entity to a persistence model.
	ToModel(entity *user.OAuthAccount) *models.OAuthAccountModel

	// ToDomain converts a persistence model to a domain entity.
	ToDomain(model *models.OAuthAccountModel) *user.OAuthAccount

	// ToDomainList converts multiple persistence models to domain entities.
	ToDomainList(models []*models.OAuthAccountModel) []*user.OAuthAccount
}

// OAuthAccountMapperImpl is the concrete implementation of OAuthAccountMapper.
type OAuthAccountMapperImpl struct{}

// NewOAuthAccountMapper creates a new OAuthAccountMapper.
func NewOAuthAccountMapper() OAuthAccountMapper {
	return &OAuthAccountMapperImpl{}
}

// ToModel converts a domain entity to a persistence model.
func (m *OAuthAccountMapperImpl) ToModel(entity *user.OAuthAccount) *models.OAuthAccountModel {
	if entity == nil {
		return nil
	}
	return &models.OAuthAccountModel{
		ID:                entity.ID,
		UserID:            entity.UserID,
		Provider:          entity.Provider,
		ProviderUserID:    entity.ProviderUserID,
		ProviderEmail:     entity.ProviderEmail,
		ProviderUsername:   entity.ProviderUsername,
		ProviderAvatarURL: entity.ProviderAvatarURL,
		RawUserInfo:       entity.RawUserInfo,
		LastLoginAt:       entity.LastLoginAt,
		LoginCount:        entity.LoginCount,
		CreatedAt:         entity.CreatedAt,
		UpdatedAt:         entity.UpdatedAt,
	}
}

// ToDomain converts a persistence model to a domain entity.
func (m *OAuthAccountMapperImpl) ToDomain(model *models.OAuthAccountModel) *user.OAuthAccount {
	if model == nil {
		return nil
	}
	return &user.OAuthAccount{
		ID:                model.ID,
		UserID:            model.UserID,
		Provider:          model.Provider,
		ProviderUserID:    model.ProviderUserID,
		ProviderEmail:     model.ProviderEmail,
		ProviderUsername:   model.ProviderUsername,
		ProviderAvatarURL: model.ProviderAvatarURL,
		RawUserInfo:       model.RawUserInfo,
		LastLoginAt:       model.LastLoginAt,
		LoginCount:        model.LoginCount,
		CreatedAt:         model.CreatedAt,
		UpdatedAt:         model.UpdatedAt,
	}
}

// ToDomainList converts multiple persistence models to domain entities.
func (m *OAuthAccountMapperImpl) ToDomainList(items []*models.OAuthAccountModel) []*user.OAuthAccount {
	return mapper.MapSlicePtr(items, m.ToDomain)
}
