package valueobjects

import "testing"

// TestForwardRuleType_IsValid tests the IsValid method for all rule types.
func TestForwardRuleType_IsValid(t *testing.T) {
	testCases := []struct {
		name     string
		ruleType ForwardRuleType
		want     bool
	}{
		{"direct is valid", ForwardRuleTypeDirect, true},
		{"entry is valid", ForwardRuleTypeEntry, true},
		{"chain is valid", ForwardRuleTypeChain, true},
		{"direct_chain is valid", ForwardRuleTypeDirectChain, true},
		{"empty string is invalid", ForwardRuleType(""), false},
		{"unknown type is invalid", ForwardRuleType("unknown"), false},
		{"invalid type is invalid", ForwardRuleType("invalid"), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ruleType.IsValid()
			if got != tc.want {
				t.Errorf("IsValid() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardRuleType_IsDirect tests the IsDirect predicate.
func TestForwardRuleType_IsDirect(t *testing.T) {
	testCases := []struct {
		name     string
		ruleType ForwardRuleType
		want     bool
	}{
		{"direct returns true", ForwardRuleTypeDirect, true},
		{"entry returns false", ForwardRuleTypeEntry, false},
		{"chain returns false", ForwardRuleTypeChain, false},
		{"direct_chain returns false", ForwardRuleTypeDirectChain, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ruleType.IsDirect()
			if got != tc.want {
				t.Errorf("IsDirect() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardRuleType_IsEntry tests the IsEntry predicate.
func TestForwardRuleType_IsEntry(t *testing.T) {
	testCases := []struct {
		name     string
		ruleType ForwardRuleType
		want     bool
	}{
		{"entry returns true", ForwardRuleTypeEntry, true},
		{"direct returns false", ForwardRuleTypeDirect, false},
		{"chain returns false", ForwardRuleTypeChain, false},
		{"direct_chain returns false", ForwardRuleTypeDirectChain, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ruleType.IsEntry()
			if got != tc.want {
				t.Errorf("IsEntry() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardRuleType_IsChain tests the IsChain predicate.
func TestForwardRuleType_IsChain(t *testing.T) {
	testCases := []struct {
		name     string
		ruleType ForwardRuleType
		want     bool
	}{
		{"chain returns true", ForwardRuleTypeChain, true},
		{"direct returns false", ForwardRuleTypeDirect, false},
		{"entry returns false", ForwardRuleTypeEntry, false},
		{"direct_chain returns false", ForwardRuleTypeDirectChain, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ruleType.IsChain()
			if got != tc.want {
				t.Errorf("IsChain() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardRuleType_IsDirectChain tests the IsDirectChain predicate.
func TestForwardRuleType_IsDirectChain(t *testing.T) {
	testCases := []struct {
		name     string
		ruleType ForwardRuleType
		want     bool
	}{
		{"direct_chain returns true", ForwardRuleTypeDirectChain, true},
		{"direct returns false", ForwardRuleTypeDirect, false},
		{"entry returns false", ForwardRuleTypeEntry, false},
		{"chain returns false", ForwardRuleTypeChain, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ruleType.IsDirectChain()
			if got != tc.want {
				t.Errorf("IsDirectChain() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardRuleType_RequiresChainAgents tests the RequiresChainAgents logic.
func TestForwardRuleType_RequiresChainAgents(t *testing.T) {
	testCases := []struct {
		name     string
		ruleType ForwardRuleType
		want     bool
	}{
		{"chain requires chain agents", ForwardRuleTypeChain, true},
		{"direct_chain requires chain agents", ForwardRuleTypeDirectChain, true},
		{"direct does not require chain agents", ForwardRuleTypeDirect, false},
		{"entry does not require chain agents", ForwardRuleTypeEntry, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ruleType.RequiresChainAgents()
			if got != tc.want {
				t.Errorf("RequiresChainAgents() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardRuleType_RequiresChainPortConfig tests the RequiresChainPortConfig logic.
func TestForwardRuleType_RequiresChainPortConfig(t *testing.T) {
	testCases := []struct {
		name     string
		ruleType ForwardRuleType
		want     bool
	}{
		{"direct_chain requires port config", ForwardRuleTypeDirectChain, true},
		{"chain does not require port config", ForwardRuleTypeChain, false},
		{"direct does not require port config", ForwardRuleTypeDirect, false},
		{"entry does not require port config", ForwardRuleTypeEntry, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ruleType.RequiresChainPortConfig()
			if got != tc.want {
				t.Errorf("RequiresChainPortConfig() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardRuleType_RequiresExitAgent tests the RequiresExitAgent logic.
func TestForwardRuleType_RequiresExitAgent(t *testing.T) {
	testCases := []struct {
		name     string
		ruleType ForwardRuleType
		want     bool
	}{
		{"entry requires exit agent", ForwardRuleTypeEntry, true},
		{"direct does not require exit agent", ForwardRuleTypeDirect, false},
		{"chain does not require exit agent", ForwardRuleTypeChain, false},
		{"direct_chain does not require exit agent", ForwardRuleTypeDirectChain, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ruleType.RequiresExitAgent()
			if got != tc.want {
				t.Errorf("RequiresExitAgent() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardRuleType_String tests the String method.
func TestForwardRuleType_String(t *testing.T) {
	testCases := []struct {
		name     string
		ruleType ForwardRuleType
		want     string
	}{
		{"direct to string", ForwardRuleTypeDirect, "direct"},
		{"entry to string", ForwardRuleTypeEntry, "entry"},
		{"chain to string", ForwardRuleTypeChain, "chain"},
		{"direct_chain to string", ForwardRuleTypeDirectChain, "direct_chain"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ruleType.String()
			if got != tc.want {
				t.Errorf("String() = %v, want %v", got, tc.want)
			}
		})
	}
}
