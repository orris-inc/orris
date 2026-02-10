package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/infrastructure/telegram/i18n"
	"github.com/orris-inc/orris/internal/shared/goroutine"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// defaultWorkerCount is the number of concurrent workers for processing updates.
	// Updates are dispatched to workers by user affinity (userID % workerCount)
	// to ensure same-user ordering while allowing cross-user concurrency.
	defaultWorkerCount = 4
)

var errMuteServiceNotConfigured = errors.New("mute service not configured")

// OffsetStore persists polling offset across restarts.
type OffsetStore interface {
	GetOffset(ctx context.Context) (int64, error)
	SaveOffset(ctx context.Context, offset int64) error
}

// UpdateHandler defines the interface for handling Telegram updates
type UpdateHandler interface {
	HandleUpdate(ctx context.Context, update *Update) error
}

// PollingService handles long polling for Telegram updates
type PollingService struct {
	botService         *BotService
	handler            UpdateHandler
	logger             logger.Interface
	offsetStore        OffsetStore // nil = in-memory only
	pollTimeout        int
	stopChan           chan struct{}
	cancelFunc         context.CancelFunc // Used to cancel ongoing HTTP requests during shutdown
	wg                 sync.WaitGroup
	lastUpdateID       int64
	processedWatermark int64 // highest update_id processed in this session (dedup safety net)
	workerCount        int
	isRunning          bool
	runningMu          sync.Mutex
}

// NewPollingService creates a new polling service.
// offsetStore is optional — pass nil for in-memory only (backward compatible).
func NewPollingService(
	botService *BotService,
	handler UpdateHandler,
	logger logger.Interface,
	offsetStore OffsetStore,
) *PollingService {
	return &PollingService{
		botService:  botService,
		handler:     handler,
		logger:      logger,
		offsetStore: offsetStore,
		pollTimeout: 30, // 30 seconds long polling timeout
		stopChan:    make(chan struct{}),
		workerCount: defaultWorkerCount,
	}
}

// Start begins polling for updates
func (s *PollingService) Start(ctx context.Context) error {
	s.runningMu.Lock()
	if s.isRunning {
		s.runningMu.Unlock()
		return nil
	}
	s.isRunning = true
	// Recreate stopChan for restart capability
	s.stopChan = make(chan struct{})
	// Create a cancellable context for HTTP requests
	pollCtx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.runningMu.Unlock()

	// Load persisted offset from store
	if s.offsetStore != nil {
		saved, err := s.offsetStore.GetOffset(ctx)
		if err != nil {
			s.logger.Warnw("failed to load polling offset, starting from 0", "error", err)
		} else if saved > 0 {
			s.lastUpdateID = saved
			s.processedWatermark = saved
			s.logger.Infow("loaded polling offset from store", "offset", saved)
		}
	}

	// Delete any existing webhook before starting polling
	if err := s.botService.DeleteWebhook(); err != nil {
		s.logger.Warnw("failed to delete webhook before polling", "error", err)
	}

	s.logger.Infow("starting telegram polling service",
		"timeout", s.pollTimeout,
		"workers", s.workerCount,
	)

	s.wg.Add(1)
	goroutine.SafeGo(s.logger, "telegram-poll-loop", func() {
		s.pollLoop(pollCtx)
	})

	return nil
}

// Stop stops the polling service
func (s *PollingService) Stop() {
	s.runningMu.Lock()
	if !s.isRunning {
		s.runningMu.Unlock()
		return
	}
	s.isRunning = false
	// Cancel ongoing HTTP requests first to unblock poll()
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	s.runningMu.Unlock()

	close(s.stopChan)
	s.wg.Wait()
	s.logger.Infow("telegram polling service stopped")
}

func (s *PollingService) pollLoop(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			s.logger.Infow("polling stopped due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Infow("polling stopped by stop signal")
			return
		default:
			s.poll(ctx)
		}
	}
}

