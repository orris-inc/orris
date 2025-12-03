package usecases

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/orris-inc/orris/internal/infrastructure/config"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type GenerateSubscriptionCommand struct {
	SubscriptionToken string
	Format            string
}

type GenerateSubscriptionResult struct {
	Content     string
	ContentType string
	Format      string
}

type SubscriptionValidationResult struct {
	SubscriptionUUID string
}

type SubscriptionTokenValidator interface {
	Validate(ctx context.Context, token string) error
	ValidateAndGetSubscription(ctx context.Context, token string) (*SubscriptionValidationResult, error)
}

type SubscriptionFormatter interface {
	Format(nodes []*Node) (string, error)
	FormatWithPassword(nodes []*Node, password string) (string, error)
	ContentType() string
}

type GenerateSubscriptionUseCase struct {
	nodeRepo       NodeRepository
	tokenValidator SubscriptionTokenValidator
	formatters     map[string]SubscriptionFormatter
	logger         logger.Interface
}

func NewGenerateSubscriptionUseCase(
	nodeRepo NodeRepository,
	tokenValidator SubscriptionTokenValidator,
	logger logger.Interface,
) *GenerateSubscriptionUseCase {
	uc := &GenerateSubscriptionUseCase{
		nodeRepo:       nodeRepo,
		tokenValidator: tokenValidator,
		formatters:     make(map[string]SubscriptionFormatter),
		logger:         logger,
	}

	uc.formatters["base64"] = NewBase64Formatter()
	uc.formatters["clash"] = NewClashFormatter()
	uc.formatters["v2ray"] = NewV2RayFormatter()
	uc.formatters["sip008"] = NewSIP008Formatter()
	uc.formatters["surge"] = NewSurgeFormatter()

	return uc
}

func (uc *GenerateSubscriptionUseCase) Execute(ctx context.Context, cmd GenerateSubscriptionCommand) (*GenerateSubscriptionResult, error) {
	// Validate subscription token and get subscription UUID
	validationResult, err := uc.tokenValidator.ValidateAndGetSubscription(ctx, cmd.SubscriptionToken)
	if err != nil {
		uc.logger.Warnw("invalid subscription token", "error", err)
		return nil, fmt.Errorf("invalid subscription token: %w", err)
	}

	// Get subscription UUID for authentication
	subscriptionUUID := validationResult.SubscriptionUUID

	nodes, err := uc.nodeRepo.GetBySubscriptionToken(ctx, cmd.SubscriptionToken)
	if err != nil {
		uc.logger.Errorw("failed to get nodes", "error", err)
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	if len(nodes) == 0 {
		uc.logger.Warnw("no available nodes found", "token", cmd.SubscriptionToken)
		return nil, fmt.Errorf("no available nodes")
	}

	formatter, ok := uc.formatters[cmd.Format]
	if !ok {
		uc.logger.Warnw("unsupported format", "format", cmd.Format)
		return nil, fmt.Errorf("unsupported format: %s", cmd.Format)
	}

	// Generate HMAC password from subscription UUID (must match agent password generation)
	hmacSecret := config.Get().Auth.JWT.Secret
	password := generateHMACPassword(subscriptionUUID, hmacSecret)

	// Pass HMAC password for node authentication
	content, err := formatter.FormatWithPassword(nodes, password)
	if err != nil {
		uc.logger.Errorw("failed to format subscription", "error", err, "format", cmd.Format)
		return nil, fmt.Errorf("failed to format subscription: %w", err)
	}

	uc.logger.Infow("subscription generated successfully",
		"format", cmd.Format,
		"node_count", len(nodes),
		"subscription_uuid", subscriptionUUID,
	)

	return &GenerateSubscriptionResult{
		Content:     content,
		ContentType: formatter.ContentType(),
		Format:      cmd.Format,
	}, nil
}

type Node struct {
	ID               uint
	Name             string
	ServerAddress    string
	ServerPort       uint16
	EncryptionMethod string
	Password         string
	Plugin           string
	PluginOpts       map[string]string
}

// generateHMACPassword generates HMAC-SHA256 password from subscription UUID
// This must match the password generation in agentdto.go for agent authentication
func generateHMACPassword(subscriptionUUID, secret string) string {
	if subscriptionUUID == "" || secret == "" {
		return ""
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(subscriptionUUID))

	return hex.EncodeToString(mac.Sum(nil))
}
