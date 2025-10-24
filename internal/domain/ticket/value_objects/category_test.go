package value_objects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCategory(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Category
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid technical category",
			input:   "technical",
			want:    CategoryTechnical,
			wantErr: false,
		},
		{
			name:    "valid account category",
			input:   "account",
			want:    CategoryAccount,
			wantErr: false,
		},
		{
			name:    "valid billing category",
			input:   "billing",
			want:    CategoryBilling,
			wantErr: false,
		},
		{
			name:    "valid feature category",
			input:   "feature",
			want:    CategoryFeature,
			wantErr: false,
		},
		{
			name:    "valid complaint category",
			input:   "complaint",
			want:    CategoryComplaint,
			wantErr: false,
		},
		{
			name:    "valid other category",
			input:   "other",
			want:    CategoryOther,
			wantErr: false,
		},
		{
			name:    "invalid category",
			input:   "invalid",
			wantErr: true,
			errMsg:  "invalid category: invalid",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
			errMsg:  "invalid category",
		},
		{
			name:    "case sensitive - uppercase",
			input:   "TECHNICAL",
			wantErr: true,
			errMsg:  "invalid category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCategory(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCategory_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		category Category
		want     bool
	}{
		{"technical is valid", CategoryTechnical, true},
		{"account is valid", CategoryAccount, true},
		{"billing is valid", CategoryBilling, true},
		{"feature is valid", CategoryFeature, true},
		{"complaint is valid", CategoryComplaint, true},
		{"other is valid", CategoryOther, true},
		{"invalid category", Category("invalid"), false},
		{"empty category", Category(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.category.IsValid())
		})
	}
}

func TestCategory_String(t *testing.T) {
	tests := []struct {
		name     string
		category Category
		want     string
	}{
		{"technical", CategoryTechnical, "technical"},
		{"account", CategoryAccount, "account"},
		{"billing", CategoryBilling, "billing"},
		{"feature", CategoryFeature, "feature"},
		{"complaint", CategoryComplaint, "complaint"},
		{"other", CategoryOther, "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.category.String())
		})
	}
}

func TestCategory_StateCheckers(t *testing.T) {
	tests := []struct {
		name     string
		category Category
		checker  string
		expected bool
	}{
		{"technical is technical", CategoryTechnical, "IsTechnical", true},
		{"account is not technical", CategoryAccount, "IsTechnical", false},

		{"account is account", CategoryAccount, "IsAccount", true},
		{"technical is not account", CategoryTechnical, "IsAccount", false},

		{"billing is billing", CategoryBilling, "IsBilling", true},
		{"account is not billing", CategoryAccount, "IsBilling", false},

		{"feature is feature", CategoryFeature, "IsFeature", true},
		{"billing is not feature", CategoryBilling, "IsFeature", false},

		{"complaint is complaint", CategoryComplaint, "IsComplaint", true},
		{"feature is not complaint", CategoryFeature, "IsComplaint", false},

		{"other is other", CategoryOther, "IsOther", true},
		{"complaint is not other", CategoryComplaint, "IsOther", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bool
			switch tt.checker {
			case "IsTechnical":
				result = tt.category.IsTechnical()
			case "IsAccount":
				result = tt.category.IsAccount()
			case "IsBilling":
				result = tt.category.IsBilling()
			case "IsFeature":
				result = tt.category.IsFeature()
			case "IsComplaint":
				result = tt.category.IsComplaint()
			case "IsOther":
				result = tt.category.IsOther()
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCategory_AllCategoriesAreValid(t *testing.T) {
	categories := []Category{
		CategoryTechnical,
		CategoryAccount,
		CategoryBilling,
		CategoryFeature,
		CategoryComplaint,
		CategoryOther,
	}

	for _, category := range categories {
		t.Run(category.String(), func(t *testing.T) {
			assert.True(t, category.IsValid(), "category %s should be valid", category)
		})
	}
}

func TestCategory_AllCategoriesHaveUniqueStrings(t *testing.T) {
	categories := []Category{
		CategoryTechnical,
		CategoryAccount,
		CategoryBilling,
		CategoryFeature,
		CategoryComplaint,
		CategoryOther,
	}

	seen := make(map[string]bool)
	for _, category := range categories {
		categoryStr := category.String()
		assert.False(t, seen[categoryStr], "category string %s should be unique", categoryStr)
		seen[categoryStr] = true
	}

	assert.Equal(t, len(categories), len(seen), "all categories should have unique string representations")
}

func TestCategory_EachCategoryHasOneChecker(t *testing.T) {
	tests := []struct {
		category Category
		checker  func(Category) bool
	}{
		{CategoryTechnical, Category.IsTechnical},
		{CategoryAccount, Category.IsAccount},
		{CategoryBilling, Category.IsBilling},
		{CategoryFeature, Category.IsFeature},
		{CategoryComplaint, Category.IsComplaint},
		{CategoryOther, Category.IsOther},
	}

	for _, tt := range tests {
		t.Run(tt.category.String(), func(t *testing.T) {
			assert.True(t, tt.checker(tt.category), "checker should return true for %s", tt.category)

			otherCategories := []Category{
				CategoryTechnical, CategoryAccount, CategoryBilling,
				CategoryFeature, CategoryComplaint, CategoryOther,
			}
			for _, other := range otherCategories {
				if other != tt.category {
					assert.False(t, tt.checker(other), "checker should return false for %s when checking %s", other, tt.category)
				}
			}
		})
	}
}

func TestCategory_ComprehensiveValidation(t *testing.T) {
	t.Run("all categories have consistent behavior", func(t *testing.T) {
		categories := []Category{
			CategoryTechnical,
			CategoryAccount,
			CategoryBilling,
			CategoryFeature,
			CategoryComplaint,
			CategoryOther,
		}

		for _, c := range categories {
			assert.True(t, c.IsValid(), "category %s should be valid", c)

			assert.NotEmpty(t, c.String(), "category %s should have non-empty string", c)

			recreated, err := NewCategory(c.String())
			require.NoError(t, err, "should be able to recreate category %s from string", c)
			assert.Equal(t, c, recreated, "recreated category should match original")
		}
	})
}

func TestCategory_CoverageOfAllCategories(t *testing.T) {
	t.Run("ensure all defined categories are tested", func(t *testing.T) {
		expectedCategories := map[Category]bool{
			CategoryTechnical: false,
			CategoryAccount:   false,
			CategoryBilling:   false,
			CategoryFeature:   false,
			CategoryComplaint: false,
			CategoryOther:     false,
		}

		for category := range expectedCategories {
			assert.True(t, category.IsValid(), "category %s should be valid", category)
			expectedCategories[category] = true
		}

		for category, tested := range expectedCategories {
			assert.True(t, tested, "category %s should be tested", category)
		}
	})
}
