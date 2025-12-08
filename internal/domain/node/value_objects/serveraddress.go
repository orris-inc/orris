package value_objects

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`)

type ServerAddress struct {
	value string
}

func NewServerAddress(address string) (ServerAddress, error) {
	// Allow empty address (will use public IP as fallback)
	if address == "" {
		return ServerAddress{}, nil
	}

	address = strings.TrimSpace(address)

	if !isValidIP(address) && !isValidDomain(address) {
		return ServerAddress{}, fmt.Errorf("invalid server address: %s", address)
	}

	return ServerAddress{value: address}, nil
}

func (sa ServerAddress) Value() string {
	return sa.value
}

func (sa ServerAddress) IsIP() bool {
	return isValidIP(sa.value)
}

func (sa ServerAddress) IsDomain() bool {
	return isValidDomain(sa.value)
}

func (sa ServerAddress) Equals(other ServerAddress) bool {
	return sa.value == other.value
}

func isValidIP(address string) bool {
	return net.ParseIP(address) != nil
}

func isValidDomain(domain string) bool {
	if len(domain) > 253 {
		return false
	}

	return domainRegex.MatchString(domain)
}
