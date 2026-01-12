package usecases

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"

	"github.com/orris-inc/orris/internal/shared/logger"
)

type ValidateNodeTokenCommand struct {
	PlainToken string
	IPAddress  string
}

type ValidateNodeTokenResult struct {
	NodeID  uint
	NodeSID string
	Name    string
}

type ValidateNodeTokenUseCase struct {
	nodeRepo NodeRepository
	logger   logger.Interface
}

func NewValidateNodeTokenUseCase(
	nodeRepo NodeRepository,
	logger logger.Interface,
) *ValidateNodeTokenUseCase {
	return &ValidateNodeTokenUseCase{
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

func (uc *ValidateNodeTokenUseCase) Execute(ctx context.Context, cmd ValidateNodeTokenCommand) (*ValidateNodeTokenResult, error) {
	tokenHash := hashToken(cmd.PlainToken)

	node, err := uc.nodeRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		uc.logger.Warnw("node token not found", "error", err)
		return nil, fmt.Errorf("invalid token")
	}

	if !verifyToken(cmd.PlainToken, node.TokenHash) {
		uc.logger.Warnw("token verification failed", "node_id", node.ID)
		return nil, fmt.Errorf("token verification failed")
	}

	// Node can be connected regardless of activation status.
	// Status is only used for business logic (e.g., subscription routing).

	uc.logger.Debugw("node token validated",
		"node_id", node.ID,
	)

	return &ValidateNodeTokenResult{
		NodeID:  node.ID,
		NodeSID: node.SID,
		Name:    node.Name,
	}, nil
}

func hashToken(plainToken string) string {
	hash := sha256.Sum256([]byte(plainToken))
	return hex.EncodeToString(hash[:])
}

func verifyToken(plainToken, tokenHash string) bool {
	computedHash := hashToken(plainToken)
	return subtle.ConstantTimeCompare([]byte(computedHash), []byte(tokenHash)) == 1
}
