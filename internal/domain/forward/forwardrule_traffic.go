package forward

import (
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
)

// UploadBytes returns the upload bytes count with traffic multiplier applied.
func (r *ForwardRule) UploadBytes() int64 {
	multiplier := r.GetEffectiveMultiplier()
	return int64(float64(r.uploadBytes) * multiplier)
}

// DownloadBytes returns the download bytes count with traffic multiplier applied.
func (r *ForwardRule) DownloadBytes() int64 {
	multiplier := r.GetEffectiveMultiplier()
	return int64(float64(r.downloadBytes) * multiplier)
}

// TotalBytes returns the total bytes count with traffic multiplier applied.
func (r *ForwardRule) TotalBytes() int64 {
	return r.UploadBytes() + r.DownloadBytes()
}

// GetRawUploadBytes returns the raw upload bytes without multiplier (for internal use).
func (r *ForwardRule) GetRawUploadBytes() int64 {
	return r.uploadBytes
}

// GetRawDownloadBytes returns the raw download bytes without multiplier (for internal use).
func (r *ForwardRule) GetRawDownloadBytes() int64 {
	return r.downloadBytes
}

// GetRawTotalBytes returns the raw total bytes without multiplier (for internal use).
func (r *ForwardRule) GetRawTotalBytes() int64 {
	return r.uploadBytes + r.downloadBytes
}

// GetTrafficMultiplier returns the configured traffic multiplier (may be nil).
func (r *ForwardRule) GetTrafficMultiplier() *float64 {
	return r.trafficMultiplier
}

// CalculateNodeCount calculates the total number of nodes in the forward chain.
func (r *ForwardRule) CalculateNodeCount() int {
	switch r.ruleType {
	case vo.ForwardRuleTypeDirect:
		return 1 // Only entry agent
	case vo.ForwardRuleTypeEntry:
		// Entry + Exit: load balancing selects one exit agent per connection
		// So the actual node count in the forwarding path is always 2
		return 2 // Entry + one Exit (load balancing selects one at a time)
	case vo.ForwardRuleTypeChain:
		// Chain: Entry -> Chain[0] -> ... -> Chain[n-1] -> Target
		chainCount := 0
		if r.chainAgentIDs != nil {
			chainCount = len(r.chainAgentIDs)
		}
		return 1 + chainCount // Entry + Chain agents
	case vo.ForwardRuleTypeDirectChain:
		chainCount := 0
		if r.chainAgentIDs != nil {
			chainCount = len(r.chainAgentIDs)
		}
		return 2 + chainCount // Entry + Chain + Exit
	case vo.ForwardRuleTypeExternal:
		return 1 // External rules have no agents, traffic multiplier calculation returns 1.0
	default:
		return 1 // Safe fallback
	}
}

// GetEffectiveMultiplier returns the effective traffic multiplier to use.
// If a multiplier is configured, it uses that value.
// Otherwise, it auto-calculates based on node count (1 / nodeCount).
func (r *ForwardRule) GetEffectiveMultiplier() float64 {
	if r.trafficMultiplier != nil {
		return *r.trafficMultiplier
	}

	nodeCount := r.CalculateNodeCount()
	if nodeCount <= 0 {
		// Safety fallback, should not happen in practice
		return 1.0
	}

	return 1.0 / float64(nodeCount)
}

// RecordTraffic records traffic bytes.
func (r *ForwardRule) RecordTraffic(upload, download int64) {
	r.uploadBytes += upload
	r.downloadBytes += download
	r.updatedAt = biztime.NowUTC()
}

// ResetTraffic resets the traffic counters.
func (r *ForwardRule) ResetTraffic() {
	r.uploadBytes = 0
	r.downloadBytes = 0
	r.updatedAt = biztime.NowUTC()
}
