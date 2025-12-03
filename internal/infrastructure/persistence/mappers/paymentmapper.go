package mappers

import (
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/payment"
	vo "github.com/orris-inc/orris/internal/domain/payment/value_objects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

func PaymentToModel(p *payment.Payment) *models.PaymentModel {
	model := &models.PaymentModel{
		ID:             p.ID(),
		OrderNo:        p.OrderNo(),
		SubscriptionID: p.SubscriptionID(),
		UserID:         p.UserID(),
		Amount:         p.Amount().AmountInCents(),
		Currency:       p.Amount().Currency(),
		PaymentMethod:  p.PaymentMethod().String(),
		PaymentStatus:  p.Status().String(),
		GatewayOrderNo: p.GatewayOrderNo(),
		TransactionID:  p.TransactionID(),
		PaymentURL:     p.PaymentURL(),
		QRCode:         p.QRCode(),
		PaidAt:         p.PaidAt(),
		ExpiredAt:      p.ExpiredAt(),
		Version:        p.Version(),
		CreatedAt:      p.CreatedAt(),
		UpdatedAt:      p.UpdatedAt(),
	}

	if len(p.Metadata()) > 0 {
		model.Metadata = p.Metadata()
	}

	return model
}

func PaymentToDomain(model *models.PaymentModel) (*payment.Payment, error) {
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

	p := &payment.Payment{}

	setPaymentFields(p, model.ID, model.OrderNo, model.SubscriptionID, model.UserID,
		amount, method, status, model.GatewayOrderNo, model.TransactionID,
		model.PaymentURL, model.QRCode, model.PaidAt, model.ExpiredAt,
		metadata, model.Version, model.CreatedAt, model.UpdatedAt)

	return p, nil
}

func setPaymentFields(p *payment.Payment, id uint, orderNo string, subscriptionID, userID uint,
	amount vo.Money, method vo.PaymentMethod, status vo.PaymentStatus,
	gatewayOrderNo, transactionID, paymentURL, qrCode *string,
	paidAt *time.Time, expiredAt time.Time, metadata map[string]interface{},
	version int, createdAt, updatedAt time.Time) {

	paymentValue := payment.ReconstructPayment(
		id, orderNo, subscriptionID, userID, amount, method, status,
		gatewayOrderNo, transactionID, paymentURL, qrCode,
		paidAt, expiredAt, metadata, version, createdAt, updatedAt,
	)

	*p = *paymentValue
}
