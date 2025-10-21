package value_objects

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
