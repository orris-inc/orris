package dto

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/domain/shared/routing"
	"github.com/orris-inc/orris/internal/shared/id"
)

// newTestRuleWithRuleSet builds a direct forward rule whose RouteConfig has a
// single route rule referencing the given rule-set tag and one rule-set entry
// mapping that tag to the given URL.
func newTestRuleWithRuleSet(t *testing.T, listenPort uint16, tag, url string) *forward.ForwardRule {
	t.Helper()

	rule, err := forward.NewForwardRule(
		1,   // agentID
		nil, // userID
		nil, // subscriptionID
		vo.ForwardRuleTypeDirect,
		0,   // exitAgentID
		nil, // exitAgents
		vo.ParseLoadBalanceStrategy(""),
		nil, // chainAgentIDs
		nil, // chainPortConfig
		nil, // tunnelHops
		vo.TunnelType(""),
		"test-rule",
		listenPort,
		"1.2.3.4", // targetAddress
		443,       // targetPort
		nil,       // targetNodeID
		"",        // bindIP
		vo.IPVersionAuto,
		vo.ForwardProtocol("tcp"),
		"", // remark
		nil,
		0,
		vo.AddressPreferenceAuto,
		id.NewForwardRuleID,
	)
	require.NoError(t, err)

	rc, err := routing.NewRouteConfig(routing.OutboundProxy)
	require.NoError(t, err)

	rr, err := routing.NewRouteRule(routing.OutboundProxy)
	require.NoError(t, err)
	rr.WithRuleSet(tag)
	require.NoError(t, rc.AddRule(*rr))

	entry, err := routing.NewRuleSetEntry(tag, url, routing.RuleSetFormatBinary, "", "1d")
	require.NoError(t, err)
	require.NoError(t, rc.SetRuleSetEntries([]routing.RuleSetEntry{*entry}))

	require.NoError(t, rule.UpdateRouteConfig(rc))
	return rule
}

// findRuleSetURL returns the URL registered for a given rule-set tag, or "".
func findRuleSetURL(entries []RuleSetEntryDTO, tag string) string {
	for _, e := range entries {
		if e.Tag == tag {
			return e.URL
		}
	}
	return ""
}

// firstInboundRuleRuleSet returns the RuleSet slice of the first route rule
// whose inbound matches the given rule SID.
func firstInboundRuleRuleSet(rules []RouteRuleDTO, ruleSID string) []string {
	for _, r := range rules {
		if len(r.Inbound) == 1 && r.Inbound[0] == ruleSID && len(r.RuleSet) > 0 {
			return r.RuleSet
		}
	}
	return nil
}

// TestMergeForwardRuleRoutes_SharedRuleSetTag verifies that two forward rules
// using the same rule-set tag with the same URL share a single rule-set entry
// and both keep the original tag reference.
func TestMergeForwardRuleRoutes_SharedRuleSetTag(t *testing.T) {
	const url = "https://example.com/geoip-cn.srs"
	r1 := newTestRuleWithRuleSet(t, 10001, "geoip-cn", url)
	r2 := newTestRuleWithRuleSet(t, 10002, "geoip-cn", url)

	var route *RouteConfigDTO
	var outbounds []OutboundDTO
	var frRoutes []ForwardRuleRouteDTO
	mergeForwardRuleRoutesCore(&route, &outbounds, &frRoutes, []*forward.ForwardRule{r1, r2})

	require.NotNil(t, route)
	// Only one shared entry should exist.
	count := 0
	for _, e := range route.RuleSetEntries {
		if e.Tag == "geoip-cn" {
			count++
		}
	}
	require.Equal(t, 1, count, "identical tag+url must be shared, not duplicated")

	// Both rules keep the original (un-namespaced) reference.
	require.Equal(t, []string{"geoip-cn"}, firstInboundRuleRuleSet(route.Rules, r1.SID()))
	require.Equal(t, []string{"geoip-cn"}, firstInboundRuleRuleSet(route.Rules, r2.SID()))
}

// TestMergeForwardRuleRoutes_ConflictingRuleSetTag verifies that when two
// forward rules use the same rule-set tag with different URLs, the second one
// is namespaced per rule and its route-rule references are rewritten so each
// rule resolves to the correct rule-set.
func TestMergeForwardRuleRoutes_ConflictingRuleSetTag(t *testing.T) {
	const urlA = "https://example.com/a.srs"
	const urlB = "https://example.com/b.srs"
	r1 := newTestRuleWithRuleSet(t, 10001, "myset", urlA)
	r2 := newTestRuleWithRuleSet(t, 10002, "myset", urlB)

	var route *RouteConfigDTO
	var outbounds []OutboundDTO
	var frRoutes []ForwardRuleRouteDTO
	mergeForwardRuleRoutesCore(&route, &outbounds, &frRoutes, []*forward.ForwardRule{r1, r2})

	require.NotNil(t, route)

	nsTag := r2.SID() + ":myset"
	// First rule keeps original tag/url; second rule gets a namespaced entry.
	require.Equal(t, urlA, findRuleSetURL(route.RuleSetEntries, "myset"))
	require.Equal(t, urlB, findRuleSetURL(route.RuleSetEntries, nsTag),
		"conflicting tag with different url must be namespaced per forward rule")

	// Each rule's route-rule references its own resolved rule-set tag.
	require.Equal(t, []string{"myset"}, firstInboundRuleRuleSet(route.Rules, r1.SID()))
	require.Equal(t, []string{nsTag}, firstInboundRuleRuleSet(route.Rules, r2.SID()),
		"second rule's reference must be rewritten to the namespaced tag")
}
