// Package services provides infrastructure services.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// maxRuleCacheSize is the maximum number of rule entries to cache.
	// Prevents memory exhaustion from malicious agents sending many different rule IDs.
	maxRuleCacheSize = 10000

	// maxTrafficPerReport is the maximum traffic bytes allowed per single report.
	// Prevents integer overflow attacks (1TB per report is more than reasonable).
	maxTrafficPerReport = 1 << 40 // 1TB

	// MsgTypeEvent is the message type for events from agents.
	// Traffic data is sent as an event with event_type: "traffic".
	MsgTypeEvent = "event"

	// EventTypeTraffic is the event type for traffic updates.
	EventTypeTraffic = "traffic"
)

// agentEventData represents an agent event payload (mirrors dto.AgentEventData).
type agentEventData struct {
	EventType string `json:"event_type"`
	Message   string `json:"message,omitempty"`
	Extra     any    `json:"extra,omitempty"`
}

// TrafficMessage represents traffic data sent from agents.
// This is the payload in Extra field when EventType is "traffic".
type TrafficMessage struct {
	Rules []TrafficItem `json:"rules"`
}

// TrafficItem represents traffic data for a single rule.
type TrafficItem struct {
	RuleID        string `json:"rule_id"` // Stripe-style SID (e.g., "fr_xxx")
	UploadBytes   int64  `json:"upload_bytes"`
	DownloadBytes int64  `json:"download_bytes"`
}

// RuleTrafficBufferWriter defines the interface for writing rule traffic entries to a buffer.
type RuleTrafficBufferWriter interface {
	AddTraffic(ruleID uint, upload, download int64)
}

// ForwardTrafficRecorder defines the interface for recording forward rule traffic to hourly cache.
// This interface is defined locally to avoid import cycle with adapters package.
type ForwardTrafficRecorder interface {
	// RecordForwardTraffic records forward rule traffic to Redis HourlyTrafficCache.
	// If subscriptionID is provided, records to that subscription's hourly bucket.
	// If subscriptionID is nil (admin rule), skip recording.
	RecordForwardTraffic(ctx context.Context, ruleID uint, subscriptionID *uint, upload, download int64) error
}

// cachedRuleInfo holds cached rule information to avoid frequent database queries.
type cachedRuleInfo struct {
	internalID          uint
	subscriptionID      *uint
	effectiveMultiplier float64
	cachedAt            time.Time
}

// TrafficMessageHandler handles traffic messages from forward agents.
// It implements the MessageHandler interface.
type TrafficMessageHandler struct {
	buffer          RuleTrafficBufferWriter
	ruleRepo        forward.Repository
	trafficRecorder ForwardTrafficRecorder
	logger          logger.Interface

	// Rule info cache with LRU eviction (avoids database queries for each traffic update)
	// Using LRU cache with size limit to prevent memory exhaustion attacks
	ruleCache *lru.Cache[string, *cachedRuleInfo]
	cacheTTL  time.Duration
}

// NewTrafficMessageHandler creates a new TrafficMessageHandler.
func NewTrafficMessageHandler(
	buffer RuleTrafficBufferWriter,
	ruleRepo forward.Repository,
	trafficRecorder ForwardTrafficRecorder,
	log logger.Interface,
) *TrafficMessageHandler {
	// Initialize LRU cache with size limit to prevent memory exhaustion
	cache, err := lru.New[string, *cachedRuleInfo](maxRuleCacheSize)
	if err != nil {
		// This should never happen with valid size, but log just in case
		log.Errorw("failed to create LRU cache, using fallback", "error", err)
		cache, _ = lru.New[string, *cachedRuleInfo](1000)
	}

	return &TrafficMessageHandler{
		buffer:          buffer,
		ruleRepo:        ruleRepo,
		trafficRecorder: trafficRecorder,
		logger:          log,
		ruleCache:       cache,
		cacheTTL:        5 * time.Minute,
	}
}

// String returns the handler name for logging purposes.
func (h *TrafficMessageHandler) String() string {
	return "TrafficMessageHandler"
}

// HandleMessage processes traffic messages from forward agents.
// Traffic is sent as an event message with event_type: "traffic".
// Returns true if the message was handled, false otherwise.
func (h *TrafficMessageHandler) HandleMessage(agentID uint, msgType string, data any) bool {
	// Only handle event messages
	if msgType != MsgTypeEvent {
		return false
	}

	// Parse event data
	dataBytes, err := json.Marshal(data)
	if err != nil {
		h.logger.Warnw("failed to marshal event data",
			"agent_id", agentID,
			"error", err,
		)
		return false
	}

	var event agentEventData
	if err := json.Unmarshal(dataBytes, &event); err != nil {
		h.logger.Warnw("failed to parse event data",
			"agent_id", agentID,
			"error", err,
		)
		return false
	}

	// Only handle traffic events
	if event.EventType != EventTypeTraffic {
		return false
	}

	// Parse traffic data from Extra field
	if event.Extra == nil {
		return true
	}

	extraBytes, err := json.Marshal(event.Extra)
	if err != nil {
		h.logger.Warnw("failed to marshal traffic extra data",
			"agent_id", agentID,
			"error", err,
		)
		return true
	}

	var msg TrafficMessage
	if err := json.Unmarshal(extraBytes, &msg); err != nil {
		h.logger.Warnw("failed to parse traffic message",
			"agent_id", agentID,
			"error", err,
		)
		return true
	}

	h.logger.Debugw("traffic event received",
		"agent_id", agentID,
		"rules_count", len(msg.Rules),
	)

	ctx := context.Background()
	for _, item := range msg.Rules {
		h.processTrafficItem(ctx, agentID, item)
	}

	return true
}

