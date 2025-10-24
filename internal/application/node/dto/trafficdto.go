package dto

import (
	"time"
)

type NodeTrafficDTO struct {
	NodeID         uint      `json:"node_id"`
	NodeName       string    `json:"node_name"`
	Upload         uint64    `json:"upload"`
	Download       uint64    `json:"download"`
	Total          uint64    `json:"total"`
	TrafficLimit   uint64    `json:"traffic_limit"`
	TrafficUsed    uint64    `json:"traffic_used"`
	UsagePercent   float64   `json:"usage_percent"`
	TrafficResetAt time.Time `json:"traffic_reset_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type TrafficStatsDTO struct {
	TotalUpload   uint64            `json:"total_upload"`
	TotalDownload uint64            `json:"total_download"`
	TotalTraffic  uint64            `json:"total_traffic"`
	TotalLimit    uint64            `json:"total_limit"`
	AverageUsage  float64           `json:"average_usage"`
	NodeStats     []*NodeTrafficDTO `json:"node_stats"`
	TopUsageNodes []*NodeTrafficDTO `json:"top_usage_nodes"`
	LowUsageNodes []*NodeTrafficDTO `json:"low_usage_nodes"`
	ExceededNodes []*NodeTrafficDTO `json:"exceeded_nodes"`
	PeriodStart   time.Time         `json:"period_start"`
	PeriodEnd     time.Time         `json:"period_end"`
}

type RecordTrafficRequest struct {
	NodeID   uint   `json:"node_id" binding:"required"`
	Upload   uint64 `json:"upload" binding:"required"`
	Download uint64 `json:"download" binding:"required"`
}

type ResetTrafficRequest struct {
	NodeID uint `json:"node_id" binding:"required"`
}

type TrafficQueryRequest struct {
	NodeIDs   []uint     `json:"node_ids,omitempty" form:"node_ids"`
	StartTime *time.Time `json:"start_time,omitempty" form:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty" form:"end_time"`
	Limit     int        `json:"limit,omitempty" form:"limit"`
}

type BulkResetTrafficRequest struct {
	NodeIDs []uint `json:"node_ids" binding:"required,min=1"`
}

type BulkResetTrafficResponse struct {
	Successful []uint   `json:"successful"`
	Failed     []uint   `json:"failed"`
	Errors     []string `json:"errors,omitempty"`
}

func CalculateUsagePercent(used, limit uint64) float64 {
	if limit == 0 {
		return 0
	}
	percent := float64(used) / float64(limit) * 100
	if percent > 100 {
		return 100
	}
	return percent
}

func NewNodeTrafficDTO(nodeID uint, nodeName string, upload, download, limit, used uint64, resetAt time.Time) *NodeTrafficDTO {
	total := upload + download
	return &NodeTrafficDTO{
		NodeID:         nodeID,
		NodeName:       nodeName,
		Upload:         upload,
		Download:       download,
		Total:          total,
		TrafficLimit:   limit,
		TrafficUsed:    used,
		UsagePercent:   CalculateUsagePercent(used, limit),
		TrafficResetAt: resetAt,
		UpdatedAt:      time.Now(),
	}
}
