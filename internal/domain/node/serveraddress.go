package node

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

type ServerAddress struct {
	value string
}

func NewServerAddress(address string) (*ServerAddress, error) {
	normalized := strings.TrimSpace(address)

	if normalized == "" {
		return nil, fmt.Errorf("server address cannot be empty")
	}

	if !isValidIP(normalized) && !isValidDomain(normalized) {
		return nil, fmt.Errorf("invalid server address: %s", address)
	}

	return &ServerAddress{value: normalized}, nil
}

func (sa *ServerAddress) Value() string {
	return sa.value
}

func (sa *ServerAddress) IsIP() bool {
	return isValidIP(sa.value)
}

func (sa *ServerAddress) IsDomain() bool {
	return isValidDomain(sa.value)
}

func (sa *ServerAddress) IsIPv4() bool {
	ip := net.ParseIP(sa.value)
	return ip != nil && ip.To4() != nil
}

func (sa *ServerAddress) IsIPv6() bool {
	ip := net.ParseIP(sa.value)
	return ip != nil && ip.To4() == nil
}

func (sa *ServerAddress) Equals(other *ServerAddress) bool {
	if sa == nil || other == nil {
		return sa == other
	}
	return sa.value == other.value
}

func isValidIP(address string) bool {
	return net.ParseIP(address) != nil
}

func isValidDomain(address string) bool {
	if len(address) > 253 {
		return false
	}
	return domainRegex.MatchString(address)
}
