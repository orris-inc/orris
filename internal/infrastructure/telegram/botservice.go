package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	gobreaker "github.com/sony/gobreaker/v2"

	sharedConfig "github.com/orris-inc/orris/internal/shared/config"
)

const (
	// maxRetryAfterSeconds caps the 429 retry_after wait to prevent excessive blocking
	maxRetryAfterSeconds = 30

	// maxNetworkRetries is the number of retries for transient network/decode errors
	maxNetworkRetries = 2
)

// allowedUpdates restricts the update types received from Telegram API.
// Only types we actually handle are listed to reduce unnecessary traffic.
var allowedUpdates = []string{"message", "callback_query"}

// BotService provides Telegram Bot API operations
type BotService struct {
	config         sharedConfig.TelegramConfig
	httpClient     *http.Client
	longPollClient *http.Client // Reusable client for long polling with extended timeout
	baseURL        string
	botUsername     string // Cached bot username from getMe
	cb             *gobreaker.CircuitBreaker[struct{}]
}

// NewBotService creates a new Telegram bot service
func NewBotService(config sharedConfig.TelegramConfig) *BotService {
	cb := gobreaker.NewCircuitBreaker[struct{}](gobreaker.Settings{
		Name:        "telegram-bot-api",
		MaxRequests: 1,                // half-open: allow 1 probe request
		Interval:    0,                // closed state does not auto-reset counts
		Timeout:     30 * time.Second, // open -> half-open wait time
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
		IsSuccessful: func(err error) bool {
			if err == nil {
				return true
			}
			// 429 (rate limit) counts as infrastructure failure to trip the breaker
			// when Telegram is persistently rate-limiting us.
			if IsRetryAfter(err) {
				return false
			}
			// Other API errors (400, 403) are "successful" from infrastructure perspective;
			// only network/timeout/decode errors count as failures.
			var apiErr *APIError
			return errors.As(err, &apiErr)
		},
	})

	s := &BotService{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		longPollClient: &http.Client{
			Timeout: 40 * time.Second, // pollTimeout(30s) + 10s buffer
		},
		baseURL: fmt.Sprintf("https://api.telegram.org/bot%s", config.BotToken),
		cb:      cb,
	}
	// Fetch and cache bot username on initialization
	if config.BotToken != "" {
		_ = s.fetchBotUsername()
	}
	return s
}

// SetWebhook sets the webhook URL for receiving updates
func (s *BotService) SetWebhook(webhookURL string) error {
	url := fmt.Sprintf("%s/setWebhook", s.baseURL)
	body := map[string]any{
		"url":             webhookURL,
		"allowed_updates": allowedUpdates,
	}
	// Include secret_token if configured for webhook verification
	if s.config.WebhookSecret != "" {
		body["secret_token"] = s.config.WebhookSecret
	}

	return s.makeRequest(url, body)
}

// DeleteWebhook removes the webhook
func (s *BotService) DeleteWebhook() error {
	url := fmt.Sprintf("%s/deleteWebhook", s.baseURL)
	return s.makeRequest(url, nil)
}

// BotCommand represents a bot command for the command menu
type BotCommand struct {
	Command     string `json:"command"`     // Command text without leading slash (e.g., "help")
	Description string `json:"description"` // Description of the command
}

// SetMyCommands sets the list of bot commands shown in the command menu
// This enables command auto-completion when users type "/"
func (s *BotService) SetMyCommands(commands []BotCommand) error {
	url := fmt.Sprintf("%s/setMyCommands", s.baseURL)
	body := map[string]any{
		"commands": commands,
	}
	return s.makeRequest(url, body)
}

// SetMyCommandsForAdmins sets commands visible only to admin users
// Uses BotCommandScopeChat to set commands for specific chat IDs
func (s *BotService) SetMyCommandsForChat(chatID int64, commands []BotCommand) error {
	url := fmt.Sprintf("%s/setMyCommands", s.baseURL)
	body := map[string]any{
		"commands": commands,
		"scope": map[string]any{
			"type":    "chat",
			"chat_id": chatID,
		},
	}
	return s.makeRequest(url, body)
}

// GetDefaultUserCommands returns the default command list for regular users
func GetDefaultUserCommands() []BotCommand {
	return []BotCommand{
		{Command: "bind", Description: "绑定账户 / Link account"},
		{Command: "status", Description: "查看状态 / View status"},
		{Command: "unbind", Description: "解绑账户 / Unlink account"},
		{Command: "help", Description: "显示帮助 / Show help"},
	}
}

