package valueobjects

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// DnsServer Tests
// =============================================================================

func TestNewDnsServer_Valid(t *testing.T) {
	s, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
	require.NoError(t, err)
	require.NotNil(t, s)

	assert.Equal(t, "remote", s.Tag())
	assert.Equal(t, "https://1.1.1.1/dns-query", s.Address())
	assert.Equal(t, "", s.AddressResolver())
	assert.Equal(t, DnsStrategy(""), s.AddressStrategy())
	assert.Equal(t, DnsStrategy(""), s.Strategy())
	assert.Equal(t, "", s.Detour())
}

func TestNewDnsServer_EmptyTag(t *testing.T) {
	_, err := NewDnsServer("", "https://1.1.1.1/dns-query")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dns server tag is required")
}

func TestNewDnsServer_EmptyAddress(t *testing.T) {
	_, err := NewDnsServer("remote", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dns server address is required")
}

func TestNewDnsServer_TagTooLong(t *testing.T) {
	longTag := strings.Repeat("a", maxDnsServerTagLength+1)
	_, err := NewDnsServer(longTag, "https://1.1.1.1/dns-query")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dns server tag too long")
}

func TestNewDnsServer_InvalidTagFormat(t *testing.T) {
	tests := []struct {
		name string
		tag  string
	}{
		{"starts with hyphen", "-remote"},
		{"starts with underscore", "_remote"},
		{"contains space", "re mote"},
		{"contains dot", "re.mote"},
		{"contains special char", "remote@1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewDnsServer(tc.tag, "https://1.1.1.1/dns-query")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid dns server tag format")
		})
	}
}

func TestDnsServer_Validate_InvalidStrategy(t *testing.T) {
	s, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
	require.NoError(t, err)

	s.WithStrategy(DnsStrategy("invalid_strategy"))
	err = s.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid strategy")
}

func TestDnsServer_Validate_InvalidAddressStrategy(t *testing.T) {
	s, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
	require.NoError(t, err)

	s.WithAddressStrategy(DnsStrategy("bad_strategy"))
	err = s.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid address strategy")
}

func TestDnsServer_Validate_InvalidDetour(t *testing.T) {
	s, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
	require.NoError(t, err)

	s.WithDetour("not_a_valid_outbound")
	err = s.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid detour")
}

func TestDnsServer_Validate_ValidDetour(t *testing.T) {
	tests := []struct {
		name   string
		detour string
	}{
		{"direct", "direct"},
		{"proxy", "proxy"},
		{"block", "block"},
		{"node reference", "node_abc123"},
		{"custom outbound", "custom_my_outbound"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
			require.NoError(t, err)
			s.WithDetour(tc.detour)
			assert.NoError(t, s.Validate())
		})
	}
}

func TestDnsServer_WithMethods(t *testing.T) {
	s, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
	require.NoError(t, err)

	// Chain builder methods
	result := s.
		WithAddressResolver("local").
		WithAddressStrategy(DnsStrategyPreferIPv4).
		WithStrategy(DnsStrategyIPv4Only).
		WithDetour("direct")

	// Verify builder returns same pointer
	assert.Same(t, s, result)

	// Verify values
	assert.Equal(t, "local", s.AddressResolver())
	assert.Equal(t, DnsStrategyPreferIPv4, s.AddressStrategy())
	assert.Equal(t, DnsStrategyIPv4Only, s.Strategy())
	assert.Equal(t, "direct", s.Detour())
}