func (s *PollingService) poll(ctx context.Context) {
	// Calculate offset: 0 for first poll (to get all pending updates), lastUpdateID+1 for subsequent polls
	offset := int64(0)
	if s.lastUpdateID > 0 {
		offset = s.lastUpdateID + 1
	}
	// Use context-aware GetUpdates for graceful shutdown support
	updates, err := s.botService.GetUpdatesWithContext(ctx, offset, s.pollTimeout)
	if err != nil {
		// Check if the error is due to context cancellation (graceful shutdown)
		if ctx.Err() != nil {
			return
		}
		s.logger.Errorw("failed to get updates", "error", err)
		// Wait a bit before retrying to avoid hammering the API on errors
		// Use select to respond to stop signals during wait
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-time.After(5 * time.Second):
			return
		}
	}

	if len(updates) == 0 {
		return
	}

	// Dedup: skip updates already processed (watermark safety net for restart overlap)
	filtered := updates[:0]
	for _, u := range updates {
		if u.UpdateID > s.processedWatermark {
			filtered = append(filtered, u)
		}
	}
	if len(filtered) == 0 {
		// Still advance lastUpdateID so Telegram won't resend these
		for _, u := range updates {
			if u.UpdateID > s.lastUpdateID {
				s.lastUpdateID = u.UpdateID
			}
		}
		return
	}

	// Dispatch updates to worker buckets by user affinity
	buckets := make([][]Update, s.workerCount)
	for i := range buckets {
		buckets[i] = make([]Update, 0)
	}
	var maxUpdateID int64
	for _, u := range filtered {
		idx := s.getUserAffinity(&u)
		buckets[idx] = append(buckets[idx], u)
		// Track max update ID (local var; commit to s.lastUpdateID after workers finish)
		if u.UpdateID > maxUpdateID {
			maxUpdateID = u.UpdateID
		}
	}

	// Process buckets concurrently
	var batchWg sync.WaitGroup
	for i, bucket := range buckets {
		if len(bucket) == 0 {
			continue
		}
		batchWg.Add(1)
		workerIdx := i
		workerBucket := bucket
		goroutine.SafeGo(s.logger, "telegram-worker-batch", func() {
			s.processWorkerBatch(ctx, &batchWg, workerIdx, workerBucket)
		})
	}
	batchWg.Wait()

	// Advance lastUpdateID and watermark only after all workers finished,
	// so a crash during processing won't skip unprocessed updates.
	s.lastUpdateID = maxUpdateID
	s.processedWatermark = maxUpdateID

	// Persist offset after processing batch.
	// Use a fresh context because the poll context may already be cancelled during shutdown.
	if s.offsetStore != nil && s.lastUpdateID > 0 {
		saveCtx, saveCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer saveCancel()
		if err := s.offsetStore.SaveOffset(saveCtx, s.lastUpdateID); err != nil {
			s.logger.Warnw("failed to save polling offset", "error", err)
		}
	}
}

// processWorkerBatch processes a slice of updates sequentially within one worker goroutine.
// Each goroutine has panic recovery to prevent a single update from crashing the entire service.
func (s *PollingService) processWorkerBatch(ctx context.Context, wg *sync.WaitGroup, workerIdx int, updates []Update) {
	defer wg.Done()

	for i := range updates {
		// Short-circuit remaining updates on shutdown to improve stop responsiveness
		if ctx.Err() != nil {
			return
		}

		func(u *Update) {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Errorw("panic recovered in update handler",
						"worker", workerIdx,
						"update_id", u.UpdateID,
						"panic", fmt.Sprintf("%v", r),
					)
				}
			}()

			if err := s.handler.HandleUpdate(ctx, u); err != nil {
				s.logger.Errorw("failed to handle update",
					"worker", workerIdx,
					"update_id", u.UpdateID,
					"error", err,
				)
			}
		}(&updates[i])
	}
}

// getUserAffinity maps an update to a worker index by user ID.
// Same user always goes to the same worker, preserving per-user ordering.
func (s *PollingService) getUserAffinity(u *Update) int {
	var userID int64
	switch {
	case u.CallbackQuery != nil && u.CallbackQuery.From != nil:
		userID = u.CallbackQuery.From.ID
	case u.Message != nil && u.Message.From != nil:
		userID = u.Message.From.ID
	default:
		// Fallback: spread by update ID
		userID = u.UpdateID
	}
	// Ensure non-negative modulo
	idx := int(userID % int64(s.workerCount))
	if idx < 0 {
		idx += s.workerCount
	}
	return idx
}

