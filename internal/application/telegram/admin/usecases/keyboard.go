package usecases

import "fmt"

// InlineKeyboardMarkup represents a Telegram inline keyboard
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton represents a button in an inline keyboard
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

// BuildMuteKeyboard builds an inline keyboard with mute button
// resourceType is "node" or "agent", resourceSID is the SID of the resource
func BuildMuteKeyboard(resourceType, resourceSID string) *InlineKeyboardMarkup {
	callbackData := fmt.Sprintf("mute:%s:%s", resourceType, resourceSID)
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{
					Text:         "🔕 静默此通知 / Mute",
					CallbackData: callbackData,
				},
			},
		},
	}
}
