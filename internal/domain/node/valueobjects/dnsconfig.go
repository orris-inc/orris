package valueobjects

import (
	"fmt"
	"regexp"
)

// DnsStrategy defines DNS resolution strategy.
// Compatible with sing-box dns.strategy field.
type DnsStrategy string

const (
	DnsStrategyPreferIPv4 DnsStrategy = "prefer_ipv4"
	DnsStrategyPreferIPv6 DnsStrategy = "prefer_ipv6"
	DnsStrategyIPv4Only   DnsStrategy = "ipv4_only"
	DnsStrategyIPv6Only   DnsStrategy = "ipv6_only"
)

// IsValid checks if the DNS strategy is valid (including empty = unspecified)
func (s DnsStrategy) IsValid() bool {
	switch s {
	case "", DnsStrategyPreferIPv4, DnsStrategyPreferIPv6, DnsStrategyIPv4Only, DnsStrategyIPv6Only:
		return true
	default:
		return false
	}
}

// String returns the string representation
func (s DnsStrategy) String() string {
	return string(s)
}

// dnsServerTagRegex validates DNS server tag format
var dnsServerTagRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

const (
	maxDnsServerTagLength = 64
	maxDnsServers         = 20
	maxDnsRules           = 50
)

// DnsServer represents a DNS server configuration.
// Compatible with sing-box dns.servers[] entry.
type DnsServer struct {
	tag             string      // unique identifier
	address         string      // DNS server address (e.g., "https://1.1.1.1/dns-query", "tls://8.8.8.8", "223.5.5.5")
	addressResolver string      // optional: tag of another DNS server used to resolve this server's address
	addressStrategy DnsStrategy // optional: strategy for resolving this server's address
	strategy        DnsStrategy // optional: DNS resolution strategy for queries sent to this server
	detour          string      // optional: outbound tag for this DNS server (direct/proxy/node_xxx/custom_xxx)
}

// NewDnsServer creates a new DNS server with validation
func NewDnsServer(tag, address string) (*DnsServer, error) {
	if tag == "" {
		return nil, fmt.Errorf("dns server tag is required")
	}
	if len(tag) > maxDnsServerTagLength {
		return nil, fmt.Errorf("dns server tag too long: %d (max %d)", len(tag), maxDnsServerTagLength)
	}
	if !dnsServerTagRegex.MatchString(tag) {
		return nil, fmt.Errorf("invalid dns server tag format: %s (must match ^[a-zA-Z0-9][a-zA-Z0-9_-]*$)", tag)
	}
	if address == "" {
		return nil, fmt.Errorf("dns server address is required")
	}
	return &DnsServer{
		tag:     tag,
		address: address,
	}, nil
}

// Tag returns the server tag
func (s *DnsServer) Tag() string { return s.tag }

// Address returns the server address
func (s *DnsServer) Address() string { return s.address }

// AddressResolver returns the address resolver tag
func (s *DnsServer) AddressResolver() string { return s.addressResolver }

// AddressStrategy returns the address resolution strategy
func (s *DnsServer) AddressStrategy() DnsStrategy { return s.addressStrategy }

// Strategy returns the DNS resolution strategy
func (s *DnsServer) Strategy() DnsStrategy { return s.strategy }

// Detour returns the outbound tag
func (s *DnsServer) Detour() string { return s.detour }

// WithAddressResolver sets the address resolver
func (s *DnsServer) WithAddressResolver(resolver string) *DnsServer {
	s.addressResolver = resolver
	return s
}

// WithAddressStrategy sets the address strategy
func (s *DnsServer) WithAddressStrategy(strategy DnsStrategy) *DnsServer {
	s.addressStrategy = strategy
	return s
}

// WithStrategy sets the DNS strategy
func (s *DnsServer) WithStrategy(strategy DnsStrategy) *DnsServer {
	s.strategy = strategy
	return s
}

// WithDetour sets the outbound detour
func (s *DnsServer) WithDetour(detour string) *DnsServer {
	s.detour = detour
	return s
}

