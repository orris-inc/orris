package usecases

import (
	"context"
	"testing"
	"time"

	"orris/internal/domain/node"
	"orris/internal/shared/query"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetNodeTrafficStats_HourlyStats(t *testing.T) {
	repo := new(mockNodeTrafficRepository)
	logger := new(mockLogger)

	logger.On("Infow", "fetching hourly traffic stats", mock.Anything).Return()
	logger.On("Infow", "hourly traffic stats retrieved successfully", mock.Anything).Return()

	nodeID := uint(1)
	from := time.Date(2025, 10, 21, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 10, 21, 23, 59, 59, 0, time.UTC)

	expectedTraffic := []*node.NodeTraffic{}
	for i := 0; i < 24; i++ {
		period := time.Date(2025, 10, 21, i, 0, 0, 0, time.UTC)
		traffic, _ := node.NewNodeTraffic(nodeID, nil, nil, period)
		_ = traffic.Accumulate(uint64(1024*i), uint64(2048*i))
		expectedTraffic = append(expectedTraffic, traffic)
	}

	filter := node.TrafficStatsFilter{
		NodeID: &nodeID,
		From:   from,
		To:     to,
	}

	repo.On("GetTrafficStats", mock.Anything, filter).Return(expectedTraffic, nil)

	result, err := repo.GetTrafficStats(context.Background(), filter)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 24)

	for i, traffic := range result {
		assert.Equal(t, nodeID, traffic.NodeID())
		assert.Equal(t, uint64(1024*i), traffic.Upload())
		assert.Equal(t, uint64(2048*i), traffic.Download())
	}

	repo.AssertExpectations(t)
}

func TestGetNodeTrafficStats_DailyStats(t *testing.T) {
	repo := new(mockNodeTrafficRepository)
	logger := new(mockLogger)

	logger.On("Infow", "fetching daily traffic stats", mock.Anything).Return()
	logger.On("Infow", "daily traffic stats retrieved successfully", mock.Anything).Return()

	nodeID := uint(1)
	from := time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC)

	expectedTraffic := []*node.NodeTraffic{}
	for i := 1; i <= 31; i++ {
		period := time.Date(2025, 10, i, 0, 0, 0, 0, time.UTC)
		traffic, _ := node.NewNodeTraffic(nodeID, nil, nil, period)
		_ = traffic.Accumulate(uint64(1024*1024*i), uint64(2048*1024*i))
		expectedTraffic = append(expectedTraffic, traffic)
	}

	repo.On("GetDailyStats", mock.Anything, nodeID, from, to).Return(expectedTraffic, nil)

	result, err := repo.GetDailyStats(context.Background(), nodeID, from, to)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 31)

	for i, traffic := range result {
		assert.Equal(t, nodeID, traffic.NodeID())
		assert.Equal(t, uint64(1024*1024*(i+1)), traffic.Upload())
		assert.Equal(t, uint64(2048*1024*(i+1)), traffic.Download())
	}

	repo.AssertExpectations(t)
}

func TestGetNodeTrafficStats_MonthlyStats(t *testing.T) {
	repo := new(mockNodeTrafficRepository)
	logger := new(mockLogger)

	logger.On("Infow", "fetching monthly traffic stats", mock.Anything).Return()
	logger.On("Infow", "monthly traffic stats retrieved successfully", mock.Anything).Return()

	nodeID := uint(1)
	year := 2025

	expectedTraffic := []*node.NodeTraffic{}
	for i := 1; i <= 12; i++ {
		period := time.Date(year, time.Month(i), 1, 0, 0, 0, 0, time.UTC)
		traffic, _ := node.NewNodeTraffic(nodeID, nil, nil, period)
		_ = traffic.Accumulate(uint64(1024*1024*1024*i), uint64(2048*1024*1024*i))
		expectedTraffic = append(expectedTraffic, traffic)
	}

	repo.On("GetMonthlyStats", mock.Anything, nodeID, year).Return(expectedTraffic, nil)

	result, err := repo.GetMonthlyStats(context.Background(), nodeID, year)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 12)

	for i, traffic := range result {
		assert.Equal(t, nodeID, traffic.NodeID())
		assert.Equal(t, uint64(1024*1024*1024*(i+1)), traffic.Upload())
		assert.Equal(t, uint64(2048*1024*1024*(i+1)), traffic.Download())
	}

	repo.AssertExpectations(t)
}

