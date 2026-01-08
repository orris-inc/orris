// Package valueobjects provides value objects for the forward domain.
package valueobjects

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

// PortRangeEntry represents a single port range entry
type PortRangeEntry struct {
	Start uint16 `json:"start"`
	End   uint16 `json:"end"`
}

// PortRange represents allowed port ranges for an agent.
// nil or empty ranges means all ports are allowed.
type PortRange struct {
	Ranges []PortRangeEntry `json:"ranges"`
}

// IsEmpty returns true if no port ranges are configured
func (p *PortRange) IsEmpty() bool {
	return p == nil || len(p.Ranges) == 0
}

// Contains checks if a port is within the allowed ranges.
// Returns true if the port range is empty (all ports allowed).
func (p *PortRange) Contains(port uint16) bool {
	if p.IsEmpty() {
		return true
	}

	for _, r := range p.Ranges {
		if port >= r.Start && port <= r.End {
			return true
		}
	}
	return false
}

// TotalPorts returns the total number of ports in all ranges.
// Returns 0 if the port range is empty (meaning all ports allowed, caller should handle this case).
func (p *PortRange) TotalPorts() int {
	if p.IsEmpty() {
		return 0
	}

	total := 0
	for _, r := range p.Ranges {
		total += int(r.End-r.Start) + 1
	}
	return total
}

// RandomPort returns a random port from the allowed ranges.
// Returns 0 if the port range is empty (caller should use default range).
func (p *PortRange) RandomPort() uint16 {
	if p.IsEmpty() {
		return 0
	}

	total := p.TotalPorts()
	if total == 0 {
		return 0
	}

	// Pick a random index within total available ports
	idx := rand.Intn(total)

	// Find which range and which port within that range
	for _, r := range p.Ranges {
		rangeSize := int(r.End-r.Start) + 1
		if idx < rangeSize {
			// Safe conversion: idx is guaranteed to be non-negative (from rand.Intn) and
			// bounded by rangeSize which is <= 65535, so uint16 conversion is safe
			// #nosec G115 -- idx is bounded by port range size (max 65535)
			return r.Start + uint16(idx)
		}
		idx -= rangeSize
	}

	// Should never reach here, but return first port as fallback
	return p.Ranges[0].Start
}

// Validate performs validation on the port range configuration
func (p *PortRange) Validate() error {
	if p.IsEmpty() {
		return nil
	}

	for i, r := range p.Ranges {
		if r.Start == 0 {
			return fmt.Errorf("port range %d: start port cannot be 0", i+1)
		}
		if r.End == 0 {
			return fmt.Errorf("port range %d: end port cannot be 0", i+1)
		}
		if r.Start > r.End {
			return fmt.Errorf("port range %d: start port (%d) cannot be greater than end port (%d)", i+1, r.Start, r.End)
		}
	}

	return nil
}

// ParsePortRange parses port range from string format "80,443,8000-9000"
func ParsePortRange(s string) (*PortRange, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	ranges := make([]PortRangeEntry, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if it's a range (e.g., "8000-9000")
		if strings.Contains(part, "-") {
			rangeParts := strings.SplitN(part, "-", 2)
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid port range format: %s", part)
			}

			start, err := parsePort(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start port in range %s: %w", part, err)
			}

			end, err := parsePort(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid end port in range %s: %w", part, err)
			}

			if start > end {
				return nil, fmt.Errorf("invalid port range %s: start (%d) cannot be greater than end (%d)", part, start, end)
			}

			ranges = append(ranges, PortRangeEntry{Start: start, End: end})
		} else {
			// Single port
			port, err := parsePort(part)
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s: %w", part, err)
			}

			ranges = append(ranges, PortRangeEntry{Start: port, End: port})
		}
	}

	if len(ranges) == 0 {
		// Input was non-empty but no valid ports were found (e.g., "," or ",,")
		// This is likely a user error, return an error instead of nil
		return nil, fmt.Errorf("no valid ports found in input")
	}

	pr := &PortRange{Ranges: ranges}
	if err := pr.Validate(); err != nil {
		return nil, err
	}

	return pr, nil
}

// parsePort parses a port string to uint16
func parsePort(s string) (uint16, error) {
	port, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid port number")
	}
	if port == 0 {
		return 0, fmt.Errorf("port cannot be 0")
	}
	return uint16(port), nil
}

// String returns the string representation "80,443,8000-9000"
func (p *PortRange) String() string {
	if p.IsEmpty() {
		return ""
	}

	// Sort ranges by start port for consistent output
	sorted := make([]PortRangeEntry, len(p.Ranges))
	copy(sorted, p.Ranges)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Start < sorted[j].Start
	})

	parts := make([]string, 0, len(sorted))
	for _, r := range sorted {
		if r.Start == r.End {
			parts = append(parts, strconv.FormatUint(uint64(r.Start), 10))
		} else {
			parts = append(parts, fmt.Sprintf("%d-%d", r.Start, r.End))
		}
	}

	return strings.Join(parts, ",")
}

// MarshalJSON implements json.Marshaler
func (p *PortRange) MarshalJSON() ([]byte, error) {
	if p.IsEmpty() {
		return []byte("null"), nil
	}
	type alias PortRange
	return json.Marshal((*alias)(p))
}

// UnmarshalJSON implements json.Unmarshaler
func (p *PortRange) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		p.Ranges = nil
		return nil
	}

	type alias PortRange
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	p.Ranges = a.Ranges
	return nil
}
