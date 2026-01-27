package setting

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/id"
)

// ValueType defines the type of a setting value
type ValueType string

const (
	ValueTypeString ValueType = "string"
	ValueTypeInt    ValueType = "int"
	ValueTypeBool   ValueType = "bool"
	ValueTypeJSON   ValueType = "json"
)

// SystemSetting represents a system configuration setting
type SystemSetting struct {
	id          uint
	sid         string    // Stripe-style ID: setting_xxx
	category    string    // Setting category (e.g., "system", "notification", "email")
	key         string    // Setting key within category
	value       string    // Setting value (stored as string, parsed based on valueType)
	valueType   ValueType // Type of the value for parsing
	description string    // Human-readable description
	updatedBy   uint      // User ID who last updated this setting
	version     int       // Optimistic locking version
	createdAt   time.Time
	updatedAt   time.Time
}

// NewSystemSetting creates a new system setting
func NewSystemSetting(category, key string, valueType ValueType, description string) (*SystemSetting, error) {
	if category == "" {
		return nil, fmt.Errorf("category is required")
	}
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}
	if !isValidValueType(valueType) {
		return nil, fmt.Errorf("invalid value type: %s", valueType)
	}

	sid, err := id.NewSettingID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SID: %w", err)
	}

	now := biztime.NowUTC()
	return &SystemSetting{
		sid:         sid,
		category:    category,
		key:         key,
		valueType:   valueType,
		description: description,
		version:     1,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// ReconstructSystemSetting reconstructs a SystemSetting from persistence layer
func ReconstructSystemSetting(
	id uint,
	sid string,
	category string,
	key string,
	value string,
	valueType ValueType,
	description string,
	updatedBy uint,
	version int,
	createdAt, updatedAt time.Time,
) *SystemSetting {
	return &SystemSetting{
		id:          id,
		sid:         sid,
		category:    category,
		key:         key,
		value:       value,
		valueType:   valueType,
		description: description,
		updatedBy:   updatedBy,
		version:     version,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

// Getters
func (s *SystemSetting) ID() uint             { return s.id }
func (s *SystemSetting) SID() string          { return s.sid }
func (s *SystemSetting) Category() string     { return s.category }
func (s *SystemSetting) Key() string          { return s.key }
func (s *SystemSetting) Value() string        { return s.value }
func (s *SystemSetting) ValueType() ValueType { return s.valueType }
func (s *SystemSetting) Description() string  { return s.description }
func (s *SystemSetting) UpdatedBy() uint      { return s.updatedBy }
func (s *SystemSetting) Version() int         { return s.version }
func (s *SystemSetting) CreatedAt() time.Time { return s.createdAt }
func (s *SystemSetting) UpdatedAt() time.Time { return s.updatedAt }

// SetID sets the setting ID (only for persistence layer use)
func (s *SystemSetting) SetID(id uint) {
	s.id = id
}

// HasValue checks if the setting has a non-empty value
func (s *SystemSetting) HasValue() bool {
	return s.value != ""
}

// GetStringValue returns the value as a string
func (s *SystemSetting) GetStringValue() string {
	return s.value
}

// GetIntValue returns the value as an integer
func (s *SystemSetting) GetIntValue() (int, error) {
	if s.value == "" {
		return 0, nil
	}
	return strconv.Atoi(s.value)
}

// GetBoolValue returns the value as a boolean
func (s *SystemSetting) GetBoolValue() (bool, error) {
	if s.value == "" {
		return false, nil
	}
	return strconv.ParseBool(s.value)
}

// GetJSONValue unmarshals the value into the provided target
func (s *SystemSetting) GetJSONValue(target interface{}) error {
	if s.value == "" {
		return nil
	}
	return json.Unmarshal([]byte(s.value), target)
}

// GetStringArrayValue returns the value as a string array (for JSON array type)
func (s *SystemSetting) GetStringArrayValue() ([]string, error) {
	if s.value == "" || s.value == "[]" {
		return []string{}, nil
	}
	var result []string
	if err := json.Unmarshal([]byte(s.value), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal string array: %w", err)
	}
	return result, nil
}

// SetStringValue sets the value as a string
func (s *SystemSetting) SetStringValue(value string, updatedBy uint) error {
	if s.valueType != ValueTypeString {
		return fmt.Errorf("value type mismatch: expected %s, got string", s.valueType)
	}
	s.value = value
	s.updatedBy = updatedBy
	s.version++
	s.updatedAt = biztime.NowUTC()
	return nil
}

// SetIntValue sets the value as an integer
func (s *SystemSetting) SetIntValue(value int, updatedBy uint) error {
	if s.valueType != ValueTypeInt {
		return fmt.Errorf("value type mismatch: expected %s, got int", s.valueType)
	}
	s.value = strconv.Itoa(value)
	s.updatedBy = updatedBy
	s.version++
	s.updatedAt = biztime.NowUTC()
	return nil
}

// SetBoolValue sets the value as a boolean
func (s *SystemSetting) SetBoolValue(value bool, updatedBy uint) error {
	if s.valueType != ValueTypeBool {
		return fmt.Errorf("value type mismatch: expected %s, got bool", s.valueType)
	}
	s.value = strconv.FormatBool(value)
	s.updatedBy = updatedBy
	s.version++
	s.updatedAt = biztime.NowUTC()
	return nil
}

// SetJSONValue sets the value as JSON
func (s *SystemSetting) SetJSONValue(value interface{}, updatedBy uint) error {
	if s.valueType != ValueTypeJSON {
		return fmt.Errorf("value type mismatch: expected %s, got json", s.valueType)
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON value: %w", err)
	}
	s.value = string(data)
	s.updatedBy = updatedBy
	s.version++
	s.updatedAt = biztime.NowUTC()
	return nil
}

// isValidValueType checks if the value type is valid
func isValidValueType(vt ValueType) bool {
	switch vt {
	case ValueTypeString, ValueTypeInt, ValueTypeBool, ValueTypeJSON:
		return true
	default:
		return false
	}
}
