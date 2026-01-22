package mappers

import (
	"encoding/json"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/mapper"
)

// PasskeyCredentialMapper handles the conversion between domain entities and persistence models
type PasskeyCredentialMapper interface {
	// ToEntity converts a persistence model to a domain entity
	ToEntity(model *models.PasskeyCredentialModel) (*user.PasskeyCredential, error)

	// ToModel converts a domain entity to a persistence model
	ToModel(entity *user.PasskeyCredential) (*models.PasskeyCredentialModel, error)

	// ToEntities converts multiple persistence models to domain entities
	ToEntities(models []*models.PasskeyCredentialModel) ([]*user.PasskeyCredential, error)

	// ToModels converts multiple domain entities to persistence models
	ToModels(entities []*user.PasskeyCredential) ([]*models.PasskeyCredentialModel, error)
}

// PasskeyCredentialMapperImpl is the concrete implementation of PasskeyCredentialMapper
type PasskeyCredentialMapperImpl struct{}

// NewPasskeyCredentialMapper creates a new passkey credential mapper
func NewPasskeyCredentialMapper() PasskeyCredentialMapper {
	return &PasskeyCredentialMapperImpl{}
}

// ToEntity converts a persistence model to a domain entity
func (m *PasskeyCredentialMapperImpl) ToEntity(model *models.PasskeyCredentialModel) (*user.PasskeyCredential, error) {
	if model == nil {
		return nil, nil
	}

	var transports []string
	if len(model.Transports) > 0 {
		if err := json.Unmarshal(model.Transports, &transports); err != nil {
			return nil, fmt.Errorf("failed to unmarshal transports: %w", err)
		}
	}

	credential, err := user.ReconstructPasskeyCredential(
		model.ID,
		model.SID,
		model.UserID,
		model.CredentialID,
		model.PublicKey,
		model.AttestationType,
		model.AAGUID,
		model.SignCount,
		model.BackupEligible,
		model.BackupState,
		transports,
		model.DeviceName,
		model.LastUsedAt,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct passkey credential entity: %w", err)
	}

	return credential, nil
}

// ToModel converts a domain entity to a persistence model
func (m *PasskeyCredentialMapperImpl) ToModel(entity *user.PasskeyCredential) (*models.PasskeyCredentialModel, error) {
	if entity == nil {
		return nil, nil
	}

	var transportsJSON []byte
	if len(entity.Transports()) > 0 {
		var err error
		transportsJSON, err = json.Marshal(entity.Transports())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal transports: %w", err)
		}
	}

	model := &models.PasskeyCredentialModel{
		ID:              entity.ID(),
		SID:             entity.SID(),
		UserID:          entity.UserID(),
		CredentialID:    entity.CredentialID(),
		PublicKey:       entity.PublicKey(),
		AttestationType: entity.AttestationType(),
		AAGUID:          entity.AAGUID(),
		SignCount:       entity.SignCount(),
		BackupEligible:  entity.BackupEligible(),
		BackupState:     entity.BackupState(),
		Transports:      transportsJSON,
		DeviceName:      entity.DeviceName(),
		LastUsedAt:      entity.LastUsedAt(),
		CreatedAt:       entity.CreatedAt(),
		UpdatedAt:       entity.UpdatedAt(),
	}

	return model, nil
}

// ToEntities converts multiple persistence models to domain entities
func (m *PasskeyCredentialMapperImpl) ToEntities(modelList []*models.PasskeyCredentialModel) ([]*user.PasskeyCredential, error) {
	return mapper.MapSlicePtrWithID(modelList, m.ToEntity, func(model *models.PasskeyCredentialModel) uint { return model.ID })
}

// ToModels converts multiple domain entities to persistence models
func (m *PasskeyCredentialMapperImpl) ToModels(entities []*user.PasskeyCredential) ([]*models.PasskeyCredentialModel, error) {
	return mapper.MapSlicePtrWithID(entities, m.ToModel, func(entity *user.PasskeyCredential) uint { return entity.ID() })
}
