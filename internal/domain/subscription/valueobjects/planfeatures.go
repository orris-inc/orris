package valueobjects

import (
	"encoding/json"
	"fmt"
)

type PlanFeatures struct {
	Features []string               `json:"features"`
	Limits   map[string]interface{} `json:"limits"`
}

func NewPlanFeatures(features []string, limits map[string]interface{}) *PlanFeatures {
	if features == nil {
		features = []string{}
	}
	if limits == nil {
		limits = make(map[string]interface{})
	}

	return &PlanFeatures{
		Features: features,
		Limits:   limits,
	}
}

func (p *PlanFeatures) HasFeature(feature string) bool {
	if p.Features == nil {
		return false
	}

	for _, f := range p.Features {
		if f == feature {
			return true
		}
	}

	return false
}

func (p *PlanFeatures) GetLimit(key string) (interface{}, bool) {
	if p.Limits == nil {
		return nil, false
	}

	value, exists := p.Limits[key]
	return value, exists
}

func (p *PlanFeatures) IsWithinLimit(key string, value interface{}) bool {
	limit, exists := p.GetLimit(key)
	if !exists {
		return true
	}

	switch limitValue := limit.(type) {
	case int:
		if intValue, ok := value.(int); ok {
			return intValue <= limitValue
		}
	case float64:
		if floatValue, ok := value.(float64); ok {
			return floatValue <= limitValue
		}
		if intValue, ok := value.(int); ok {
			return float64(intValue) <= limitValue
		}
	case string:
		if strValue, ok := value.(string); ok {
			return strValue == limitValue
		}
	case bool:
		if boolValue, ok := value.(bool); ok {
			return boolValue == limitValue
		}
	}

	return false
}

func (p *PlanFeatures) AddFeature(feature string) {
	if p.Features == nil {
		p.Features = []string{}
	}

	if !p.HasFeature(feature) {
		p.Features = append(p.Features, feature)
	}
}

func (p *PlanFeatures) RemoveFeature(feature string) {
	if p.Features == nil {
		return
	}

	for i, f := range p.Features {
		if f == feature {
			p.Features = append(p.Features[:i], p.Features[i+1:]...)
			return
		}
	}
}

func (p *PlanFeatures) SetLimit(key string, value interface{}) {
	if p.Limits == nil {
		p.Limits = make(map[string]interface{})
	}

	p.Limits[key] = value
}

func (p *PlanFeatures) RemoveLimit(key string) {
	if p.Limits == nil {
		return
	}

	delete(p.Limits, key)
}

func (p *PlanFeatures) GetIntLimit(key string) (int, error) {
	value, exists := p.GetLimit(key)
	if !exists {
		return 0, fmt.Errorf("limit not found: %s", key)
	}

	switch v := value.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("limit is not an integer: %s", key)
	}
}

func (p *PlanFeatures) GetStringLimit(key string) (string, error) {
	value, exists := p.GetLimit(key)
	if !exists {
		return "", fmt.Errorf("limit not found: %s", key)
	}

	if strValue, ok := value.(string); ok {
		return strValue, nil
	}

	return "", fmt.Errorf("limit is not a string: %s", key)
}

func (p *PlanFeatures) GetBoolLimit(key string) (bool, error) {
	value, exists := p.GetLimit(key)
	if !exists {
		return false, fmt.Errorf("limit not found: %s", key)
	}

	if boolValue, ok := value.(bool); ok {
		return boolValue, nil
	}

	return false, fmt.Errorf("limit is not a boolean: %s", key)
}

func (p *PlanFeatures) Clone() *PlanFeatures {
	features := make([]string, len(p.Features))
	copy(features, p.Features)

	limits := make(map[string]interface{})
	for k, v := range p.Limits {
		limits[k] = v
	}

	return &PlanFeatures{
		Features: features,
		Limits:   limits,
	}
}

func (p *PlanFeatures) IsEmpty() bool {
	return len(p.Features) == 0 && len(p.Limits) == 0
}

func (p *PlanFeatures) FeatureCount() int {
	return len(p.Features)
}

func (p *PlanFeatures) LimitCount() int {
	return len(p.Limits)
}

func (p *PlanFeatures) Equals(other *PlanFeatures) bool {
	if other == nil {
		return false
	}

	if len(p.Features) != len(other.Features) {
		return false
	}

	featureMap := make(map[string]bool)
	for _, f := range p.Features {
		featureMap[f] = true
	}

	for _, f := range other.Features {
		if !featureMap[f] {
			return false
		}
	}

	if len(p.Limits) != len(other.Limits) {
		return false
	}

	for k, v := range p.Limits {
		otherV, exists := other.Limits[k]
		if !exists {
			return false
		}
		if v != otherV {
			return false
		}
	}

	return true
}

func (p *PlanFeatures) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Features []string               `json:"features"`
		Limits   map[string]interface{} `json:"limits"`
	}{
		Features: p.Features,
		Limits:   p.Limits,
	})
}

