// Package dto provides data transfer objects for forward operations.
package dto

// BatchLimit defines the maximum number of items per batch operation.
const BatchLimit = 100

// BatchOperationResult represents the result of a batch operation.
type BatchOperationResult struct {
	Succeeded []string            `json:"succeeded"`
	Failed    []BatchOperationErr `json:"failed,omitempty"`
}

// BatchOperationErr represents an error for a single item in batch operation.
type BatchOperationErr struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// BatchCreateResult represents a single successful creation result.
type BatchCreateResult struct {
	Index int    `json:"index"` // original index in request
	ID    string `json:"id"`    // created rule SID
}

// BatchCreateResponse represents the response of batch creation.
type BatchCreateResponse struct {
	Succeeded []BatchCreateResult `json:"succeeded"`
	Failed    []BatchOperationErr `json:"failed,omitempty"`
}
