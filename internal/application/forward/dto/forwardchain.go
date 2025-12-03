// Package dto provides data transfer objects for the forward domain.
package dto

import (
	"github.com/orris-inc/orris/internal/domain/forward"
)

// ChainNodeDTO represents a node in the forward chain.
type ChainNodeDTO struct {
	AgentID    uint   `json:"agent_id"`
	ListenPort uint16 `json:"listen_port"`
	Sequence   int    `json:"sequence"`
}

// ForwardChainDTO represents the data transfer object for forward chains.
type ForwardChainDTO struct {
	ID            uint           `json:"id"`
	Name          string         `json:"name"`
	Protocol      string         `json:"protocol"`
	Status        string         `json:"status"`
	Nodes         []ChainNodeDTO `json:"nodes"`
	NodeCount     int            `json:"node_count"`
	TargetAddress string         `json:"target_address"`
	TargetPort    uint16         `json:"target_port"`
	Remark        string         `json:"remark"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
}

// ToForwardChainDTO converts a domain forward chain to DTO.
func ToForwardChainDTO(chain *forward.ForwardChain) *ForwardChainDTO {
	if chain == nil {
		return nil
	}

	nodes := make([]ChainNodeDTO, len(chain.Nodes()))
	for i, node := range chain.Nodes() {
		nodes[i] = ChainNodeDTO{
			AgentID:    node.AgentID,
			ListenPort: node.ListenPort,
			Sequence:   node.Sequence,
		}
	}

	return &ForwardChainDTO{
		ID:            chain.ID(),
		Name:          chain.Name(),
		Protocol:      chain.Protocol().String(),
		Status:        chain.Status().String(),
		Nodes:         nodes,
		NodeCount:     chain.NodeCount(),
		TargetAddress: chain.TargetAddress(),
		TargetPort:    chain.TargetPort(),
		Remark:        chain.Remark(),
		CreatedAt:     chain.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     chain.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ToForwardChainDTOs converts a slice of domain forward chains to DTOs.
func ToForwardChainDTOs(chains []*forward.ForwardChain) []*ForwardChainDTO {
	dtos := make([]*ForwardChainDTO, len(chains))
	for i, chain := range chains {
		dtos[i] = ToForwardChainDTO(chain)
	}
	return dtos
}
