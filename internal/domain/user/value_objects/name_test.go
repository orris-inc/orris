package value_objects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		expected  string
	}{
		{
			name:      "valid name",
			input:     "John Doe",
			wantError: false,
			expected:  "John Doe",
		},
		{
			name:      "valid name with extra spaces",
			input:     "  John Doe  ",
			wantError: false,
			expected:  "John Doe",
		},
		{
			name:      "valid single name",
			input:     "Madonna",
			wantError: false,
			expected:  "Madonna",
		},
		{
			name:      "valid name with hyphen",
			input:     "Mary-Jane Watson",
			wantError: false,
			expected:  "Mary-Jane Watson",
		},
		{
			name:      "valid name with apostrophe",
			input:     "O'Brien",
			wantError: false,
			expected:  "O'Brien",
		},
		{
			name:      "valid name with period",
			input:     "Dr. Smith",
			wantError: false,
			expected:  "Dr. Smith",
		},
		{
			name:      "empty name",
			input:     "",
			wantError: true,
		},
		{
			name:      "only spaces",
			input:     "   ",
			wantError: true,
		},
		{
			name:      "too short - 1 character",
			input:     "A",
			wantError: true,
		},
		{
			name:      "name with numbers",
			input:     "John123",
			wantError: true,
		},
		{
			name:      "name with special characters",
			input:     "John@Doe",
			wantError: true,
		},
		{
			name:      "consecutive spaces",
			input:     "John  Doe",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := NewName(tt.input)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, name)
			} else {
				require.NoError(t, err)
				require.NotNil(t, name)
				assert.Equal(t, tt.expected, name.String())
			}
		})
	}
}

func TestName_FirstName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "full name",
			input:    "John Doe",
			expected: "John",
		},
		{
			name:     "three part name",
			input:    "John Michael Doe",
			expected: "John",
		},
		{
			name:     "single name",
			input:    "Madonna",
			expected: "Madonna",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := NewName(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, name.FirstName())
		})
	}
}

func TestName_LastName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "full name",
			input:    "John Doe",
			expected: "Doe",
		},
		{
			name:     "three part name",
			input:    "John Michael Doe",
			expected: "Doe",
		},
		{
			name:     "single name",
			input:    "Madonna",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := NewName(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, name.LastName())
		})
	}
}

func TestName_MiddleNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "no middle names",
			input:    "John Doe",
			expected: []string{},
		},
		{
			name:     "one middle name",
			input:    "John Michael Doe",
			expected: []string{"Michael"},
		},
		{
			name:     "two middle names",
			input:    "John Michael Robert Doe",
			expected: []string{"Michael", "Robert"},
		},
		{
			name:     "single name",
			input:    "Madonna",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := NewName(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, name.MiddleNames())
		})
	}
}

func TestName_Initials(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "two part name",
			input:    "John Doe",
			expected: "JD",
		},
		{
			name:     "three part name",
			input:    "John Michael Doe",
			expected: "JMD",
		},
		{
			name:     "single name",
			input:    "Madonna",
			expected: "M",
		},
		{
			name:     "lowercase name",
			input:    "john doe",
			expected: "JD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := NewName(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, name.Initials())
		})
	}
}

func TestName_DisplayName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase name",
			input:    "john doe",
			expected: "John Doe",
		},
		{
			name:     "uppercase name",
			input:    "JOHN DOE",
			expected: "John Doe",
		},
		{
			name:     "mixed case name",
			input:    "JoHn DoE",
			expected: "John Doe",
		},
		{
			name:     "already proper case",
			input:    "John Doe",
			expected: "John Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := NewName(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, name.DisplayName())
		})
	}
}

func TestName_IsMononym(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		isMononym  bool
	}{
		{
			name:      "single name",
			input:     "Madonna",
			isMononym: true,
		},
		{
			name:      "two part name",
			input:     "John Doe",
			isMononym: false,
		},
		{
			name:      "three part name",
			input:     "John Michael Doe",
			isMononym: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := NewName(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.isMononym, name.IsMononym())
		})
	}
}

func TestName_Equals(t *testing.T) {
	tests := []struct {
		name     string
		name1    string
		name2    string
		expected bool
	}{
		{
			name:     "same name",
			name1:    "John Doe",
			name2:    "John Doe",
			expected: true,
		},
		{
			name:     "different case",
			name1:    "John Doe",
			name2:    "john doe",
			expected: true,
		},
		{
			name:     "different names",
			name1:    "John Doe",
			name2:    "Jane Doe",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name1, err := NewName(tt.name1)
			require.NoError(t, err)

			name2, err := NewName(tt.name2)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, name1.Equals(name2))
		})
	}
}

func TestName_JSON(t *testing.T) {
	name, err := NewName("John Doe")
	require.NoError(t, err)

	// Test MarshalJSON
	data, err := name.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `"John Doe"`, string(data))

	// Test UnmarshalJSON
	var name2 Name
	err = name2.UnmarshalJSON([]byte(`"Jane Smith"`))
	require.NoError(t, err)
	assert.Equal(t, "Jane Smith", name2.String())
}
