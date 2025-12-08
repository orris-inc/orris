// Package dto provides data transfer objects for the forward domain.
package dto

import (
	"github.com/orris-inc/orris/internal/domain/forward"
)

// ForwardRuleDTO represents the data transfer object for forward rules.
type ForwardRuleDTO struct {
	ID            uint   `json:"id"`
	AgentID       uint   `json:"agent_id"`
	RuleType      string `json:"rule_type"`                // direct, entry, exit
	ExitAgentID   uint   `json:"exit_agent_id,omitempty"`  // for entry type
	WsListenPort  uint16 `json:"ws_listen_port,omitempty"` // for exit type
	Name          string `json:"name"`
	ListenPort    uint16 `json:"listen_port"`
	TargetAddress string `json:"target_address,omitempty"` // for direct and exit types
	TargetPort    uint16 `json:"target_port,omitempty"`    // for direct and exit types
	TargetNodeID  *uint  `json:"target_node_id,omitempty"` // for dynamic node address resolution
	IPVersion     string `json:"ip_version"`               // auto, ipv4, ipv6
	Protocol      string `json:"protocol"`
	Status        string `json:"status"`
	Remark        string `json:"remark"`
	UploadBytes   int64  `json:"upload_bytes"`
	DownloadBytes int64  `json:"download_bytes"`
	TotalBytes    int64  `json:"total_bytes"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`

	// Target node info (populated when targetNodeID is set)
	TargetNodeServerAddress string  `json:"target_node_server_address,omitempty"` // node's configured server address
	TargetNodePublicIPv4    *string `json:"target_node_public_ipv4,omitempty"`    // node's reported public IPv4
	TargetNodePublicIPv6    *string `json:"target_node_public_ipv6,omitempty"`    // node's reported public IPv6
}

// ToForwardRuleDTO converts a domain forward rule to DTO.
// Note: TargetNode* fields are NOT populated by this function.
// Use PopulateTargetNodeInfo to fill them after getting node data.
func ToForwardRuleDTO(rule *forward.ForwardRule) *ForwardRuleDTO {
	if rule == nil {
		return nil
	}

	return &ForwardRuleDTO{
		ID:            rule.ID(),
		AgentID:       rule.AgentID(),
		RuleType:      rule.RuleType().String(),
		ExitAgentID:   rule.ExitAgentID(),
		WsListenPort:  rule.WsListenPort(),
		Name:          rule.Name(),
		ListenPort:    rule.ListenPort(),
		TargetAddress: rule.TargetAddress(),
		TargetPort:    rule.TargetPort(),
		TargetNodeID:  rule.TargetNodeID(),
		IPVersion:     rule.IPVersion().String(),
		Protocol:      rule.Protocol().String(),
		Status:        rule.Status().String(),
		Remark:        rule.Remark(),
		UploadBytes:   rule.UploadBytes(),
		DownloadBytes: rule.DownloadBytes(),
		TotalBytes:    rule.TotalBytes(),
		CreatedAt:     rule.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     rule.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}

// TargetNodeInfo contains node information for target address resolution.
type TargetNodeInfo struct {
	ServerAddress string
	PublicIPv4    *string
	PublicIPv6    *string
}

// PopulateTargetNodeInfo fills in the target node info fields.
func (d *ForwardRuleDTO) PopulateTargetNodeInfo(info *TargetNodeInfo) {
	if info == nil {
		return
	}
	d.TargetNodeServerAddress = info.ServerAddress
	d.TargetNodePublicIPv4 = info.PublicIPv4
	d.TargetNodePublicIPv6 = info.PublicIPv6
}

// ToForwardRuleDTOs converts a slice of domain forward rules to DTOs.
func ToForwardRuleDTOs(rules []*forward.ForwardRule) []*ForwardRuleDTO {
	dtos := make([]*ForwardRuleDTO, len(rules))
	for i, rule := range rules {
		dtos[i] = ToForwardRuleDTO(rule)
	}
	return dtos
}
