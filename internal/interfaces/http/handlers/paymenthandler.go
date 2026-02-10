package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	paymentUsecases "github.com/orris-inc/orris/internal/application/payment/usecases"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

type PaymentHandler struct {
	createPaymentUC  *paymentUsecases.CreatePaymentUseCase
	handleCallbackUC *paymentUsecases.HandlePaymentCallbackUseCase
	subscriptionRepo subscription.SubscriptionRepository
	logger           logger.Interface
}

func NewPaymentHandler(
	createPaymentUC *paymentUsecases.CreatePaymentUseCase,
	handleCallbackUC *paymentUsecases.HandlePaymentCallbackUseCase,
	subscriptionRepo subscription.SubscriptionRepository,
	logger logger.Interface,
) *PaymentHandler {
	return &PaymentHandler{
		createPaymentUC:  createPaymentUC,
		handleCallbackUC: handleCallbackUC,
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
	}
}

type CreatePaymentRequest struct {
	SubscriptionSID string `json:"subscription_id" binding:"required"` // Stripe-style SID (sub_xxx)
	BillingCycle    string `json:"billing_cycle" binding:"required,oneof=monthly quarterly semi_annual yearly"`
	PaymentMethod   string `json:"payment_method" binding:"required,oneof=alipay wechat stripe usdt_pol usdt_trc"`
	ReturnURL       string `json:"return_url"`
}

type CreatePaymentResponse struct {
	OrderNo    string `json:"order_no"`
	PaymentURL string `json:"payment_url"`
	QRCode     string `json:"qr_code,omitempty"`
	ExpiredAt  string `json:"expired_at"`
	// USDT-specific fields (only present for USDT payments)
	ChainType        string  `json:"chain_type,omitempty"`
	USDTAmount       float64 `json:"usdt_amount,omitempty"`
	ReceivingAddress string  `json:"receiving_address,omitempty"`
	ExchangeRate     float64 `json:"exchange_rate,omitempty"`
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorw("failed to bind request", "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	// Convert SID to internal ID
	sub, err := h.subscriptionRepo.GetBySID(c.Request.Context(), req.SubscriptionSID)
	if err != nil {
		h.logger.Warnw("subscription not found", "sid", req.SubscriptionSID, "error", err)
		utils.ErrorResponse(c, http.StatusNotFound, "subscription not found")
		return
	}

	cmd := paymentUsecases.CreatePaymentCommand{
		SubscriptionID: sub.ID(),
		UserID:         userID,
		BillingCycle:   req.BillingCycle,
		PaymentMethod:  req.PaymentMethod,
		ReturnURL:      req.ReturnURL,
	}

	result, err := h.createPaymentUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("failed to create payment", "error", err, "user_id", userID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	response := CreatePaymentResponse{
		OrderNo:    result.Payment.OrderNo(),
		PaymentURL: result.PaymentURL,
		QRCode:     result.QRCode,
		ExpiredAt:  result.Payment.ExpiredAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	// Add USDT-specific fields if present
	if result.Payment.IsUSDTPayment() {
		if ct := result.Payment.ChainType(); ct != nil {
			response.ChainType = ct.String()
		}
		if ua := result.Payment.USDTAmountRaw(); ua != nil {
			// Convert raw amount to float for display (1 USDT = 1000000 units)
			response.USDTAmount = float64(*ua) / 1000000.0
		}
		if ra := result.Payment.ReceivingAddress(); ra != nil {
			response.ReceivingAddress = *ra
		}
		if er := result.Payment.ExchangeRate(); er != nil {
			response.ExchangeRate = *er
		}
	}

	utils.CreatedResponse(c, response, "payment created successfully")
}

func (h *PaymentHandler) HandleCallback(c *gin.Context) {
	if err := h.handleCallbackUC.Execute(c.Request.Context(), c.Request); err != nil {
		h.logger.Errorw("failed to handle payment callback", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "callback processed successfully", nil)
}
