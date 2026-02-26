package valueobjects

import "fmt"

// RouteConfig defines the routing configuration for a node.
// It specifies how traffic should be routed based on matching rules.
// Compatible with sing-box route configuration.
type RouteConfig struct {
	rules            []RouteRule      // Ordered list of routing rules
	finalAction      OutboundType     // Default action when no rules match
	customOutbounds  []CustomOutbound // User-defined outbound configurations referenced by route rules via custom_xxx tags
	ruleSetEntries   []RuleSetEntry   // Remote rule-set sources referenced by rules via rule_set tags
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

// CustomOutbounds returns a copy of the custom outbounds
func (c *RouteConfig) CustomOutbounds() []CustomOutbound {
	if c.customOutbounds == nil {
		return nil
	}
	result := make([]CustomOutbound, len(c.customOutbounds))
	copy(result, c.customOutbounds)
	return result
}

// maxCustomOutbounds is the maximum number of custom outbounds per route config
const maxCustomOutbounds = 20

// SetCustomOutbounds replaces all custom outbounds after validation.
// Validates each outbound and ensures tag uniqueness.
func (c *RouteConfig) SetCustomOutbounds(outbounds []CustomOutbound) error {
	if len(outbounds) > maxCustomOutbounds {
		return fmt.Errorf("too many custom outbounds: %d (max %d)", len(outbounds), maxCustomOutbounds)
	}
	// Validate each outbound and check tag uniqueness
	seen := make(map[string]bool, len(outbounds))
	for i, co := range outbounds {
		if err := co.Validate(); err != nil {
			return fmt.Errorf("invalid custom outbound at index %d: %w", i, err)
		}
		if seen[co.Tag()] {
			return fmt.Errorf("duplicate custom outbound tag: %s", co.Tag())
		}
		seen[co.Tag()] = true
	}
	// Defensive copy to prevent caller from modifying internal state
	cp := make([]CustomOutbound, len(outbounds))
	copy(cp, outbounds)
	c.customOutbounds = cp
	return nil
}

// HasCustomOutbounds checks if the route config has custom outbounds
func (c *RouteConfig) HasCustomOutbounds() bool {
	return len(c.customOutbounds) > 0
}

// RuleSetEntries returns a copy of the rule-set entries
func (c *RouteConfig) RuleSetEntries() []RuleSetEntry {
	if c.ruleSetEntries == nil {
		return nil
	}
	result := make([]RuleSetEntry, len(c.ruleSetEntries))
	copy(result, c.ruleSetEntries)
	return result
}

// SetRuleSetEntries replaces all rule-set entries after validation.
// Validates each entry and ensures tag uniqueness.
func (c *RouteConfig) SetRuleSetEntries(entries []RuleSetEntry) error {
	if len(entries) > maxRuleSetEntries {
		return fmt.Errorf("too many rule-set entries: %d (max %d)", len(entries), maxRuleSetEntries)
	}
	seen := make(map[string]bool, len(entries))
	for i, e := range entries {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("invalid rule-set entry at index %d: %w", i, err)
		}
		if seen[e.Tag()] {
			return fmt.Errorf("duplicate rule-set entry tag: %s", e.Tag())
		}
		seen[e.Tag()] = true
	}
	cp := make([]RuleSetEntry, len(entries))
	copy(cp, entries)
	c.ruleSetEntries = cp
	return nil
}

// HasRuleSetEntries checks if the route config has rule-set entries
func (c *RouteConfig) HasRuleSetEntries() bool {
	return len(c.ruleSetEntries) > 0
}

