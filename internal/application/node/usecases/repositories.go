package usecases

import "context"

// NodeMode constants for subscription node filtering
const (
	NodeModeAll     = "all"     // Return all nodes (origin + forwarded)
	NodeModeForward = "forward" // Return only forwarded nodes
	NodeModeOrigin  = "origin"  // Return only origin nodes
)

type NodeRepository interface {
	GetBySubscriptionToken(ctx context.Context, token string, mode string) ([]*Node, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (NodeData, error)
}

type NodeData struct {
	ID        uint
	SID       string
	Name      string
	TokenHash string
	Status    string
}