// GetAdminCommands returns the command list including admin commands
func GetAdminCommands() []BotCommand {
	return []BotCommand{
		{Command: "bind", Description: "绑定账户 / Link account"},
		{Command: "status", Description: "查看状态 / View status"},
		{Command: "unbind", Description: "解绑账户 / Unlink account"},
		{Command: "adminbind", Description: "绑定管理员 / Link admin"},
		{Command: "adminstatus", Description: "管理员状态 / Admin status"},
		{Command: "adminunbind", Description: "解绑管理员 / Unlink admin"},
		{Command: "help", Description: "显示帮助 / Show help"},
	}
}

// GetUpdates retrieves updates using long polling
// offset: Identifier of the first update to be returned
// timeout: Timeout in seconds for long polling (0-60)
func (s *BotService) GetUpdates(offset int64, timeout int) ([]Update, error) {
	return s.GetUpdatesWithContext(context.Background(), offset, timeout)
}

// GetUpdatesWithContext retrieves updates using long polling with context support.
// The context can be used to cancel the long polling request for graceful shutdown.
func (s *BotService) GetUpdatesWithContext(ctx context.Context, offset int64, timeout int) ([]Update, error) {
	apiURL := fmt.Sprintf("%s/getUpdates", s.baseURL)

	body := map[string]any{
		"timeout":         timeout,
		"allowed_updates": allowedUpdates,
	}
	if offset > 0 {
		body["offset"] = offset
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.longPollClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var result getUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		apiErr := &APIError{
			ErrorCode:   result.ErrorCode,
			Description: result.Description,
		}
		if result.Parameters != nil {
			apiErr.RetryAfter = result.Parameters.RetryAfter
		}
		return nil, apiErr
	}

	return result.Result, nil
}

// SendMessage sends a plain text message to a chat (HTML format).
// Long messages are automatically split into multiple chunks.
func (s *BotService) SendMessage(chatID int64, text string) error {
	chunks := splitMessage(text, maxMessageLength)
	for _, chunk := range chunks {
		url := fmt.Sprintf("%s/sendMessage", s.baseURL)
		body := map[string]any{
			"chat_id":    chatID,
			"text":       chunk,
			"parse_mode": "HTML",
		}
		if err := s.makeRequest(url, body); err != nil {
			return err
		}
	}
	return nil
}

// SendMessagePlain sends a plain text message without any formatting.
// Long messages are automatically split into multiple chunks.
func (s *BotService) SendMessagePlain(chatID int64, text string) error {
	chunks := splitMessage(text, maxMessageLength)
	for _, chunk := range chunks {
		url := fmt.Sprintf("%s/sendMessage", s.baseURL)
		body := map[string]any{
			"chat_id": chatID,
			"text":    chunk,
		}
		if err := s.makeRequest(url, body); err != nil {
			return err
		}
	}
	return nil
}

// SendMessageMarkdown sends a markdown formatted message to a chat.
// Deprecated: Use SendMessage with HTML format instead.
// Long messages are automatically split into multiple chunks.
func (s *BotService) SendMessageMarkdown(chatID int64, text string) error {
	chunks := splitMessage(text, maxMessageLength)
	for _, chunk := range chunks {
		url := fmt.Sprintf("%s/sendMessage", s.baseURL)
		body := map[string]any{
			"chat_id":    chatID,
			"text":       chunk,
			"parse_mode": "Markdown",
		}
		if err := s.makeRequest(url, body); err != nil {
			return err
		}
	}
	return nil
}

// SendMessageWithKeyboard sends a message with a reply keyboard (HTML format)
func (s *BotService) SendMessageWithKeyboard(chatID int64, text string, keyboard any) error {
	url := fmt.Sprintf("%s/sendMessage", s.baseURL)
	body := map[string]any{
		"chat_id":      chatID,
		"text":         text,
		"parse_mode":   "HTML",
		"reply_markup": keyboard,
	}

	return s.makeRequest(url, body)
}

