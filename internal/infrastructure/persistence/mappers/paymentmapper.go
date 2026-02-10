package mappers

import (
	"fmt"

	"github.com/orris-inc/orris/internal/domain/payment"
	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// PaymentMapper handles the conversion between domain entities and persistence models
type PaymentMapper interface {
	// ToModel converts a domain entity to a persistence model
	ToModel(p *payment.Payment) *models.PaymentModel

	// ToDomain converts a persistence model to a domain entity
	ToDomain(model *models.PaymentModel) (*payment.Payment, error)
}

// PaymentMapperImpl is the concrete implementation of PaymentMapper
type PaymentMapperImpl struct{}

// NewPaymentMapper creates a new payment mapper
func NewPaymentMapper() PaymentMapper {
	return &PaymentMapperImpl{}
}

// ToModel converts a domain entity to a persistence model
func (m *PaymentMapperImpl) ToModel(p *payment.Payment) *models.PaymentModel {
	model := &models.PaymentModel{
		ID:               p.ID(),
		OrderNo:          p.OrderNo(),
		SubscriptionID:   p.SubscriptionID(),
		UserID:           p.UserID(),
		Amount:           p.Amount().AmountInCents(),
		Currency:         p.Amount().Currency(),
		PaymentMethod:    p.PaymentMethod().String(),
		PaymentStatus:    p.Status().String(),
		GatewayOrderNo:   p.GatewayOrderNo(),
		TransactionID:    p.TransactionID(),
		PaymentURL:       p.PaymentURL(),
		QRCode:           p.QRCode(),
		PaidAt:           p.PaidAt(),
		ExpiredAt:        p.ExpiredAt(),
		USDTAmountRaw:    p.USDTAmountRaw(),
		ReceivingAddress: p.ReceivingAddress(),
		ExchangeRate:     p.ExchangeRate(),
		TxHash:           p.TxHash(),
		BlockNumber:      p.BlockNumber(),
		ConfirmedAt:      p.ConfirmedAt(),
		Version:          p.Version(),
		CreatedAt:        p.CreatedAt(),
		UpdatedAt:        p.UpdatedAt(),
	}

	// Map ChainType
	if ct := p.ChainType(); ct != nil {
		ctStr := ct.String()
		model.ChainType = &ctStr
	}

	if len(p.Metadata()) > 0 {
		model.Metadata = p.Metadata()
	}

	return model
}

// ToDomain converts a persistence model to a domain entity
func (m *PaymentMapperImpl) ToDomain(model *models.PaymentModel) (*payment.Payment, error) {
	amount := vo.NewMoney(model.Amount, model.Currency)

	method, err := vo.NewPaymentMethod(model.PaymentMethod)
	if err != nil {
		return nil, fmt.Errorf("invalid payment method: %w", err)
	}

	status := vo.PaymentStatus(model.PaymentStatus)
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid payment status: %s", model.PaymentStatus)
	}

	metadata := model.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Parse ChainType if present
	var chainType *vo.ChainType
	if model.ChainType != nil && *model.ChainType != "" {
		ct, err := vo.NewChainType(*model.ChainType)
		if err == nil {
			chainType = &ct
		}
	}

	return payment.ReconstructPaymentWithParams(payment.PaymentReconstructParams{
		ID:               model.ID,
		OrderNo:          model.OrderNo,
		SubscriptionID:   model.SubscriptionID,
		UserID:           model.UserID,
		Amount:           amount,
		PaymentMethod:    method,
		Status:           status,
		GatewayOrderNo:   model.GatewayOrderNo,
		TransactionID:    model.TransactionID,
		PaymentURL:       model.PaymentURL,
		QRCode:           model.QRCode,
		PaidAt:           model.PaidAt,
		ExpiredAt:        model.ExpiredAt,
		Metadata:         metadata,
		Version:          model.Version,
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
		ChainType:        chainType,
		USDTAmountRaw:    model.USDTAmountRaw,
		ReceivingAddress: model.ReceivingAddress,
		ExchangeRate:     model.ExchangeRate,
		TxHash:           model.TxHash,
		BlockNumber:      model.BlockNumber,
		ConfirmedAt:      model.ConfirmedAt,
	}), nil
}
