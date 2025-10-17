package value_objects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmail(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		expected  string
	}{
		{
			name:      "valid email",
			input:     "test@example.com",
			wantError: false,
			expected:  "test@example.com",
		},
		{
			name:      "valid email with uppercase",
			input:     "Test@Example.COM",
			wantError: false,
			expected:  "test@example.com", // Should be normalized to lowercase
		},
		{
			name:      "valid email with subdomain",
			input:     "user@mail.example.com",
			wantError: false,
			expected:  "user@mail.example.com",
		},
		{
			name:      "empty email",
			input:     "",
			wantError: true,
		},
		{
			name:      "email with spaces",
			input:     " test@example.com ",
			wantError: false,
			expected:  "test@example.com", // Spaces should be trimmed
		},
		{
			name:      "invalid format - no @",
			input:     "testexample.com",
			wantError: true,
		},
		{
			name:      "invalid format - no domain",
			input:     "test@",
			wantError: true,
		},
		{
			name:      "invalid format - no local part",
			input:     "@example.com",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := NewEmail(tt.input)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, email)
			} else {
				require.NoError(t, err)
				require.NotNil(t, email)
				assert.Equal(t, tt.expected, email.String())
			}
		})
	}
}

func TestEmail_Domain(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "simple domain",
			email:    "test@example.com",
			expected: "example.com",
		},
		{
			name:     "subdomain",
			email:    "user@mail.google.com",
			expected: "mail.google.com",
		},
		{
			name:     "uppercase domain",
			email:    "Test@EXAMPLE.COM",
			expected: "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := NewEmail(tt.email)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, email.Domain())
		})
	}
}

func TestEmail_LocalPart(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "simple local part",
			email:    "test@example.com",
			expected: "test",
		},
		{
			name:     "complex local part",
			email:    "user.name+tag@example.com",
			expected: "user.name+tag",
		},
		{
			name:     "uppercase local part",
			email:    "Test@example.com",
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := NewEmail(tt.email)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, email.LocalPart())
		})
	}
}

func TestEmail_IsBusinessEmail(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		isBusiness bool
	}{
		{
			name:       "gmail is not business",
			email:      "user@gmail.com",
			isBusiness: false,
		},
		{
			name:       "yahoo is not business",
			email:      "user@yahoo.com",
			isBusiness: false,
		},
		{
			name:       "hotmail is not business",
			email:      "user@hotmail.com",
			isBusiness: false,
		},
		{
			name:       "outlook is not business",
			email:      "user@outlook.com",
			isBusiness: false,
		},
		{
			name:       "custom domain is business",
			email:      "user@company.com",
			isBusiness: true,
		},
		{
			name:       "subdomain of free provider",
			email:      "user@mail.gmail.com",
			isBusiness: true, // IsBusinessEmail checks the full domain, not just the root
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := NewEmail(tt.email)
			require.NoError(t, err)
			assert.Equal(t, tt.isBusiness, email.IsBusinessEmail())
		})
	}
}

func TestEmail_Equals(t *testing.T) {
	tests := []struct {
		name     string
		email1   string
		email2   string
		expected bool
	}{
		{
			name:     "same email",
			email1:   "test@example.com",
			email2:   "test@example.com",
			expected: true,
		},
		{
			name:     "different case",
			email1:   "Test@Example.com",
			email2:   "test@example.com",
			expected: true,
		},
		{
			name:     "different email",
			email1:   "test1@example.com",
			email2:   "test2@example.com",
			expected: false,
		},
		{
			name:     "nil comparison",
			email1:   "test@example.com",
			email2:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email1, err := NewEmail(tt.email1)
			require.NoError(t, err)

			if tt.email2 == "" {
				assert.Equal(t, tt.expected, email1.Equals(nil))
			} else {
				email2, err := NewEmail(tt.email2)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, email1.Equals(email2))
			}
		})
	}
}

func TestEmail_MarshalJSON(t *testing.T) {
	email, err := NewEmail("test@example.com")
	require.NoError(t, err)

	data, err := email.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `"test@example.com"`, string(data))
}

func TestEmail_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		expected  string
	}{
		{
			name:      "valid JSON",
			input:     `"test@example.com"`,
			wantError: false,
			expected:  "test@example.com",
		},
		{
			name:      "invalid email in JSON",
			input:     `"invalid-email"`,
			wantError: true,
		},
		{
			name:      "empty JSON string",
			input:     `""`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var email Email
			err := email.UnmarshalJSON([]byte(tt.input))

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, email.String())
			}
		})
	}
}