func TestDnsServer_Equals(t *testing.T) {
	t.Run("equal servers", func(t *testing.T) {
		s1, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
		require.NoError(t, err)
		s1.WithDetour("proxy").WithStrategy(DnsStrategyIPv4Only)

		s2, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
		require.NoError(t, err)
		s2.WithDetour("proxy").WithStrategy(DnsStrategyIPv4Only)

		assert.True(t, s1.Equals(s2))
	})

	t.Run("different tag", func(t *testing.T) {
		s1, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
		require.NoError(t, err)
		s2, err := NewDnsServer("local", "https://1.1.1.1/dns-query")
		require.NoError(t, err)
		assert.False(t, s1.Equals(s2))
	})

	t.Run("different address", func(t *testing.T) {
		s1, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
		require.NoError(t, err)
		s2, err := NewDnsServer("remote", "https://8.8.8.8/dns-query")
		require.NoError(t, err)
		assert.False(t, s1.Equals(s2))
	})

	t.Run("different detour", func(t *testing.T) {
		s1, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
		require.NoError(t, err)
		s1.WithDetour("direct")
		s2, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
		require.NoError(t, err)
		s2.WithDetour("proxy")
		assert.False(t, s1.Equals(s2))
	})

	t.Run("nil comparisons", func(t *testing.T) {
		s, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
		require.NoError(t, err)

		assert.True(t, (*DnsServer)(nil).Equals(nil))
		assert.False(t, s.Equals(nil))
		assert.False(t, (*DnsServer)(nil).Equals(s))
	})
}

func TestDnsServer_Validate_AllStrategies(t *testing.T) {
	validStrategies := []DnsStrategy{
		"",
		DnsStrategyPreferIPv4,
		DnsStrategyPreferIPv6,
		DnsStrategyIPv4Only,
		DnsStrategyIPv6Only,
	}
	for _, strat := range validStrategies {
		t.Run("strategy_"+string(strat), func(t *testing.T) {
			s, err := NewDnsServer("remote", "1.1.1.1")
			require.NoError(t, err)
			s.WithStrategy(strat)
			assert.NoError(t, s.Validate())
		})
	}
}

// =============================================================================
// DnsRule Tests
// =============================================================================

func TestNewDnsRule_Valid(t *testing.T) {
	r, err := NewDnsRule("remote")
	require.NoError(t, err)
	require.NotNil(t, r)

	assert.Equal(t, "remote", r.Server())
	assert.False(t, r.DisableCache())
	assert.Nil(t, r.Domain())
	assert.Nil(t, r.DomainSuffix())
	assert.Nil(t, r.DomainKeyword())
	assert.Nil(t, r.DomainRegex())
	assert.Nil(t, r.Geosite())
	assert.Nil(t, r.GeoIP())
	assert.Nil(t, r.RuleSet())
	assert.Nil(t, r.Outbound())
}

func TestNewDnsRule_EmptyServer(t *testing.T) {
	_, err := NewDnsRule("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dns rule server is required")
}

func TestDnsRule_Validate_NoCondition(t *testing.T) {
	r, err := NewDnsRule("remote")
	require.NoError(t, err)

	err = r.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dns rule must have at least one matching condition")
}

func TestDnsRule_WithMethods(t *testing.T) {
	r, err := NewDnsRule("local")
	require.NoError(t, err)

	// Set all builder methods
	r.WithDomain("example.com", "test.com").
		WithDomainSuffix(".cn", ".jp").
		WithDomainKeyword("google", "youtube").
		WithDomainRegex(`^ads\..*`).
		WithGeosite("cn", "geolocation-cn").
		WithGeoIP("CN", "JP").
		WithRuleSet("geosite-cn", "geoip-cn").
		WithOutbound("direct", "proxy").
		WithDisableCache(true)

	// Verify values
	assert.Equal(t, []string{"example.com", "test.com"}, r.Domain())
	assert.Equal(t, []string{".cn", ".jp"}, r.DomainSuffix())
	assert.Equal(t, []string{"google", "youtube"}, r.DomainKeyword())
	assert.Equal(t, []string{`^ads\..*`}, r.DomainRegex())
	assert.Equal(t, []string{"cn", "geolocation-cn"}, r.Geosite())
	assert.Equal(t, []string{"CN", "JP"}, r.GeoIP())
	assert.Equal(t, []string{"geosite-cn", "geoip-cn"}, r.RuleSet())
	assert.Equal(t, []string{"direct", "proxy"}, r.Outbound())
	assert.True(t, r.DisableCache())

	// Verify validation passes with conditions set
	assert.NoError(t, r.Validate())

	// Verify defensive copy: mutating returned slice does not affect original
	domains := r.Domain()
	domains[0] = "mutated.com"
	assert.Equal(t, "example.com", r.Domain()[0], "getter must return defensive copy")
}

