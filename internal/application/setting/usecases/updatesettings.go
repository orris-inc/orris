package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/setting/dto"
	"github.com/orris-inc/orris/internal/domain/setting"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SettingChangeNotifier defines the interface for notifying setting changes
type SettingChangeNotifier interface {
	NotifyChange(ctx context.Context, category string, changes map[string]any) error
}

// UpdateSettingsUseCase handles updating system settings
type UpdateSettingsUseCase struct {
	settingRepo setting.Repository
	notifier    SettingChangeNotifier
	logger      logger.Interface
}

// NewUpdateSettingsUseCase creates a new UpdateSettingsUseCase
func NewUpdateSettingsUseCase(
	settingRepo setting.Repository,
	notifier SettingChangeNotifier,
	logger logger.Interface,
) *UpdateSettingsUseCase {
	return &UpdateSettingsUseCase{
		settingRepo: settingRepo,
		notifier:    notifier,
		logger:      logger,
	}
}

// UpdateCategorySettings batch updates settings in a category
func (uc *UpdateSettingsUseCase) UpdateCategorySettings(
	ctx context.Context,
	category string,
	request dto.UpdateCategorySettingsRequest,
	updatedBy uint,
) error {
	if len(request.Settings) == 0 {
		return nil
	}

	changes := make(map[string]any)

	for key, value := range request.Settings {
		if err := uc.updateSingleSetting(ctx, category, key, value, updatedBy); err != nil {
			uc.logger.Errorw("failed to update setting",
				"category", category,
				"key", key,
				"error", err,
			)
			return fmt.Errorf("failed to update setting %s.%s: %w", category, key, err)
		}
		changes[key] = value
	}

	// Notify subscribers of the changes
	if uc.notifier != nil && len(changes) > 0 {
		if err := uc.notifier.NotifyChange(ctx, category, changes); err != nil {
			uc.logger.Warnw("failed to notify setting changes",
				"category", category,
				"error", err,
			)
			// Don't fail the update if notification fails
		}
	}

	return nil
}

// UpdateTelegramConfig updates Telegram configuration settings
func (uc *UpdateSettingsUseCase) UpdateTelegramConfig(
	ctx context.Context,
	request dto.UpdateTelegramConfigRequest,
	updatedBy uint,
) error {
	changes := make(map[string]any)

	if request.Enabled != nil {
		if err := uc.updateSingleSetting(ctx, "telegram", "enabled", *request.Enabled, updatedBy); err != nil {
			return fmt.Errorf("failed to update telegram.enabled: %w", err)
		}
		changes["enabled"] = *request.Enabled
	}

	if request.BotToken != nil {
		if err := uc.updateSingleSetting(ctx, "telegram", "bot_token", *request.BotToken, updatedBy); err != nil {
			return fmt.Errorf("failed to update telegram.bot_token: %w", err)
		}
		changes["bot_token"] = *request.BotToken
	}

	if request.WebhookURL != nil {
		if err := uc.updateSingleSetting(ctx, "telegram", "webhook_url", *request.WebhookURL, updatedBy); err != nil {
			return fmt.Errorf("failed to update telegram.webhook_url: %w", err)
		}
		changes["webhook_url"] = *request.WebhookURL
	}

	if request.WebhookSecret != nil {
		if err := uc.updateSingleSetting(ctx, "telegram", "webhook_secret", *request.WebhookSecret, updatedBy); err != nil {
			return fmt.Errorf("failed to update telegram.webhook_secret: %w", err)
		}
		changes["webhook_secret"] = *request.WebhookSecret
	}

	// Notify subscribers of the changes
	if uc.notifier != nil && len(changes) > 0 {
		if err := uc.notifier.NotifyChange(ctx, "telegram", changes); err != nil {
			uc.logger.Warnw("failed to notify telegram config changes",
				"error", err,
			)
			// Don't fail the update if notification fails
		}
	}

	return nil
}

// updateSingleSetting updates or creates a single setting
func (uc *UpdateSettingsUseCase) updateSingleSetting(
	ctx context.Context,
	category, key string,
	value any,
	updatedBy uint,
) error {
	// Get existing setting or create new one
	existingSetting, err := uc.settingRepo.GetByKey(ctx, category, key)
	if err != nil && err != setting.ErrSettingNotFound {
		return err
	}

	var s *setting.SystemSetting

	if existingSetting != nil {
		// Update existing setting
		s = existingSetting
	} else {
		// Create new setting with inferred type
		valueType := uc.inferValueType(value)
		s, err = setting.NewSystemSetting(category, key, valueType, "")
		if err != nil {
			return err
		}
	}

	// Set the value based on type
	if err := uc.setValueByType(s, value, updatedBy); err != nil {
		return err
	}

	return uc.settingRepo.Upsert(ctx, s)
}

// inferValueType infers the value type from the Go value
func (uc *UpdateSettingsUseCase) inferValueType(value any) setting.ValueType {
	switch value.(type) {
	case bool:
		return setting.ValueTypeBool
	case int, int32, int64, float32, float64:
		return setting.ValueTypeInt
	case string:
		return setting.ValueTypeString
	default:
		return setting.ValueTypeJSON
	}
}

// setValueByType sets the value on the setting based on its type
func (uc *UpdateSettingsUseCase) setValueByType(s *setting.SystemSetting, value any, updatedBy uint) error {
	switch s.ValueType() {
	case setting.ValueTypeBool:
		boolVal, ok := value.(bool)
		if !ok {
			return fmt.Errorf("expected bool value for key %s", s.Key())
		}
		return s.SetBoolValue(boolVal, updatedBy)

	case setting.ValueTypeInt:
		var intVal int
		switch v := value.(type) {
		case int:
			intVal = v
		case int32:
			intVal = int(v)
		case int64:
			intVal = int(v)
		case float64:
			intVal = int(v)
		case float32:
			intVal = int(v)
		default:
			return fmt.Errorf("expected int value for key %s", s.Key())
		}
		return s.SetIntValue(intVal, updatedBy)

	case setting.ValueTypeString:
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string value for key %s", s.Key())
		}
		return s.SetStringValue(strVal, updatedBy)

	case setting.ValueTypeJSON:
		return s.SetJSONValue(value, updatedBy)

	default:
		return fmt.Errorf("unsupported value type: %s", s.ValueType())
	}
}

// UpsertSetting creates or updates a setting directly
func (uc *UpdateSettingsUseCase) UpsertSetting(ctx context.Context, s *setting.SystemSetting) error {
	return uc.settingRepo.Upsert(ctx, s)
}
