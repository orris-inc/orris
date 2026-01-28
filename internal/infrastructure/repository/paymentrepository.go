package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/domain/payment"
	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/mappers"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/db"
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(ctx context.Context, p *payment.Payment) error {
	model := mappers.PaymentToModel(p)

	if err := db.GetTxFromContext(ctx, r.db).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	// Write back the auto-generated ID to the domain object
	p.SetID(model.ID)

	return nil
}

func (r *PaymentRepository) Update(ctx context.Context, p *payment.Payment) error {
	model := mappers.PaymentToModel(p)

	result := db.GetTxFromContext(ctx, r.db).
		Model(&models.PaymentModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]interface{}{
			"payment_status":   model.PaymentStatus,
			"transaction_id":   model.TransactionID,
			"paid_at":          model.PaidAt,
			"metadata":         model.Metadata,
			"version":          model.Version,
			"updated_at":       model.UpdatedAt,
			"gateway_order_no": model.GatewayOrderNo,
			"payment_url":      model.PaymentURL,
			"qr_code":          model.QRCode,
			// USDT-specific fields
			"chain_type":        model.ChainType,
			"usdt_amount_raw":   model.USDTAmountRaw,
			"receiving_address": model.ReceivingAddress,
			"exchange_rate":     model.ExchangeRate,
			"tx_hash":           model.TxHash,
			"block_number":      model.BlockNumber,
			"confirmed_at":      model.ConfirmedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update payment: %w", result.Error)
	}

	// Note: RowsAffected may be 0 when updated values are identical to existing values.

	return nil
}

func (r *PaymentRepository) GetByID(ctx context.Context, id uint) (*payment.Payment, error) {
	var model models.PaymentModel

	if err := db.GetTxFromContext(ctx, r.db).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment not found")
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	return mappers.PaymentToDomain(&model)
}

func (r *PaymentRepository) GetByOrderNo(ctx context.Context, orderNo string) (*payment.Payment, error) {
	var model models.PaymentModel

	if err := db.GetTxFromContext(ctx, r.db).
		Where("order_no = ?", orderNo).
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment not found")
		}
		return nil, fmt.Errorf("failed to get payment by order_no: %w", err)
	}

	return mappers.PaymentToDomain(&model)
}

func (r *PaymentRepository) GetByGatewayOrderNo(ctx context.Context, gatewayOrderNo string) (*payment.Payment, error) {
	var model models.PaymentModel

	if err := db.GetTxFromContext(ctx, r.db).
		Where("gateway_order_no = ?", gatewayOrderNo).
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment not found")
		}
		return nil, fmt.Errorf("failed to get payment by gateway_order_no: %w", err)
	}

	return mappers.PaymentToDomain(&model)
}

func (r *PaymentRepository) GetBySubscriptionID(ctx context.Context, subscriptionID uint) ([]*payment.Payment, error) {
	var paymentModels []models.PaymentModel

	if err := db.GetTxFromContext(ctx, r.db).
		Where("subscription_id = ?", subscriptionID).
		Order("created_at DESC").
		Find(&paymentModels).Error; err != nil {
		return nil, fmt.Errorf("failed to get payments by subscription_id: %w", err)
	}

	payments := make([]*payment.Payment, len(paymentModels))
	for i, model := range paymentModels {
		p, err := mappers.PaymentToDomain(&model)
		if err != nil {
			return nil, err
		}
		payments[i] = p
	}

	return payments, nil
}

func (r *PaymentRepository) GetPendingBySubscriptionID(ctx context.Context, subscriptionID uint) (*payment.Payment, error) {
	var model models.PaymentModel

	if err := db.GetTxFromContext(ctx, r.db).
		Where("subscription_id = ? AND payment_status = ?", subscriptionID, vo.PaymentStatusPending).
		Order("created_at DESC").
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get pending payment: %w", err)
	}

	return mappers.PaymentToDomain(&model)
}

// HasPendingPaymentBySubscriptionID checks if there are any pending payments for a subscription
func (r *PaymentRepository) HasPendingPaymentBySubscriptionID(ctx context.Context, subscriptionID uint) (bool, error) {
	var count int64

	if err := db.GetTxFromContext(ctx, r.db).
		Model(&models.PaymentModel{}).
		Where("subscription_id = ? AND payment_status = ?", subscriptionID, vo.PaymentStatusPending).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check pending payments: %w", err)
	}

	return count > 0, nil
}