func TestDnsRule_WithMethods_DefensiveCopy(t *testing.T) {
	r, err := NewDnsRule("local")
	require.NoError(t, err)

	// Set domains from an external slice
	external := []string{"a.com", "b.com"}
	r.WithDomain(external...)

	// Mutate the external slice
	external[0] = "mutated.com"

	// The rule's internal data should not be affected
	assert.Equal(t, "a.com", r.Domain()[0], "WithDomain must make defensive copy")
}

func TestDnsRule_Equals(t *testing.T) {
	t.Run("equal rules", func(t *testing.T) {
		r1, err := NewDnsRule("local")
		require.NoError(t, err)
		r1.WithGeosite("cn").WithDisableCache(true)

		r2, err := NewDnsRule("local")
		require.NoError(t, err)
		r2.WithGeosite("cn").WithDisableCache(true)

		assert.True(t, r1.Equals(r2))
	})

	t.Run("different server", func(t *testing.T) {
		r1, err := NewDnsRule("local")
		require.NoError(t, err)
		r1.WithGeosite("cn")

		r2, err := NewDnsRule("remote")
		require.NoError(t, err)
		r2.WithGeosite("cn")

		assert.False(t, r1.Equals(r2))
	})

	t.Run("different conditions", func(t *testing.T) {
		r1, err := NewDnsRule("local")
		require.NoError(t, err)
		r1.WithGeosite("cn")

		r2, err := NewDnsRule("local")
		require.NoError(t, err)
		r2.WithGeosite("jp")

		assert.False(t, r1.Equals(r2))
	})

	t.Run("different disable cache", func(t *testing.T) {
		r1, err := NewDnsRule("local")
		require.NoError(t, err)
		r1.WithGeosite("cn").WithDisableCache(false)

		r2, err := NewDnsRule("local")
		require.NoError(t, err)
		r2.WithGeosite("cn").WithDisableCache(true)

		assert.False(t, r1.Equals(r2))
	})

	t.Run("nil comparisons", func(t *testing.T) {
		r, err := NewDnsRule("local")
		require.NoError(t, err)
		r.WithGeosite("cn")

		assert.True(t, (*DnsRule)(nil).Equals(nil))
		assert.False(t, r.Equals(nil))
		assert.False(t, (*DnsRule)(nil).Equals(r))
	})
}

func TestReconstructDnsRule(t *testing.T) {
	domain := []string{"example.com"}
	domainSuffix := []string{".cn"}
	domainKeyword := []string{"google"}
	domainRegex := []string{`^ads\..*`}
	geosite := []string{"cn"}
	geoip := []string{"CN"}
	ruleSet := []string{"geosite-cn"}
	outbound := []string{"direct"}

	r := ReconstructDnsRule(
		domain, domainSuffix, domainKeyword, domainRegex,
		geosite, geoip, ruleSet, outbound,
		"local", true,
	)
	require.NotNil(t, r)

	assert.Equal(t, "local", r.Server())
	assert.True(t, r.DisableCache())
	assert.Equal(t, []string{"example.com"}, r.Domain())
	assert.Equal(t, []string{".cn"}, r.DomainSuffix())
	assert.Equal(t, []string{"google"}, r.DomainKeyword())
	assert.Equal(t, []string{`^ads\..*`}, r.DomainRegex())
	assert.Equal(t, []string{"cn"}, r.Geosite())
	assert.Equal(t, []string{"CN"}, r.GeoIP())
	assert.Equal(t, []string{"geosite-cn"}, r.RuleSet())
	assert.Equal(t, []string{"direct"}, r.Outbound())

	// Verify defensive copy: mutating the input slice should not affect the rule
	domain[0] = "mutated.com"
	assert.Equal(t, "example.com", r.Domain()[0], "ReconstructDnsRule must make defensive copy")
}

// =============================================================================
// DnsConfig Tests
// =============================================================================