func TestGetNodeTrafficStats_TimeRangeQuery(t *testing.T) {
	repo := new(mockNodeTrafficRepository)
	logger := new(mockLogger)

	logger.On("Infow", "fetching traffic stats for time range", mock.Anything).Return()
	logger.On("Infow", "traffic stats retrieved successfully", mock.Anything).Return()

	nodeID := uint(1)
	from := time.Date(2025, 10, 21, 8, 0, 0, 0, time.UTC)
	to := time.Date(2025, 10, 21, 12, 0, 0, 0, time.UTC)

	expectedTraffic := []*node.NodeTraffic{}
	for i := 8; i <= 12; i++ {
		period := time.Date(2025, 10, 21, i, 0, 0, 0, time.UTC)
		traffic, _ := node.NewNodeTraffic(nodeID, nil, nil, period)
		_ = traffic.Accumulate(uint64(1024*i), uint64(2048*i))
		expectedTraffic = append(expectedTraffic, traffic)
	}

	filter := node.TrafficStatsFilter{
		NodeID: &nodeID,
		From:   from,
		To:     to,
	}

	repo.On("GetTrafficStats", mock.Anything, filter).Return(expectedTraffic, nil)

	result, err := repo.GetTrafficStats(context.Background(), filter)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 5)

	for i, traffic := range result {
		hour := i + 8
		assert.Equal(t, nodeID, traffic.NodeID())
		assert.Equal(t, uint64(1024*hour), traffic.Upload())
		assert.Equal(t, uint64(2048*hour), traffic.Download())
	}

	repo.AssertExpectations(t)
}

func TestGetNodeTrafficStats_TotalTraffic(t *testing.T) {
	repo := new(mockNodeTrafficRepository)
	logger := new(mockLogger)

	logger.On("Infow", "fetching total traffic", mock.Anything).Return()
	logger.On("Infow", "total traffic retrieved successfully", mock.Anything).Return()

	nodeID := uint(1)
	from := time.Date(2025, 10, 21, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 10, 21, 23, 59, 59, 0, time.UTC)

	expectedSummary := &node.TrafficSummary{
		NodeID:   nodeID,
		Upload:   uint64(1024 * 1024 * 100),
		Download: uint64(2048 * 1024 * 200),
		Total:    uint64(1024*1024*100 + 2048*1024*200),
		From:     from,
		To:       to,
	}

	repo.On("GetTotalTraffic", mock.Anything, nodeID, from, to).Return(expectedSummary, nil)

	result, err := repo.GetTotalTraffic(context.Background(), nodeID, from, to)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, nodeID, result.NodeID)
	assert.Equal(t, expectedSummary.Upload, result.Upload)
	assert.Equal(t, expectedSummary.Download, result.Download)
	assert.Equal(t, expectedSummary.Total, result.Total)

	repo.AssertExpectations(t)
}

func TestGetNodeTrafficStats_WithPagination(t *testing.T) {
	repo := new(mockNodeTrafficRepository)
	logger := new(mockLogger)

	logger.On("Infow", "fetching paginated traffic stats", mock.Anything).Return()

	nodeID := uint(1)
	from := time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC)

	expectedTraffic := []*node.NodeTraffic{}
	for i := 1; i <= 10; i++ {
		period := time.Date(2025, 10, i, 0, 0, 0, 0, time.UTC)
		traffic, _ := node.NewNodeTraffic(nodeID, nil, nil, period)
		_ = traffic.Accumulate(uint64(1024*i), uint64(2048*i))
		expectedTraffic = append(expectedTraffic, traffic)
	}

	filter := node.TrafficStatsFilter{
		PageFilter: query.PageFilter{
			Page:     1,
			PageSize: 10,
		},
		NodeID: &nodeID,
		From:   from,
		To:     to,
	}

	repo.On("GetTrafficStats", mock.Anything, filter).Return(expectedTraffic, nil)

	result, err := repo.GetTrafficStats(context.Background(), filter)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 10)

	repo.AssertExpectations(t)
}

func TestGetNodeTrafficStats_EmptyResult(t *testing.T) {
	repo := new(mockNodeTrafficRepository)
	logger := new(mockLogger)

	logger.On("Infow", "fetching traffic stats", mock.Anything).Return()
	logger.On("Infow", "no traffic stats found", mock.Anything).Return()

	nodeID := uint(999)
	from := time.Date(2025, 10, 21, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 10, 21, 23, 59, 59, 0, time.UTC)

	filter := node.TrafficStatsFilter{
		NodeID: &nodeID,
		From:   from,
		To:     to,
	}

	repo.On("GetTrafficStats", mock.Anything, filter).Return([]*node.NodeTraffic{}, nil)

	result, err := repo.GetTrafficStats(context.Background(), filter)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)

	repo.AssertExpectations(t)
}
