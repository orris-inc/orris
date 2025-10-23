package node

import (
	"strings"
	"testing"
)

func TestNewServerAddress_ValidIPv4(t *testing.T) {
	tests := []struct {
		name    string
		address string
	}{
		{"standard IPv4", "192.168.1.1"},
		{"loopback", "127.0.0.1"},
		{"public IPv4", "8.8.8.8"},
		{"with whitespace", "  10.0.0.1  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := NewServerAddress(tt.address)
			if err != nil {
				t.Errorf("NewServerAddress(%q) error = %v, want nil", tt.address, err)
				return
			}
			if !addr.IsIP() {
				t.Errorf("IsIP() = false, want true")
			}
			if !addr.IsIPv4() {
				t.Errorf("IsIPv4() = false, want true")
			}
			if addr.IsIPv6() {
				t.Errorf("IsIPv6() = true, want false")
			}
			if addr.IsDomain() {
				t.Errorf("IsDomain() = true, want false")
			}
		})
	}
}

func TestNewServerAddress_ValidIPv6(t *testing.T) {
	tests := []struct {
		name    string
		address string
	}{
		{"full IPv6", "2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
		{"compressed IPv6", "2001:db8::1"},
		{"loopback", "::1"},
		{"with whitespace", "  fe80::1  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := NewServerAddress(tt.address)
			if err != nil {
				t.Errorf("NewServerAddress(%q) error = %v, want nil", tt.address, err)
				return
			}
			if !addr.IsIP() {
				t.Errorf("IsIP() = false, want true")
			}
			if !addr.IsIPv6() {
				t.Errorf("IsIPv6() = false, want true")
			}
			if addr.IsIPv4() {
				t.Errorf("IsIPv4() = true, want false")
			}
			if addr.IsDomain() {
				t.Errorf("IsDomain() = true, want false")
			}
		})
	}
}

func TestNewServerAddress_ValidDomain(t *testing.T) {
	tests := []struct {
		name   string
		domain string
	}{
		{"standard domain", "example.com"},
		{"subdomain", "api.example.com"},
		{"multi-level subdomain", "v1.api.example.com"},
		{"with hyphen", "my-server.example.com"},
		{"long TLD", "example.online"},
		{"with whitespace", "  example.com  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := NewServerAddress(tt.domain)
			if err != nil {
				t.Errorf("NewServerAddress(%q) error = %v, want nil", tt.domain, err)
				return
			}
			if !addr.IsDomain() {
				t.Errorf("IsDomain() = false, want true")
			}
			if addr.IsIP() {
				t.Errorf("IsIP() = true, want false")
			}
			if addr.IsIPv4() {
				t.Errorf("IsIPv4() = true, want false")
			}
			if addr.IsIPv6() {
				t.Errorf("IsIPv6() = true, want false")
			}
		})
	}
}

func TestNewServerAddress_InvalidAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"invalid IP", "999.999.999.999"},
		{"invalid domain - no TLD", "example"},
		{"invalid domain - starts with hyphen", "-example.com"},
		{"invalid domain - ends with hyphen", "example-.com"},
		{"invalid domain - too long", "a" + string(make([]byte, 300)) + ".com"},
		{"invalid characters", "exam ple.com"},
		{"just a dot", "."},
		{"double dots", "example..com"},
		{"starts with dot", ".example.com"},
		{"ends with dot", "example.com."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewServerAddress(tt.address)
			if err == nil {
				t.Errorf("NewServerAddress(%q) error = nil, want error", tt.address)
			}
		})
	}
}

func TestServerAddress_Value(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"IPv4", "192.168.1.1", "192.168.1.1"},
		{"domain", "example.com", "example.com"},
		{"trimmed whitespace", "  example.com  ", "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := NewServerAddress(tt.input)
			if err != nil {
				t.Fatalf("NewServerAddress(%q) error = %v", tt.input, err)
			}
			if addr.Value() != tt.expected {
				t.Errorf("Value() = %q, want %q", addr.Value(), tt.expected)
			}
		})
	}
}

func TestServerAddress_Equals(t *testing.T) {
	addr1, _ := NewServerAddress("192.168.1.1")
	addr2, _ := NewServerAddress("192.168.1.1")
	addr3, _ := NewServerAddress("192.168.1.2")
	addr4, _ := NewServerAddress("example.com")

	tests := []struct {
		name     string
		addr1    *ServerAddress
		addr2    *ServerAddress
		expected bool
	}{
		{"same IP addresses", addr1, addr2, true},
		{"different IP addresses", addr1, addr3, false},
		{"IP vs domain", addr1, addr4, false},
		{"nil equals nil", nil, nil, true},
		{"nil vs non-nil", nil, addr1, false},
		{"non-nil vs nil", addr1, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.addr1.Equals(tt.addr2)
			if result != tt.expected {
				t.Errorf("Equals() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestServerAddress_BoundaryConditions(t *testing.T) {
	t.Run("domain max length 253", func(t *testing.T) {
		validDomain := strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 60) + ".com"

		if len(validDomain) <= 253 {
			_, err := NewServerAddress(validDomain)
			if err != nil {
				t.Errorf("NewServerAddress() with %d char domain should succeed, got error: %v", len(validDomain), err)
			}
		}
	})

	t.Run("domain exceeds max length", func(t *testing.T) {
		longDomain := strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 63) + "." + strings.Repeat("e", 63) + ".com"

		_, err := NewServerAddress(longDomain)
		if err == nil {
			t.Error("NewServerAddress() with domain > 253 chars should fail")
		}
	})

	t.Run("single char labels", func(t *testing.T) {
		_, err := NewServerAddress("a.b.com")
		if err != nil {
			t.Errorf("NewServerAddress() with single char labels should succeed, got: %v", err)
		}
	})
}