// GetCustomOutboundByTag returns a copy of a custom outbound by its tag, or nil if not found
func (c *RouteConfig) GetCustomOutboundByTag(tag string) *CustomOutbound {
	for i := range c.customOutbounds {
		if c.customOutbounds[i].Tag() == tag {
			co := c.customOutbounds[i] // value copy
			return &co
		}
	}
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

	// Validate custom outbounds: uniqueness and individual validity
	customTags := make(map[string]bool, len(c.customOutbounds))
	for i, co := range c.customOutbounds {
		if err := co.Validate(); err != nil {
			return fmt.Errorf("invalid custom outbound at index %d: %w", i, err)
		}
		if customTags[co.Tag()] {
			return fmt.Errorf("duplicate custom outbound tag: %s", co.Tag())
		}
		customTags[co.Tag()] = true
	}

	// Validate that all custom outbound references in rules have corresponding definitions
	for i, rule := range c.rules {
		if rule.outbound.IsCustomOutbound() {
			tag := rule.outbound.CustomOutboundTag()
			if !customTags[tag] {
				return fmt.Errorf("rule at index %d references undefined custom outbound: %s", i, tag)
			}
		}
	}
	// Also check finalAction
	if c.finalAction.IsCustomOutbound() {
		tag := c.finalAction.CustomOutboundTag()
		if !customTags[tag] {
			return fmt.Errorf("final action references undefined custom outbound: %s", tag)
		}
	}

	// Validate rule-set entries: uniqueness and individual validity
	rsTags := make(map[string]bool, len(c.ruleSetEntries))
	for i, e := range c.ruleSetEntries {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("invalid rule-set entry at index %d: %w", i, err)
		}
		if rsTags[e.Tag()] {
			return fmt.Errorf("duplicate rule-set entry tag: %s", e.Tag())
		}
		rsTags[e.Tag()] = true
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
	if len(c.customOutbounds) != len(other.customOutbounds) {
		return false
	}
	for i, co := range c.customOutbounds {
		if !co.Equals(&other.customOutbounds[i]) {
			return false
		}
	}
	if len(c.ruleSetEntries) != len(other.ruleSetEntries) {
		return false
	}
	for i, e := range c.ruleSetEntries {
		if !e.Equals(&other.ruleSetEntries[i]) {
			return false
		}
	}
	return true
}

// GetReferencedNodeSIDs returns all unique node SIDs referenced in outbound rules.
// This includes both rule outbounds and finalAction if they reference nodes.
// Custom outbound references (custom_xxx) are excluded.
func (c *RouteConfig) GetReferencedNodeSIDs() []string {
	if c == nil {
		return nil
	}

	seen := make(map[string]bool)
	var sids []string

	// Check rules (skip custom outbound references)
	for _, rule := range c.rules {
		if rule.outbound.IsNodeReference() {
			sid := rule.outbound.NodeSID()
			if !seen[sid] {
				sids = append(sids, sid)
				seen[sid] = true
			}
		}
	}

	// Check finalAction (skip custom outbound references)
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
func ReconstructRouteConfig(rules []RouteRule, finalAction OutboundType, customOutbounds []CustomOutbound, ruleSetEntries []RuleSetEntry) *RouteConfig {
	return &RouteConfig{
		rules:           rules,
		finalAction:     finalAction,
		customOutbounds: customOutbounds,
		ruleSetEntries:  ruleSetEntries,
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
		ruleSetEntries: []RuleSetEntry{
			*ReconstructRuleSetEntry("geoip-cn", "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs", RuleSetFormatBinary, "", "1d"),
			*ReconstructRuleSetEntry("geosite-cn", "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs", RuleSetFormatBinary, "", "1d"),
		},
	}

	// Rule 1: Private IPs direct
	privateRule := mustNewRouteRule(OutboundDirect)
	privateRule.WithIPIsPrivate(true)
	config.rules = append(config.rules, *privateRule)

	// Rule 2: China IPs direct (via rule-set)
	geoIPRule := mustNewRouteRule(OutboundDirect)
	geoIPRule.WithRuleSet("geoip-cn")
	config.rules = append(config.rules, *geoIPRule)

	// Rule 3: China sites direct (via rule-set)
	geoSiteRule := mustNewRouteRule(OutboundDirect)
	geoSiteRule.WithRuleSet("geosite-cn")
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
// - Only specified rule-set tags go through proxy
// - Everything else goes direct
func NewWhitelistRouteConfig(ruleSetTags ...string) *RouteConfig {
	config := &RouteConfig{
		rules:       make([]RouteRule, 0, 1),
		finalAction: OutboundDirect,
	}

	if len(ruleSetTags) > 0 {
		proxyRule := mustNewRouteRule(OutboundProxy)
		proxyRule.WithRuleSet(ruleSetTags...)
		config.rules = append(config.rules, *proxyRule)
	}

	return config
}

// NewBlockAdsRouteConfig creates a route config that blocks ads:
// - Ad domains are blocked (via rule-set)
// - Everything else uses specified default action
// Returns error if defaultAction is invalid
func NewBlockAdsRouteConfig(defaultAction OutboundType) (*RouteConfig, error) {
	if !defaultAction.IsValid() {
		return nil, fmt.Errorf("invalid default action: %s", defaultAction)
	}

	config := &RouteConfig{
		rules:       make([]RouteRule, 0, 1),
		finalAction: defaultAction,
		ruleSetEntries: []RuleSetEntry{
			*ReconstructRuleSetEntry("geosite-category-ads-all", "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs", RuleSetFormatBinary, "", "1d"),
		},
	}

	// Block ad categories via rule-set
	blockRule := mustNewRouteRule(OutboundBlock)
	blockRule.WithRuleSet("geosite-category-ads-all")
	config.rules = append(config.rules, *blockRule)

	return config, nil
}
