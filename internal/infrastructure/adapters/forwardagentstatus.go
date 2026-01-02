package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	forwardAgentStatusKeyPrefix = "forward_agent:%d:status"
	forwardAgentStatusTTL       = 5 * time.Minute
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

	// Store status in Redis hash with TTL
	pipe := a.redisClient.Pipeline()
	pipe.HSet(ctx, key, map[string]interface{}{
		"cpu_percent":        fmt.Sprintf("%.2f", status.CPUPercent),
		"memory_percent":     fmt.Sprintf("%.2f", status.MemoryPercent),
		"memory_used":        status.MemoryUsed,
		"memory_total":       status.MemoryTotal,
		"memory_avail":       status.MemoryAvail,
		"disk_percent":       fmt.Sprintf("%.2f", status.DiskPercent),
		"disk_used":          status.DiskUsed,
		"disk_total":         status.DiskTotal,
		"uptime_seconds":     status.UptimeSeconds,
		"load_avg_1":         fmt.Sprintf("%.2f", status.LoadAvg1),
		"load_avg_5":         fmt.Sprintf("%.2f", status.LoadAvg5),
		"load_avg_15":        fmt.Sprintf("%.2f", status.LoadAvg15),
		"network_rx_bytes":   status.NetworkRxBytes,
		"network_tx_bytes":   status.NetworkTxBytes,
		"network_rx_rate":    status.NetworkRxRate,
		"network_tx_rate":    status.NetworkTxRate,
		"tcp_connections":    status.TCPConnections,
		"udp_connections":    status.UDPConnections,
		"public_ipv4":        status.PublicIPv4,
		"public_ipv6":        status.PublicIPv6,
		"active_rules":       status.ActiveRules,
		"active_connections": status.ActiveConnections,
		"ws_listen_port":     status.WsListenPort,
		"tls_listen_port":    status.TlsListenPort,
		"agent_version":      status.AgentVersion,
		"platform":           status.Platform,
		"arch":               status.Arch,
		"updated_at":         biztime.NowUTC().Unix(),
	})

	// Store tunnel status as JSON if present
	if len(status.TunnelStatus) > 0 {
		tunnelJSON, err := json.Marshal(status.TunnelStatus)
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
	status := &dto.AgentStatusDTO{}

	// Parse float values
	fmt.Sscanf(values["cpu_percent"], "%f", &status.CPUPercent)
	fmt.Sscanf(values["memory_percent"], "%f", &status.MemoryPercent)
	fmt.Sscanf(values["disk_percent"], "%f", &status.DiskPercent)
	fmt.Sscanf(values["load_avg_1"], "%f", &status.LoadAvg1)
	fmt.Sscanf(values["load_avg_5"], "%f", &status.LoadAvg5)
	fmt.Sscanf(values["load_avg_15"], "%f", &status.LoadAvg15)

	// Parse uint64 values
	fmt.Sscanf(values["memory_used"], "%d", &status.MemoryUsed)
	fmt.Sscanf(values["memory_total"], "%d", &status.MemoryTotal)
	fmt.Sscanf(values["memory_avail"], "%d", &status.MemoryAvail)
	fmt.Sscanf(values["disk_used"], "%d", &status.DiskUsed)
	fmt.Sscanf(values["disk_total"], "%d", &status.DiskTotal)
	fmt.Sscanf(values["network_rx_bytes"], "%d", &status.NetworkRxBytes)
	fmt.Sscanf(values["network_tx_bytes"], "%d", &status.NetworkTxBytes)
	fmt.Sscanf(values["network_rx_rate"], "%d", &status.NetworkRxRate)
	fmt.Sscanf(values["network_tx_rate"], "%d", &status.NetworkTxRate)

	// Parse int64 values
	fmt.Sscanf(values["uptime_seconds"], "%d", &status.UptimeSeconds)

	// Parse int values
	fmt.Sscanf(values["tcp_connections"], "%d", &status.TCPConnections)
	fmt.Sscanf(values["udp_connections"], "%d", &status.UDPConnections)
	fmt.Sscanf(values["active_rules"], "%d", &status.ActiveRules)
	fmt.Sscanf(values["active_connections"], "%d", &status.ActiveConnections)

	// Parse uint16 values
	fmt.Sscanf(values["ws_listen_port"], "%d", &status.WsListenPort)
	fmt.Sscanf(values["tls_listen_port"], "%d", &status.TlsListenPort)

	// Parse tunnel status JSON
	if tunnelJSON, ok := values["tunnel_status"]; ok && tunnelJSON != "" {
		var tunnelStatus map[string]string
		if err := json.Unmarshal([]byte(tunnelJSON), &tunnelStatus); err == nil {
			status.TunnelStatus = tunnelStatus
		}
	}

	// Parse public IP addresses
	status.PublicIPv4 = values["public_ipv4"]
	status.PublicIPv6 = values["public_ipv6"]

	// Parse agent info
	status.AgentVersion = values["agent_version"]
	status.Platform = values["platform"]
	status.Arch = values["arch"]

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
