package dto

import (
	"time"

	"orris/internal/domain/node"
)

type NodeDTO struct {
	ID                uint                   `json:"id"`
	Name              string                 `json:"name"`
	ServerAddress     string                 `json:"server_address"`
	ServerPort        uint16                 `json:"server_port"`
	EncryptionMethod  string                 `json:"encryption_method"`
	Password          string                 `json:"password,omitempty"`
	Plugin            string                 `json:"plugin,omitempty"`
	PluginOpts        map[string]string      `json:"plugin_opts,omitempty"`
	Status            string                 `json:"status"`
	Country           string                 `json:"country,omitempty"`
	Region            string                 `json:"region,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	CustomFields      map[string]interface{} `json:"custom_fields,omitempty"`
	MaxUsers          uint                   `json:"max_users"`
	TrafficLimit      uint64                 `json:"traffic_limit"`
	TrafficUsed       uint64                 `json:"traffic_used"`
	TrafficResetAt    time.Time              `json:"traffic_reset_at"`
	SortOrder         int                    `json:"sort_order"`
	MaintenanceReason *string                `json:"maintenance_reason,omitempty"`
	IsAvailable       bool                   `json:"is_available"`
	Version           int                    `json:"version"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

type CreateNodeDTO struct {
	Name             string                 `json:"name" binding:"required,min=2,max=100"`
	ServerAddress    string                 `json:"server_address" binding:"required"`
	ServerPort       uint16                 `json:"server_port" binding:"required,min=1,max=65535"`
	EncryptionMethod string                 `json:"encryption_method" binding:"required"`
	Password         string                 `json:"password" binding:"required"`
	Plugin           string                 `json:"plugin,omitempty"`
	PluginOpts       map[string]string      `json:"plugin_opts,omitempty"`
	Country          string                 `json:"country,omitempty"`
	Region           string                 `json:"region,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	CustomFields     map[string]interface{} `json:"custom_fields,omitempty"`
	MaxUsers         uint                   `json:"max_users" binding:"required,min=1"`
	TrafficLimit     uint64                 `json:"traffic_limit" binding:"required,min=1"`
	SortOrder        int                    `json:"sort_order"`
}

type UpdateNodeDTO struct {
	Name             *string                `json:"name,omitempty" binding:"omitempty,min=2,max=100"`
	ServerAddress    *string                `json:"server_address,omitempty"`
	ServerPort       *uint16                `json:"server_port,omitempty" binding:"omitempty,min=1,max=65535"`
	EncryptionMethod *string                `json:"encryption_method,omitempty"`
	Password         *string                `json:"password,omitempty"`
	Plugin           *string                `json:"plugin,omitempty"`
	PluginOpts       map[string]string      `json:"plugin_opts,omitempty"`
	Country          *string                `json:"country,omitempty"`
	Region           *string                `json:"region,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	CustomFields     map[string]interface{} `json:"custom_fields,omitempty"`
	MaxUsers         *uint                  `json:"max_users,omitempty" binding:"omitempty,min=1"`
	TrafficLimit     *uint64                `json:"traffic_limit,omitempty" binding:"omitempty,min=1"`
	SortOrder        *int                   `json:"sort_order,omitempty"`
}

type NodeListDTO struct {
	Nodes      []*NodeDTO         `json:"nodes"`
	Pagination PaginationResponse `json:"pagination"`
}

type PaginationResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ListNodesRequest struct {
	Page     int      `json:"page" form:"page"`
	PageSize int      `json:"page_size" form:"page_size"`
	Status   string   `json:"status,omitempty" form:"status"`
	Country  string   `json:"country,omitempty" form:"country"`
	Region   string   `json:"region,omitempty" form:"region"`
	Tags     []string `json:"tags,omitempty" form:"tags"`
	OrderBy  string   `json:"order_by,omitempty" form:"order_by"`
	Order    string   `json:"order,omitempty" form:"order" binding:"omitempty,oneof=asc desc"`
}

type ActivateNodeRequest struct {
	NodeID uint `json:"node_id" binding:"required"`
}

type DeactivateNodeRequest struct {
	NodeID uint `json:"node_id" binding:"required"`
}

type MaintenanceNodeRequest struct {
	NodeID uint   `json:"node_id" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

func ToNodeDTO(n *node.Node) *NodeDTO {
	if n == nil {
		return nil
	}

	dto := &NodeDTO{
		ID:                n.ID(),
		Name:              n.Name(),
		ServerAddress:     n.ServerAddress().Value(),
		ServerPort:        n.ServerPort(),
		EncryptionMethod:  n.EncryptionConfig().Method(),
		Status:            n.Status().String(),
		MaxUsers:          n.MaxUsers(),
		TrafficLimit:      n.TrafficLimit(),
		TrafficUsed:       n.TrafficUsed(),
		TrafficResetAt:    n.TrafficResetAt(),
		SortOrder:         n.SortOrder(),
		MaintenanceReason: n.MaintenanceReason(),
		IsAvailable:       n.IsAvailable(),
		Version:           n.Version(),
		CreatedAt:         n.CreatedAt(),
		UpdatedAt:         n.UpdatedAt(),
	}

	if n.PluginConfig() != nil {
		dto.Plugin = n.PluginConfig().Plugin()
		dto.PluginOpts = n.PluginConfig().Opts()
	}

	metadata := n.Metadata()
	if metadata.Country() != "" {
		dto.Country = metadata.Country()
	}
	if metadata.Region() != "" {
		dto.Region = metadata.Region()
	}
	if len(metadata.Tags()) > 0 {
		dto.Tags = metadata.Tags()
	}

	return dto
}

func ToNodeDTOList(nodes []*node.Node) []*NodeDTO {
	if nodes == nil {
		return nil
	}

	dtos := make([]*NodeDTO, 0, len(nodes))
	for _, n := range nodes {
		if dto := ToNodeDTO(n); dto != nil {
			dtos = append(dtos, dto)
		}
	}

	return dtos
}