func (p *PlanFeatures) UnmarshalJSON(data []byte) error {
	aux := &struct {
		Features []string               `json:"features"`
		Limits   map[string]interface{} `json:"limits"`
	}{}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	p.Features = aux.Features
	p.Limits = aux.Limits

	if p.Features == nil {
		p.Features = []string{}
	}
	if p.Limits == nil {
		p.Limits = make(map[string]interface{})
	}

	return nil
}

// Standard limit keys for traffic and resource management
const (
	// LimitKeyTraffic represents monthly traffic limit in bytes
	LimitKeyTraffic = "traffic_limit"
	// LimitKeyDeviceCount represents maximum number of concurrent devices
	LimitKeyDeviceCount = "device_limit"
	// LimitKeySpeedLimit represents download/upload speed limit in Mbps
	LimitKeySpeedLimit = "speed_limit"
	// LimitKeyConnectionLimit represents maximum concurrent connections
	LimitKeyConnectionLimit = "connection_limit"
)

// Standard feature keys for plan capabilities
const (
	// FeatureForward represents the forward capability feature
	FeatureForward = "forward"
)

// Forward-related limit keys
const (
	// LimitKeyForwardRuleCount represents maximum number of forward rules
	LimitKeyForwardRuleCount = "forward_rule_limit"
	// LimitKeyForwardTraffic represents monthly forward traffic limit in bytes
	LimitKeyForwardTraffic = "forward_traffic_limit"
	// LimitKeyForwardRuleTypes represents allowed forward rule types
	LimitKeyForwardRuleTypes = "forward_rule_types"
)

// GetTrafficLimit returns the monthly traffic limit in bytes
// Returns 0 if unlimited or not set
func (p *PlanFeatures) GetTrafficLimit() (uint64, error) {
	value, exists := p.GetLimit(LimitKeyTraffic)
	if !exists {
		return 0, nil // 0 means unlimited
	}

	switch v := value.(type) {
	case uint64:
		return v, nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("traffic limit cannot be negative")
		}
		return uint64(v), nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("traffic limit cannot be negative")
		}
		return uint64(v), nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("traffic limit cannot be negative")
		}
		return uint64(v), nil
	default:
		return 0, fmt.Errorf("invalid traffic limit type: %T", v)
	}
}

// SetTrafficLimit sets the monthly traffic limit in bytes
// Use 0 for unlimited
func (p *PlanFeatures) SetTrafficLimit(bytes uint64) {
	p.SetLimit(LimitKeyTraffic, bytes)
}

// GetDeviceLimit returns the maximum number of concurrent devices
// Returns 0 if unlimited or not set
func (p *PlanFeatures) GetDeviceLimit() (int, error) {
	value, exists := p.GetLimit(LimitKeyDeviceCount)
	if !exists {
		return 0, nil // 0 means unlimited
	}

	switch v := value.(type) {
	case int:
		if v < 0 {
			return 0, fmt.Errorf("device limit cannot be negative")
		}
		return v, nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("device limit cannot be negative")
		}
		return int(v), nil
	default:
		return 0, fmt.Errorf("invalid device limit type: %T", v)
	}
}

// SetDeviceLimit sets the maximum number of concurrent devices
// Use 0 for unlimited
func (p *PlanFeatures) SetDeviceLimit(count int) error {
	if count < 0 {
		return fmt.Errorf("device limit cannot be negative")
	}
	p.SetLimit(LimitKeyDeviceCount, count)
	return nil
}

// GetSpeedLimit returns the speed limit in Mbps
// Returns 0 if unlimited or not set
func (p *PlanFeatures) GetSpeedLimit() (int, error) {
	value, exists := p.GetLimit(LimitKeySpeedLimit)
	if !exists {
		return 0, nil // 0 means unlimited
	}

	switch v := value.(type) {
	case int:
		if v < 0 {
			return 0, fmt.Errorf("speed limit cannot be negative")
		}
		return v, nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("speed limit cannot be negative")
		}
		return int(v), nil
	default:
		return 0, fmt.Errorf("invalid speed limit type: %T", v)
	}
}

// SetSpeedLimit sets the speed limit in Mbps
// Use 0 for unlimited
func (p *PlanFeatures) SetSpeedLimit(mbps int) error {
	if mbps < 0 {
		return fmt.Errorf("speed limit cannot be negative")
	}
	p.SetLimit(LimitKeySpeedLimit, mbps)
	return nil
}

// GetConnectionLimit returns the maximum number of concurrent connections
// Returns 0 if unlimited or not set
func (p *PlanFeatures) GetConnectionLimit() (int, error) {
	value, exists := p.GetLimit(LimitKeyConnectionLimit)
	if !exists {
		return 0, nil // 0 means unlimited
	}

	switch v := value.(type) {
	case int:
		if v < 0 {
			return 0, fmt.Errorf("connection limit cannot be negative")
		}
		return v, nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("connection limit cannot be negative")
		}
		return int(v), nil
	default:
		return 0, fmt.Errorf("invalid connection limit type: %T", v)
	}
}

