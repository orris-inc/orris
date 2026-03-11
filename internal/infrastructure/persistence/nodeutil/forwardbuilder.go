package nodeutil

import (
	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

// ForwardedNodeBuilder builds subscription nodes from forward rules.
type ForwardedNodeBuilder struct {
	agentMap map[uint]*models.ForwardAgentModel
	configs  ProtocolConfigs
}

// NewForwardedNodeBuilder creates a new ForwardedNodeBuilder.
func NewForwardedNodeBuilder(agentMap map[uint]*models.ForwardAgentModel, configs ProtocolConfigs) *ForwardedNodeBuilder {
	return &ForwardedNodeBuilder{
		agentMap: agentMap,
		configs:  configs,
	}
}

// ResolveRuleServerAddress determines the server address for a forward rule.
// For external rules, returns the rule's server address.
// For internal rules, returns the agent's public address.
// Returns empty string if address cannot be determined.
func (b *ForwardedNodeBuilder) ResolveRuleServerAddress(rule *forward.ForwardRule) string {
	if rule.IsExternal() {
		return rule.ServerAddress()
	}

	agent, ok := b.agentMap[rule.AgentID()]
	if !ok || agent.PublicAddress == "" {
		return ""
	}
	return agent.PublicAddress
}

// BuildFromNodeModel builds a forwarded node from a forward rule and target NodeModel.
func (b *ForwardedNodeBuilder) BuildFromNodeModel(rule *forward.ForwardRule, targetNode *models.NodeModel) *usecases.Node {
	if rule.TargetNodeID() == nil || targetNode == nil {
		return nil
	}

	serverAddress := b.ResolveRuleServerAddress(rule)
	if serverAddress == "" {
		return nil
	}

	source := NodeSource{
		ID:        targetNode.ID,
		Name:      rule.Name(),
		Address:   serverAddress,
		Port:      rule.ListenPort(),
		Protocol:  targetNode.Protocol,
		TokenHash: targetNode.TokenHash,
		SortOrder: rule.SortOrder(),
	}

	return BuildNode(source, b.configs)
}

// BuildFromUsecaseNode builds a forwarded node from a forward rule and target usecases.Node.
// This is used when the original node is already loaded as a usecases.Node.
func (b *ForwardedNodeBuilder) BuildFromUsecaseNode(rule *forward.ForwardRule, originalNode *usecases.Node) *usecases.Node {
	if rule.TargetNodeID() == nil || originalNode == nil {
		return nil
	}

	serverAddress := b.ResolveRuleServerAddress(rule)
	if serverAddress == "" {
		return nil
	}

	forwardedNode := &usecases.Node{
		ID:               originalNode.ID,
		Name:             rule.Name(),
		ServerAddress:    serverAddress,
		SubscriptionPort: rule.ListenPort(),
		Protocol:         originalNode.Protocol,
		TokenHash:        originalNode.TokenHash,
		Password:         originalNode.Password,
		SortOrder:        rule.SortOrder(),
	}

	// Copy protocol-specific fields from the original node
	CopyProtocolFieldsFromNode(forwardedNode, originalNode)

	return forwardedNode
}

// BuildForwardedNodesFromModels builds forwarded nodes from rules using NodeModel targets.
func (b *ForwardedNodeBuilder) BuildForwardedNodesFromModels(
	rules []*forward.ForwardRule,
	nodeMap map[uint]*models.NodeModel,
) []*usecases.Node {
	var nodes []*usecases.Node

	for _, rule := range rules {
		if rule.TargetNodeID() == nil {
			continue
		}

		targetNode, ok := nodeMap[*rule.TargetNodeID()]
		if !ok {
			continue
		}

		if node := b.BuildFromNodeModel(rule, targetNode); node != nil {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

// BuildForwardedNodesFromUsecaseNodes builds forwarded nodes from rules using usecases.Node targets.
func (b *ForwardedNodeBuilder) BuildForwardedNodesFromUsecaseNodes(
	rules []*forward.ForwardRule,
	nodeMap map[uint]*usecases.Node,
) []*usecases.Node {
	var nodes []*usecases.Node

	for _, rule := range rules {
		if rule.TargetNodeID() == nil {
			continue
		}

		originalNode, ok := nodeMap[*rule.TargetNodeID()]
		if !ok {
			continue
		}

		if node := b.BuildFromUsecaseNode(rule, originalNode); node != nil {
			nodes = append(nodes, node)
		}
	}

	return nodes
}