// Validate validates the DNS server configuration
func (s *DnsServer) Validate() error {
	if s.tag == "" {
		return fmt.Errorf("dns server tag is required")
	}
	if len(s.tag) > maxDnsServerTagLength {
		return fmt.Errorf("dns server tag too long: %d (max %d)", len(s.tag), maxDnsServerTagLength)
	}
	if !dnsServerTagRegex.MatchString(s.tag) {
		return fmt.Errorf("invalid dns server tag format: %s", s.tag)
	}
	if s.address == "" {
		return fmt.Errorf("dns server address is required")
	}
	if !s.addressStrategy.IsValid() {
		return fmt.Errorf("invalid address strategy: %s", s.addressStrategy)
	}
	if !s.strategy.IsValid() {
		return fmt.Errorf("invalid strategy: %s", s.strategy)
	}
	if s.detour != "" {
		ot := OutboundType(s.detour)
		if !ot.IsValid() {
			return fmt.Errorf("invalid detour: %s", s.detour)
		}
	}
	return nil
}

// Equals compares two DNS servers
func (s *DnsServer) Equals(other *DnsServer) bool {
	if s == nil && other == nil {
		return true
	}
	if s == nil || other == nil {
		return false
	}
	return s.tag == other.tag &&
		s.address == other.address &&
		s.addressResolver == other.addressResolver &&
		s.addressStrategy == other.addressStrategy &&
		s.strategy == other.strategy &&
		s.detour == other.detour
}

// ReconstructDnsServer reconstructs a DnsServer from persistence data without validation
func ReconstructDnsServer(tag, address, addressResolver string, addressStrategy, strategy DnsStrategy, detour string) *DnsServer {
	return &DnsServer{
		tag:             tag,
		address:         address,
		addressResolver: addressResolver,
		addressStrategy: addressStrategy,
		strategy:        strategy,
		detour:          detour,
	}
}

// DnsRule represents a DNS routing rule.
// Compatible with sing-box dns.rules[] entry.
type DnsRule struct {
	// Matching conditions
	domain        []string // exact domain match
	domainSuffix  []string // domain suffix match
	domainKeyword []string // domain keyword match
	domainRegex   []string // domain regex match
	geosite       []string // GeoSite categories
	geoip         []string // GeoIP country codes (for response IP matching)
	ruleSet       []string // rule set references
	outbound      []string // match by outbound tag

	// Action
	server       string // required: DNS server tag to use
	disableCache bool   // optional: disable cache for matched queries
}

// NewDnsRule creates a new DNS rule with a target server tag
func NewDnsRule(server string) (*DnsRule, error) {
	if server == "" {
		return nil, fmt.Errorf("dns rule server is required")
	}
	return &DnsRule{
		server: server,
	}, nil
}

// Server returns the target DNS server tag
func (r *DnsRule) Server() string { return r.server }

// DisableCache returns whether caching is disabled
func (r *DnsRule) DisableCache() bool { return r.disableCache }

// Domain returns domain conditions (copy)
func (r *DnsRule) Domain() []string {
	if r.domain == nil {
		return nil
	}
	result := make([]string, len(r.domain))
	copy(result, r.domain)
	return result
}

// DomainSuffix returns domain suffix conditions (copy)
func (r *DnsRule) DomainSuffix() []string {
	if r.domainSuffix == nil {
		return nil
	}
	result := make([]string, len(r.domainSuffix))
	copy(result, r.domainSuffix)
	return result
}

// DomainKeyword returns domain keyword conditions (copy)
func (r *DnsRule) DomainKeyword() []string {
	if r.domainKeyword == nil {
		return nil
	}
	result := make([]string, len(r.domainKeyword))
	copy(result, r.domainKeyword)
	return result
}

// DomainRegex returns domain regex conditions (copy)
func (r *DnsRule) DomainRegex() []string {
	if r.domainRegex == nil {
		return nil
	}
	result := make([]string, len(r.domainRegex))
	copy(result, r.domainRegex)
	return result
}

// Geosite returns geosite conditions (copy)
func (r *DnsRule) Geosite() []string {
	if r.geosite == nil {
		return nil
	}
	result := make([]string, len(r.geosite))
	copy(result, r.geosite)
	return result
}

// GeoIP returns geoip conditions (copy)
func (r *DnsRule) GeoIP() []string {
	if r.geoip == nil {
		return nil
	}
	result := make([]string, len(r.geoip))
	copy(result, r.geoip)
	return result
}

// RuleSet returns rule set conditions (copy)
func (r *DnsRule) RuleSet() []string {
	if r.ruleSet == nil {
		return nil
	}
	result := make([]string, len(r.ruleSet))
	copy(result, r.ruleSet)
	return result
}