// processTrafficItem processes a single traffic item and adds it to the buffer.
func (h *TrafficMessageHandler) processTrafficItem(ctx context.Context, agentID uint, item TrafficItem) {
	// Validate traffic data - reject negative values
	if item.UploadBytes < 0 || item.DownloadBytes < 0 {
		h.logger.Warnw("negative traffic rejected",
			"rule_id", item.RuleID,
			"agent_id", agentID,
		)
		return
	}

	// Validate traffic data - reject excessively large values to prevent integer overflow
	if item.UploadBytes > maxTrafficPerReport || item.DownloadBytes > maxTrafficPerReport {
		h.logger.Warnw("excessive traffic rejected",
			"rule_id", item.RuleID,
			"agent_id", agentID,
			"upload_bytes", item.UploadBytes,
			"download_bytes", item.DownloadBytes,
			"max_allowed", maxTrafficPerReport,
		)
		return
	}

	// Skip zero traffic
	if item.UploadBytes == 0 && item.DownloadBytes == 0 {
		return
	}

	// Get rule internal ID (with caching)
	ruleInfo, err := h.getRuleInfo(ctx, item.RuleID)
	if err != nil {
		h.logger.Debugw("skip traffic for invalid rule",
			"rule_id", item.RuleID,
			"agent_id", agentID,
			"error", err,
		)
		return
	}

	// Add to buffer (writes to forward_rules table)
	h.buffer.AddTraffic(ruleInfo.internalID, item.UploadBytes, item.DownloadBytes)

	// Also record traffic to HourlyTrafficCache for subscription usage tracking.
	// Apply traffic multiplier before recording.
	// Only record if rule has a subscription (user rule); skip admin rules.
	if h.trafficRecorder != nil && ruleInfo.subscriptionID != nil {
		// Apply multiplier to get the effective traffic for billing/usage tracking
		// Use safe multiplication to prevent integer overflow
		effectiveUpload := safeMultiplyTraffic(item.UploadBytes, ruleInfo.effectiveMultiplier)
		effectiveDownload := safeMultiplyTraffic(item.DownloadBytes, ruleInfo.effectiveMultiplier)
		if err := h.trafficRecorder.RecordForwardTraffic(ctx, ruleInfo.internalID, ruleInfo.subscriptionID, effectiveUpload, effectiveDownload); err != nil {
			// Log warning but don't fail - buffer update already succeeded
			h.logger.Warnw("failed to record forward traffic to hourly cache",
				"rule_id", item.RuleID,
				"internal_id", ruleInfo.internalID,
				"subscription_id", *ruleInfo.subscriptionID,
				"error", err,
			)
		}
	}
}

// getRuleInfo retrieves rule information with caching.
func (h *TrafficMessageHandler) getRuleInfo(ctx context.Context, ruleSID string) (*cachedRuleInfo, error) {
	// Check cache
	if info, ok := h.ruleCache.Get(ruleSID); ok {
		if time.Since(info.cachedAt) < h.cacheTTL {
			return info, nil
		}
		// Cache entry expired, will refresh below
	}

	// Fetch from database
	rule, err := h.ruleRepo.GetBySID(ctx, ruleSID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}
	if rule == nil {
		return nil, fmt.Errorf("rule not found: %s", ruleSID)
	}

	// Cache the result (LRU will automatically evict oldest entries if full)
	info := &cachedRuleInfo{
		internalID:          rule.ID(),
		subscriptionID:      rule.SubscriptionID(),
		effectiveMultiplier: rule.GetEffectiveMultiplier(),
		cachedAt:            time.Now(),
	}
	h.ruleCache.Add(ruleSID, info)

	return info, nil
}

// InvalidateCache removes a rule from the cache.
// This should be called when a rule is deleted.
func (h *TrafficMessageHandler) InvalidateCache(ruleSID string) {
	h.ruleCache.Remove(ruleSID)
}

// ClearCache clears all cached rule information.
func (h *TrafficMessageHandler) ClearCache() {
	h.ruleCache.Purge()
}

// safeMultiplyTraffic safely multiplies traffic bytes by a multiplier,
// capping at math.MaxInt64 to prevent integer overflow.
func safeMultiplyTraffic(bytes int64, multiplier float64) int64 {
	if bytes <= 0 || multiplier <= 0 {
		return 0
	}

	result := float64(bytes) * multiplier

	// Cap at MaxInt64 to prevent overflow when converting back to int64
	if result > float64(math.MaxInt64) {
		return math.MaxInt64
	}

	return int64(result)
}
