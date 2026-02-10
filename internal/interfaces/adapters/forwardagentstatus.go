package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/interfaces/adapters/systemstatus"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	forwardAgentStatusKeyPrefix = "forward_agent:%d:status"
	forwardAgentStatusTTL       = 5 * time.Minute

	// MaxTunnelStatusEntries limits the number of tunnel status entries to prevent memory exhaustion.
	MaxTunnelStatusEntries = 100

	// MaxTunnelStatusValueLen limits the length of each tunnel status value.
	MaxTunnelStatusValueLen = 64

	// MaxTunnelStatusKeyLen limits the length of each tunnel status key.
	// Rule SID format: "fr_xK9mP2vL3nQ" (15 chars max)
	MaxTunnelStatusKeyLen = 32
)

// ForwardAgentStatusUpdater updates forward agent status in Redis.
type ForwardAgentStatusUpdater interface {
	UpdateStatus(ctx context.Context, agentID uint, status *dto.AgentStatusDTO) error
}

// ForwardAgentStatusQuerier queries forward agent status from Redis.
type ForwardAgentStatusQuerier interface {
	GetStatus(ctx context.Context, agentID uint) (*dto.AgentStatusDTO, error)
	GetMultipleStatus(ctx context.Context, agentIDs []uint) (map[uint]*dto.AgentStatusDTO, error)
}

// ForwardAgentStatusAdapter implements both ForwardAgentStatusUpdater and ForwardAgentStatusQuerier.
type ForwardAgentStatusAdapter struct {
	redisClient *redis.Client
	logger      logger.Interface
}

// NewForwardAgentStatusAdapter creates a new forward agent status adapter.
func NewForwardAgentStatusAdapter(
	redisClient *redis.Client,
	logger logger.Interface,
) *ForwardAgentStatusAdapter {
	return &ForwardAgentStatusAdapter{
		redisClient: redisClient,
		logger:      logger,
	}
}

// UpdateStatus updates forward agent status in Redis.
func (a *ForwardAgentStatusAdapter) UpdateStatus(ctx context.Context, agentID uint, status *dto.AgentStatusDTO) error {
	key := fmt.Sprintf(forwardAgentStatusKeyPrefix, agentID)

	// Get base system status fields
	fields := systemstatus.ToRedisFields(&status.SystemStatus)

	// Add forward-specific fields
	fields["active_rules"] = status.ActiveRules
	fields["active_connections"] = status.ActiveConnections
	fields["ws_listen_port"] = status.WsListenPort
	fields["tls_listen_port"] = status.TlsListenPort
	fields["updated_at"] = biztime.NowUTC().Unix()

	// Store status in Redis hash with TTL
	pipe := a.redisClient.Pipeline()
	pipe.HSet(ctx, key, fields)

	// Store tunnel status as JSON if present (with size limits to prevent memory exhaustion)
	if len(status.TunnelStatus) > 0 {
		// Limit map size, key lengths, and value lengths
		limitedTunnelStatus := make(map[string]string)
		count := 0
		for k, v := range status.TunnelStatus {
			if count >= MaxTunnelStatusEntries {
				a.logger.Warnw("tunnel status entries truncated",
					"agent_id", agentID,
					"max", MaxTunnelStatusEntries,
				)
				break
			}
			// Skip entries with keys that are too long (invalid rule SID)
			if len(k) > MaxTunnelStatusKeyLen {
				continue
			}
			// Truncate value if too long
			if len(v) > MaxTunnelStatusValueLen {
				v = v[:MaxTunnelStatusValueLen]
			}
			limitedTunnelStatus[k] = v
			count++
		}

		tunnelJSON, err := json.Marshal(limitedTunnelStatus)
		if err == nil {
			pipe.HSet(ctx, key, "tunnel_status", string(tunnelJSON))
		}
	}

	pipe.Expire(ctx, key, forwardAgentStatusTTL)

	_, err := pipe.Exec(ctx)
	if err != nil {
		a.logger.Errorw("failed to store forward agent status in redis",
			"error", err,
			"agent_id", agentID,
		)
		return fmt.Errorf("failed to store forward agent status: %w", err)
	}

	a.logger.Debugw("forward agent status updated in redis",
		"agent_id", agentID,
		"cpu", status.CPUPercent,
		"memory", status.MemoryPercent,
		"active_rules", status.ActiveRules,
		"agent_version", status.AgentVersion,
		"platform", status.Platform,
		"arch", status.Arch,
	)

	return nil
}