// Outbound returns outbound conditions (copy)
func (r *DnsRule) Outbound() []string {
	if r.outbound == nil {
		return nil
	}
	result := make([]string, len(r.outbound))
	copy(result, r.outbound)
	return result
}

// Builder methods for setting conditions

// WithDomain sets exact domain match conditions
func (r *DnsRule) WithDomain(domains ...string) *DnsRule {
	r.domain = append([]string(nil), domains...)
	return r
}

// WithDomainSuffix sets domain suffix match conditions
func (r *DnsRule) WithDomainSuffix(suffixes ...string) *DnsRule {
	r.domainSuffix = append([]string(nil), suffixes...)
	return r
}

// WithDomainKeyword sets domain keyword match conditions
func (r *DnsRule) WithDomainKeyword(keywords ...string) *DnsRule {
	r.domainKeyword = append([]string(nil), keywords...)
	return r
}

// WithDomainRegex sets domain regex match conditions
func (r *DnsRule) WithDomainRegex(regexes ...string) *DnsRule {
	r.domainRegex = append([]string(nil), regexes...)
	return r
}

// WithGeosite sets geosite match conditions
func (r *DnsRule) WithGeosite(categories ...string) *DnsRule {
	r.geosite = append([]string(nil), categories...)
	return r
}

// WithGeoIP sets geoip match conditions
func (r *DnsRule) WithGeoIP(codes ...string) *DnsRule {
	r.geoip = append([]string(nil), codes...)
	return r
}

// WithRuleSet sets rule set match conditions
func (r *DnsRule) WithRuleSet(sets ...string) *DnsRule {
	r.ruleSet = append([]string(nil), sets...)
	return r
}

// WithOutbound sets outbound tag match conditions
func (r *DnsRule) WithOutbound(tags ...string) *DnsRule {
	r.outbound = append([]string(nil), tags...)
	return r
}

// WithDisableCache sets the disable cache flag
func (r *DnsRule) WithDisableCache(disable bool) *DnsRule {
	r.disableCache = disable
	return r
}

// hasCondition checks if the rule has at least one matching condition
func (r *DnsRule) hasCondition() bool {
	return len(r.domain) > 0 || len(r.domainSuffix) > 0 ||
		len(r.domainKeyword) > 0 || len(r.domainRegex) > 0 ||
		len(r.geosite) > 0 || len(r.geoip) > 0 ||
		len(r.ruleSet) > 0 || len(r.outbound) > 0
}

// Validate validates the DNS rule
func (r *DnsRule) Validate() error {
	if r.server == "" {
		return fmt.Errorf("dns rule server is required")
	}
	if !r.hasCondition() {
		return fmt.Errorf("dns rule must have at least one matching condition")
	}
	return nil
}

// Equals compares two DNS rules
func (r *DnsRule) Equals(other *DnsRule) bool {
	if r == nil && other == nil {
		return true
	}
	if r == nil || other == nil {
		return false
	}
	if r.server != other.server || r.disableCache != other.disableCache {
		return false
	}
	return stringSliceEqual(r.domain, other.domain) &&
		stringSliceEqual(r.domainSuffix, other.domainSuffix) &&
		stringSliceEqual(r.domainKeyword, other.domainKeyword) &&
		stringSliceEqual(r.domainRegex, other.domainRegex) &&
		stringSliceEqual(r.geosite, other.geosite) &&
		stringSliceEqual(r.geoip, other.geoip) &&
		stringSliceEqual(r.ruleSet, other.ruleSet) &&
		stringSliceEqual(r.outbound, other.outbound)
}

// ReconstructDnsRule reconstructs a DnsRule from persistence data without validation.
// Defensive copies are made for all slice parameters.
func ReconstructDnsRule(
	domain, domainSuffix, domainKeyword, domainRegex,
	geosite, geoip, ruleSet, outbound []string,
	server string, disableCache bool,
) *DnsRule {
	return &DnsRule{
		domain:        copyStringSlice(domain),
		domainSuffix:  copyStringSlice(domainSuffix),
		domainKeyword: copyStringSlice(domainKeyword),
		domainRegex:   copyStringSlice(domainRegex),
		geosite:       copyStringSlice(geosite),
		geoip:         copyStringSlice(geoip),
		ruleSet:       copyStringSlice(ruleSet),
		outbound:      copyStringSlice(outbound),
		server:        server,
		disableCache:  disableCache,
	}
}

