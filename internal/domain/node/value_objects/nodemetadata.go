package value_objects

type NodeMetadata struct {
	region      string
	tags        []string
	description string
}

func NewNodeMetadata(region string, tags []string, description string) NodeMetadata {
	if tags == nil {
		tags = []string{}
	}

	return NodeMetadata{
		region:      region,
		tags:        tags,
		description: description,
	}
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
	if nm.region == "" {
		return "Unknown"
	}
	return nm.region
}

func (nm NodeMetadata) Equals(other NodeMetadata) bool {
	if nm.region != other.region || nm.description != other.description {
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
