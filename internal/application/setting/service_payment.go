package setting

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/setting/dto"
	paymentVO "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
)

// ============================================================================
// USDT Payment Settings
// ============================================================================

// GetUSDTSettings retrieves USDT payment settings
func (s *ServiceDDD) GetUSDTSettings(ctx context.Context) (*dto.USDTSettingsResponse, error) {
	return &dto.USDTSettingsResponse{
		Enabled:               s.getSettingWithSourceBool(ctx, "usdt", "enabled"),
		POLReceivingAddresses: s.getSettingWithSourceStringArray(ctx, "usdt", "pol_receiving_addresses"),
		TRCReceivingAddresses: s.getSettingWithSourceStringArray(ctx, "usdt", "trc_receiving_addresses"),
		PolygonScanAPIKey:     s.getSettingWithSourceMasked(ctx, "usdt", "polygonscan_api_key"),
		TronGridAPIKey:        s.getSettingWithSourceMasked(ctx, "usdt", "trongrid_api_key"),
		PaymentTTLMinutes:     s.getSettingWithSourceInt(ctx, "usdt", "payment_ttl_minutes", 10),
		POLConfirmations:      s.getSettingWithSourceInt(ctx, "usdt", "pol_confirmations", 12),
		TRCConfirmations:      s.getSettingWithSourceInt(ctx, "usdt", "trc_confirmations", 19),
	}, nil
}

// UpdateUSDTSettings updates USDT payment settings
func (s *ServiceDDD) UpdateUSDTSettings(ctx context.Context, req dto.UpdateUSDTSettingsRequest, updatedBy uint) error {
	// Validation constants
	const (
		maxConfirmations = 100
		minConfirmations = 1
		maxPaymentTTL    = 1440 // 24 hours in minutes
		minPaymentTTL    = 5
		maxAddresses     = 10 // Maximum number of addresses per chain
	)

	// Validate confirmation counts
	if req.POLConfirmations != nil {
		if *req.POLConfirmations < minConfirmations || *req.POLConfirmations > maxConfirmations {
			return fmt.Errorf("pol_confirmations must be between %d and %d", minConfirmations, maxConfirmations)
		}
	}
	if req.TRCConfirmations != nil {
		if *req.TRCConfirmations < minConfirmations || *req.TRCConfirmations > maxConfirmations {
			return fmt.Errorf("trc_confirmations must be between %d and %d", minConfirmations, maxConfirmations)
		}
	}
	if req.PaymentTTLMinutes != nil {
		if *req.PaymentTTLMinutes < minPaymentTTL || *req.PaymentTTLMinutes > maxPaymentTTL {
			return fmt.Errorf("payment_ttl_minutes must be between %d and %d", minPaymentTTL, maxPaymentTTL)
		}
	}
	// Validate address arrays
	if req.POLReceivingAddresses != nil && len(*req.POLReceivingAddresses) > maxAddresses {
		return fmt.Errorf("pol_receiving_addresses cannot exceed %d addresses", maxAddresses)
	}
	if req.TRCReceivingAddresses != nil && len(*req.TRCReceivingAddresses) > maxAddresses {
		return fmt.Errorf("trc_receiving_addresses cannot exceed %d addresses", maxAddresses)
	}

	// Validate address formats
	if req.POLReceivingAddresses != nil {
		for i, addr := range *req.POLReceivingAddresses {
			if err := paymentVO.ChainTypePOL.ValidateAddress(addr); err != nil {
				return fmt.Errorf("invalid Polygon address at index %d: %w", i, err)
			}
		}
	}
	if req.TRCReceivingAddresses != nil {
		for i, addr := range *req.TRCReceivingAddresses {
			if err := paymentVO.ChainTypeTRC.ValidateAddress(addr); err != nil {
				return fmt.Errorf("invalid Tron address at index %d: %w", i, err)
			}
		}
	}

	changes := make(map[string]any)

	if req.Enabled != nil {
		if err := s.upsertSettingBool(ctx, "usdt", "enabled", *req.Enabled, updatedBy); err != nil {
			return err
		}
		changes["enabled"] = *req.Enabled
	}
	if req.POLReceivingAddresses != nil {
		if err := s.upsertSettingStringArray(ctx, "usdt", "pol_receiving_addresses", *req.POLReceivingAddresses, updatedBy); err != nil {
			return err
		}
		changes["pol_receiving_addresses"] = *req.POLReceivingAddresses
	}
	if req.TRCReceivingAddresses != nil {
		if err := s.upsertSettingStringArray(ctx, "usdt", "trc_receiving_addresses", *req.TRCReceivingAddresses, updatedBy); err != nil {
			return err
		}
		changes["trc_receiving_addresses"] = *req.TRCReceivingAddresses
	}
	if req.PolygonScanAPIKey != nil {
		if err := s.upsertSetting(ctx, "usdt", "polygonscan_api_key", *req.PolygonScanAPIKey, updatedBy); err != nil {
			return err
		}
		changes["polygonscan_api_key"] = "[REDACTED]"
	}
	if req.TronGridAPIKey != nil {
		if err := s.upsertSetting(ctx, "usdt", "trongrid_api_key", *req.TronGridAPIKey, updatedBy); err != nil {
			return err
		}
		changes["trongrid_api_key"] = "[REDACTED]"
	}
	if req.PaymentTTLMinutes != nil {
		if err := s.upsertSettingInt(ctx, "usdt", "payment_ttl_minutes", *req.PaymentTTLMinutes, updatedBy); err != nil {
			return err
		}
		changes["payment_ttl_minutes"] = *req.PaymentTTLMinutes
	}
	if req.POLConfirmations != nil {
		if err := s.upsertSettingInt(ctx, "usdt", "pol_confirmations", *req.POLConfirmations, updatedBy); err != nil {
			return err
		}
		changes["pol_confirmations"] = *req.POLConfirmations
	}
	if req.TRCConfirmations != nil {
		if err := s.upsertSettingInt(ctx, "usdt", "trc_confirmations", *req.TRCConfirmations, updatedBy); err != nil {
			return err
		}
		changes["trc_confirmations"] = *req.TRCConfirmations
	}

	if len(changes) > 0 {
		if err := s.settingProvider.NotifyChange(ctx, "usdt", changes); err != nil {
			s.logger.Warnw("failed to notify USDT setting changes", "error", err)
		}
	}
	return nil
}

// ============================================================================
// Subscription Settings
// ============================================================================

// GetSubscriptionSettings retrieves subscription settings
func (s *ServiceDDD) GetSubscriptionSettings(ctx context.Context) (*dto.SubscriptionSettingsResponse, error) {
	return &dto.SubscriptionSettingsResponse{
		ShowInfoNodes: s.getSettingWithSourceBool(ctx, "subscription", "show_info_nodes"),
	}, nil
}

// UpdateSubscriptionSettings updates subscription settings
func (s *ServiceDDD) UpdateSubscriptionSettings(ctx context.Context, req dto.UpdateSubscriptionSettingsRequest, updatedBy uint) error {
	changes := make(map[string]any)

	if req.ShowInfoNodes != nil {
		if err := s.upsertSettingBool(ctx, "subscription", "show_info_nodes", *req.ShowInfoNodes, updatedBy); err != nil {
			return err
		}
		changes["show_info_nodes"] = *req.ShowInfoNodes
	}

	if len(changes) > 0 {
		if err := s.settingProvider.NotifyChange(ctx, "subscription", changes); err != nil {
			s.logger.Warnw("failed to notify subscription setting changes", "error", err)
		}
	}
	return nil
}
