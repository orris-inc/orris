package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	paymentUsecases "orris/internal/application/payment/usecases"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

type PaymentHandler struct {
	createPaymentUC  *paymentUsecases.CreatePaymentUseCase
	handleCallbackUC *paymentUsecases.HandlePaymentCallbackUseCase
	logger           logger.Interface
}

func NewPaymentHandler(
	createPaymentUC *paymentUsecases.CreatePaymentUseCase,
	handleCallbackUC *paymentUsecases.HandlePaymentCallbackUseCase,
	logger logger.Interface,
) *PaymentHandler {
	return &PaymentHandler{
		createPaymentUC:  createPaymentUC,
		handleCallbackUC: handleCallbackUC,
		logger:           logger,
	}
}

type CreatePaymentRequest struct {
	SubscriptionID uint   `json:"subscription_id" binding:"required"`
	PaymentMethod  string `json:"payment_method" binding:"required,oneof=alipay wechat stripe"`
	ReturnURL      string `json:"return_url"`
}

type CreatePaymentResponse struct {
	PaymentID  uint   `json:"payment_id"`
	OrderNo    string `json:"order_no"`
	PaymentURL string `json:"payment_url"`
	QRCode     string `json:"qr_code,omitempty"`
	ExpiredAt  string `json:"expired_at"`
}

// @Summary		Create payment
// @Description	Create a payment for subscription
// @Tags			payments
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			payment	body		CreatePaymentRequest							true	"Payment data"
// @Success		200		{object}	utils.APIResponse{data=CreatePaymentResponse}	"Payment created successfully"
// @Failure		400		{object}	utils.APIResponse								"Bad request"
// @Failure		401		{object}	utils.APIResponse								"Unauthorized"
// @Failure		500		{object}	utils.APIResponse								"Internal server error"
// @Router			/payments [post]
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorw("failed to bind request", "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	cmd := paymentUsecases.CreatePaymentCommand{
		SubscriptionID: req.SubscriptionID,
		UserID:         userID.(uint),
		PaymentMethod:  req.PaymentMethod,
		ReturnURL:      req.ReturnURL,
	}

	result, err := h.createPaymentUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("failed to create payment", "error", err, "user_id", userID)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to create payment: "+err.Error())
		return
	}

	response := CreatePaymentResponse{
		PaymentID:  result.Payment.ID(),
		OrderNo:    result.Payment.OrderNo(),
		PaymentURL: result.PaymentURL,
		QRCode:     result.QRCode,
		ExpiredAt:  result.Payment.ExpiredAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	utils.SuccessResponse(c, http.StatusOK, "payment created successfully", response)
}

// @Summary		Handle payment callback
// @Description	Handle payment gateway callback notification
// @Tags			payments
// @Accept			json
// @Produce		json
// @Success		200	{object}	utils.APIResponse	"Callback processed successfully"
// @Failure		400	{object}	utils.APIResponse	"Bad request"
// @Failure		500	{object}	utils.APIResponse	"Internal server error"
// @Router			/payments/callback [post]
func (h *PaymentHandler) HandleCallback(c *gin.Context) {
	if err := h.handleCallbackUC.Execute(c.Request.Context(), c.Request); err != nil {
		h.logger.Errorw("failed to handle payment callback", "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, "failed to process callback: "+err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "callback processed successfully", nil)
}
