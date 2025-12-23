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
