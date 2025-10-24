package node

import (
	"fmt"
	"strings"
)

type NodeMetadata struct {
	country     string
	region      string
	tags        []string
	description string
}

func NewNodeMetadata(country, region string, tags []string, description string) (*NodeMetadata, error) {
	normalizedCountry := strings.ToUpper(strings.TrimSpace(country))

	if normalizedCountry == "" {
		return nil, fmt.Errorf("country code cannot be empty")
	}

	if len(normalizedCountry) != 2 {
		return nil, fmt.Errorf("country code must be 2 characters (ISO 3166-1 alpha-2)")
	}

	normalizedRegion := strings.TrimSpace(region)
	if normalizedRegion == "" {
		return nil, fmt.Errorf("region cannot be empty")
	}

	normalizedTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			normalizedTags = append(normalizedTags, trimmed)
		}
	}

	return &NodeMetadata{
		country:     normalizedCountry,
		region:      normalizedRegion,
		tags:        normalizedTags,
		description: strings.TrimSpace(description),
	}, nil
}

func (nm *NodeMetadata) Country() string {
	return nm.country
}

func (nm *NodeMetadata) Region() string {
	return nm.region
}

func (nm *NodeMetadata) Tags() []string {
	tagsCopy := make([]string, len(nm.tags))
	copy(tagsCopy, nm.tags)
	return tagsCopy
}

func (nm *NodeMetadata) Description() string {
	return nm.description
}

func (nm *NodeMetadata) HasTag(tag string) bool {
	normalizedTag := strings.TrimSpace(tag)
	for _, t := range nm.tags {
		if strings.EqualFold(t, normalizedTag) {
			return true
		}
	}
	return false
}

func (nm *NodeMetadata) TagCount() int {
	return len(nm.tags)
}

func (nm *NodeMetadata) DisplayName() string {
	return fmt.Sprintf("%s - %s", nm.country, nm.region)
}

func (nm *NodeMetadata) FullDisplayName() string {
	if nm.description != "" {
		return fmt.Sprintf("%s - %s (%s)", nm.country, nm.region, nm.description)
	}
	return nm.DisplayName()
}

func (nm *NodeMetadata) WithDescription(description string) *NodeMetadata {
	return &NodeMetadata{
		country:     nm.country,
		region:      nm.region,
		tags:        nm.Tags(),
		description: strings.TrimSpace(description),
	}
}

func (nm *NodeMetadata) WithTags(tags []string) *NodeMetadata {
	normalizedTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			normalizedTags = append(normalizedTags, trimmed)
		}
	}

	return &NodeMetadata{
		country:     nm.country,
		region:      nm.region,
		tags:        normalizedTags,
		description: nm.description,
	}
}

func (nm *NodeMetadata) AddTag(tag string) *NodeMetadata {
	trimmed := strings.TrimSpace(tag)
	if trimmed == "" || nm.HasTag(trimmed) {
		return nm
	}

	newTags := append(nm.Tags(), trimmed)
	return &NodeMetadata{
		country:     nm.country,
		region:      nm.region,
		tags:        newTags,
		description: nm.description,
	}
}

func (nm *NodeMetadata) RemoveTag(tag string) *NodeMetadata {
	normalizedTag := strings.TrimSpace(tag)
	newTags := make([]string, 0, len(nm.tags))

	for _, t := range nm.tags {
		if !strings.EqualFold(t, normalizedTag) {
			newTags = append(newTags, t)
		}
	}

	return &NodeMetadata{
		country:     nm.country,
		region:      nm.region,
		tags:        newTags,
		description: nm.description,
	}
}

func (nm *NodeMetadata) Equals(other *NodeMetadata) bool {
	if nm == nil || other == nil {
		return nm == other
	}

	if nm.country != other.country || nm.region != other.region || nm.description != other.description {
		return false
	}

	if len(nm.tags) != len(other.tags) {
		return false
	}

	tagMap := make(map[string]bool)
	for _, tag := range nm.tags {
		tagMap[strings.ToLower(tag)] = true
	}

	for _, tag := range other.tags {
		if !tagMap[strings.ToLower(tag)] {
			return false
		}
	}

	return true
}
