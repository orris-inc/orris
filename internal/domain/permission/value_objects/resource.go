package value_objects

import "fmt"

type Resource string

func NewResource(resource string) (Resource, error) {
	if resource == "" {
		return "", fmt.Errorf("resource cannot be empty")
	}
	if len(resource) > 50 {
		return "", fmt.Errorf("resource too long (max 50 characters)")
	}
	return Resource(resource), nil
}

func (r Resource) String() string {
	return string(r)
}

func (r Resource) Equals(other Resource) bool {
	return r == other
}