// copyStringSlice returns a defensive copy of a string slice, preserving nil.
func copyStringSlice(s []string) []string {
	if s == nil {
		return nil
	}
	cp := make([]string, len(s))
	copy(cp, s)
	return cp
}

// DnsConfig represents the DNS configuration for a node.
// Compatible with sing-box dns configuration.
type DnsConfig struct {
	servers          []DnsServer
	rules            []DnsRule
	final            string      // required: default DNS server tag
	strategy         DnsStrategy // global DNS strategy
	disableCache     bool
	disableExpire    bool
	independentCache bool
	reverseMapping   bool
}

// NewDnsConfig creates a new DNS configuration with a final (default) server tag
func NewDnsConfig(final string) (*DnsConfig, error) {
	if final == "" {
		return nil, fmt.Errorf("dns config final server tag is required")
	}
	return &DnsConfig{
		servers: make([]DnsServer, 0),
		rules:   make([]DnsRule, 0),
		final:   final,
	}, nil
}

// Servers returns a copy of the DNS servers
func (c *DnsConfig) Servers() []DnsServer {
	if c.servers == nil {
		return nil
	}
	result := make([]DnsServer, len(c.servers))
	copy(result, c.servers)
	return result
}

// Rules returns a copy of the DNS rules
func (c *DnsConfig) Rules() []DnsRule {
	if c.rules == nil {
		return nil
	}
	result := make([]DnsRule, len(c.rules))
	copy(result, c.rules)
	return result
}

// Final returns the default DNS server tag
func (c *DnsConfig) Final() string { return c.final }

// Strategy returns the global DNS strategy
func (c *DnsConfig) Strategy() DnsStrategy { return c.strategy }

// DisableCache returns whether DNS cache is disabled
func (c *DnsConfig) DisableCache() bool { return c.disableCache }

// DisableExpire returns whether DNS cache expiration is disabled
func (c *DnsConfig) DisableExpire() bool { return c.disableExpire }

// IndependentCache returns whether independent cache per server is enabled
func (c *DnsConfig) IndependentCache() bool { return c.independentCache }

// ReverseMapping returns whether reverse DNS mapping is enabled
func (c *DnsConfig) ReverseMapping() bool { return c.reverseMapping }

// SetServers replaces all DNS servers
func (c *DnsConfig) SetServers(servers []DnsServer) error {
	if len(servers) > maxDnsServers {
		return fmt.Errorf("too many dns servers: %d (max %d)", len(servers), maxDnsServers)
	}
	tags := make(map[string]bool, len(servers))
	for i, s := range servers {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("invalid dns server at index %d: %w", i, err)
		}
		if tags[s.tag] {
			return fmt.Errorf("duplicate dns server tag at index %d: %s", i, s.tag)
		}
		tags[s.tag] = true
	}
	cp := make([]DnsServer, len(servers))
	copy(cp, servers)
	c.servers = cp
	return nil
}

// SetRules replaces all DNS rules
func (c *DnsConfig) SetRules(rules []DnsRule) error {
	if len(rules) > maxDnsRules {
		return fmt.Errorf("too many dns rules: %d (max %d)", len(rules), maxDnsRules)
	}
	for i, r := range rules {
		if err := r.Validate(); err != nil {
			return fmt.Errorf("invalid dns rule at index %d: %w", i, err)
		}
	}
	cp := make([]DnsRule, len(rules))
	copy(cp, rules)
	c.rules = cp
	return nil
}

// SetFinal sets the default DNS server tag
func (c *DnsConfig) SetFinal(tag string) error {
	if tag == "" {
		return fmt.Errorf("dns config final server tag is required")
	}
	c.final = tag
	return nil
}

// SetStrategy sets the global DNS strategy
func (c *DnsConfig) SetStrategy(strategy DnsStrategy) error {
	if !strategy.IsValid() {
		return fmt.Errorf("invalid dns strategy: %s", strategy)
	}
	c.strategy = strategy
	return nil
}

// SetDisableCache sets the disable cache flag
func (c *DnsConfig) SetDisableCache(v bool) { c.disableCache = v }

// SetDisableExpire sets the disable expire flag
func (c *DnsConfig) SetDisableExpire(v bool) { c.disableExpire = v }

// SetIndependentCache sets the independent cache flag
func (c *DnsConfig) SetIndependentCache(v bool) { c.independentCache = v }

// SetReverseMapping sets the reverse mapping flag
func (c *DnsConfig) SetReverseMapping(v bool) { c.reverseMapping = v }

