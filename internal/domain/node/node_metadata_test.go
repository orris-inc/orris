package node

import (
	"strings"
	"testing"
)

func TestNewNodeMetadata_Valid(t *testing.T) {
	tests := []struct {
		name        string
		country     string
		region      string
		tags        []string
		description string
	}{
		{"standard metadata", "US", "California", []string{"premium", "fast"}, "High speed server"},
		{"minimal metadata", "CN", "Beijing", []string{}, ""},
		{"with whitespace", "  us  ", "  California  ", []string{" tag1 ", " tag2 "}, "  description  "},
		{"lowercase country", "us", "California", []string{}, ""},
		{"empty description", "US", "California", []string{"tag"}, ""},
		{"empty tags", "US", "California", []string{}, "description"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := NewNodeMetadata(tt.country, tt.region, tt.tags, tt.description)
			if err != nil {
				t.Errorf("NewNodeMetadata() error = %v, want nil", err)
				return
			}
			if metadata.Country() == "" {
				t.Error("Country() returned empty string")
			}
			if metadata.Region() == "" {
				t.Error("Region() returned empty string")
			}
		})
	}
}

func TestNewNodeMetadata_InvalidCountry(t *testing.T) {
	tests := []struct {
		name    string
		country string
	}{
		{"empty country", ""},
		{"whitespace only", "   "},
		{"too short", "U"},
		{"too long", "USA"},
		{"three chars", "USA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewNodeMetadata(tt.country, "Region", []string{}, "")
			if err == nil {
				t.Errorf("NewNodeMetadata(%q, _, _, _) error = nil, want error", tt.country)
			}
		})
	}
}

func TestNewNodeMetadata_InvalidRegion(t *testing.T) {
	tests := []struct {
		name   string
		region string
	}{
		{"empty region", ""},
		{"whitespace only", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewNodeMetadata("US", tt.region, []string{}, "")
			if err == nil {
				t.Errorf("NewNodeMetadata(_, %q, _, _) error = nil, want error", tt.region)
			}
		})
	}
}

func TestNodeMetadata_CountryNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercase", "us", "US"},
		{"uppercase", "US", "US"},
		{"mixed case", "Us", "US"},
		{"with whitespace", "  us  ", "US"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := NewNodeMetadata(tt.input, "Region", []string{}, "")
			if err != nil {
				t.Fatalf("NewNodeMetadata() error = %v", err)
			}
			if metadata.Country() != tt.expected {
				t.Errorf("Country() = %q, want %q", metadata.Country(), tt.expected)
			}
		})
	}
}

func TestNodeMetadata_TagsFiltering(t *testing.T) {
	tests := []struct {
		name          string
		inputTags     []string
		expectedCount int
		expectedTags  []string
	}{
		{"normal tags", []string{"tag1", "tag2"}, 2, []string{"tag1", "tag2"}},
		{"with empty strings", []string{"tag1", "", "tag2"}, 2, []string{"tag1", "tag2"}},
		{"with whitespace", []string{" tag1 ", "  ", "tag2"}, 2, []string{"tag1", "tag2"}},
		{"all empty", []string{"", "  ", ""}, 0, []string{}},
		{"nil tags", nil, 0, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := NewNodeMetadata("US", "Region", tt.inputTags, "")
			if err != nil {
				t.Fatalf("NewNodeMetadata() error = %v", err)
			}
			if metadata.TagCount() != tt.expectedCount {
				t.Errorf("TagCount() = %d, want %d", metadata.TagCount(), tt.expectedCount)
			}

			tags := metadata.Tags()
			for _, expectedTag := range tt.expectedTags {
				found := false
				for _, tag := range tags {
					if tag == expectedTag {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Tags() missing expected tag: %q", expectedTag)
				}
			}
		})
	}
}

func TestNodeMetadata_HasTag(t *testing.T) {
	metadata, err := NewNodeMetadata("US", "California", []string{"premium", "Fast", "RELIABLE"}, "")
	if err != nil {
		t.Fatalf("NewNodeMetadata() error = %v", err)
	}

	tests := []struct {
		name     string
		tag      string
		expected bool
	}{
		{"exact match", "premium", true},
		{"case insensitive match", "PREMIUM", true},
		{"mixed case match", "Premium", true},
		{"case insensitive fast", "fast", true},
		{"case insensitive reliable", "reliable", true},
		{"non-existent tag", "nonexistent", false},
		{"empty tag", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.HasTag(tt.tag)
			if result != tt.expected {
				t.Errorf("HasTag(%q) = %v, want %v", tt.tag, result, tt.expected)
			}
		})
	}
}

