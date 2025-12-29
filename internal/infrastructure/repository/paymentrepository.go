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
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(ctx context.Context, p *payment.Payment) error {
	model := mappers.PaymentToModel(p)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	return nil
}

func (r *PaymentRepository) Update(ctx context.Context, p *payment.Payment) error {
	model := mappers.PaymentToModel(p)

	result := r.db.WithContext(ctx).
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
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update payment: %w", result.Error)
	}

	// Note: RowsAffected may be 0 when updated values are identical to existing values.

	return nil
}

func (r *PaymentRepository) GetByID(ctx context.Context, id uint) (*payment.Payment, error) {
	var model models.PaymentModel

	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("payment not found")
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	return mappers.PaymentToDomain(&model)
}

func (r *PaymentRepository) GetByOrderNo(ctx context.Context, orderNo string) (*payment.Payment, error) {
	var model models.PaymentModel

	if err := r.db.WithContext(ctx).
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

	if err := r.db.WithContext(ctx).
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

	if err := r.db.WithContext(ctx).
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

	if err := r.db.WithContext(ctx).
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

func (r *PaymentRepository) GetExpiredPayments(ctx context.Context) ([]*payment.Payment, error) {
	var paymentModels []models.PaymentModel

	if err := r.db.WithContext(ctx).
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
