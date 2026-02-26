package valueobjects

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// RuleSetFormat defines the format of a remote rule-set resource.
type RuleSetFormat string

const (
	RuleSetFormatBinary RuleSetFormat = "binary"
	RuleSetFormatSource RuleSetFormat = "source"
)

// IsValid checks if the format is a recognized rule-set format.
func (f RuleSetFormat) IsValid() bool {
	switch f {
	case RuleSetFormatBinary, RuleSetFormatSource:
		return true
	default:
		return false
	}
}

// String returns the string representation of the format.
func (f RuleSetFormat) String() string { return string(f) }

// ruleSetTagPattern validates a rule-set tag: alphanumeric, hyphens, underscores.
var ruleSetTagPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// maxRuleSetEntries is the maximum number of rule-set entries per route config.
const maxRuleSetEntries = 50

// RuleSetEntry represents a remote rule-set source for sing-box route configuration.
// Each entry defines a tag that route/DNS rules can reference via the rule_set field.
type RuleSetEntry struct {
	tag            string        // Unique identifier referenced by rules
	url            string        // Remote URL of the rule-set resource
	format         RuleSetFormat // Resource format (binary or source)
	downloadDetour string        // Optional outbound tag for downloading
	updateInterval string        // Optional update interval (e.g. "1d", "12h")
}

// NewRuleSetEntry creates a new RuleSetEntry with validation.
func NewRuleSetEntry(tag, rawURL string, format RuleSetFormat, downloadDetour, updateInterval string) (*RuleSetEntry, error) {
	e := &RuleSetEntry{
		tag:            tag,
		url:            rawURL,
		format:         format,
		downloadDetour: downloadDetour,
		updateInterval: updateInterval,
	}
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return e, nil
}

// Getters

func (e *RuleSetEntry) Tag() string            { return e.tag }
func (e *RuleSetEntry) URL() string            { return e.url }
func (e *RuleSetEntry) Format() RuleSetFormat  { return e.format }
func (e *RuleSetEntry) DownloadDetour() string { return e.downloadDetour }
func (e *RuleSetEntry) UpdateInterval() string { return e.updateInterval }

// updateIntervalPattern matches valid duration strings like "1d", "12h", "30m", "1d12h".
var updateIntervalPattern = regexp.MustCompile(`^(\d+[dhm])+$`)

// Validate checks all fields for correctness.
func (e *RuleSetEntry) Validate() error {
	// Tag: required, pattern
	if e.tag == "" {
		return fmt.Errorf("rule-set entry tag is required")
	}
	if len(e.tag) > 128 {
		return fmt.Errorf("rule-set entry tag too long: max 128 characters")
	}
	if !ruleSetTagPattern.MatchString(e.tag) {
		return fmt.Errorf("rule-set entry tag contains invalid characters: %s (only alphanumeric, hyphens, underscores allowed, must start with alphanumeric)", e.tag)
	}

	// URL: required, valid HTTP(S)
	if e.url == "" {
		return fmt.Errorf("rule-set entry URL is required")
	}
	if len(e.url) > 2048 {
		return fmt.Errorf("rule-set entry URL too long: max 2048 characters")
	}
	parsed, err := url.Parse(e.url)
	if err != nil {
		return fmt.Errorf("rule-set entry URL is invalid: %w", err)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("rule-set entry URL must use http or https scheme, got: %s", parsed.Scheme)
	}

	// Format: must be valid
	if !e.format.IsValid() {
		return fmt.Errorf("rule-set entry format must be 'binary' or 'source', got: %s", e.format)
	}

	// DownloadDetour: optional, max length
	if len(e.downloadDetour) > 128 {
		return fmt.Errorf("rule-set entry download_detour too long: max 128 characters")
	}

	// UpdateInterval: optional, pattern
	if e.updateInterval != "" {
		if !updateIntervalPattern.MatchString(e.updateInterval) {
			return fmt.Errorf("rule-set entry update_interval format invalid: %s (expected e.g. '1d', '12h', '30m')", e.updateInterval)
		}
	}

	return nil
}

// Equals compares two RuleSetEntry instances for equality.
func (e *RuleSetEntry) Equals(other *RuleSetEntry) bool {
	if e == nil && other == nil {
		return true
	}
	if e == nil || other == nil {
		return false
	}
	return e.tag == other.tag &&
		e.url == other.url &&
		e.format == other.format &&
		e.downloadDetour == other.downloadDetour &&
		e.updateInterval == other.updateInterval
}

// ReconstructRuleSetEntry rebuilds a RuleSetEntry from persistence data without validation.
func ReconstructRuleSetEntry(tag, rawURL string, format RuleSetFormat, downloadDetour, updateInterval string) *RuleSetEntry {
	return &RuleSetEntry{
		tag:            tag,
		url:            rawURL,
		format:         format,
		downloadDetour: downloadDetour,
		updateInterval: updateInterval,
	}
}
