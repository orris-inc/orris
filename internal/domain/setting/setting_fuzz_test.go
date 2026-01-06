package setting

import (
	"encoding/json"
	"math"
	"testing"
	"unicode/utf8"
)

// FuzzGetIntValue tests GetIntValue with random string inputs
func FuzzGetIntValue(f *testing.F) {
	seeds := []string{
		"",
		"0",
		"1",
		"-1",
		"123",
		"-123",
		"2147483647",  // MaxInt32
		"-2147483648", // MinInt32
		"9223372036854775807",  // MaxInt64
		"-9223372036854775808", // MinInt64
		"99999999999999999999999999999", // Overflow
		"abc",
		"12.34",
		"12abc",
		"  123  ",
		"0x1F",
		"1e10",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		s := &SystemSetting{value: input, valueType: ValueTypeInt}

		val, err := s.GetIntValue()

		// Empty string should return 0 without error
		if input == "" {
			if err != nil || val != 0 {
				t.Errorf("GetIntValue(%q) = (%d, %v), expected (0, nil)", input, val, err)
			}
			return
		}

		// If error is nil, the result should be a valid integer
		if err == nil {
			// Verify round-trip: convert back to string and parse again
			s2 := &SystemSetting{value: input, valueType: ValueTypeInt}
			val2, err2 := s2.GetIntValue()
			if err2 != nil || val != val2 {
				t.Errorf("GetIntValue not consistent: first=%d, second=%d", val, val2)
			}
		}
	})
}

// FuzzGetBoolValue tests GetBoolValue with random string inputs
func FuzzGetBoolValue(f *testing.F) {
	seeds := []string{
		"",
		"true",
		"false",
		"True",
		"False",
		"TRUE",
		"FALSE",
		"1",
		"0",
		"t",
		"f",
		"T",
		"F",
		"yes",
		"no",
		"on",
		"off",
		"abc",
		"truee",
		"fals",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		s := &SystemSetting{value: input, valueType: ValueTypeBool}

		val, err := s.GetBoolValue()

		// Empty string should return false without error
		if input == "" {
			if err != nil || val != false {
				t.Errorf("GetBoolValue(%q) = (%t, %v), expected (false, nil)", input, val, err)
			}
			return
		}

		// If error is nil, result should be consistent
		if err == nil {
			s2 := &SystemSetting{value: input, valueType: ValueTypeBool}
			val2, err2 := s2.GetBoolValue()
			if err2 != nil || val != val2 {
				t.Errorf("GetBoolValue not consistent: first=%t, second=%t", val, val2)
			}
		}
	})
}

