package value_objects

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

func NewNodeMetadata(country, region string, tags []string, description string) NodeMetadata {
	if tags == nil {
		tags = []string{}
	}

	return NodeMetadata{
		country:     strings.ToUpper(country),
		region:      region,
		tags:        tags,
		description: description,
	}
}

func (nm NodeMetadata) Country() string {
	return nm.country
}

func (nm NodeMetadata) Region() string {
	return nm.region
}

func (nm NodeMetadata) Tags() []string {
	return nm.tags
}

func (nm NodeMetadata) Description() string {
	return nm.description
}

func (nm NodeMetadata) HasTag(tag string) bool {
	for _, t := range nm.tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (nm NodeMetadata) DisplayName() string {
	if nm.country == "" && nm.region == "" {
		return "Unknown"
	}
	if nm.region == "" {
		return nm.country
	}
	return fmt.Sprintf("%s - %s", nm.country, nm.region)
}

func (nm NodeMetadata) Equals(other NodeMetadata) bool {
	if nm.country != other.country || nm.region != other.region || nm.description != other.description {
		return false
	}
	if len(nm.tags) != len(other.tags) {
		return false
	}
	tagMap := make(map[string]bool)
	for _, tag := range nm.tags {
		tagMap[tag] = true
	}
	for _, tag := range other.tags {
		if !tagMap[tag] {
			return false
		}
	}
	return true
}
