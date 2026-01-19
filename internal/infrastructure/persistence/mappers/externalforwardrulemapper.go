package mappers

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"

	"github.com/orris-inc/orris/internal/domain/externalforward"
	vo "github.com/orris-inc/orris/internal/domain/externalforward/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// ExternalForwardRuleMapper provides mapping between domain and persistence models.
type ExternalForwardRuleMapper struct{}

// NewExternalForwardRuleMapper creates a new mapper.
func NewExternalForwardRuleMapper() *ExternalForwardRuleMapper {
	return &ExternalForwardRuleMapper{}
}

// ToModel converts a domain entity to a persistence model.
func (m *ExternalForwardRuleMapper) ToModel(rule *externalforward.ExternalForwardRule) (*models.ExternalForwardRuleModel, error) {
	if rule == nil {
		return nil, nil
	}

	// Serialize group_ids to JSON
	var groupIDsJSON datatypes.JSON
	if len(rule.GroupIDs()) > 0 {
		jsonBytes, err := json.Marshal(rule.GroupIDs())
		if err != nil {
			return nil, fmt.Errorf("failed to serialize group_ids: %w", err)
		}
		groupIDsJSON = jsonBytes
	}

	return &models.ExternalForwardRuleModel{
		ID:             rule.ID(),
		SID:            rule.SID(),
		SubscriptionID: rule.SubscriptionID(),
		UserID:         rule.UserID(),
		NodeID:         rule.NodeID(),
		Name:           rule.Name(),
		ServerAddress:  rule.ServerAddress(),
		ListenPort:     rule.ListenPort(),
		ExternalSource: rule.ExternalSource(),
		ExternalRuleID: rule.ExternalRuleID(),
		Status:         rule.Status().String(),
		SortOrder:      rule.SortOrder(),
		Remark:         rule.Remark(),
		GroupIDs:       groupIDsJSON,
		CreatedAt:      rule.CreatedAt(),
		UpdatedAt:      rule.UpdatedAt(),
	}, nil
}

// ToDomain converts a persistence model to a domain entity.
func (m *ExternalForwardRuleMapper) ToDomain(model *models.ExternalForwardRuleModel) (*externalforward.ExternalForwardRule, error) {
	if model == nil {
		return nil, nil
	}

	// Parse group_ids JSON
	var groupIDs []uint
	if len(model.GroupIDs) > 0 {
		if err := json.Unmarshal(model.GroupIDs, &groupIDs); err != nil {
			return nil, fmt.Errorf("failed to parse group_ids: %w", err)
		}
	}

	return externalforward.ReconstructExternalForwardRule(
		model.ID,
		model.SID,
		model.SubscriptionID,
		model.UserID,
		model.NodeID,
		model.Name,
		model.ServerAddress,
		model.ListenPort,
		model.ExternalSource,
		model.ExternalRuleID,
		vo.Status(model.Status),
		model.SortOrder,
		model.Remark,
		groupIDs,
		model.CreatedAt,
		model.UpdatedAt,
	)
}

// ToDomainList converts a slice of persistence models to domain entities.
func (m *ExternalForwardRuleMapper) ToDomainList(modelList []*models.ExternalForwardRuleModel) ([]*externalforward.ExternalForwardRule, error) {
	if modelList == nil {
		return nil, nil
	}

	rules := make([]*externalforward.ExternalForwardRule, 0, len(modelList))
	for _, model := range modelList {
		rule, err := m.ToDomain(model)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, nil
}