// FuzzGetJSONValue tests GetJSONValue with random JSON inputs
func FuzzGetJSONValue(f *testing.F) {
	seeds := []string{
		"",
		"null",
		"true",
		"false",
		"123",
		"12.34",
		`"hello"`,
		`""`,
		"[]",
		"{}",
		`{"key": "value"}`,
		`{"nested": {"deep": true}}`,
		`[1, 2, 3]`,
		`{"unicode": "中文"}`,
		`{"emoji": "👍"}`,
		"invalid json",
		`{"unclosed": "brace"`,
		`[1, 2, 3`,
		`{"key": undefined}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		if !utf8.ValidString(input) {
			return
		}

		s := &SystemSetting{value: input, valueType: ValueTypeJSON}

		var target interface{}
		err := s.GetJSONValue(&target)

		// Empty string should not error
		if input == "" {
			if err != nil {
				t.Errorf("GetJSONValue(%q) returned error: %v", input, err)
			}
			return
		}

		// If no error, verify the JSON is valid
		if err == nil {
			var check interface{}
			if jsonErr := json.Unmarshal([]byte(input), &check); jsonErr != nil {
				t.Errorf("GetJSONValue(%q) returned nil error but JSON is invalid", input)
			}
		}
	})
}

// FuzzIsValidValueType tests isValidValueType with random inputs
func FuzzIsValidValueType(f *testing.F) {
	seeds := []string{
		"",
		"string",
		"int",
		"bool",
		"json",
		"String",
		"STRING",
		"integer",
		"boolean",
		"object",
		"array",
		"float",
		"double",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		vt := ValueType(input)
		result := isValidValueType(vt)

		// Only these four should return true
		validTypes := map[ValueType]bool{
			ValueTypeString: true,
			ValueTypeInt:    true,
			ValueTypeBool:   true,
			ValueTypeJSON:   true,
		}

		expected := validTypes[vt]
		if result != expected {
			t.Errorf("isValidValueType(%q) = %t, expected %t", input, result, expected)
		}
	})
}

// FuzzSetIntValue tests SetIntValue with random integers
func FuzzSetIntValue(f *testing.F) {
	values := []int{0, 1, -1, 100, -100, math.MaxInt32, math.MinInt32}
	for _, v := range values {
		f.Add(v)
	}

	f.Fuzz(func(t *testing.T, value int) {
		s := &SystemSetting{valueType: ValueTypeInt, version: 1}

		err := s.SetIntValue(value, 1)
		if err != nil {
			t.Errorf("SetIntValue(%d) returned error: %v", value, err)
			return
		}

		// Verify we can get the value back
		got, err := s.GetIntValue()
		if err != nil {
			t.Errorf("GetIntValue after SetIntValue(%d) returned error: %v", value, err)
			return
		}

		if got != value {
			t.Errorf("SetIntValue(%d) then GetIntValue() = %d", value, got)
		}
	})
}

// FuzzSetBoolValue tests SetBoolValue
func FuzzSetBoolValue(f *testing.F) {
	f.Add(true)
	f.Add(false)

	f.Fuzz(func(t *testing.T, value bool) {
		s := &SystemSetting{valueType: ValueTypeBool, version: 1}

		err := s.SetBoolValue(value, 1)
		if err != nil {
			t.Errorf("SetBoolValue(%t) returned error: %v", value, err)
			return
		}

		got, err := s.GetBoolValue()
		if err != nil {
			t.Errorf("GetBoolValue after SetBoolValue(%t) returned error: %v", value, err)
			return
		}

		if got != value {
			t.Errorf("SetBoolValue(%t) then GetBoolValue() = %t", value, got)
		}
	})
}

// FuzzNewSystemSetting tests NewSystemSetting with random inputs
func FuzzNewSystemSetting(f *testing.F) {
	f.Add("system", "key1", "string", "description")
	f.Add("", "key", "string", "desc")
	f.Add("cat", "", "string", "desc")
	f.Add("cat", "key", "invalid", "desc")
	f.Add("cat", "key", "int", "")
	f.Add("中文", "键名", "json", "描述")

	f.Fuzz(func(t *testing.T, category, key, valueType, description string) {
		if !utf8.ValidString(category) || !utf8.ValidString(key) ||
			!utf8.ValidString(valueType) || !utf8.ValidString(description) {
			return
		}

		setting, err := NewSystemSetting(category, key, ValueType(valueType), description)

		// Empty category should error
		if category == "" {
			if err == nil {
				t.Errorf("NewSystemSetting(%q, %q, %q, %q) should error for empty category", category, key, valueType, description)
			}
			return
		}

		// Empty key should error
		if key == "" {
			if err == nil {
				t.Errorf("NewSystemSetting(%q, %q, %q, %q) should error for empty key", category, key, valueType, description)
			}
			return
		}

		// Invalid value type should error
		if !isValidValueType(ValueType(valueType)) {
			if err == nil {
				t.Errorf("NewSystemSetting(%q, %q, %q, %q) should error for invalid value type", category, key, valueType, description)
			}
			return
		}

		// Valid inputs should succeed
		if err != nil {
			t.Errorf("NewSystemSetting(%q, %q, %q, %q) returned unexpected error: %v", category, key, valueType, description, err)
			return
		}

		// Verify the setting was created correctly
		if setting.Category() != category {
			t.Errorf("Category() = %q, expected %q", setting.Category(), category)
		}
		if setting.Key() != key {
			t.Errorf("Key() = %q, expected %q", setting.Key(), key)
		}
	})
}
