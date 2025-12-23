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
	config     sharedConfig.TelegramConfig
	httpClient *http.Client
	baseURL    string
}

// NewBotService creates a new Telegram bot service
func NewBotService(config sharedConfig.TelegramConfig) *BotService {
	return &BotService{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: fmt.Sprintf("https://api.telegram.org/bot%s", config.BotToken),
	}
}

// SetWebhook sets the webhook URL for receiving updates
func (s *BotService) SetWebhook(webhookURL string) error {
	url := fmt.Sprintf("%s/setWebhook", s.baseURL)
	body := map[string]interface{}{
		"url": webhookURL,
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
	body := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}

	return s.makeRequest(url, body)
}

// SendMessageMarkdown sends a markdown formatted message to a chat
func (s *BotService) SendMessageMarkdown(chatID int64, text string) error {
	url := fmt.Sprintf("%s/sendMessage", s.baseURL)
	body := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	return s.makeRequest(url, body)
}

// apiResponse represents a Telegram API response
type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

func (s *BotService) makeRequest(url string, body map[string]interface{}) error {
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