// SetConnectionLimit sets the maximum number of concurrent connections
// Use 0 for unlimited
func (p *PlanFeatures) SetConnectionLimit(count int) error {
	if count < 0 {
		return fmt.Errorf("connection limit cannot be negative")
	}
	p.SetLimit(LimitKeyConnectionLimit, count)
	return nil
}

// IsUnlimitedTraffic checks if the plan has unlimited traffic
func (p *PlanFeatures) IsUnlimitedTraffic() bool {
	limit, err := p.GetTrafficLimit()
	return err == nil && limit == 0
}

// HasTrafficRemaining checks if the used traffic is within the plan limit
// Returns true if unlimited or within limit
func (p *PlanFeatures) HasTrafficRemaining(usedBytes uint64) (bool, error) {
	limit, err := p.GetTrafficLimit()
	if err != nil {
		return false, err
	}

	// 0 means unlimited
	if limit == 0 {
		return true, nil
	}

	return usedBytes < limit, nil
}

// GetForwardRuleLimit returns the maximum number of forward rules
// Returns 0 if unlimited or not set
func (p *PlanFeatures) GetForwardRuleLimit() (int, error) {
	value, exists := p.GetLimit(LimitKeyForwardRuleCount)
	if !exists {
		return 0, nil // 0 means unlimited
	}

	switch v := value.(type) {
	case int:
		if v < 0 {
			return 0, fmt.Errorf("forward rule limit cannot be negative")
		}
		return v, nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("forward rule limit cannot be negative")
		}
		return int(v), nil
	default:
		return 0, fmt.Errorf("invalid forward rule limit type: %T", v)
	}
}

// SetForwardRuleLimit sets the maximum number of forward rules
// Use 0 for unlimited
func (p *PlanFeatures) SetForwardRuleLimit(count int) error {
	if count < 0 {
		return fmt.Errorf("forward rule limit cannot be negative")
	}
	p.SetLimit(LimitKeyForwardRuleCount, count)
	return nil
}

// GetForwardTrafficLimit returns the monthly forward traffic limit in bytes
// Returns 0 if unlimited or not set
func (p *PlanFeatures) GetForwardTrafficLimit() (uint64, error) {
	value, exists := p.GetLimit(LimitKeyForwardTraffic)
	if !exists {
		return 0, nil // 0 means unlimited
	}

	switch v := value.(type) {
	case uint64:
		return v, nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("forward traffic limit cannot be negative")
		}
		return uint64(v), nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("forward traffic limit cannot be negative")
		}
		return uint64(v), nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("forward traffic limit cannot be negative")
		}
		return uint64(v), nil
	default:
		return 0, fmt.Errorf("invalid forward traffic limit type: %T", v)
	}
}

// SetForwardTrafficLimit sets the monthly forward traffic limit in bytes
// Use 0 for unlimited
func (p *PlanFeatures) SetForwardTrafficLimit(bytes uint64) {
	p.SetLimit(LimitKeyForwardTraffic, bytes)
}

// GetAllowedForwardRuleTypes returns the list of allowed forward rule types
// Returns empty slice if all types are allowed
func (p *PlanFeatures) GetAllowedForwardRuleTypes() ([]string, error) {
	value, exists := p.GetLimit(LimitKeyForwardRuleTypes)
	if !exists {
		return []string{}, nil // empty means all types allowed
	}

	switch v := value.(type) {
	case []string:
		return v, nil
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			switch ruleType := item.(type) {
			case string:
				result = append(result, ruleType)
			default:
				return nil, fmt.Errorf("invalid forward rule type: %T", item)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("invalid forward rule types type: %T", v)
	}
}

// SetAllowedForwardRuleTypes sets the list of allowed forward rule types
// Use empty slice for all types allowed
func (p *PlanFeatures) SetAllowedForwardRuleTypes(ruleTypes []string) {
	p.SetLimit(LimitKeyForwardRuleTypes, ruleTypes)
}

// IsUnlimitedForwardTraffic checks if the plan has unlimited forward traffic
func (p *PlanFeatures) IsUnlimitedForwardTraffic() bool {
	limit, err := p.GetForwardTrafficLimit()
	return err == nil && limit == 0
}

// HasForwardTrafficRemaining checks if the used forward traffic is within the plan limit
// Returns true if unlimited or within limit
func (p *PlanFeatures) HasForwardTrafficRemaining(usedBytes uint64) (bool, error) {
	limit, err := p.GetForwardTrafficLimit()
	if err != nil {
		return false, err
	}

	// 0 means unlimited
	if limit == 0 {
		return true, nil
	}

	return usedBytes < limit, nil
}

// IsForwardRuleTypeAllowed checks if the given rule type is allowed in the plan
// Returns true if all types allowed or the specific type is in the allowed list
func (p *PlanFeatures) IsForwardRuleTypeAllowed(ruleType string) (bool, error) {
	allowedTypes, err := p.GetAllowedForwardRuleTypes()
	if err != nil {
		return false, err
	}

	// empty means all types allowed
	if len(allowedTypes) == 0 {
		return true, nil
	}

	for _, allowed := range allowedTypes {
		if allowed == ruleType {
			return true, nil
		}
	}

	return false, nil
}
