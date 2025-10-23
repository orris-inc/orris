package dto

import (
	"time"
)

type SubscriptionResponseDTO struct {
	Content     string    `json:"content"`
	Format      string    `json:"format"`
	NodeCount   int       `json:"node_count"`
	GeneratedAt time.Time `json:"generated_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	UserAgent   string    `json:"user_agent,omitempty"`
}

type GenerateSubscriptionRequestDTO struct {
	UserID uint   `json:"user_id" binding:"required"`
	Format string `json:"format" binding:"required,oneof=base64 clash surge quantumult quantumultx shadowrocket"`
}

type SubscriptionConfigDTO struct {
	Format        string   `json:"format"`
	IncludeNodes  []uint   `json:"include_nodes,omitempty"`
	ExcludeNodes  []uint   `json:"exclude_nodes,omitempty"`
	IncludeGroups []uint   `json:"include_groups,omitempty"`
	ExcludeGroups []uint   `json:"exclude_groups,omitempty"`
	Countries     []string `json:"countries,omitempty"`
	Regions       []string `json:"regions,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	MaxNodes      int      `json:"max_nodes,omitempty"`
	SortBy        string   `json:"sort_by,omitempty" binding:"omitempty,oneof=name speed latency traffic"`
	SortOrder     string   `json:"sort_order,omitempty" binding:"omitempty,oneof=asc desc"`
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

type ClashConfigDTO struct {
	Port               int                         `json:"port" yaml:"port"`
	SocksPort          int                         `json:"socks-port" yaml:"socks-port"`
	AllowLan           bool                        `json:"allow-lan" yaml:"allow-lan"`
	Mode               string                      `json:"mode" yaml:"mode"`
	LogLevel           string                      `json:"log-level" yaml:"log-level"`
	ExternalController string                      `json:"external-controller" yaml:"external-controller"`
	Proxies            []map[string]interface{}    `json:"proxies" yaml:"proxies"`
	ProxyGroups        []map[string]interface{}    `json:"proxy-groups" yaml:"proxy-groups"`
	Rules              []string                    `json:"rules" yaml:"rules"`
}

type SurgeConfigDTO struct {
	General  map[string]interface{}   `json:"general"`
	Replica  map[string]interface{}   `json:"replica"`
	Proxies  []map[string]interface{} `json:"proxies"`
	Groups   []map[string]interface{} `json:"groups"`
	Rules    []string                 `json:"rules"`
}

type SubscriptionStatsDTO struct {
	TotalSubscriptions   int                `json:"total_subscriptions"`
	ActiveSubscriptions  int                `json:"active_subscriptions"`
	SubscriptionsByFormat map[string]int    `json:"subscriptions_by_format"`
	SubscriptionsByUser   map[uint]int      `json:"subscriptions_by_user"`
	LastGenerated        *time.Time         `json:"last_generated,omitempty"`
	MostPopularFormat    string             `json:"most_popular_format"`
	AverageNodesPerSub   float64            `json:"average_nodes_per_sub"`
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