// SendMessageWithInlineKeyboard sends a message with an inline keyboard (HTML format)
func (s *BotService) SendMessageWithInlineKeyboard(chatID int64, text string, keyboard any) error {
	url := fmt.Sprintf("%s/sendMessage", s.baseURL)
	body := map[string]any{
		"chat_id":      chatID,
		"text":         text,
		"parse_mode":   "HTML",
		"reply_markup": keyboard,
	}

	return s.makeRequest(url, body)
}

// EditMessageText edits the text of a message (HTML format)
func (s *BotService) EditMessageText(chatID int64, messageID int64, text string) error {
	url := fmt.Sprintf("%s/editMessageText", s.baseURL)
	body := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
		"parse_mode": "HTML",
	}

	return s.makeRequest(url, body)
}

// EditMessageWithInlineKeyboard edits a message with an inline keyboard (HTML format)
func (s *BotService) EditMessageWithInlineKeyboard(chatID int64, messageID int64, text string, keyboard any) error {
	url := fmt.Sprintf("%s/editMessageText", s.baseURL)
	body := map[string]any{
		"chat_id":      chatID,
		"message_id":   messageID,
		"text":         text,
		"parse_mode":   "HTML",
		"reply_markup": keyboard,
	}

	return s.makeRequest(url, body)
}

// EditMessageReplyMarkup edits only the inline keyboard of a message
func (s *BotService) EditMessageReplyMarkup(chatID int64, messageID int64, keyboard any) error {
	url := fmt.Sprintf("%s/editMessageReplyMarkup", s.baseURL)
	body := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
	}
	if keyboard != nil {
		body["reply_markup"] = keyboard
	}

	return s.makeRequest(url, body)
}

// AnswerCallbackQuery answers a callback query from an inline keyboard
func (s *BotService) AnswerCallbackQuery(callbackQueryID string, text string, showAlert bool) error {
	url := fmt.Sprintf("%s/answerCallbackQuery", s.baseURL)
	body := map[string]any{
		"callback_query_id": callbackQueryID,
	}
	if text != "" {
		body["text"] = text
	}
	if showAlert {
		body["show_alert"] = true
	}

	return s.makeRequest(url, body)
}

// GetDefaultReplyKeyboard returns the default reply keyboard with common commands
func (s *BotService) GetDefaultReplyKeyboard() any {
	return &ReplyKeyboardMarkup{
		Keyboard: [][]KeyboardButton{
			{{Text: "/status"}, {Text: "/help"}},
			{{Text: "/unbind"}},
		},
		ResizeKeyboard: true,
	}
}

// KeyboardButton represents a button in a reply keyboard
type KeyboardButton struct {
	Text string `json:"text"`
}

// ReplyKeyboardMarkup represents a custom reply keyboard
type ReplyKeyboardMarkup struct {
	Keyboard        [][]KeyboardButton `json:"keyboard"`
	ResizeKeyboard  bool               `json:"resize_keyboard,omitempty"`
	OneTimeKeyboard bool               `json:"one_time_keyboard,omitempty"`
}

// InlineKeyboardButton represents a button in an inline keyboard
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

// InlineKeyboardMarkup represents an inline keyboard
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// NewInlineKeyboard creates a new inline keyboard with the given rows
func NewInlineKeyboard(rows ...[]InlineKeyboardButton) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

// NewInlineKeyboardRow creates a row of inline buttons
func NewInlineKeyboardRow(buttons ...InlineKeyboardButton) []InlineKeyboardButton {
	return buttons
}

// NewInlineKeyboardButton creates a callback button
func NewInlineKeyboardButton(text, callbackData string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         text,
		CallbackData: callbackData,
	}
}

// NewInlineKeyboardButtonURL creates a URL button
func NewInlineKeyboardButtonURL(text, url string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: text,
		URL:  url,
	}
}

// apiResponse represents a Telegram API response
type apiResponse struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code,omitempty"`
	Description string `json:"description,omitempty"`
	Parameters  *responseParameters `json:"parameters,omitempty"`
}

// responseParameters contains additional response parameters from Telegram API
type responseParameters struct {
	RetryAfter int `json:"retry_after,omitempty"`
}

// Update represents a Telegram update from getUpdates or webhook
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

// CallbackQuery represents a callback query from an inline keyboard
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

// Message represents a Telegram message
type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      *Chat  `json:"chat"`
	Date      int64  `json:"date"`
	Text      string `json:"text,omitempty"`
}

// User represents a Telegram user
type User struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

