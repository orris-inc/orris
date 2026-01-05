package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	sharedConfig "github.com/orris-inc/orris/internal/shared/config"
)

// BotService provides Telegram Bot API operations
type BotService struct {
	config      sharedConfig.TelegramConfig
	httpClient  *http.Client
	baseURL     string
	botUsername string // Cached bot username from getMe
}

// NewBotService creates a new Telegram bot service
func NewBotService(config sharedConfig.TelegramConfig) *BotService {
	s := &BotService{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: fmt.Sprintf("https://api.telegram.org/bot%s", config.BotToken),
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
		"url": webhookURL,
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

// GetUpdates retrieves updates using long polling
// offset: Identifier of the first update to be returned
// timeout: Timeout in seconds for long polling (0-60)
func (s *BotService) GetUpdates(offset int64, timeout int) ([]Update, error) {
	apiURL := fmt.Sprintf("%s/getUpdates", s.baseURL)

	body := map[string]any{
		"timeout": timeout,
	}
	if offset > 0 {
		body["offset"] = offset
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create a client with extended timeout for long polling
	client := &http.Client{
		Timeout: time.Duration(timeout+10) * time.Second,
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var result getUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("telegram API error: %s", result.Description)
	}

	return result.Result, nil
}

// SendMessage sends a plain text message to a chat
func (s *BotService) SendMessage(chatID int64, text string) error {
	url := fmt.Sprintf("%s/sendMessage", s.baseURL)
	body := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}

	return s.makeRequest(url, body)
}

// SendMessageMarkdown sends a markdown formatted message to a chat
func (s *BotService) SendMessageMarkdown(chatID int64, text string) error {
	url := fmt.Sprintf("%s/sendMessage", s.baseURL)
	body := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	return s.makeRequest(url, body)
}

// SendMessageWithKeyboard sends a message with a reply keyboard
func (s *BotService) SendMessageWithKeyboard(chatID int64, text string, keyboard any) error {
	url := fmt.Sprintf("%s/sendMessage", s.baseURL)
	body := map[string]any{
		"chat_id":      chatID,
		"text":         text,
		"parse_mode":   "Markdown",
		"reply_markup": keyboard,
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

// apiResponse represents a Telegram API response
type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// Update represents a Telegram update from getUpdates or webhook
type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
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
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// Chat represents a Telegram chat
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// getUpdatesResponse represents the response from getUpdates API
type getUpdatesResponse struct {
	OK          bool     `json:"ok"`
	Result      []Update `json:"result"`
	Description string   `json:"description,omitempty"`
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

// GetBotLink returns the Telegram bot link (https://t.me/username)
func (s *BotService) GetBotLink() string {
	if s.botUsername == "" {
		return ""
	}
	return fmt.Sprintf("https://t.me/%s", s.botUsername)
}

func (s *BotService) makeRequest(url string, body map[string]any) error {
	var req *http.Request
	var err error

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		req, err = http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(http.MethodPost, url, nil)
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
		return fmt.Errorf("telegram API error: %s", result.Description)
	}

	return nil
}
