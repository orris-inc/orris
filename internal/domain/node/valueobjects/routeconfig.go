package valueobjects

import "fmt"

// RouteConfig defines the routing configuration for a node.
// It specifies how traffic should be routed based on matching rules.
// Compatible with sing-box route configuration.
type RouteConfig struct {
	rules       []RouteRule  // Ordered list of routing rules
	finalAction OutboundType // Default action when no rules match
}

// NewRouteConfig creates a new route configuration
func NewRouteConfig(finalAction OutboundType) (*RouteConfig, error) {
	if !finalAction.IsValid() {
		return nil, fmt.Errorf("invalid final action: %s", finalAction)
	}
	return &RouteConfig{
		rules:       make([]RouteRule, 0),
		finalAction: finalAction,
	}, nil
}

// Rules returns a copy of the routing rules to prevent external modification
func (c *RouteConfig) Rules() []RouteRule {
	if c.rules == nil {
		return nil
	}
	result := make([]RouteRule, len(c.rules))
	copy(result, c.rules)
	return result
}

// FinalAction returns the default action when no rules match
func (c *RouteConfig) FinalAction() OutboundType {
	return c.finalAction
}

// AddRule adds a routing rule to the configuration
func (c *RouteConfig) AddRule(rule RouteRule) error {
	if err := rule.Validate(); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}
	c.rules = append(c.rules, rule)
	return nil
}

// SetRules replaces all routing rules
func (c *RouteConfig) SetRules(rules []RouteRule) error {
	for i, rule := range rules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("invalid rule at index %d: %w", i, err)
		}
	}
	c.rules = rules
	return nil
}

// SetFinalAction sets the default action
func (c *RouteConfig) SetFinalAction(action OutboundType) error {
	if !action.IsValid() {
		return fmt.Errorf("invalid final action: %s", action)
	}
	c.finalAction = action
	return nil
}

// Validate validates the route configuration
func (c *RouteConfig) Validate() error {
	if !c.finalAction.IsValid() {
		return fmt.Errorf("invalid final action: %s", c.finalAction)
	}
	for i, rule := range c.rules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("invalid rule at index %d: %w", i, err)
		}
	}
	return nil
}

// IsEmpty checks if the route config has no rules
func (c *RouteConfig) IsEmpty() bool {
	return len(c.rules) == 0
}

// RuleCount returns the number of rules
func (c *RouteConfig) RuleCount() int {
	return len(c.rules)
}

// Equals compares two route configurations for equality
func (c *RouteConfig) Equals(other *RouteConfig) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	if c.finalAction != other.finalAction {
		return false
	}
	if len(c.rules) != len(other.rules) {
		return false
	}
	for i, rule := range c.rules {
		if !rule.Equals(&other.rules[i]) {
			return false
		}
	}
	return true
}

// GetReferencedNodeSIDs returns all unique node SIDs referenced in outbound rules.
// This includes both rule outbounds and finalAction if they reference nodes.
func (c *RouteConfig) GetReferencedNodeSIDs() []string {
	if c == nil {
		return nil
	}

	seen := make(map[string]bool)
	var sids []string

	// Check rules
	for _, rule := range c.rules {
		if rule.outbound.IsNodeReference() {
			sid := rule.outbound.NodeSID()
			if !seen[sid] {
				sids = append(sids, sid)
				seen[sid] = true
			}
		}
	}

	// Check finalAction
	if c.finalAction.IsNodeReference() {
		sid := c.finalAction.NodeSID()
		if !seen[sid] {
			sids = append(sids, sid)
		}
	}

	return sids
}

// HasNodeReferences checks if the route config references any nodes
func (c *RouteConfig) HasNodeReferences() bool {
	if c == nil {
		return false
	}

	for _, rule := range c.rules {
		if rule.outbound.IsNodeReference() {
			return true
		}
	}

	return c.finalAction.IsNodeReference()
}

// ReconstructRouteConfig reconstructs a RouteConfig from persistence data
func ReconstructRouteConfig(rules []RouteRule, finalAction OutboundType) *RouteConfig {
	return &RouteConfig{
		rules:       rules,
		finalAction: finalAction,
	}
}

// Preset route configurations

// mustNewRouteRule creates a new RouteRule and panics if the outbound type is invalid.
// This is safe to use with known-valid OutboundType constants.
func mustNewRouteRule(outbound OutboundType) *RouteRule {
	rule, err := NewRouteRule(outbound)
	if err != nil {
		panic(fmt.Sprintf("mustNewRouteRule: %v", err))
	}
	return rule
}

// NewCNDirectRouteConfig creates a "China Direct" route config:
// - China IPs and domains go direct
// - Private IPs go direct
// - Everything else goes through proxy
func NewCNDirectRouteConfig() *RouteConfig {
	config := &RouteConfig{
		rules:       make([]RouteRule, 0, 3),
		finalAction: OutboundProxy,
	}

	// Rule 1: Private IPs direct
	privateRule := mustNewRouteRule(OutboundDirect)
	privateRule.WithIPIsPrivate(true)
	config.rules = append(config.rules, *privateRule)

	// Rule 2: China GeoIP direct
	geoIPRule := mustNewRouteRule(OutboundDirect)
	geoIPRule.WithGeoIP("cn")
	config.rules = append(config.rules, *geoIPRule)

	// Rule 3: China GeoSite direct
	geoSiteRule := mustNewRouteRule(OutboundDirect)
	geoSiteRule.WithGeoSite("cn")
	config.rules = append(config.rules, *geoSiteRule)

	return config
}

// NewGlobalProxyRouteConfig creates a "Global Proxy" route config:
// - Private IPs go direct
// - Everything else goes through proxy
func NewGlobalProxyRouteConfig() *RouteConfig {
	config := &RouteConfig{
		rules:       make([]RouteRule, 0, 1),
		finalAction: OutboundProxy,
	}

	// Rule: Private IPs direct
	privateRule := mustNewRouteRule(OutboundDirect)
	privateRule.WithIPIsPrivate(true)
	config.rules = append(config.rules, *privateRule)

	return config
}

// NewWhitelistRouteConfig creates a "Whitelist" route config:
// - Only specified categories go through proxy
// - Everything else goes direct
func NewWhitelistRouteConfig(proxyCategories ...string) *RouteConfig {
	config := &RouteConfig{
		rules:       make([]RouteRule, 0, 1),
		finalAction: OutboundDirect,
	}

	if len(proxyCategories) > 0 {
		proxyRule := mustNewRouteRule(OutboundProxy)
		proxyRule.WithGeoSite(proxyCategories...)
		config.rules = append(config.rules, *proxyRule)
	}

	return config
}

// NewBlockAdsRouteConfig creates a route config that blocks ads:
// - Ad domains are blocked
// - Everything else uses specified default action
// Returns error if defaultAction is invalid
func NewBlockAdsRouteConfig(defaultAction OutboundType) (*RouteConfig, error) {
	if !defaultAction.IsValid() {
		return nil, fmt.Errorf("invalid default action: %s", defaultAction)
	}

	config := &RouteConfig{
		rules:       make([]RouteRule, 0, 1),
		finalAction: defaultAction,
	}

	// Block ad categories
	blockRule := mustNewRouteRule(OutboundBlock)
	blockRule.WithGeoSite("category-ads", "category-ads-all")
	config.rules = append(config.rules, *blockRule)

	return config, nil
}
