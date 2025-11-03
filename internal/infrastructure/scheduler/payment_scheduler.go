package scheduler

import (
	"context"
	"time"

	paymentUsecases "orris/internal/application/payment/usecases"
	"orris/internal/shared/logger"
)

type PaymentScheduler struct {
	expirePaymentsUC *paymentUsecases.ExpirePaymentsUseCase
	logger           logger.Interface
	stopChan         chan struct{}
	interval         time.Duration
}

func NewPaymentScheduler(
	expirePaymentsUC *paymentUsecases.ExpirePaymentsUseCase,
	logger logger.Interface,
) *PaymentScheduler {
	return &PaymentScheduler{
		expirePaymentsUC: expirePaymentsUC,
		logger:           logger,
		stopChan:         make(chan struct{}),
		interval:         5 * time.Minute,
	}
}

func (s *PaymentScheduler) Start(ctx context.Context) {
	s.logger.Infow("starting payment scheduler", "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Infow("payment scheduler stopped due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Infow("payment scheduler stopped")
			return
		case <-ticker.C:
			s.processExpiredPayments(ctx)
		}
	}
}

func (s *PaymentScheduler) Stop() {
	close(s.stopChan)
}

func (s *PaymentScheduler) processExpiredPayments(ctx context.Context) {
	s.logger.Debugw("processing expired payments task started")

	count, err := s.expirePaymentsUC.Execute(ctx)
	if err != nil {
		s.logger.Errorw("failed to process expired payments", "error", err)
		return
	}

	if count > 0 {
		s.logger.Infow("expired payments processed", "count", count)
	}
}
