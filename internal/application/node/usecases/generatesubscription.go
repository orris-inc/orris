package usecases

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/config"
	"github.com/orris-inc/orris/internal/infrastructure/template"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type GenerateSubscriptionCommand struct {
	SubscriptionToken string
	Format            string
	NodeMode          string // "all" | "forward" | "origin", defaults to "all"
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
	templateLoader *template.SubscriptionTemplateLoader,
	logger logger.Interface,
) *GenerateSubscriptionUseCase {
	uc := &GenerateSubscriptionUseCase{
		nodeRepo:       nodeRepo,
		tokenValidator: tokenValidator,
		formatters:     make(map[string]SubscriptionFormatter),
		logger:         logger,
	}

	// Create template renderer
	renderer := NewTemplateRenderer(templateLoader)

	// Use template-aware formatters for clash and surge
	uc.formatters["clash"] = NewTemplateClashFormatter(renderer)
	uc.formatters["surge"] = NewTemplateSurgeFormatter(renderer)

	// Keep original formatters for other formats
	uc.formatters["base64"] = NewBase64Formatter()
	uc.formatters["v2ray"] = NewV2RayFormatter()
	uc.formatters["sip008"] = NewSIP008Formatter()

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

	// Default node mode to "all" if not specified
	nodeMode := cmd.NodeMode
	if nodeMode == "" {
		nodeMode = NodeModeAll
	}

	nodes, err := uc.nodeRepo.GetBySubscriptionToken(ctx, cmd.SubscriptionToken, nodeMode)
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

	// Debug: log if password is empty
	if password == "" {
		uc.logger.Warnw("generated empty password",
			"subscription_uuid", subscriptionUUID,
			"uuid_empty", subscriptionUUID == "",
			"secret_empty", hmacSecret == "",
		)
	}

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
	SubscriptionPort uint16 // port for client subscriptions (effective port)
	Protocol         string // shadowsocks, trojan, vless, vmess, hysteria2, tuic
	EncryptionMethod string // for shadowsocks
	TokenHash        string // Node token hash for SS2022 ServerKey derivation
	Password         string
	Plugin           string
	PluginOpts       map[string]string
	// Trojan specific fields
	TransportProtocol string // tcp, ws, grpc
	Host              string // WebSocket host / gRPC service name
	Path              string // WebSocket path
	SNI               string // TLS Server Name Indication
	AllowInsecure     bool   // Allow insecure TLS connection
	// New protocol specific fields
	VLESSConfig     *valueobjects.VLESSConfig
	VMessConfig     *valueobjects.VMessConfig
	Hysteria2Config *valueobjects.Hysteria2Config
	TUICConfig      *valueobjects.TUICConfig
}

// ToTrojanURI generates a Trojan URI string for subscription
// Delegates to domain layer TrojanConfig.ToURI for consistent URI generation
func (n *Node) ToTrojanURI(password string) string {
	// Default transport protocol to tcp if not specified
	transportProtocol := n.TransportProtocol
	if transportProtocol == "" {
		transportProtocol = "tcp"
	}

	// Create TrojanConfig from Node fields (validation already done at node creation)
	config, err := valueobjects.NewTrojanConfig(
		password,
		transportProtocol,
		n.Host,
		n.Path,
		n.AllowInsecure,
		n.SNI,
	)
	if err != nil {
		// Fallback: should not happen as node was already validated
		return fmt.Sprintf("trojan://%s@%s:%d#%s", password, n.ServerAddress, n.SubscriptionPort, n.Name)
	}

	return config.ToURI(n.ServerAddress, n.SubscriptionPort, n.Name)
}

// generateHMACPassword generates HMAC-SHA256 password from subscription UUID
// Returns hex-encoded password for traditional SS compatibility
// This must match the password generation in agentdto.go for agent authentication
func generateHMACPassword(subscriptionUUID, secret string) string {
	if subscriptionUUID == "" || secret == "" {
		return ""
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(subscriptionUUID))

	return hex.EncodeToString(mac.Sum(nil))
}