func TestNodeMetadata_DisplayName(t *testing.T) {
	metadata, err := NewNodeMetadata("US", "California", []string{}, "")
	if err != nil {
		t.Fatalf("NewNodeMetadata() error = %v", err)
	}

	displayName := metadata.DisplayName()
	expected := "US - California"

	if displayName != expected {
		t.Errorf("DisplayName() = %q, want %q", displayName, expected)
	}
}

func TestNodeMetadata_FullDisplayName(t *testing.T) {
	tests := []struct {
		name        string
		description string
		expected    string
	}{
		{"with description", "High speed server", "US - California (High speed server)"},
		{"without description", "", "US - California"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := NewNodeMetadata("US", "California", []string{}, tt.description)
			if err != nil {
				t.Fatalf("NewNodeMetadata() error = %v", err)
			}

			result := metadata.FullDisplayName()
			if result != tt.expected {
				t.Errorf("FullDisplayName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNodeMetadata_WithDescription(t *testing.T) {
	original, err := NewNodeMetadata("US", "California", []string{"tag1"}, "original")
	if err != nil {
		t.Fatalf("NewNodeMetadata() error = %v", err)
	}

	updated := original.WithDescription("new description")

	if updated.Description() != "new description" {
		t.Errorf("WithDescription() description = %q, want %q", updated.Description(), "new description")
	}

	if original.Description() != "original" {
		t.Error("WithDescription() modified original metadata")
	}

	if updated.Country() != original.Country() || updated.Region() != original.Region() {
		t.Error("WithDescription() changed country or region")
	}
}

func TestNodeMetadata_WithTags(t *testing.T) {
	original, err := NewNodeMetadata("US", "California", []string{"tag1"}, "description")
	if err != nil {
		t.Fatalf("NewNodeMetadata() error = %v", err)
	}

	newTags := []string{"tag2", "tag3"}
	updated := original.WithTags(newTags)

	if updated.TagCount() != 2 {
		t.Errorf("WithTags() TagCount() = %d, want 2", updated.TagCount())
	}

	if !updated.HasTag("tag2") || !updated.HasTag("tag3") {
		t.Error("WithTags() did not set new tags correctly")
	}

	if original.TagCount() != 1 {
		t.Error("WithTags() modified original metadata")
	}
}

func TestNodeMetadata_AddTag(t *testing.T) {
	tests := []struct {
		name          string
		initialTags   []string
		tagToAdd      string
		expectedCount int
		shouldContain string
	}{
		{"add new tag", []string{"tag1"}, "tag2", 2, "tag2"},
		{"add duplicate tag", []string{"tag1"}, "tag1", 1, "tag1"},
		{"add empty tag", []string{"tag1"}, "", 1, ""},
		{"add whitespace tag", []string{"tag1"}, "  ", 1, ""},
		{"case insensitive duplicate", []string{"Tag1"}, "tag1", 1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := NewNodeMetadata("US", "California", tt.initialTags, "")
			if err != nil {
				t.Fatalf("NewNodeMetadata() error = %v", err)
			}

			updated := metadata.AddTag(tt.tagToAdd)

			if updated.TagCount() != tt.expectedCount {
				t.Errorf("AddTag(%q) TagCount() = %d, want %d", tt.tagToAdd, updated.TagCount(), tt.expectedCount)
			}

			if metadata.TagCount() != len(tt.initialTags) {
				t.Error("AddTag() modified original metadata")
			}
		})
	}
}

func TestNodeMetadata_RemoveTag(t *testing.T) {
	tests := []struct {
		name          string
		initialTags   []string
		tagToRemove   string
		expectedCount int
	}{
		{"remove existing tag", []string{"tag1", "tag2"}, "tag1", 1},
		{"remove non-existing tag", []string{"tag1", "tag2"}, "tag3", 2},
		{"case insensitive remove", []string{"Tag1", "tag2"}, "tag1", 1},
		{"remove from empty", []string{}, "tag1", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := NewNodeMetadata("US", "California", tt.initialTags, "")
			if err != nil {
				t.Fatalf("NewNodeMetadata() error = %v", err)
			}

			updated := metadata.RemoveTag(tt.tagToRemove)

			if updated.TagCount() != tt.expectedCount {
				t.Errorf("RemoveTag(%q) TagCount() = %d, want %d", tt.tagToRemove, updated.TagCount(), tt.expectedCount)
			}

			if metadata.TagCount() != len(tt.initialTags) {
				t.Error("RemoveTag() modified original metadata")
			}
		})
	}
}

func TestNodeMetadata_Equals(t *testing.T) {
	metadata1, _ := NewNodeMetadata("US", "California", []string{"tag1", "tag2"}, "description")
	metadata2, _ := NewNodeMetadata("US", "California", []string{"tag1", "tag2"}, "description")
	metadata3, _ := NewNodeMetadata("CN", "California", []string{"tag1", "tag2"}, "description")
	metadata4, _ := NewNodeMetadata("US", "Beijing", []string{"tag1", "tag2"}, "description")
	metadata5, _ := NewNodeMetadata("US", "California", []string{"tag1"}, "description")
	metadata6, _ := NewNodeMetadata("US", "California", []string{"tag1", "tag2"}, "different")
	metadata7, _ := NewNodeMetadata("US", "California", []string{"tag2", "tag1"}, "description")

	tests := []struct {
		name      string
		metadata1 *NodeMetadata
		metadata2 *NodeMetadata
		expected  bool
	}{
		{"same metadata", metadata1, metadata2, true},
		{"different country", metadata1, metadata3, false},
		{"different region", metadata1, metadata4, false},
		{"different tags", metadata1, metadata5, false},
		{"different description", metadata1, metadata6, false},
		{"tags in different order", metadata1, metadata7, true},
		{"nil equals nil", nil, nil, true},
		{"nil vs non-nil", nil, metadata1, false},
		{"non-nil vs nil", metadata1, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metadata1.Equals(tt.metadata2)
			if result != tt.expected {
				t.Errorf("Equals() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNodeMetadata_TagsImmutability(t *testing.T) {
	metadata, err := NewNodeMetadata("US", "California", []string{"tag1", "tag2"}, "")
	if err != nil {
		t.Fatalf("NewNodeMetadata() error = %v", err)
	}

	tags := metadata.Tags()
	tags[0] = "modified"
	tags = append(tags, "new-tag")

	if metadata.HasTag("modified") {
		t.Error("modifying returned Tags() slice affected internal state")
	}

	if metadata.HasTag("new-tag") {
		t.Error("appending to returned Tags() slice affected internal state")
	}

	if metadata.TagCount() != 2 {
		t.Errorf("TagCount() = %d, want 2", metadata.TagCount())
	}
}

func TestNodeMetadata_CaseInsensitiveTagComparison(t *testing.T) {
	metadata1, _ := NewNodeMetadata("US", "California", []string{"Premium", "Fast"}, "")
	metadata2, _ := NewNodeMetadata("US", "California", []string{"premium", "fast"}, "")

	if !metadata1.Equals(metadata2) {
		t.Error("Equals() should be case-insensitive for tags")
	}
}

func TestNodeMetadata_BoundaryConditions(t *testing.T) {
	t.Run("very long description", func(t *testing.T) {
		longDesc := strings.Repeat("a", 1000)
		_, err := NewNodeMetadata("US", "California", []string{}, longDesc)
		if err != nil {
			t.Errorf("NewNodeMetadata() with long description should succeed, got: %v", err)
		}
	})

	t.Run("many tags", func(t *testing.T) {
		manyTags := make([]string, 100)
		for i := range manyTags {
			manyTags[i] = "tag" + string(rune(i))
		}
		metadata, err := NewNodeMetadata("US", "California", manyTags, "")
		if err != nil {
			t.Errorf("NewNodeMetadata() with many tags should succeed, got: %v", err)
		}
		if metadata.TagCount() != 100 {
			t.Errorf("TagCount() = %d, want 100", metadata.TagCount())
		}
	})

	t.Run("very long region", func(t *testing.T) {
		longRegion := strings.Repeat("a", 1000)
		_, err := NewNodeMetadata("US", longRegion, []string{}, "")
		if err != nil {
			t.Errorf("NewNodeMetadata() with long region should succeed, got: %v", err)
		}
	})
}