func TestNewDnsConfig_Valid(t *testing.T) {
	config, err := NewDnsConfig("remote")
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "remote", config.Final())
	assert.Empty(t, config.Servers())
	assert.Empty(t, config.Rules())
	assert.Equal(t, DnsStrategy(""), config.Strategy())
	assert.False(t, config.DisableCache())
	assert.False(t, config.DisableExpire())
	assert.False(t, config.IndependentCache())
	assert.False(t, config.ReverseMapping())
}

func TestNewDnsConfig_EmptyFinal(t *testing.T) {
	_, err := NewDnsConfig("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dns config final server tag is required")
}

func TestDnsConfig_SetServers_Valid(t *testing.T) {
	config, err := NewDnsConfig("remote")
	require.NoError(t, err)

	s1, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
	require.NoError(t, err)
	s1.WithDetour("proxy")

	s2, err := NewDnsServer("local", "223.5.5.5")
	require.NoError(t, err)
	s2.WithDetour("direct")

	err = config.SetServers([]DnsServer{*s1, *s2})
	require.NoError(t, err)

	servers := config.Servers()
	require.Len(t, servers, 2)
	assert.Equal(t, "remote", servers[0].Tag())
	assert.Equal(t, "local", servers[1].Tag())
}

func TestDnsConfig_SetServers_DuplicateTag(t *testing.T) {
	config, err := NewDnsConfig("remote")
	require.NoError(t, err)

	s1, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
	require.NoError(t, err)
	s1.WithDetour("proxy")

	s2, err := NewDnsServer("remote", "223.5.5.5")
	require.NoError(t, err)
	s2.WithDetour("direct")

	err = config.SetServers([]DnsServer{*s1, *s2})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate dns server tag")
}

func TestDnsConfig_SetServers_TooMany(t *testing.T) {
	config, err := NewDnsConfig("s0")
	require.NoError(t, err)

	servers := make([]DnsServer, maxDnsServers+1)
	for i := range servers {
		tag := fmt.Sprintf("s%d", i)
		s, err := NewDnsServer(tag, "1.1.1.1")
		require.NoError(t, err)
		s.WithDetour("direct")
		servers[i] = *s
	}

	err = config.SetServers(servers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many dns servers")
}

func TestDnsConfig_SetRules_Valid(t *testing.T) {
	config, err := NewDnsConfig("remote")
	require.NoError(t, err)

	r1, err := NewDnsRule("local")
	require.NoError(t, err)
	r1.WithGeosite("cn")

	r2, err := NewDnsRule("remote")
	require.NoError(t, err)
	r2.WithDomainSuffix(".com")

	err = config.SetRules([]DnsRule{*r1, *r2})
	require.NoError(t, err)

	rules := config.Rules()
	require.Len(t, rules, 2)
	assert.Equal(t, "local", rules[0].Server())
	assert.Equal(t, "remote", rules[1].Server())
}

func TestDnsConfig_SetRules_InvalidRule(t *testing.T) {
	config, err := NewDnsConfig("remote")
	require.NoError(t, err)

	// Rule without any condition
	r, err := NewDnsRule("local")
	require.NoError(t, err)

	err = config.SetRules([]DnsRule{*r})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dns rule at index 0")
}

func TestDnsConfig_Validate_FinalReferencesUndefinedServer(t *testing.T) {
	config, err := NewDnsConfig("nonexistent")
	require.NoError(t, err)

	s, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
	require.NoError(t, err)
	s.WithDetour("proxy")
	require.NoError(t, config.SetServers([]DnsServer{*s}))

	err = config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dns final references undefined server tag: nonexistent")
}

func TestDnsConfig_Validate_RuleReferencesUndefinedServer(t *testing.T) {
	config, err := NewDnsConfig("remote")
	require.NoError(t, err)

	s, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
	require.NoError(t, err)
	s.WithDetour("proxy")
	require.NoError(t, config.SetServers([]DnsServer{*s}))

	// Manually inject a rule referencing an undefined server.
	// SetRules does not validate server references, but Validate does.
	rule, err := NewDnsRule("nonexistent")
	require.NoError(t, err)
	rule.WithGeosite("cn")
	require.NoError(t, config.SetRules([]DnsRule{*rule}))

	err = config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dns rule at index 0 references undefined server tag: nonexistent")
}

func TestDnsConfig_Validate_AddressResolverSelfReference(t *testing.T) {
	config, err := NewDnsConfig("remote")
	require.NoError(t, err)

	s, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
	require.NoError(t, err)
	s.WithDetour("proxy").WithAddressResolver("remote") // self-reference
	require.NoError(t, config.SetServers([]DnsServer{*s}))

	err = config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot reference itself as addressResolver")
}

func TestDnsConfig_Validate_AddressResolverUndefinedServer(t *testing.T) {
	config, err := NewDnsConfig("remote")
	require.NoError(t, err)

	s, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
	require.NoError(t, err)
	s.WithDetour("proxy").WithAddressResolver("nonexistent")
	require.NoError(t, config.SetServers([]DnsServer{*s}))

	err = config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "addressResolver references undefined server tag: nonexistent")
}

