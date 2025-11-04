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

type SubscriptionNodeDTO struct {
	Name             string            `json:"name"`
	ServerAddress    string            `json:"server_address"`
	ServerPort       uint16            `json:"server_port"`
	EncryptionMethod string            `json:"encryption_method"`
	Password         string            `json:"password"`
	Plugin           string            `json:"plugin,omitempty"`
	PluginOpts       map[string]string `json:"plugin_opts,omitempty"`
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

func ToSubscriptionNodeDTO(node *NodeDTO) *SubscriptionNodeDTO {
	if node == nil {
		return nil
	}

	return &SubscriptionNodeDTO{
		Name:             node.Name,
		ServerAddress:    node.ServerAddress,
		ServerPort:       node.ServerPort,
		EncryptionMethod: node.EncryptionMethod,
		Password:         node.Password,
		Plugin:           node.Plugin,
		PluginOpts:       node.PluginOpts,
	}
}

func ToSubscriptionNodeDTOList(nodes []*NodeDTO) []*SubscriptionNodeDTO {
	if nodes == nil {
		return nil
	}

	dtos := make([]*SubscriptionNodeDTO, 0, len(nodes))
	for _, node := range nodes {
		if dto := ToSubscriptionNodeDTO(node); dto != nil {
			dtos = append(dtos, dto)
		}
	}

	return dtos
}
