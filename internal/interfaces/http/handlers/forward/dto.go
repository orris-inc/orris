package forward

// CreateForwardRuleRequest represents a request to create a forward rule.
// Required fields by rule type:
// - direct: agent_id, listen_port, (target_address+target_port OR target_node_id)
// - entry: agent_id, exit_agent_id, listen_port, (target_address+target_port OR target_node_id)
// - chain: agent_id, chain_agent_ids, listen_port, (target_address+target_port OR target_node_id)
// - direct_chain: agent_id, chain_agent_ids, chain_port_config, (target_address+target_port OR target_node_id)
type CreateForwardRuleRequest struct {
	AgentID         string            `json:"agent_id" binding:"required" example:"fa_xK9mP2vL3nQ"`
	RuleType        string            `json:"rule_type" binding:"required,oneof=direct entry chain direct_chain" example:"direct"`
	ExitAgentID     string            `json:"exit_agent_id,omitempty" example:"fa_yL8nQ3wM4oR"`
	ChainAgentIDs   []string          `json:"chain_agent_ids,omitempty" example:"[\"fa_aaa\",\"fa_bbb\"]"`
	ChainPortConfig map[string]uint16 `json:"chain_port_config,omitempty" example:"{\"fa_xK9mP2vL3nQ\":8080,\"fa_yL8nQ3wM4oR\":9090}"`
	Name            string            `json:"name" binding:"required" example:"MySQL-Forward"`
	ListenPort      uint16            `json:"listen_port,omitempty" example:"13306"`
	TargetAddress   string            `json:"target_address,omitempty" example:"192.168.1.100"`
	TargetPort      uint16            `json:"target_port,omitempty" example:"3306"`
	TargetNodeID    string            `json:"target_node_id,omitempty" example:"node_xK9mP2vL3nQ"`
	BindIP          string            `json:"bind_ip,omitempty" example:"192.168.1.1"`
	Protocol        string            `json:"protocol" binding:"required,oneof=tcp udp both" example:"tcp"`
	Remark          string            `json:"remark,omitempty" example:"Forward to internal MySQL server"`
}

// UpdateForwardRuleRequest represents a request to update a forward rule.
type UpdateForwardRuleRequest struct {
	Name            *string           `json:"name,omitempty" example:"MySQL-Forward-Updated"`
	AgentID         *string           `json:"agent_id,omitempty" example:"fa_xK9mP2vL3nQ"`
	ExitAgentID     *string           `json:"exit_agent_id,omitempty" example:"fa_yL8nQ3wM4oR"`
	ChainAgentIDs   []string          `json:"chain_agent_ids,omitempty" example:"[\"fa_aaa\",\"fa_bbb\"]"`
	ChainPortConfig map[string]uint16 `json:"chain_port_config,omitempty" example:"{\"fa_xK9mP2vL3nQ\":8080,\"fa_yL8nQ3wM4oR\":9090}"`
	ListenPort      *uint16           `json:"listen_port,omitempty" example:"13307"`
	TargetAddress   *string           `json:"target_address,omitempty" example:"192.168.1.101"`
	TargetPort      *uint16           `json:"target_port,omitempty" example:"3307"`
	TargetNodeID    *string           `json:"target_node_id,omitempty" example:"node_xK9mP2vL3nQ"`
	BindIP          *string           `json:"bind_ip,omitempty" example:"192.168.1.1"`
	IPVersion       *string           `json:"ip_version,omitempty" binding:"omitempty,oneof=auto ipv4 ipv6" example:"auto"`
	Protocol        *string           `json:"protocol,omitempty" binding:"omitempty,oneof=tcp udp both" example:"tcp"`
	Remark          *string           `json:"remark,omitempty" example:"Updated remark"`
}

// UpdateStatusRequest represents a request to update forward rule status.
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=enabled disabled" example:"enabled"`
}

// ProbeRuleRequest represents the request body for probing a rule.
type ProbeRuleRequest struct {
	IPVersion string `json:"ip_version"` // optional: auto, ipv4, ipv6
}
