package usecases

import (
	"context"
	"fmt"

	"orris/internal/shared/logger"
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

type SubscriptionTokenValidator interface {
	Validate(ctx context.Context, token string) error
}

type SubscriptionFormatter interface {
	Format(nodes []*Node) (string, error)
	ContentType() string
}

type GenerateSubscriptionUseCase struct {
	nodeRepo          NodeRepository
	tokenValidator    SubscriptionTokenValidator
	formatters        map[string]SubscriptionFormatter
	logger            logger.Interface
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
	if err := uc.tokenValidator.Validate(ctx, cmd.SubscriptionToken); err != nil {
		uc.logger.Warnw("invalid subscription token", "error", err)
		return nil, fmt.Errorf("invalid subscription token: %w", err)
	}

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

	content, err := formatter.Format(nodes)
	if err != nil {
		uc.logger.Errorw("failed to format subscription", "error", err, "format", cmd.Format)
		return nil, fmt.Errorf("failed to format subscription: %w", err)
	}

	uc.logger.Infow("subscription generated successfully",
		"format", cmd.Format,
		"node_count", len(nodes),
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