// Validate validates the entire DNS configuration
func (c *DnsConfig) Validate() error {
	if c.final == "" {
		return fmt.Errorf("dns config final server tag is required")
	}
	if !c.strategy.IsValid() {
		return fmt.Errorf("invalid global dns strategy: %s", c.strategy)
	}
	if len(c.servers) > maxDnsServers {
		return fmt.Errorf("too many dns servers: %d (max %d)", len(c.servers), maxDnsServers)
	}
	if len(c.rules) > maxDnsRules {
		return fmt.Errorf("too many dns rules: %d (max %d)", len(c.rules), maxDnsRules)
	}

	// Validate each server and check tag uniqueness
	serverTags := make(map[string]bool, len(c.servers))
	for i, s := range c.servers {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("invalid dns server at index %d: %w", i, err)
		}
		if serverTags[s.tag] {
			return fmt.Errorf("duplicate dns server tag: %s", s.tag)
		}
		serverTags[s.tag] = true
	}

	// Validate final references an existing server
	if !serverTags[c.final] {
		return fmt.Errorf("dns final references undefined server tag: %s", c.final)
	}

	// Validate each rule
	for i, r := range c.rules {
		if err := r.Validate(); err != nil {
			return fmt.Errorf("invalid dns rule at index %d: %w", i, err)
		}
		// Validate rule.server references an existing server
		if !serverTags[r.server] {
			return fmt.Errorf("dns rule at index %d references undefined server tag: %s", i, r.server)
		}
	}

	// Validate addressResolver references
	for i, s := range c.servers {
		if s.addressResolver != "" {
			if s.addressResolver == s.tag {
				return fmt.Errorf("dns server at index %d cannot reference itself as addressResolver: %s", i, s.tag)
			}
			if !serverTags[s.addressResolver] {
				return fmt.Errorf("dns server at index %d addressResolver references undefined server tag: %s", i, s.addressResolver)
			}
		}
	}

	return nil
}

// HasNodeReferences checks if any DNS server detour references a node outbound (node_xxx format)
func (c *DnsConfig) HasNodeReferences() bool {
	if c == nil {
		return false
	}
	for _, s := range c.servers {
		ot := OutboundType(s.detour)
		if ot.IsNodeReference() {
			return true
		}
	}
	return false
}

// GetReferencedNodeSIDs returns all unique node SIDs referenced by DNS server detours
func (c *DnsConfig) GetReferencedNodeSIDs() []string {
	if c == nil {
		return nil
	}
	seen := make(map[string]bool)
	var sids []string
	for _, s := range c.servers {
		ot := OutboundType(s.detour)
		if ot.IsNodeReference() {
			sid := ot.NodeSID()
			if !seen[sid] {
				sids = append(sids, sid)
				seen[sid] = true
			}
		}
	}
	return sids
}

// Equals compares two DNS configurations for equality
func (c *DnsConfig) Equals(other *DnsConfig) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	if c.final != other.final || c.strategy != other.strategy ||
		c.disableCache != other.disableCache || c.disableExpire != other.disableExpire ||
		c.independentCache != other.independentCache || c.reverseMapping != other.reverseMapping {
		return false
	}
	if len(c.servers) != len(other.servers) {
		return false
	}
	for i := range c.servers {
		if !c.servers[i].Equals(&other.servers[i]) {
			return false
		}
	}
	if len(c.rules) != len(other.rules) {
		return false
	}
	for i := range c.rules {
		if !c.rules[i].Equals(&other.rules[i]) {
			return false
		}
	}
	return true
}

// ReconstructDnsConfig reconstructs a DnsConfig from persistence data without validation.
// Defensive copies are made for slice parameters.
func ReconstructDnsConfig(
	servers []DnsServer,
	rules []DnsRule,
	final string,
	strategy DnsStrategy,
	disableCache, disableExpire, independentCache, reverseMapping bool,
) *DnsConfig {
	svrs := make([]DnsServer, len(servers))
	copy(svrs, servers)
	rls := make([]DnsRule, len(rules))
	copy(rls, rules)
	return &DnsConfig{
		servers:          svrs,
		rules:            rls,
		final:            final,
		strategy:         strategy,
		disableCache:     disableCache,
		disableExpire:    disableExpire,
		independentCache: independentCache,
		reverseMapping:   reverseMapping,
	}
}
