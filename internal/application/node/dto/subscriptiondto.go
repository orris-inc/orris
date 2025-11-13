package dto

import (
	"time"
)

type SubscriptionResponseDTO struct {
	Content     string     `json:"content"`
	Format      string     `json:"format"`
	NodeCount   int        `json:"node_count"`
	GeneratedAt time.Time  `json:"generated_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	UserAgent   string     `json:"user_agent,omitempty"`
}

type GenerateSubscriptionRequestDTO struct {
	UserID uint   `json:"user_id" binding:"required"`
	Format string `json:"format" binding:"required,oneof=base64 clash surge quantumult quantumultx shadowrocket"`
}

func ToSubscriptionResponseDTO(content, format string, nodeCount int, userAgent string) *SubscriptionResponseDTO {
	return &SubscriptionResponseDTO{
		Content:     content,
		Format:      format,
		NodeCount:   nodeCount,
		GeneratedAt: time.Now(),
		UserAgent:   userAgent,
	}
}