// Chat represents a Telegram chat
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// getUpdatesResponse represents the response from getUpdates API
type getUpdatesResponse struct {
	OK          bool                `json:"ok"`
	ErrorCode   int                 `json:"error_code,omitempty"`
	Result      []Update            `json:"result"`
	Description string              `json:"description,omitempty"`
	Parameters  *responseParameters `json:"parameters,omitempty"`
}

// getMeResponse represents the response from getMe API
type getMeResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		ID        int64  `json:"id"`
		IsBot     bool   `json:"is_bot"`
		FirstName string `json:"first_name"`
		Username  string `json:"username"`
	} `json:"result"`
	Description string `json:"description,omitempty"`
}

// fetchBotUsername fetches and caches the bot username from Telegram API
func (s *BotService) fetchBotUsername() error {
	url := fmt.Sprintf("%s/getMe", s.baseURL)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var result getMeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("telegram API error: %s", result.Description)
	}

	s.botUsername = result.Result.Username
	return nil
}

// GetBotUsername returns the cached bot username
func (s *BotService) GetBotUsername() string {
	return s.botUsername
}

// GetBotLink returns the Telegram bot link (https://t.me/username)
func (s *BotService) GetBotLink() string {
	if s.botUsername == "" {
		return ""
	}
	return fmt.Sprintf("https://t.me/%s", s.botUsername)
}

// doRequest performs a single HTTP request and returns a typed *APIError for API failures.
func (s *BotService) doRequest(apiURL string, body map[string]any) error {
	var req *http.Request
	var err error

	if body != nil {
		jsonBody, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal request body: %w", marshalErr)
		}
		req, err = http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(http.MethodPost, apiURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		apiErr := &APIError{
			ErrorCode:   result.ErrorCode,
			Description: result.Description,
		}
		if result.Parameters != nil {
			apiErr.RetryAfter = result.Parameters.RetryAfter
		}
		return apiErr
	}

	return nil
}

// SendChatAction sends a chat action (e.g., "typing") to a chat.
// This is a fire-and-forget operation that skips retry logic but still
// respects the circuit breaker to avoid requests when the API is down.
func (s *BotService) SendChatAction(chatID int64, action string) error {
	if s.cb.State() == gobreaker.StateOpen {
		return ErrCircuitOpen
	}
	url := fmt.Sprintf("%s/sendChatAction", s.baseURL)
	body := map[string]any{"chat_id": chatID, "action": action}
	return s.doRequest(url, body)
}

// makeRequest wraps makeRequestInternal with a circuit breaker.
// When the breaker is open, returns ErrCircuitOpen immediately.
func (s *BotService) makeRequest(apiURL string, body map[string]any) error {
	_, err := s.cb.Execute(func() (struct{}, error) {
		return struct{}{}, s.makeRequestInternal(apiURL, body)
	})
	if err != nil && (errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests)) {
		return ErrCircuitOpen
	}
	return err
}

// makeRequestInternal performs an HTTP request with retry logic:
//   - 429 Too Many Requests: wait retry_after seconds (capped at 30s), retry once
//   - Network/decode errors: exponential backoff (500ms, 1s), up to 2 retries
//   - 400/403 (non-retryable API errors): return immediately, no retry
func (s *BotService) makeRequestInternal(apiURL string, body map[string]any) error {
	err := s.doRequest(apiURL, body)
	if err == nil {
		return nil
	}

	// 429: wait and retry once
	if IsRetryAfter(err) {
		waitSec := GetRetryAfter(err)
		if waitSec > maxRetryAfterSeconds {
			waitSec = maxRetryAfterSeconds
		}
		if waitSec < 1 {
			waitSec = 1
		}
		time.Sleep(time.Duration(waitSec) * time.Second)
		return s.doRequest(apiURL, body)
	}

	// Non-retryable API errors (400, 403): return immediately
	if isNonRetryable(err) {
		return err
	}

	// Network/decode errors: exponential backoff retries
	backoff := 500 * time.Millisecond
	for i := 0; i < maxNetworkRetries; i++ {
		time.Sleep(backoff)
		backoff *= 2

		err = s.doRequest(apiURL, body)
		if err == nil {
			return nil
		}

		// If the retry returned a non-retryable API error, stop
		if isNonRetryable(err) || IsRetryAfter(err) {
			return err
		}
	}

	return err
}
