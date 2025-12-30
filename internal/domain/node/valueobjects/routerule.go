package valueobjects

import "fmt"

// RouteRule represents a single routing rule for traffic matching.
// Compatible with sing-box route rule configuration.
type RouteRule struct {
	// Match conditions
	domain        []string // Exact domain match
	domainSuffix  []string // Domain suffix match (e.g., ".cn", ".google.com")
	domainKeyword []string // Domain keyword match
	domainRegex   []string // Domain regex match
	ipCIDR        []string // Destination IP CIDR match
	sourceIPCIDR  []string // Source IP CIDR match
	ipIsPrivate   bool     // Match private/LAN IP addresses
	geoIP         []string // GeoIP country codes (e.g., "cn", "us")
	geoSite       []string // GeoSite categories (e.g., "cn", "google", "telegram")
	port          []uint16 // Destination port match
	sourcePort    []uint16 // Source port match
	protocol      []string // Sniffed protocol match (http, tls, quic, etc.)
	network       []string // Network type match (tcp, udp)
	ruleSet       []string // Rule set references

	// Action
	outbound OutboundType // Action to take when matched
}

// NewRouteRule creates a new route rule with the specified outbound action
func NewRouteRule(outbound OutboundType) (*RouteRule, error) {
	if !outbound.IsValid() {
		return nil, fmt.Errorf("invalid outbound type: %s", outbound)
	}
	return &RouteRule{
		outbound: outbound,
	}, nil
}

// Domain getters
func (r *RouteRule) Domain() []string        { return r.domain }
func (r *RouteRule) DomainSuffix() []string  { return r.domainSuffix }
func (r *RouteRule) DomainKeyword() []string { return r.domainKeyword }
func (r *RouteRule) DomainRegex() []string   { return r.domainRegex }

// IP getters
func (r *RouteRule) IPCIDR() []string       { return r.ipCIDR }
func (r *RouteRule) SourceIPCIDR() []string { return r.sourceIPCIDR }
func (r *RouteRule) IPIsPrivate() bool      { return r.ipIsPrivate }

// Geo getters
func (r *RouteRule) GeoIP() []string   { return r.geoIP }
func (r *RouteRule) GeoSite() []string { return r.geoSite }

// Port getters
func (r *RouteRule) Port() []uint16       { return r.port }
func (r *RouteRule) SourcePort() []uint16 { return r.sourcePort }

// Protocol/Network getters
func (r *RouteRule) Protocol() []string { return r.protocol }
func (r *RouteRule) Network() []string  { return r.network }

// RuleSet getter
func (r *RouteRule) RuleSet() []string { return r.ruleSet }

// Outbound getter
func (r *RouteRule) Outbound() OutboundType { return r.outbound }

// Builder pattern setters for fluent API
func (r *RouteRule) WithDomain(domains ...string) *RouteRule {
	r.domain = domains
	return r
}

func (r *RouteRule) WithDomainSuffix(suffixes ...string) *RouteRule {
	r.domainSuffix = suffixes
	return r
}

func (r *RouteRule) WithDomainKeyword(keywords ...string) *RouteRule {
	r.domainKeyword = keywords
	return r
}

func (r *RouteRule) WithDomainRegex(patterns ...string) *RouteRule {
	r.domainRegex = patterns
	return r
}

func (r *RouteRule) WithIPCIDR(cidrs ...string) *RouteRule {
	r.ipCIDR = cidrs
	return r
}

func (r *RouteRule) WithSourceIPCIDR(cidrs ...string) *RouteRule {
	r.sourceIPCIDR = cidrs
	return r
}

func (r *RouteRule) WithIPIsPrivate(isPrivate bool) *RouteRule {
	r.ipIsPrivate = isPrivate
	return r
}

func (r *RouteRule) WithGeoIP(countries ...string) *RouteRule {
	r.geoIP = countries
	return r
}

func (r *RouteRule) WithGeoSite(categories ...string) *RouteRule {
	r.geoSite = categories
	return r
}

