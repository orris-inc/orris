package usecases

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/infrastructure/config"
	"github.com/orris-inc/orris/internal/infrastructure/template"
	"github.com/orris-inc/orris/internal/shared/biztime"
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
	UserInfo    *SubscriptionUserInfo
}

// SubscriptionUserInfo contains traffic usage and subscription expiration info
// for the Subscription-Userinfo response header.
type SubscriptionUserInfo struct {
	Upload   uint64 // Current period upload bytes
	Download uint64 // Current period download bytes
	Total    uint64 // Traffic limit in bytes (0 = unlimited)
	Expire   int64  // Subscription end time as Unix timestamp
}

type SubscriptionValidationResult struct {
	SubscriptionID     uint
	SubscriptionUUID   string
	PlanID             uint
	EndDate            time.Time
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
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

// SubscriptionSettingProvider provides subscription-related settings.
type SubscriptionSettingProvider interface {
	// IsShowInfoNodesEnabled returns whether to show info nodes (expire/traffic) in subscription.
	IsShowInfoNodesEnabled(ctx context.Context) bool
}

type GenerateSubscriptionUseCase struct {
	nodeRepo        NodeRepository
	tokenValidator  SubscriptionTokenValidator
	planRepo        subscription.PlanRepository
	usageStatsRepo  subscription.SubscriptionUsageStatsRepository
	hourlyCache     cache.HourlyTrafficCache
	settingProvider SubscriptionSettingProvider
	formatters      map[string]SubscriptionFormatter
	logger          logger.Interface
}

func NewGenerateSubscriptionUseCase(
	nodeRepo NodeRepository,
	tokenValidator SubscriptionTokenValidator,
	templateLoader *template.SubscriptionTemplateLoader,
	planRepo subscription.PlanRepository,
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyCache cache.HourlyTrafficCache,
	settingProvider SubscriptionSettingProvider,
	logger logger.Interface,
) *GenerateSubscriptionUseCase {
	uc := &GenerateSubscriptionUseCase{
		nodeRepo:        nodeRepo,
		tokenValidator:  tokenValidator,
		planRepo:        planRepo,
		usageStatsRepo:  usageStatsRepo,
		hourlyCache:     hourlyCache,
		settingProvider: settingProvider,
		formatters:      make(map[string]SubscriptionFormatter),
		logger:          logger,
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
	// Validate subscription token and get subscription info
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
		uc.logger.Warnw("no available nodes found, returning empty subscription", "token", cmd.SubscriptionToken, "mode", nodeMode)
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

	// Build subscription user info for response header
	userInfo := uc.buildUserInfo(ctx, validationResult)

	// Optionally prepend info nodes (expire time and traffic usage) to the node list
	// This is controlled by admin setting "subscription.show_info_nodes"
	nodesToFormat := nodes
	if uc.settingProvider != nil && uc.settingProvider.IsShowInfoNodesEnabled(ctx) {
		nodesToFormat = uc.prependInfoNodes(nodes, userInfo)
	}

	// Pass HMAC password for node authentication
	content, err := formatter.FormatWithPassword(nodesToFormat, password)
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
		UserInfo:    userInfo,
	}, nil
}

// prependInfoNodes creates two info nodes (expire time and traffic usage) and prepends them to the node list.
// The info nodes copy the first real node's configuration but with descriptive names.
func (uc *GenerateSubscriptionUseCase) prependInfoNodes(nodes []*Node, userInfo *SubscriptionUserInfo) []*Node {
	if len(nodes) == 0 || userInfo == nil {
		return nodes
	}

	// Copy first node as template for info nodes
	templateNode := nodes[0]

	// Create expire info node
	expireNode := uc.createInfoNode(templateNode, uc.formatExpireInfo(userInfo.Expire), -2)

	// Create traffic info node
	trafficNode := uc.createInfoNode(templateNode, uc.formatTrafficInfo(userInfo), -1)

	// Prepend info nodes to the list
	result := make([]*Node, 0, len(nodes)+2)
	result = append(result, expireNode, trafficNode)
	result = append(result, nodes...)

	return result
}

// createInfoNode creates a copy of the template node with a new name for displaying info.
// Note: This creates a shallow copy which is safe for read-only usage in formatters.
// The pointer fields (VLESSConfig, etc.) and map fields (PluginOpts) are shared with the original node.
func (uc *GenerateSubscriptionUseCase) createInfoNode(template *Node, name string, sortOrder int) *Node {
	// Deep copy PluginOpts map to prevent unintended modifications
	var pluginOpts map[string]string
	if template.PluginOpts != nil {
		pluginOpts = make(map[string]string, len(template.PluginOpts))
		for k, v := range template.PluginOpts {
			pluginOpts[k] = v
		}
	}

	return &Node{
		ID:                template.ID,
		Name:              name,
		ServerAddress:     template.ServerAddress,
		SubscriptionPort:  template.SubscriptionPort,
		Protocol:          template.Protocol,
		EncryptionMethod:  template.EncryptionMethod,
		TokenHash:         template.TokenHash,
		Password:          template.Password,
		Plugin:            template.Plugin,
		PluginOpts:        pluginOpts,
		TransportProtocol: template.TransportProtocol,
		Host:              template.Host,
		Path:              template.Path,
		SNI:               template.SNI,
		AllowInsecure:     template.AllowInsecure,
		// Pointer fields are shared (shallow copy) - safe for read-only usage in formatters
		VLESSConfig:     template.VLESSConfig,
		VMessConfig:     template.VMessConfig,
		Hysteria2Config: template.Hysteria2Config,
		TUICConfig:      template.TUICConfig,
		AnyTLSConfig:    template.AnyTLSConfig,
		SortOrder:       sortOrder,
	}
}

// formatExpireInfo formats the expiration time for display in node name.
func (uc *GenerateSubscriptionUseCase) formatExpireInfo(expireUnix int64) string {
	if expireUnix == 0 {
		return "📅 到期: 永久有效"
	}
	expireTime := time.Unix(expireUnix, 0).In(biztime.Location())
	return fmt.Sprintf("📅 到期: %s", expireTime.Format("2006-01-02"))
}

// formatTrafficInfo formats the traffic usage for display in node name.
func (uc *GenerateSubscriptionUseCase) formatTrafficInfo(userInfo *SubscriptionUserInfo) string {
	used := userInfo.Upload + userInfo.Download
	total := userInfo.Total

	if total == 0 {
		// Unlimited traffic
		return fmt.Sprintf("📊 流量: %s / 无限制", formatBytes(used))
	}

	return fmt.Sprintf("📊 流量: %s / %s", formatBytes(used), formatBytes(total))
}

// formatBytes formats bytes to human-readable string (KB, MB, GB, TB).
func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2fTB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2fGB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// buildUserInfo constructs SubscriptionUserInfo with traffic usage and expiration info.
// Uses a hybrid approach: Redis for recent traffic (last 24h) and MySQL stats for historical data.
func (uc *GenerateSubscriptionUseCase) buildUserInfo(ctx context.Context, validation *SubscriptionValidationResult) *SubscriptionUserInfo {
	// Get plan to determine traffic limit
	plan, err := uc.planRepo.GetByID(ctx, validation.PlanID)
	if err != nil {
		uc.logger.Warnw("failed to get plan for user info",
			"plan_id", validation.PlanID,
			"error", err,
		)
		// Return minimal info with just expiration
		return &SubscriptionUserInfo{
			Expire: validation.EndDate.Unix(),
		}
	}

	if plan == nil {
		uc.logger.Warnw("plan not found for user info",
			"plan_id", validation.PlanID,
		)
		// Return minimal info with just expiration
		return &SubscriptionUserInfo{
			Expire: validation.EndDate.Unix(),
		}
	}

	// Get traffic limit from plan (0 = unlimited)
	trafficLimit, err := plan.GetTrafficLimit()
	if err != nil {
		uc.logger.Warnw("failed to get traffic limit from plan",
			"plan_id", validation.PlanID,
			"error", err,
		)
		trafficLimit = 0 // Treat as unlimited on error
	}

	// Calculate current period traffic usage
	upload, download := uc.calculatePeriodTraffic(ctx, validation, plan)

	return &SubscriptionUserInfo{
		Upload:   upload,
		Download: download,
		Total:    trafficLimit,
		Expire:   validation.EndDate.Unix(),
	}
}

// calculatePeriodTraffic calculates upload and download traffic for the current traffic period.
// Uses the plan's traffic_reset_mode to determine period boundaries:
// - calendar_month: business timezone month boundaries (default, backward compatible)
// - billing_cycle: subscription's CurrentPeriodStart/CurrentPeriodEnd
// Uses Redis for recent data (last 24h) and MySQL stats for historical data.
func (uc *GenerateSubscriptionUseCase) calculatePeriodTraffic(ctx context.Context, validation *SubscriptionValidationResult, plan *subscription.Plan) (upload, download uint64) {
	now := biztime.NowUTC()

	// Determine period based on plan's traffic reset mode
	var periodStart, periodEnd time.Time
	mode := subscription.GetTrafficResetMode(plan)
	if mode == subscription.TrafficResetBillingCycle {
		periodStart = validation.CurrentPeriodStart
		periodEnd = validation.CurrentPeriodEnd
	} else {
		bizNow := biztime.ToBizTimezone(now)
		periodStart = biztime.StartOfMonthUTC(bizNow.Year(), bizNow.Month())
		periodEnd = biztime.EndOfMonthUTC(bizNow.Year(), bizNow.Month())
	}

	// If period end is in the future, use current time
	if periodEnd.After(now) {
		periodEnd = now
	}

	// Use start of yesterday's business day as batch/speed boundary (Lambda architecture)
	// MySQL: complete days before yesterday; Redis: yesterday + today (within 48h TTL)
	recentBoundary := biztime.StartOfDayUTC(now.AddDate(0, 0, -1))

	// Get historical traffic from MySQL stats (complete days before yesterday)
	if periodStart.Before(recentBoundary) {
		historicalTo := recentBoundary.Add(-time.Second)
		if historicalTo.After(periodEnd) {
			historicalTo = periodEnd
		}
		// Aggregate all resource types (nil = no filter)
		historicalTraffic, err := uc.usageStatsRepo.GetTotalBySubscriptionIDs(
			ctx, []uint{validation.SubscriptionID}, nil, subscription.GranularityDaily, periodStart, historicalTo,
		)
		if err != nil {
			uc.logger.Warnw("failed to get historical traffic from stats",
				"subscription_id", validation.SubscriptionID,
				"from", periodStart,
				"to", historicalTo,
				"error", err,
			)
		} else if historicalTraffic != nil {
			upload += historicalTraffic.Upload
			download += historicalTraffic.Download
		}
	}

	// Get recent traffic from Redis (yesterday + today)
	recentFrom := periodStart
	if recentFrom.Before(recentBoundary) {
		recentFrom = recentBoundary
	}

	if recentFrom.Before(periodEnd) && recentFrom.Before(now) {
		recentTo := periodEnd
		if recentTo.After(now) {
			recentTo = now
		}
		// Aggregate all resource types (empty string = no filter)
		recentTraffic, err := uc.hourlyCache.GetTotalTrafficBySubscriptionIDs(
			ctx, []uint{validation.SubscriptionID}, "", recentFrom, recentTo,
		)
		if err != nil {
			uc.logger.Warnw("failed to get recent traffic from Redis",
				"subscription_id", validation.SubscriptionID,
				"from", recentFrom,
				"to", recentTo,
				"error", err,
			)
		} else {
			for _, t := range recentTraffic {
				upload += t.Upload
				download += t.Download
			}
		}
	}

	return upload, download
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
	AnyTLSConfig    *valueobjects.AnyTLSConfig
	// Sorting field for subscription output ordering
	SortOrder int
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
