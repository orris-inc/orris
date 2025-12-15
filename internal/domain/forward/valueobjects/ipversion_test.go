package valueobjects

import "testing"

// TestIPVersion_IsValid tests the IsValid method for all IP versions.
func TestIPVersion_IsValid(t *testing.T) {
	testCases := []struct {
		name    string
		version IPVersion
		want    bool
	}{
		{"auto is valid", IPVersionAuto, true},
		{"ipv4 is valid", IPVersionIPv4, true},
		{"ipv6 is valid", IPVersionIPv6, true},
		{"empty string is invalid", IPVersion(""), false},
		{"unknown version is invalid", IPVersion("unknown"), false},
		{"invalid version is invalid", IPVersion("ipv5"), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.version.IsValid()
			if got != tc.want {
				t.Errorf("IsValid() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIPVersion_IsAuto tests the IsAuto predicate.
func TestIPVersion_IsAuto(t *testing.T) {
	testCases := []struct {
		name    string
		version IPVersion
		want    bool
	}{
		{"auto returns true", IPVersionAuto, true},
		{"ipv4 returns false", IPVersionIPv4, false},
		{"ipv6 returns false", IPVersionIPv6, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.version.IsAuto()
			if got != tc.want {
				t.Errorf("IsAuto() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIPVersion_IsIPv4 tests the IsIPv4 predicate.
func TestIPVersion_IsIPv4(t *testing.T) {
	testCases := []struct {
		name    string
		version IPVersion
		want    bool
	}{
		{"ipv4 returns true", IPVersionIPv4, true},
		{"auto returns false", IPVersionAuto, false},
		{"ipv6 returns false", IPVersionIPv6, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.version.IsIPv4()
			if got != tc.want {
				t.Errorf("IsIPv4() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIPVersion_IsIPv6 tests the IsIPv6 predicate.
func TestIPVersion_IsIPv6(t *testing.T) {
	testCases := []struct {
		name    string
		version IPVersion
		want    bool
	}{
		{"ipv6 returns true", IPVersionIPv6, true},
		{"auto returns false", IPVersionAuto, false},
		{"ipv4 returns false", IPVersionIPv4, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.version.IsIPv6()
			if got != tc.want {
				t.Errorf("IsIPv6() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIPVersion_String tests the String method.
func TestIPVersion_String(t *testing.T) {
	testCases := []struct {
		name    string
		version IPVersion
		want    string
	}{
		{"auto to string", IPVersionAuto, "auto"},
		{"ipv4 to string", IPVersionIPv4, "ipv4"},
		{"ipv6 to string", IPVersionIPv6, "ipv6"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.version.String()
			if got != tc.want {
				t.Errorf("String() = %v, want %v", got, tc.want)
			}
		})
	}
}
