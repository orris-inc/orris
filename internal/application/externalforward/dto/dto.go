// Package dto provides data transfer objects for the external forward application layer.
package dto

import (
	"github.com/orris-inc/orris/internal/domain/externalforward"
)

// ExternalForwardRuleDTO represents an external forward rule for API responses.
type ExternalForwardRuleDTO struct {
	ID             string `json:"id"`
	SubscriptionID string `json:"subscription_id,omitempty"`
	NodeID         string `json:"node_id,omitempty"`
	Name           string `json:"name"`
	ServerAddress  string `json:"server_address"`
	ListenPort     uint16 `json:"listen_port"`
	ExternalSource string `json:"external_source"`
	ExternalRuleID string `json:"external_rule_id,omitempty"`
	Status         string `json:"status"`
	SortOrder      int    `json:"sort_order"`
	Remark         string `json:"remark,omitempty"`
	GroupSIDs      []string `json:"group_ids,omitempty"`
	// Node details (populated when node is assigned)
	NodeServerAddress string `json:"node_server_address,omitempty"`
	NodePublicIPv4    string `json:"node_public_ipv4,omitempty"`
	NodePublicIPv6    string `json:"node_public_ipv6,omitempty"`
	NodeProtocol      string `json:"node_protocol,omitempty"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// NodeInfo contains node details for populating DTOs.
type NodeInfo struct {
	SID           string
	Name          string
	ServerAddress string
	PublicIPv4    string
	PublicIPv6    string
	Protocol      string
}

// FromDomain converts a domain entity to a DTO.
// nodeInfo contains the node details if the rule has a node assigned.
func FromDomain(rule *externalforward.ExternalForwardRule, subscriptionSID string, nodeInfo *NodeInfo) *ExternalForwardRuleDTO {
	if rule == nil {
		return nil
	}

	dto := &ExternalForwardRuleDTO{
		ID:             rule.SID(),
		SubscriptionID: subscriptionSID,
		Name:           rule.Name(),
		ServerAddress:  rule.ServerAddress(),
		ListenPort:     rule.ListenPort(),
		ExternalSource: rule.ExternalSource(),
		ExternalRuleID: rule.ExternalRuleID(),
		Status:         rule.Status().String(),
		SortOrder:      rule.SortOrder(),
		Remark:         rule.Remark(),
		CreatedAt:      rule.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      rule.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	if nodeInfo != nil {
		dto.NodeID = nodeInfo.SID
		dto.NodeServerAddress = nodeInfo.ServerAddress
		dto.NodePublicIPv4 = nodeInfo.PublicIPv4
		dto.NodePublicIPv6 = nodeInfo.PublicIPv6
		dto.NodeProtocol = nodeInfo.Protocol
	}

	return dto
}

// FromDomainList converts a list of domain entities to DTOs.
// Nil elements in the input slice are skipped to prevent nil elements in the output.
// nodeIDToInfo is an optional map to get node details by node IDs.
func FromDomainList(rules []*externalforward.ExternalForwardRule, subscriptionSID string, nodeIDToInfo map[uint]*NodeInfo) []*ExternalForwardRuleDTO {
	if rules == nil {
		return nil
	}

	dtos := make([]*ExternalForwardRuleDTO, 0, len(rules))
	for _, rule := range rules {
		var nodeInfo *NodeInfo
		if rule.NodeID() != nil && nodeIDToInfo != nil {
			nodeInfo = nodeIDToInfo[*rule.NodeID()]
		}
		if dto := FromDomain(rule, subscriptionSID, nodeInfo); dto != nil {
			dtos = append(dtos, dto)
		}
	}
	return dtos
}

// AdminExternalForwardRuleDTO represents an external forward rule for admin API responses.
// Note: Internal numeric IDs are not exposed for security reasons. Use SIDs for external references.
type AdminExternalForwardRuleDTO struct {
	ID             string   `json:"id"`
	SubscriptionSID string  `json:"subscription_id,omitempty"` // Stripe-style SID (sub_xxx), not internal ID
	UserSID        string   `json:"user_id,omitempty"`         // Stripe-style SID (usr_xxx), not internal ID
	NodeSID        string   `json:"node_id,omitempty"`         // Stripe-style SID (node_xxx), not internal ID
	Name           string   `json:"name"`
	ServerAddress  string   `json:"server_address"`
	ListenPort     uint16   `json:"listen_port"`
	ExternalSource string   `json:"external_source"`
	ExternalRuleID string   `json:"external_rule_id,omitempty"`
	Status         string   `json:"status"`
	SortOrder      int      `json:"sort_order"`
	Remark         string   `json:"remark,omitempty"`
	GroupSIDs      []string `json:"group_ids,omitempty"`
	// Node details (populated when node is assigned)
	NodeName          string `json:"node_name,omitempty"`
	NodeServerAddress string `json:"node_server_address,omitempty"`
	NodePublicIPv4    string `json:"node_public_ipv4,omitempty"`
	NodePublicIPv6    string `json:"node_public_ipv6,omitempty"`
	NodeProtocol      string `json:"node_protocol,omitempty"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// AdminDTOLookups contains optional lookup maps for converting internal IDs to SIDs.
type AdminDTOLookups struct {
	GroupIDToSID        map[uint]string   // resource group ID -> SID
	NodeIDToInfo        map[uint]*NodeInfo // node ID -> NodeInfo
	SubscriptionIDToSID map[uint]string   // subscription ID -> SID
	UserIDToSID         map[uint]string   // user ID -> SID
}

// FromDomainToAdmin converts a domain entity to an admin DTO.
// lookups contains optional maps for converting internal IDs to SIDs and fetching related info.
func FromDomainToAdmin(rule *externalforward.ExternalForwardRule, lookups *AdminDTOLookups) *AdminExternalForwardRuleDTO {
	if rule == nil {
		return nil
	}

	if lookups == nil {
		lookups = &AdminDTOLookups{}
	}

	dto := &AdminExternalForwardRuleDTO{
		ID:             rule.SID(),
		Name:           rule.Name(),
		ServerAddress:  rule.ServerAddress(),
		ListenPort:     rule.ListenPort(),
		ExternalSource: rule.ExternalSource(),
		ExternalRuleID: rule.ExternalRuleID(),
		Status:         rule.Status().String(),
		SortOrder:      rule.SortOrder(),
		Remark:         rule.Remark(),
		CreatedAt:      rule.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      rule.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	// Convert subscription ID to SID
	if rule.SubscriptionID() != nil && lookups.SubscriptionIDToSID != nil {
		if sid, ok := lookups.SubscriptionIDToSID[*rule.SubscriptionID()]; ok && sid != "" {
			dto.SubscriptionSID = sid
		}
	}

	// Convert user ID to SID
	if rule.UserID() != nil && lookups.UserIDToSID != nil {
		if sid, ok := lookups.UserIDToSID[*rule.UserID()]; ok && sid != "" {
			dto.UserSID = sid
		}
	}

	// Populate node details
	if rule.NodeID() != nil && lookups.NodeIDToInfo != nil {
		if info, ok := lookups.NodeIDToInfo[*rule.NodeID()]; ok && info != nil {
			dto.NodeSID = info.SID
			dto.NodeName = info.Name
			dto.NodeServerAddress = info.ServerAddress
			dto.NodePublicIPv4 = info.PublicIPv4
			dto.NodePublicIPv6 = info.PublicIPv6
			dto.NodeProtocol = info.Protocol
		}
	}

	// Convert group IDs to SIDs
	if len(rule.GroupIDs()) > 0 && lookups.GroupIDToSID != nil {
		groupSIDs := make([]string, 0, len(rule.GroupIDs()))
		for _, gid := range rule.GroupIDs() {
			if sid, ok := lookups.GroupIDToSID[gid]; ok && sid != "" {
				groupSIDs = append(groupSIDs, sid)
			}
		}
		dto.GroupSIDs = groupSIDs
	}

	return dto
}

// FromDomainListToAdmin converts a list of domain entities to admin DTOs.
// Nil elements in the input slice are skipped to prevent nil elements in the output.
// lookups contains optional maps for converting internal IDs to SIDs and fetching related info.
func FromDomainListToAdmin(rules []*externalforward.ExternalForwardRule, lookups *AdminDTOLookups) []*AdminExternalForwardRuleDTO {
	if rules == nil {
		return nil
	}

	dtos := make([]*AdminExternalForwardRuleDTO, 0, len(rules))
	for _, rule := range rules {
		if dto := FromDomainToAdmin(rule, lookups); dto != nil {
			dtos = append(dtos, dto)
		}
	}
	return dtos
}