// TelegramServiceForPolling defines the interface for telegram service operations needed by polling
type TelegramServiceForPolling interface {
	BindFromWebhookForPolling(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error
	UnbindByTelegramID(ctx context.Context, telegramUserID int64) error
	IsBoundByTelegramID(ctx context.Context, telegramUserID int64) (bool, error)
	SendBotMessage(chatID int64, text string) error
	SendBotMessageWithKeyboard(chatID int64, text string) error
	SendBotChatAction(chatID int64, action string) error
	UpdateBindingLanguage(ctx context.Context, telegramUserID int64, language string) error
	UpdateAdminBindingLanguage(ctx context.Context, telegramUserID int64, language string) error
	// Admin binding
	AdminBindFromPolling(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error
	AdminUnbindByTelegramID(ctx context.Context, telegramUserID int64) error
	// Callback query handling
	IsAdminBound(ctx context.Context, telegramUserID int64) (bool, error)
	MuteAgentNotification(ctx context.Context, agentSID string) error
	MuteNodeNotification(ctx context.Context, nodeSID string) error
	UnmuteAgentNotification(ctx context.Context, agentSID string) error
	UnmuteNodeNotification(ctx context.Context, nodeSID string) error
	AnswerCallbackQuery(callbackQueryID string, text string, showAlert bool) error
	EditMessageWithInlineKeyboard(chatID int64, messageID int64, text string, keyboard any) error
	EditMessageReplyMarkup(chatID int64, messageID int64, keyboard any) error
}

// TelegramBinderService defines the interface for binding operations
type TelegramBinderService interface {
	UnbindByTelegramID(ctx context.Context, telegramUserID int64) error
}

// BotServiceGetter provides access to the current BotService instance
type BotServiceGetter interface {
	GetBotService() *BotService
}

// AdminBinderService defines the interface for admin binding operations
type AdminBinderService interface {
	BindFromWebhook(ctx context.Context, verifyCode string, telegramUserID int64, telegramUsername string) (any, error)
	UnbindByTelegramID(ctx context.Context, telegramUserID int64) error
	GetBindingByTelegramID(ctx context.Context, telegramUserID int64) (any, error)
}

// MuteNotificationService defines the interface for muting resource notifications
type MuteNotificationService interface {
	MuteAgentNotification(ctx context.Context, agentSID string) error
	MuteNodeNotification(ctx context.Context, nodeSID string) error
	UnmuteAgentNotification(ctx context.Context, agentSID string) error
	UnmuteNodeNotification(ctx context.Context, nodeSID string) error
}

// CallbackAnswerer defines the interface for answering Telegram callback queries
type CallbackAnswerer interface {
	AnswerCallbackQuery(callbackQueryID string, text string, showAlert bool) error
	EditMessageWithInlineKeyboard(chatID int64, messageID int64, text string, keyboard any) error
	EditMessageReplyMarkup(chatID int64, messageID int64, keyboard any) error
}

// ServiceAdapter wraps the telegram ServiceDDD to implement TelegramServiceForPolling
type ServiceAdapter struct {
	binder               TelegramBinderService
	adminBinder          AdminBinderService
	botServiceGetter     BotServiceGetter
	muteService          MuteNotificationService
	callbackAnswerer     CallbackAnswerer
	bindFunc             func(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error
	getBindingStatus     func(ctx context.Context, telegramUserID int64) (bool, error)
	updateLanguageFunc   func(ctx context.Context, telegramUserID int64, language string) error
	updateAdminLangFunc  func(ctx context.Context, telegramUserID int64, language string) error
}

// NewServiceAdapter creates a new service adapter from telegram ServiceDDD
func NewServiceAdapter(service interface {
	UnbindByTelegramID(ctx context.Context, telegramUserID int64) error
},
	bindFunc func(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error,
	getBindingStatus func(ctx context.Context, telegramUserID int64) (bool, error),
	updateLanguageFunc func(ctx context.Context, telegramUserID int64, language string) error,
	updateAdminLangFunc func(ctx context.Context, telegramUserID int64, language string) error,
) *ServiceAdapter {
	return &ServiceAdapter{
		binder:              service,
		bindFunc:            bindFunc,
		getBindingStatus:    getBindingStatus,
		updateLanguageFunc:  updateLanguageFunc,
		updateAdminLangFunc: updateAdminLangFunc,
	}
}

// SetBotServiceGetter sets the bot service getter (used to break circular dependency)
func (a *ServiceAdapter) SetBotServiceGetter(getter BotServiceGetter) {
	a.botServiceGetter = getter
}

// BindFromWebhookForPolling implements TelegramServiceForPolling
func (a *ServiceAdapter) BindFromWebhookForPolling(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error {
	return a.bindFunc(ctx, telegramUserID, telegramUsername, verifyCode)
}

// UnbindByTelegramID implements TelegramServiceForPolling
func (a *ServiceAdapter) UnbindByTelegramID(ctx context.Context, telegramUserID int64) error {
	return a.binder.UnbindByTelegramID(ctx, telegramUserID)
}

// IsBoundByTelegramID implements TelegramServiceForPolling
func (a *ServiceAdapter) IsBoundByTelegramID(ctx context.Context, telegramUserID int64) (bool, error) {
	return a.getBindingStatus(ctx, telegramUserID)
}

// SendBotMessage implements TelegramServiceForPolling
func (a *ServiceAdapter) SendBotMessage(chatID int64, text string) error {
	botService := a.botServiceGetter.GetBotService()
	if botService == nil {
		return nil
	}
	return botService.SendMessage(chatID, text)
}

// SendBotMessageWithKeyboard implements TelegramServiceForPolling
func (a *ServiceAdapter) SendBotMessageWithKeyboard(chatID int64, text string) error {
	botService := a.botServiceGetter.GetBotService()
	if botService == nil {
		return nil
	}
	keyboard := botService.GetDefaultReplyKeyboard()
	return botService.SendMessageWithKeyboard(chatID, text, keyboard)
}

// SendBotChatAction implements TelegramServiceForPolling
func (a *ServiceAdapter) SendBotChatAction(chatID int64, action string) error {
	botService := a.botServiceGetter.GetBotService()
	if botService == nil {
		return nil
	}
	return botService.SendChatAction(chatID, action)
}

// SetAdminBinder sets the admin binder service (used to break circular dependency)
func (a *ServiceAdapter) SetAdminBinder(binder AdminBinderService) {
	a.adminBinder = binder
}

// AdminBindFromPolling implements TelegramServiceForPolling for admin binding
func (a *ServiceAdapter) AdminBindFromPolling(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error {
	if a.adminBinder == nil {
		return nil
	}
	_, err := a.adminBinder.BindFromWebhook(ctx, verifyCode, telegramUserID, telegramUsername)
	return err
}

// AdminUnbindByTelegramID implements TelegramServiceForPolling for admin unbinding
func (a *ServiceAdapter) AdminUnbindByTelegramID(ctx context.Context, telegramUserID int64) error {
	if a.adminBinder == nil {
		return nil
	}
	return a.adminBinder.UnbindByTelegramID(ctx, telegramUserID)
}

// SetMuteService sets the mute notification service (used to break circular dependency)
func (a *ServiceAdapter) SetMuteService(muteService MuteNotificationService) {
	a.muteService = muteService
}

// SetCallbackAnswerer sets the callback answerer (used to break circular dependency)
func (a *ServiceAdapter) SetCallbackAnswerer(answerer CallbackAnswerer) {
	a.callbackAnswerer = answerer
}

// IsAdminBound implements TelegramServiceForPolling to check if a telegram user is a bound admin
func (a *ServiceAdapter) IsAdminBound(ctx context.Context, telegramUserID int64) (bool, error) {
	if a.adminBinder == nil {
		return false, nil
	}
	binding, err := a.adminBinder.GetBindingByTelegramID(ctx, telegramUserID)
	if err != nil {
		return false, nil // Treat error as not bound
	}
	return binding != nil, nil
}

// MuteAgentNotification implements TelegramServiceForPolling
func (a *ServiceAdapter) MuteAgentNotification(ctx context.Context, agentSID string) error {
	if a.muteService == nil {
		return errMuteServiceNotConfigured
	}
	return a.muteService.MuteAgentNotification(ctx, agentSID)
}

// MuteNodeNotification implements TelegramServiceForPolling
func (a *ServiceAdapter) MuteNodeNotification(ctx context.Context, nodeSID string) error {
	if a.muteService == nil {
		return errMuteServiceNotConfigured
	}
	return a.muteService.MuteNodeNotification(ctx, nodeSID)
}

// UnmuteAgentNotification implements TelegramServiceForPolling
func (a *ServiceAdapter) UnmuteAgentNotification(ctx context.Context, agentSID string) error {
	if a.muteService == nil {
		return errMuteServiceNotConfigured
	}
	return a.muteService.UnmuteAgentNotification(ctx, agentSID)
}

// UnmuteNodeNotification implements TelegramServiceForPolling
func (a *ServiceAdapter) UnmuteNodeNotification(ctx context.Context, nodeSID string) error {
	if a.muteService == nil {
		return errMuteServiceNotConfigured
	}
	return a.muteService.UnmuteNodeNotification(ctx, nodeSID)
}

// AnswerCallbackQuery implements TelegramServiceForPolling
func (a *ServiceAdapter) AnswerCallbackQuery(callbackQueryID string, text string, showAlert bool) error {
	if a.callbackAnswerer == nil {
		return nil
	}
	return a.callbackAnswerer.AnswerCallbackQuery(callbackQueryID, text, showAlert)
}

// EditMessageWithInlineKeyboard implements TelegramServiceForPolling
func (a *ServiceAdapter) EditMessageWithInlineKeyboard(chatID int64, messageID int64, text string, keyboard any) error {
	if a.callbackAnswerer == nil {
		return nil
	}
	return a.callbackAnswerer.EditMessageWithInlineKeyboard(chatID, messageID, text, keyboard)
}

// EditMessageReplyMarkup implements TelegramServiceForPolling
func (a *ServiceAdapter) EditMessageReplyMarkup(chatID int64, messageID int64, keyboard any) error {
	if a.callbackAnswerer == nil {
		return nil
	}
	return a.callbackAnswerer.EditMessageReplyMarkup(chatID, messageID, keyboard)
}

// UpdateBindingLanguage implements TelegramServiceForPolling
func (a *ServiceAdapter) UpdateBindingLanguage(ctx context.Context, telegramUserID int64, language string) error {
	if a.updateLanguageFunc == nil {
		return nil
	}
	return a.updateLanguageFunc(ctx, telegramUserID, language)
}

// UpdateAdminBindingLanguage implements TelegramServiceForPolling
func (a *ServiceAdapter) UpdateAdminBindingLanguage(ctx context.Context, telegramUserID int64, language string) error {
	if a.updateAdminLangFunc == nil {
		return nil
	}
	return a.updateAdminLangFunc(ctx, telegramUserID, language)
}

// PollingUpdateHandler implements UpdateHandler for the telegram service
type PollingUpdateHandler struct {
	service TelegramServiceForPolling
	logger  logger.Interface
}

// NewPollingUpdateHandler creates a new polling update handler
func NewPollingUpdateHandler(
	service TelegramServiceForPolling,
	logger logger.Interface,
) *PollingUpdateHandler {
	return &PollingUpdateHandler{
		service: service,
		logger:  logger,
	}
}

// HandleUpdate processes a single Telegram update
func (h *PollingUpdateHandler) HandleUpdate(ctx context.Context, update *Update) error {
	// Handle callback query from inline keyboard buttons
	if update.CallbackQuery != nil {
		return h.handleCallbackQuery(ctx, update.CallbackQuery)
	}

	if update.Message == nil || update.Message.From == nil {
		return nil
	}

	text := strings.TrimSpace(update.Message.Text)
	telegramUserID := update.Message.From.ID
	username := update.Message.From.Username
	langCode := update.Message.From.LanguageCode
	lang := i18n.DetectLang(langCode)

	switch {
	case strings.HasPrefix(text, "/start "):
		// Deep link: /start <payload>
		payload := strings.TrimSpace(strings.TrimPrefix(text, "/start "))
		return h.handleStartPayload(ctx, telegramUserID, username, lang, payload)
	case strings.HasPrefix(text, "/bind "):
		code := strings.TrimSpace(strings.TrimPrefix(text, "/bind "))
		return h.handleBindCommand(ctx, telegramUserID, username, lang, code)
	case text == "/unbind":
		return h.handleUnbindCommand(ctx, telegramUserID, lang)
	case text == "/status":
		return h.handleStatusCommand(ctx, telegramUserID, lang)
	case strings.HasPrefix(text, "/adminbind "):
		code := strings.TrimSpace(strings.TrimPrefix(text, "/adminbind "))
		return h.handleAdminBindCommand(ctx, telegramUserID, username, lang, code)
	case text == "/adminunbind":
		return h.handleAdminUnbindCommand(ctx, telegramUserID, lang)
	case text == "/adminstatus":
		return h.handleAdminStatusCommand(ctx, telegramUserID, lang)
	case text == "/start" || text == "/help":
		return h.handleHelpCommand(telegramUserID, lang)
	default:
		return h.handleHelpCommand(telegramUserID, lang)
	}
}

func (h *PollingUpdateHandler) handleStartPayload(ctx context.Context, telegramUserID int64, username string, lang i18n.Lang, payload string) error {
	// Reject overly long payloads to prevent abuse
	const maxPayloadLen = 128
	if len(payload) > maxPayloadLen {
		return h.handleHelpCommand(telegramUserID, lang)
	}

	switch {
	case strings.HasPrefix(payload, "bind_"):
		code := strings.TrimPrefix(payload, "bind_")
		return h.handleBindCommand(ctx, telegramUserID, username, lang, code)
	case strings.HasPrefix(payload, "adminbind_"):
		code := strings.TrimPrefix(payload, "adminbind_")
		return h.handleAdminBindCommand(ctx, telegramUserID, username, lang, code)
	default:
		return h.handleHelpCommand(telegramUserID, lang)
	}
}

func (h *PollingUpdateHandler) handleBindCommand(ctx context.Context, telegramUserID int64, username string, lang i18n.Lang, code string) error {
	if code == "" {
		return h.service.SendBotMessage(telegramUserID, i18n.MsgBindMissingCode(lang))
	}

	_ = h.service.SendBotChatAction(telegramUserID, "typing")
	err := h.service.BindFromWebhookForPolling(ctx, telegramUserID, username, code)
	if err != nil {
		h.logger.Errorw("failed to bind telegram from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return h.service.SendBotMessage(telegramUserID, i18n.MsgBindFailed(lang))
	}

	// Update language after successful binding
	if err := h.service.UpdateBindingLanguage(ctx, telegramUserID, string(lang)); err != nil {
		h.logger.Debugw("failed to update binding language", "telegram_user_id", telegramUserID, "error", err)
	}

	return h.service.SendBotMessageWithKeyboard(telegramUserID, i18n.MsgBindSuccess(lang))
}

func (h *PollingUpdateHandler) handleUnbindCommand(ctx context.Context, telegramUserID int64, lang i18n.Lang) error {
	err := h.service.UnbindByTelegramID(ctx, telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to unbind telegram from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return h.service.SendBotMessage(telegramUserID, i18n.MsgUnbindFailed(lang))
	}

	return h.service.SendBotMessage(telegramUserID, i18n.MsgUnbindSuccess(lang))
}

func (h *PollingUpdateHandler) handleStatusCommand(ctx context.Context, telegramUserID int64, lang i18n.Lang) error {
	_ = h.service.SendBotChatAction(telegramUserID, "typing")
	isBound, err := h.service.IsBoundByTelegramID(ctx, telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to get binding status from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return h.service.SendBotMessage(telegramUserID, i18n.MsgStatusError(lang))
	}

	if !isBound {
		return h.service.SendBotMessage(telegramUserID, i18n.MsgStatusNotConnected(lang))
	}

	// Update language if bound
	if err := h.service.UpdateBindingLanguage(ctx, telegramUserID, string(lang)); err != nil {
		h.logger.Debugw("failed to update binding language", "telegram_user_id", telegramUserID, "error", err)
	}

	// For bound status, just send a generic message since we don't have detailed info in polling mode
	return h.service.SendBotMessage(telegramUserID, i18n.MsgStatusConnectedSimple(lang))
}

func (h *PollingUpdateHandler) handleHelpCommand(telegramUserID int64, lang i18n.Lang) error {
	return h.service.SendBotMessageWithKeyboard(telegramUserID, i18n.MsgHelpFull(lang))
}

func (h *PollingUpdateHandler) handleAdminBindCommand(ctx context.Context, telegramUserID int64, username string, lang i18n.Lang, code string) error {
	if code == "" {
		return h.service.SendBotMessage(telegramUserID, i18n.MsgAdminBindMissingCode(lang))
	}

	err := h.service.AdminBindFromPolling(ctx, telegramUserID, username, code)
	if err != nil {
		h.logger.Errorw("failed to bind admin telegram from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return h.service.SendBotMessage(telegramUserID, i18n.MsgAdminBindFailedPolling(lang))
	}

	// Update language after successful binding
	if err := h.service.UpdateAdminBindingLanguage(ctx, telegramUserID, string(lang)); err != nil {
		h.logger.Debugw("failed to update admin binding language", "telegram_user_id", telegramUserID, "error", err)
	}

	return h.service.SendBotMessageWithKeyboard(telegramUserID, i18n.MsgAdminBindSuccessPolling(lang))
}

func (h *PollingUpdateHandler) handleAdminUnbindCommand(ctx context.Context, telegramUserID int64, lang i18n.Lang) error {
	err := h.service.AdminUnbindByTelegramID(ctx, telegramUserID)
	if err != nil {
		h.logger.Errorw("failed to unbind admin telegram from polling",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return h.service.SendBotMessage(telegramUserID, i18n.MsgAdminUnbindFailed(lang))
	}

	return h.service.SendBotMessage(telegramUserID, i18n.MsgAdminUnbindSuccess(lang))
}

func (h *PollingUpdateHandler) handleAdminStatusCommand(ctx context.Context, telegramUserID int64, lang i18n.Lang) error {
	isAdmin, err := h.service.IsAdminBound(ctx, telegramUserID)
	if err != nil || !isAdmin {
		return h.service.SendBotMessage(telegramUserID, i18n.MsgAdminStatusNotBound(lang))
	}

	// Update language if bound
	if err := h.service.UpdateAdminBindingLanguage(ctx, telegramUserID, string(lang)); err != nil {
		h.logger.Debugw("failed to update admin binding language", "telegram_user_id", telegramUserID, "error", err)
	}

	return h.service.SendBotMessage(telegramUserID, i18n.MsgAdminStatusBound(lang))
}

// handleCallbackQuery handles callback queries from inline keyboard buttons
func (h *PollingUpdateHandler) handleCallbackQuery(ctx context.Context, query *CallbackQuery) error {
	if query == nil || query.Data == "" {
		return nil
	}

	// Detect language from callback query user
	lang := i18n.ZH
	if query.From != nil && query.From.LanguageCode != "" {
		lang = i18n.DetectLang(query.From.LanguageCode)
	}

	// Parse callback data: format is "action:type:sid"
	// Example: "mute:agent:fa_xxx" or "mute:node:nd_xxx"
	parts := strings.SplitN(query.Data, ":", 3)
	if len(parts) != 3 {
		h.logger.Warnw("invalid callback data format", "data", query.Data)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackInvalidAction(lang), true)
		return nil
	}

	action := parts[0]
	resourceType := parts[1]
	resourceSID := parts[2]

	// Validate SID format (defense-in-depth)
	if _, _, err := id.ParsePrefixedID(resourceSID); err != nil {
		h.logger.Warnw("invalid resource SID in callback", "sid", resourceSID)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackInvalidAction(lang), true)
		return nil
	}

	switch action {
	case "mute":
		return h.handleMuteCallback(ctx, query, lang, resourceType, resourceSID)
	case "unmute":
		return h.handleUnmuteCallback(ctx, query, lang, resourceType, resourceSID)
	default:
		h.logger.Warnw("unknown callback action", "action", action)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackUnknownAction(lang), true)
		return nil
	}
}

// handleMuteCallback handles the mute notification callback
func (h *PollingUpdateHandler) handleMuteCallback(ctx context.Context, query *CallbackQuery, lang i18n.Lang, resourceType, resourceSID string) error {
	// Verify the user is a bound admin (security check)
	if query.From == nil {
		h.logger.Warnw("callback query missing from user")
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackInvalidRequest(lang), true)
		return nil
	}

	isAdmin, err := h.service.IsAdminBound(ctx, query.From.ID)
	if err != nil || !isAdmin {
		h.logger.Warnw("mute callback from non-admin user",
			"telegram_user_id", query.From.ID,
		)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackPermissionDenied(lang), true)
		return nil
	}

	// Execute mute operation
	var muteErr error
	switch resourceType {
	case "agent":
		muteErr = h.service.MuteAgentNotification(ctx, resourceSID)
	case "node":
		muteErr = h.service.MuteNodeNotification(ctx, resourceSID)
	default:
		h.logger.Warnw("unknown resource type for mute", "type", resourceType)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackUnknownResourceType(lang), true)
		return nil
	}

	if muteErr != nil {
		h.logger.Errorw("failed to mute notification",
			"resource_type", resourceType,
			"resource_sid", resourceSID,
			"error", muteErr,
		)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackOperationFailed(lang), true)
		return nil
	}

	// Answer callback with success message
	successMsg := i18n.MsgCallbackMuteSuccess(lang) + i18n.ResourceName(lang, resourceType)
	_ = h.service.AnswerCallbackQuery(query.ID, successMsg, false)

	// Update the button to show unmute option
	if query.Message != nil && query.Message.Chat != nil {
		chatID := query.Message.Chat.ID
		messageID := query.Message.MessageID
		unmuteKeyboard := i18n.BuildUnmuteKeyboard(lang, resourceType, resourceSID)
		if editErr := h.service.EditMessageReplyMarkup(chatID, messageID, unmuteKeyboard); editErr != nil {
			h.logger.Errorw("failed to update message reply markup after mute",
				"chat_id", chatID,
				"message_id", messageID,
				"error", editErr,
			)
		}
	}

	h.logger.Infow("notification muted via telegram callback (polling)",
		"resource_type", resourceType,
		"resource_sid", resourceSID,
		"telegram_user_id", query.From.ID,
	)

	return nil
}

// handleUnmuteCallback handles the unmute notification callback
func (h *PollingUpdateHandler) handleUnmuteCallback(ctx context.Context, query *CallbackQuery, lang i18n.Lang, resourceType, resourceSID string) error {
	// Verify the user is a bound admin (security check)
	if query.From == nil {
		h.logger.Warnw("callback query missing from user")
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackInvalidRequest(lang), true)
		return nil
	}

	isAdmin, err := h.service.IsAdminBound(ctx, query.From.ID)
	if err != nil || !isAdmin {
		h.logger.Warnw("unmute callback from non-admin user",
			"telegram_user_id", query.From.ID,
		)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackPermissionDenied(lang), true)
		return nil
	}

	// Execute unmute operation
	var unmuteErr error
	switch resourceType {
	case "agent":
		unmuteErr = h.service.UnmuteAgentNotification(ctx, resourceSID)
	case "node":
		unmuteErr = h.service.UnmuteNodeNotification(ctx, resourceSID)
	default:
		h.logger.Warnw("unknown resource type for unmute", "type", resourceType)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackUnknownResourceType(lang), true)
		return nil
	}

	if unmuteErr != nil {
		h.logger.Errorw("failed to unmute notification",
			"resource_type", resourceType,
			"resource_sid", resourceSID,
			"error", unmuteErr,
		)
		_ = h.service.AnswerCallbackQuery(query.ID, i18n.MsgCallbackOperationFailed(lang), true)
		return nil
	}

	// Answer callback with success message
	successMsg := i18n.MsgCallbackUnmuteSuccess(lang) + i18n.ResourceName(lang, resourceType)
	_ = h.service.AnswerCallbackQuery(query.ID, successMsg, false)

	// Update the button to show mute option again
	if query.Message != nil && query.Message.Chat != nil {
		chatID := query.Message.Chat.ID
		messageID := query.Message.MessageID
		muteKeyboard := i18n.BuildMuteKeyboard(lang, resourceType, resourceSID)
		if editErr := h.service.EditMessageReplyMarkup(chatID, messageID, muteKeyboard); editErr != nil {
			h.logger.Errorw("failed to update message reply markup after unmute",
				"chat_id", chatID,
				"message_id", messageID,
				"error", editErr,
			)
		}
	}

	h.logger.Infow("notification unmuted via telegram callback (polling)",
		"resource_type", resourceType,
		"resource_sid", resourceSID,
		"telegram_user_id", query.From.ID,
	)

	return nil
}