func TestDnsConfig_Validate_DuplicateServerTag(t *testing.T) {
	// Use ReconstructDnsConfig to bypass SetServers duplicate check
	s1 := ReconstructDnsServer("remote", "1.1.1.1", "", "", "", "direct")
	s2 := ReconstructDnsServer("remote", "8.8.8.8", "", "", "", "proxy")

	config := ReconstructDnsConfig(
		[]DnsServer{*s1, *s2},
		nil,
		"remote",
		"",
		false, false, false, false,
	)

	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate dns server tag: remote")
}

func TestDnsConfig_Validate_FullValid(t *testing.T) {
	config := validDnsConfig(t)
	assert.NoError(t, config.Validate())
}

func TestDnsConfig_SetFinal(t *testing.T) {
	config, err := NewDnsConfig("remote")
	require.NoError(t, err)

	err = config.SetFinal("local")
	require.NoError(t, err)
	assert.Equal(t, "local", config.Final())

	// Empty final should fail
	err = config.SetFinal("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dns config final server tag is required")
}

func TestDnsConfig_SetStrategy(t *testing.T) {
	config, err := NewDnsConfig("remote")
	require.NoError(t, err)

	err = config.SetStrategy(DnsStrategyPreferIPv4)
	require.NoError(t, err)
	assert.Equal(t, DnsStrategyPreferIPv4, config.Strategy())

	// Invalid strategy should fail
	err = config.SetStrategy(DnsStrategy("bad"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dns strategy")
}

func TestDnsConfig_BoolSetters(t *testing.T) {
	config, err := NewDnsConfig("remote")
	require.NoError(t, err)

	config.SetDisableCache(true)
	assert.True(t, config.DisableCache())

	config.SetDisableExpire(true)
	assert.True(t, config.DisableExpire())

	config.SetIndependentCache(true)
	assert.True(t, config.IndependentCache())

	config.SetReverseMapping(true)
	assert.True(t, config.ReverseMapping())
}

func TestDnsConfig_Equals(t *testing.T) {
	t.Run("equal configs", func(t *testing.T) {
		c1 := validDnsConfig(t)
		c2 := validDnsConfig(t)
		assert.True(t, c1.Equals(c2))
	})

	t.Run("different final", func(t *testing.T) {
		c1 := validDnsConfig(t)
		c2 := validDnsConfig(t)
		require.NoError(t, c2.SetFinal("local"))
		assert.False(t, c1.Equals(c2))
	})

	t.Run("different strategy", func(t *testing.T) {
		c1 := validDnsConfig(t)
		c2 := validDnsConfig(t)
		require.NoError(t, c2.SetStrategy(DnsStrategyIPv6Only))
		assert.False(t, c1.Equals(c2))
	})

	t.Run("different number of servers", func(t *testing.T) {
		c1 := validDnsConfig(t)

		c2, err := NewDnsConfig("remote")
		require.NoError(t, err)
		s, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
		require.NoError(t, err)
		s.WithDetour("proxy")
		require.NoError(t, c2.SetServers([]DnsServer{*s}))

		assert.False(t, c1.Equals(c2))
	})

	t.Run("different number of rules", func(t *testing.T) {
		c1 := validDnsConfig(t)

		c2 := validDnsConfig(t)
		require.NoError(t, c2.SetRules(nil))
		assert.False(t, c1.Equals(c2))
	})

	t.Run("different bool flags", func(t *testing.T) {
		c1 := validDnsConfig(t)
		c2 := validDnsConfig(t)
		c2.SetDisableCache(true)
		assert.False(t, c1.Equals(c2))
	})

	t.Run("nil comparisons", func(t *testing.T) {
		c := validDnsConfig(t)

		assert.True(t, (*DnsConfig)(nil).Equals(nil))
		assert.False(t, c.Equals(nil))
		assert.False(t, (*DnsConfig)(nil).Equals(c))
	})
}

func TestReconstructDnsConfig(t *testing.T) {
	s1 := ReconstructDnsServer("remote", "https://1.1.1.1/dns-query", "local", DnsStrategyPreferIPv4, DnsStrategyIPv4Only, "proxy")
	s2 := ReconstructDnsServer("local", "223.5.5.5", "", "", "", "direct")

	r1 := ReconstructDnsRule(
		[]string{"example.com"}, nil, nil, nil,
		[]string{"cn"}, nil, nil, nil,
		"local", true,
	)

	config := ReconstructDnsConfig(
		[]DnsServer{*s1, *s2},
		[]DnsRule{*r1},
		"remote",
		DnsStrategyPreferIPv4,
		true, true, true, true,
	)
	require.NotNil(t, config)

	assert.Equal(t, "remote", config.Final())
	assert.Equal(t, DnsStrategyPreferIPv4, config.Strategy())
	assert.True(t, config.DisableCache())
	assert.True(t, config.DisableExpire())
	assert.True(t, config.IndependentCache())
	assert.True(t, config.ReverseMapping())

	servers := config.Servers()
	require.Len(t, servers, 2)
	assert.Equal(t, "remote", servers[0].Tag())
	assert.Equal(t, "https://1.1.1.1/dns-query", servers[0].Address())
	assert.Equal(t, "local", servers[0].AddressResolver())
	assert.Equal(t, DnsStrategyPreferIPv4, servers[0].AddressStrategy())
	assert.Equal(t, DnsStrategyIPv4Only, servers[0].Strategy())
	assert.Equal(t, "proxy", servers[0].Detour())
	assert.Equal(t, "local", servers[1].Tag())

	rules := config.Rules()
	require.Len(t, rules, 1)
	assert.Equal(t, "local", rules[0].Server())
	assert.True(t, rules[0].DisableCache())
	assert.Equal(t, []string{"example.com"}, rules[0].Domain())
	assert.Equal(t, []string{"cn"}, rules[0].Geosite())
}

func TestDnsConfig_Servers_DefensiveCopy(t *testing.T) {
	config := validDnsConfig(t)
	servers := config.Servers()

	// Mutate the returned slice
	servers[0] = DnsServer{}

	// Original should be unchanged
	assert.Equal(t, "remote", config.Servers()[0].Tag())
}

func TestDnsConfig_Rules_DefensiveCopy(t *testing.T) {
	config := validDnsConfig(t)
	rules := config.Rules()

	// Mutate the returned slice
	rules[0] = DnsRule{}

	// Original should be unchanged
	assert.Equal(t, "local", config.Rules()[0].Server())
}

// =============================================================================
// DnsStrategy Tests
// =============================================================================

func TestDnsStrategy_IsValid(t *testing.T) {
	tests := []struct {
		strategy DnsStrategy
		valid    bool
	}{
		{"", true},
		{DnsStrategyPreferIPv4, true},
		{DnsStrategyPreferIPv6, true},
		{DnsStrategyIPv4Only, true},
		{DnsStrategyIPv6Only, true},
		{"invalid", false},
		{"prefer_ipv4_only", false},
	}
	for _, tc := range tests {
		t.Run("strategy_"+string(tc.strategy), func(t *testing.T) {
			assert.Equal(t, tc.valid, tc.strategy.IsValid())
		})
	}
}

func TestDnsStrategy_String(t *testing.T) {
	assert.Equal(t, "prefer_ipv4", DnsStrategyPreferIPv4.String())
	assert.Equal(t, "", DnsStrategy("").String())
}

// =============================================================================
// ReconstructDnsServer Tests
// =============================================================================

func TestReconstructDnsServer(t *testing.T) {
	s := ReconstructDnsServer("remote", "https://1.1.1.1/dns-query", "local", DnsStrategyPreferIPv4, DnsStrategyIPv4Only, "proxy")
	require.NotNil(t, s)

	assert.Equal(t, "remote", s.Tag())
	assert.Equal(t, "https://1.1.1.1/dns-query", s.Address())
	assert.Equal(t, "local", s.AddressResolver())
	assert.Equal(t, DnsStrategyPreferIPv4, s.AddressStrategy())
	assert.Equal(t, DnsStrategyIPv4Only, s.Strategy())
	assert.Equal(t, "proxy", s.Detour())
}

// =============================================================================
// DnsConfig Node Reference Tests
// =============================================================================

func TestDnsConfig_HasNodeReferences(t *testing.T) {
	t.Run("no node references", func(t *testing.T) {
		config := validDnsConfig(t) // servers have detour "proxy" and "direct"
		assert.False(t, config.HasNodeReferences())
	})

	t.Run("with node reference", func(t *testing.T) {
		s := ReconstructDnsServer("remote", "https://1.1.1.1/dns-query", "", "", "", "node_abc123")
		config := ReconstructDnsConfig(
			[]DnsServer{*s}, nil, "remote", "", false, false, false, false,
		)
		assert.True(t, config.HasNodeReferences())
	})

	t.Run("nil config", func(t *testing.T) {
		var config *DnsConfig
		assert.False(t, config.HasNodeReferences())
	})
}

func TestDnsConfig_GetReferencedNodeSIDs(t *testing.T) {
	t.Run("no references", func(t *testing.T) {
		config := validDnsConfig(t)
		assert.Nil(t, config.GetReferencedNodeSIDs())
	})

	t.Run("with references", func(t *testing.T) {
		s1 := ReconstructDnsServer("remote", "1.1.1.1", "", "", "", "node_abc123")
		s2 := ReconstructDnsServer("local", "8.8.8.8", "", "", "", "direct")
		s3 := ReconstructDnsServer("backup", "9.9.9.9", "", "", "", "node_def456")
		config := ReconstructDnsConfig(
			[]DnsServer{*s1, *s2, *s3}, nil, "remote", "", false, false, false, false,
		)
		sids := config.GetReferencedNodeSIDs()
		assert.ElementsMatch(t, []string{"node_abc123", "node_def456"}, sids)
	})

	t.Run("deduplicates", func(t *testing.T) {
		s1 := ReconstructDnsServer("remote", "1.1.1.1", "", "", "", "node_abc123")
		s2 := ReconstructDnsServer("backup", "8.8.8.8", "", "", "", "node_abc123")
		config := ReconstructDnsConfig(
			[]DnsServer{*s1, *s2}, nil, "remote", "", false, false, false, false,
		)
		sids := config.GetReferencedNodeSIDs()
		assert.Equal(t, []string{"node_abc123"}, sids)
	})

	t.Run("nil config", func(t *testing.T) {
		var config *DnsConfig
		assert.Nil(t, config.GetReferencedNodeSIDs())
	})
}

// =============================================================================
// Test Helpers
// =============================================================================

// validDnsConfig creates a complete, valid DnsConfig for testing.
func validDnsConfig(t *testing.T) *DnsConfig {
	t.Helper()
	s1, err := NewDnsServer("remote", "https://1.1.1.1/dns-query")
	require.NoError(t, err)
	s1.WithDetour("proxy")

	s2, err := NewDnsServer("local", "223.5.5.5")
	require.NoError(t, err)
	s2.WithDetour("direct")

	config, err := NewDnsConfig("remote")
	require.NoError(t, err)
	require.NoError(t, config.SetServers([]DnsServer{*s1, *s2}))

	rule, err := NewDnsRule("local")
	require.NoError(t, err)
	rule.WithGeosite("cn")
	require.NoError(t, config.SetRules([]DnsRule{*rule}))

	require.NoError(t, config.Validate())
	return config
}