func (r *PaymentRepository) GetExpiredPayments(ctx context.Context) ([]*payment.Payment, error) {
	var paymentModels []models.PaymentModel

	if err := db.GetTxFromContext(ctx, r.db).
		Where("payment_status = ? AND expired_at < ?", vo.PaymentStatusPending, biztime.NowUTC()).
		Find(&paymentModels).Error; err != nil {
		return nil, fmt.Errorf("failed to get expired payments: %w", err)
	}

	payments := make([]*payment.Payment, len(paymentModels))
	for i, model := range paymentModels {
		p, err := mappers.PaymentToDomain(&model)
		if err != nil {
			return nil, err
		}
		payments[i] = p
	}

	return payments, nil
}

// GetPendingUSDTPayments returns all pending USDT payments that haven't expired
func (r *PaymentRepository) GetPendingUSDTPayments(ctx context.Context) ([]*payment.Payment, error) {
	var paymentModels []models.PaymentModel

	if err := db.GetTxFromContext(ctx, r.db).
		Where("payment_status = ? AND payment_method IN ? AND expired_at > ?",
			vo.PaymentStatusPending,
			[]string{string(vo.PaymentMethodUSDTPOL), string(vo.PaymentMethodUSDTTRC)},
			biztime.NowUTC(),
		).
		Find(&paymentModels).Error; err != nil {
		return nil, fmt.Errorf("failed to get pending USDT payments: %w", err)
	}

	payments := make([]*payment.Payment, len(paymentModels))
	for i, model := range paymentModels {
		p, err := mappers.PaymentToDomain(&model)
		if err != nil {
			return nil, err
		}
		payments[i] = p
	}

	return payments, nil
}

// GetConfirmedUSDTPaymentsNeedingActivation returns confirmed USDT payments
// that have subscription_activation_pending=true in metadata
func (r *PaymentRepository) GetConfirmedUSDTPaymentsNeedingActivation(ctx context.Context) ([]*payment.Payment, error) {
	var paymentModels []models.PaymentModel

	// Query for paid USDT payments with subscription_activation_pending in metadata
	// Using JSON_EXTRACT for MySQL/MariaDB compatibility
	if err := db.GetTxFromContext(ctx, r.db).
		Where("payment_status = ? AND payment_method IN ? AND JSON_EXTRACT(metadata, '$.subscription_activation_pending') = true",
			vo.PaymentStatusPaid,
			[]string{string(vo.PaymentMethodUSDTPOL), string(vo.PaymentMethodUSDTTRC)},
		).
		Find(&paymentModels).Error; err != nil {
		return nil, fmt.Errorf("failed to get payments needing activation: %w", err)
	}

	payments := make([]*payment.Payment, len(paymentModels))
	for i, model := range paymentModels {
		p, err := mappers.PaymentToDomain(&model)
		if err != nil {
			return nil, err
		}
		payments[i] = p
	}

	return payments, nil
}

// CountPendingUSDTPaymentsByUser returns the count of pending USDT payments for a user
func (r *PaymentRepository) CountPendingUSDTPaymentsByUser(ctx context.Context, userID uint) (int, error) {
	var count int64

	if err := db.GetTxFromContext(ctx, r.db).
		Model(&models.PaymentModel{}).
		Where("user_id = ? AND payment_status = ? AND payment_method IN ?",
			userID,
			vo.PaymentStatusPending,
			[]string{string(vo.PaymentMethodUSDTPOL), string(vo.PaymentMethodUSDTTRC)},
		).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count pending USDT payments: %w", err)
	}

	return int(count), nil
}

// GetPaidPaymentsNeedingActivation returns paid non-USDT payments
// that have subscription_activation_pending=true in metadata
func (r *PaymentRepository) GetPaidPaymentsNeedingActivation(ctx context.Context) ([]*payment.Payment, error) {
	var paymentModels []models.PaymentModel

	if err := db.GetTxFromContext(ctx, r.db).
		Where("payment_status = ? AND payment_method NOT IN ? AND JSON_EXTRACT(metadata, '$.subscription_activation_pending') = true",
			vo.PaymentStatusPaid,
			[]string{string(vo.PaymentMethodUSDTPOL), string(vo.PaymentMethodUSDTTRC)},
		).
		Find(&paymentModels).Error; err != nil {
		return nil, fmt.Errorf("failed to get non-USDT payments needing activation: %w", err)
	}

	payments := make([]*payment.Payment, len(paymentModels))
	for i, model := range paymentModels {
		p, err := mappers.PaymentToDomain(&model)
		if err != nil {
			return nil, err
		}
		payments[i] = p
	}

	return payments, nil
}
