package valueobjects

import "testing"

// TestForwardProtocol_IsValid tests the IsValid method for all protocols.
func TestForwardProtocol_IsValid(t *testing.T) {
	testCases := []struct {
		name     string
		protocol ForwardProtocol
		want     bool
	}{
		{"tcp is valid", ForwardProtocolTCP, true},
		{"udp is valid", ForwardProtocolUDP, true},
		{"both is valid", ForwardProtocolBoth, true},
		{"empty string is invalid", ForwardProtocol(""), false},
		{"unknown protocol is invalid", ForwardProtocol("unknown"), false},
		{"invalid protocol is invalid", ForwardProtocol("icmp"), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.protocol.IsValid()
			if got != tc.want {
				t.Errorf("IsValid() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardProtocol_IsTCP tests the IsTCP predicate.
// Business rule: TCP is supported for both "tcp" and "both" protocols.
func TestForwardProtocol_IsTCP(t *testing.T) {
	testCases := []struct {
		name     string
		protocol ForwardProtocol
		want     bool
	}{
		{"tcp returns true", ForwardProtocolTCP, true},
		{"both returns true", ForwardProtocolBoth, true},
		{"udp returns false", ForwardProtocolUDP, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.protocol.IsTCP()
			if got != tc.want {
				t.Errorf("IsTCP() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardProtocol_IsUDP tests the IsUDP predicate.
// Business rule: UDP is supported for both "udp" and "both" protocols.
func TestForwardProtocol_IsUDP(t *testing.T) {
	testCases := []struct {
		name     string
		protocol ForwardProtocol
		want     bool
	}{
		{"udp returns true", ForwardProtocolUDP, true},
		{"both returns true", ForwardProtocolBoth, true},
		{"tcp returns false", ForwardProtocolTCP, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.protocol.IsUDP()
			if got != tc.want {
				t.Errorf("IsUDP() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardProtocol_IsBoth tests the IsBoth predicate.
func TestForwardProtocol_IsBoth(t *testing.T) {
	testCases := []struct {
		name     string
		protocol ForwardProtocol
		want     bool
	}{
		{"both returns true", ForwardProtocolBoth, true},
		{"tcp returns false", ForwardProtocolTCP, false},
		{"udp returns false", ForwardProtocolUDP, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.protocol.IsBoth()
			if got != tc.want {
				t.Errorf("IsBoth() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestForwardProtocol_String tests the String method.
func TestForwardProtocol_String(t *testing.T) {
	testCases := []struct {
		name     string
		protocol ForwardProtocol
		want     string
	}{
		{"tcp to string", ForwardProtocolTCP, "tcp"},
		{"udp to string", ForwardProtocolUDP, "udp"},
		{"both to string", ForwardProtocolBoth, "both"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.protocol.String()
			if got != tc.want {
				t.Errorf("String() = %v, want %v", got, tc.want)
			}
		})
	}
}
