package utils

import (
	"fmt"
	"net"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/orris-inc/orris/internal/shared/errors"
)

var validate *validator.Validate

// init initializes the validator
func init() {
	validate = validator.New()

	// Use JSON tag names for validation errors
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// ValidateStruct validates a struct and returns a user-friendly error
func ValidateStruct(s interface{}) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	validationErrors := err.(validator.ValidationErrors)
	if len(validationErrors) == 0 {
		return nil
	}

	// Create a detailed error message
	var errorMessages []string
	for _, fieldError := range validationErrors {
		errorMessages = append(errorMessages, getFieldErrorMessage(fieldError))
	}

	return errors.NewValidationError(
		"Validation failed",
		strings.Join(errorMessages, "; "),
	)
}

// FormatFieldError formats a validation error into a user-friendly message.
// The field parameter allows callers to customize the field name (e.g., snake_case).
func FormatFieldError(field, tag, param string, kind reflect.Kind) string {
	switch tag {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email address"
	case "min":
		if kind == reflect.String {
			return field + " must be at least " + param + " characters long"
		}
		return field + " must be at least " + param
	case "max":
		if kind == reflect.String {
			return field + " must be at most " + param + " characters long"
		}
		return field + " must be at most " + param
	case "len":
		return field + " must be exactly " + param + " characters long"
	case "gt":
		return field + " must be greater than " + param
	case "gte":
		return field + " must be greater than or equal to " + param
	case "lt":
		return field + " must be less than " + param
	case "lte":
		return field + " must be less than or equal to " + param
	case "oneof":
		return field + " must be one of: " + param
	case "uuid":
		return field + " must be a valid UUID"
	case "alphanum":
		return field + " must contain only alphanumeric characters"
	case "alpha":
		return field + " must contain only alphabetic characters"
	case "numeric":
		return field + " must be a valid number"
	case "url":
		return field + " must be a valid URL"
	case "uri":
		return field + " must be a valid URI"
	default:
		return field + " failed validation: " + tag
	}
}

// getFieldErrorMessage returns a user-friendly error message for a field validation error.
func getFieldErrorMessage(fe validator.FieldError) string {
	return FormatFieldError(fe.Field(), fe.Tag(), fe.Param(), fe.Kind())
}

// ValidateID validates that an ID string is not empty and follows expected format
func ValidateID(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.NewValidationError("ID cannot be empty")
	}
	return nil
}

// ValidateServerAddress validates that a server address is a valid IP or domain
// and is not a private/internal network address (SSRF protection).
func ValidateServerAddress(address string) error {
	if address == "" {
		return errors.NewValidationError("server_address is required")
	}

	address = strings.TrimSpace(address)

	// Check if it's a valid IP address
	if ip := parseIP(address); ip != nil {
		if isPrivateOrReservedIP(ip) {
			return errors.NewValidationError("server_address cannot be a private or reserved IP address")
		}
		return nil
	}

	// Check if it's a valid domain
	if !isValidDomain(address) {
		return errors.NewValidationError("server_address must be a valid IP address or domain name")
	}

	// Check for localhost variants
	lowerAddr := strings.ToLower(address)
	if lowerAddr == "localhost" || strings.HasSuffix(lowerAddr, ".localhost") ||
		strings.HasSuffix(lowerAddr, ".local") || strings.HasSuffix(lowerAddr, ".internal") {
		return errors.NewValidationError("server_address cannot be localhost or internal domain")
	}

	return nil
}

// parseIP parses an IP address string, handling IPv4-mapped IPv6 addresses.
func parseIP(address string) net.IP {
	ip := net.ParseIP(address)
	if ip == nil {
		return nil
	}
	// Convert IPv4-mapped IPv6 to IPv4 for consistent checking
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	return ip
}

// isPrivateOrReservedIP checks if an IP address is private, loopback, or reserved.
func isPrivateOrReservedIP(ip net.IP) bool {
	// Loopback (127.0.0.0/8, ::1)
	if ip.IsLoopback() {
		return true
	}

	// Private networks (RFC 1918: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
	if ip.IsPrivate() {
		return true
	}

	// Link-local (169.254.0.0/16, fe80::/10) - includes AWS metadata endpoint
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Unspecified (0.0.0.0, ::)
	if ip.IsUnspecified() {
		return true
	}

	// Check against pre-parsed reserved networks
	for _, network := range reservedNetworks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// isValidDomain checks if a string is a valid domain name.
var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`)

// reservedNetworks contains pre-parsed reserved CIDR networks for efficient lookup.
var reservedNetworks []*net.IPNet

func init() {
	// Parse reserved CIDR ranges once at startup
	cidrs := []string{
		"100.64.0.0/10",   // Carrier-grade NAT (RFC 6598)
		"192.0.0.0/24",    // IETF Protocol Assignments
		"192.0.2.0/24",    // TEST-NET-1
		"198.51.100.0/24", // TEST-NET-2
		"203.0.113.0/24",  // TEST-NET-3
		"224.0.0.0/4",     // Multicast
		"240.0.0.0/4",     // Reserved for future use
	}
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			reservedNetworks = append(reservedNetworks, network)
		}
	}
}

func isValidDomain(domain string) bool {
	if len(domain) > 253 {
		return false
	}
	return domainRegex.MatchString(domain)
}

// ValidateListenPort validates that a port number is in a safe range.
// Excludes system reserved ports (1-1023) and commonly dangerous ports.
func ValidateListenPort(port uint16) error {
	if port == 0 {
		return errors.NewValidationError("listen_port is required")
	}

	// Reject system reserved ports (requires root privileges)
	if port < 1024 {
		return errors.NewValidationError("listen_port must be 1024 or higher (system ports are not allowed)")
	}

	// Reject commonly dangerous/sensitive ports
	dangerousPorts := map[uint16]string{
		3306:  "MySQL",
		5432:  "PostgreSQL",
		6379:  "Redis",
		27017: "MongoDB",
		11211: "Memcached",
	}

	if service, isDangerous := dangerousPorts[port]; isDangerous {
		return errors.NewValidationError(fmt.Sprintf("listen_port %d (%s) is not allowed for security reasons", port, service))
	}

	return nil
}