// GetStatus retrieves forward agent status from Redis.
func (a *ForwardAgentStatusAdapter) GetStatus(ctx context.Context, agentID uint) (*dto.AgentStatusDTO, error) {
	key := fmt.Sprintf(forwardAgentStatusKeyPrefix, agentID)

	values, err := a.redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		a.logger.Errorw("failed to get forward agent status from redis",
			"error", err,
			"agent_id", agentID,
		)
		return nil, fmt.Errorf("failed to get forward agent status: %w", err)
	}

	if len(values) == 0 {
		return nil, nil
	}

	return a.parseStatus(values), nil
}

// GetMultipleStatus retrieves status for multiple forward agents in batch.
func (a *ForwardAgentStatusAdapter) GetMultipleStatus(ctx context.Context, agentIDs []uint) (map[uint]*dto.AgentStatusDTO, error) {
	result := make(map[uint]*dto.AgentStatusDTO)

	if len(agentIDs) == 0 {
		return result, nil
	}

	// Use pipeline for efficient batch querying
	pipe := a.redisClient.Pipeline()
	cmds := make(map[uint]*redis.MapStringStringCmd)

	for _, agentID := range agentIDs {
		key := fmt.Sprintf(forwardAgentStatusKeyPrefix, agentID)
		cmds[agentID] = pipe.HGetAll(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		a.logger.Errorw("failed to get multiple forward agent statuses from redis",
			"error", err,
			"agent_count", len(agentIDs),
		)
		return result, fmt.Errorf("failed to get forward agent statuses: %w", err)
	}

	for agentID, cmd := range cmds {
		values, err := cmd.Result()
		if err != nil || len(values) == 0 {
			continue
		}
		result[agentID] = a.parseStatus(values)
	}

	return result, nil
}

func (a *ForwardAgentStatusAdapter) parseStatus(values map[string]string) *dto.AgentStatusDTO {
	status := &dto.AgentStatusDTO{
		SystemStatus: systemstatus.ParseSystemStatus(values),
	}

	// Parse forward-specific fields
	fmt.Sscanf(values["active_rules"], "%d", &status.ActiveRules)
	fmt.Sscanf(values["active_connections"], "%d", &status.ActiveConnections)
	fmt.Sscanf(values["ws_listen_port"], "%d", &status.WsListenPort)
	fmt.Sscanf(values["tls_listen_port"], "%d", &status.TlsListenPort)

	// Parse tunnel status JSON
	if tunnelJSON, ok := values["tunnel_status"]; ok && tunnelJSON != "" {
		var tunnelStatus map[string]string
		if err := json.Unmarshal([]byte(tunnelJSON), &tunnelStatus); err == nil {
			status.TunnelStatus = tunnelStatus
		}
	}

	return status
}

// AgentLastSeenUpdater defines the interface for updating agent last seen time.
type AgentLastSeenUpdater interface {
	UpdateLastSeen(ctx context.Context, agentID uint) error
}

// AgentLastSeenUpdaterAdapter adapts the AgentRepository to AgentLastSeenUpdater interface.
type AgentLastSeenUpdaterAdapter struct {
	repo forward.AgentRepository
}

// NewAgentLastSeenUpdaterAdapter creates a new AgentLastSeenUpdaterAdapter.
func NewAgentLastSeenUpdaterAdapter(repo forward.AgentRepository) *AgentLastSeenUpdaterAdapter {
	return &AgentLastSeenUpdaterAdapter{repo: repo}
}

// UpdateLastSeen updates the last_seen_at timestamp for an agent.
func (a *AgentLastSeenUpdaterAdapter) UpdateLastSeen(ctx context.Context, agentID uint) error {
	return a.repo.UpdateLastSeen(ctx, agentID)
}

// AgentInfoUpdaterAdapter adapts the AgentRepository to AgentInfoUpdater interface.
type AgentInfoUpdaterAdapter struct {
	repo forward.AgentRepository
}

// NewAgentInfoUpdaterAdapter creates a new AgentInfoUpdaterAdapter.
func NewAgentInfoUpdaterAdapter(repo forward.AgentRepository) *AgentInfoUpdaterAdapter {
	return &AgentInfoUpdaterAdapter{repo: repo}
}

// UpdateAgentInfo updates the agent info (version, platform, arch) for an agent.
func (a *AgentInfoUpdaterAdapter) UpdateAgentInfo(ctx context.Context, agentID uint, agentVersion, platform, arch string) error {
	return a.repo.UpdateAgentInfo(ctx, agentID, agentVersion, platform, arch)
}