func (r *RouteRule) WithPort(ports ...uint16) *RouteRule {
	r.port = ports
	return r
}

func (r *RouteRule) WithSourcePort(ports ...uint16) *RouteRule {
	r.sourcePort = ports
	return r
}

func (r *RouteRule) WithProtocol(protocols ...string) *RouteRule {
	r.protocol = protocols
	return r
}

func (r *RouteRule) WithNetwork(networks ...string) *RouteRule {
	r.network = networks
	return r
}

func (r *RouteRule) WithRuleSet(ruleSets ...string) *RouteRule {
	r.ruleSet = ruleSets
	return r
}

// HasConditions checks if the rule has any match conditions
func (r *RouteRule) HasConditions() bool {
	return len(r.domain) > 0 ||
		len(r.domainSuffix) > 0 ||
		len(r.domainKeyword) > 0 ||
		len(r.domainRegex) > 0 ||
		len(r.ipCIDR) > 0 ||
		len(r.sourceIPCIDR) > 0 ||
		r.ipIsPrivate ||
		len(r.geoIP) > 0 ||
		len(r.geoSite) > 0 ||
		len(r.port) > 0 ||
		len(r.sourcePort) > 0 ||
		len(r.protocol) > 0 ||
		len(r.network) > 0 ||
		len(r.ruleSet) > 0
}

// Validate validates the route rule
func (r *RouteRule) Validate() error {
	if !r.outbound.IsValid() {
		return fmt.Errorf("invalid outbound type: %s", r.outbound)
	}
	// A rule without conditions is valid (acts as catch-all)
	return nil
}

// Equals compares two route rules for equality
func (r *RouteRule) Equals(other *RouteRule) bool {
	if r == nil && other == nil {
		return true
	}
	if r == nil || other == nil {
		return false
	}
	if r.outbound != other.outbound {
		return false
	}
	if r.ipIsPrivate != other.ipIsPrivate {
		return false
	}
	if !stringSliceEqual(r.domain, other.domain) ||
		!stringSliceEqual(r.domainSuffix, other.domainSuffix) ||
		!stringSliceEqual(r.domainKeyword, other.domainKeyword) ||
		!stringSliceEqual(r.domainRegex, other.domainRegex) ||
		!stringSliceEqual(r.ipCIDR, other.ipCIDR) ||
		!stringSliceEqual(r.sourceIPCIDR, other.sourceIPCIDR) ||
		!stringSliceEqual(r.geoIP, other.geoIP) ||
		!stringSliceEqual(r.geoSite, other.geoSite) ||
		!stringSliceEqual(r.protocol, other.protocol) ||
		!stringSliceEqual(r.network, other.network) ||
		!stringSliceEqual(r.ruleSet, other.ruleSet) {
		return false
	}
	if !uint16SliceEqual(r.port, other.port) ||
		!uint16SliceEqual(r.sourcePort, other.sourcePort) {
		return false
	}
	return true
}

// ReconstructRouteRule reconstructs a RouteRule from persistence data
func ReconstructRouteRule(
	domain, domainSuffix, domainKeyword, domainRegex []string,
	ipCIDR, sourceIPCIDR []string,
	ipIsPrivate bool,
	geoIP, geoSite []string,
	port, sourcePort []uint16,
	protocol, network []string,
	ruleSet []string,
	outbound OutboundType,
) *RouteRule {
	return &RouteRule{
		domain:        domain,
		domainSuffix:  domainSuffix,
		domainKeyword: domainKeyword,
		domainRegex:   domainRegex,
		ipCIDR:        ipCIDR,
		sourceIPCIDR:  sourceIPCIDR,
		ipIsPrivate:   ipIsPrivate,
		geoIP:         geoIP,
		geoSite:       geoSite,
		port:          port,
		sourcePort:    sourcePort,
		protocol:      protocol,
		network:       network,
		ruleSet:       ruleSet,
		outbound:      outbound,
	}
}

// helper functions
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func uint16SliceEqual(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
