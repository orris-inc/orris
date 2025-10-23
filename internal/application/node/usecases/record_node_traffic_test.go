package usecases

import (
	"context"
	"testing"
	"time"

	"orris/internal/domain/node"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockNodeTrafficRepository struct {
	mock.Mock
}

func (m *mockNodeTrafficRepository) RecordTraffic(ctx context.Context, traffic *node.NodeTraffic) error {
	args := m.Called(ctx, traffic)
	return args.Error(0)
}

func (m *mockNodeTrafficRepository) GetTrafficStats(ctx context.Context, filter node.TrafficStatsFilter) ([]*node.NodeTraffic, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*node.NodeTraffic), args.Error(1)
}

func (m *mockNodeTrafficRepository) GetTotalTraffic(ctx context.Context, nodeID uint, from, to time.Time) (*node.TrafficSummary, error) {
	args := m.Called(ctx, nodeID, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*node.TrafficSummary), args.Error(1)
}

func (m *mockNodeTrafficRepository) AggregateDaily(ctx context.Context, date time.Time) error {
	args := m.Called(ctx, date)
	return args.Error(0)
}

func (m *mockNodeTrafficRepository) AggregateMonthly(ctx context.Context, year int, month int) error {
	args := m.Called(ctx, year, month)
	return args.Error(0)
}

func (m *mockNodeTrafficRepository) GetDailyStats(ctx context.Context, nodeID uint, from, to time.Time) ([]*node.NodeTraffic, error) {
	args := m.Called(ctx, nodeID, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*node.NodeTraffic), args.Error(1)
}

func (m *mockNodeTrafficRepository) GetMonthlyStats(ctx context.Context, nodeID uint, year int) ([]*node.NodeTraffic, error) {
	args := m.Called(ctx, nodeID, year)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*node.NodeTraffic), args.Error(1)
}

func (m *mockNodeTrafficRepository) DeleteOldRecords(ctx context.Context, before time.Time) error {
	args := m.Called(ctx, before)
	return args.Error(0)
}

func TestRecordNodeTraffic_Success(t *testing.T) {
	_ = new(mockNodeTrafficRepository)
	logger := new(mockLogger)

	logger.On("Infow", "recording node traffic", mock.Anything).Return()
	logger.On("Infow", "node traffic recorded successfully", mock.Anything).Return()

	nodeID := uint(1)
	upload := uint64(1024)
	download := uint64(2048)
	period := time.Now().Truncate(time.Hour)

	traffic, err := node.NewNodeTraffic(nodeID, nil, nil, period)
	assert.NoError(t, err)
	assert.NotNil(t, traffic)

	err = traffic.Accumulate(upload, download)
	assert.NoError(t, err)

	assert.Equal(t, upload, traffic.Upload())
	assert.Equal(t, download, traffic.Download())
	assert.Equal(t, upload+download, traffic.Total())
}

func TestRecordNodeTraffic_Accumulation(t *testing.T) {
	_ = new(mockLogger)

	nodeID := uint(1)
	period := time.Now().Truncate(time.Hour)

	traffic, err := node.NewNodeTraffic(nodeID, nil, nil, period)
	assert.NoError(t, err)

	err = traffic.Accumulate(1024, 2048)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1024), traffic.Upload())
	assert.Equal(t, uint64(2048), traffic.Download())
	assert.Equal(t, uint64(3072), traffic.Total())

	err = traffic.Accumulate(512, 1024)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1536), traffic.Upload())
	assert.Equal(t, uint64(3072), traffic.Download())
	assert.Equal(t, uint64(4608), traffic.Total())

	err = traffic.Accumulate(256, 512)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1792), traffic.Upload())
	assert.Equal(t, uint64(3584), traffic.Download())
	assert.Equal(t, uint64(5376), traffic.Total())
}

func TestRecordNodeTraffic_HourlyAggregation(t *testing.T) {
	nodeID := uint(1)
	period1 := time.Date(2025, 10, 21, 10, 0, 0, 0, time.UTC)
	period2 := time.Date(2025, 10, 21, 11, 0, 0, 0, time.UTC)

	traffic1, err := node.NewNodeTraffic(nodeID, nil, nil, period1)
	assert.NoError(t, err)
	err = traffic1.Accumulate(1024, 2048)
	assert.NoError(t, err)

	traffic2, err := node.NewNodeTraffic(nodeID, nil, nil, period2)
	assert.NoError(t, err)
	err = traffic2.Accumulate(512, 1024)
	assert.NoError(t, err)

	assert.Equal(t, period1, traffic1.Period())
	assert.Equal(t, period2, traffic2.Period())
	assert.Equal(t, uint64(3072), traffic1.Total())
	assert.Equal(t, uint64(1536), traffic2.Total())
}

func TestRecordNodeTraffic_WithUserAndSubscription(t *testing.T) {
	nodeID := uint(1)
	userID := uint(100)
	subscriptionID := uint(200)
	period := time.Now().Truncate(time.Hour)

	traffic, err := node.NewNodeTraffic(nodeID, &userID, &subscriptionID, period)
	assert.NoError(t, err)
	assert.NotNil(t, traffic)

	err = traffic.Accumulate(1024, 2048)
	assert.NoError(t, err)

	assert.Equal(t, nodeID, traffic.NodeID())
	assert.Equal(t, &userID, traffic.UserID())
	assert.Equal(t, &subscriptionID, traffic.SubscriptionID())
	assert.Equal(t, uint64(1024), traffic.Upload())
	assert.Equal(t, uint64(2048), traffic.Download())
}

func TestRecordNodeTraffic_ZeroTraffic(t *testing.T) {
	nodeID := uint(1)
	period := time.Now().Truncate(time.Hour)

	traffic, err := node.NewNodeTraffic(nodeID, nil, nil, period)
	assert.NoError(t, err)

	err = traffic.Accumulate(0, 0)
	assert.NoError(t, err)

	assert.True(t, traffic.IsEmpty())
	assert.Equal(t, uint64(0), traffic.Upload())
	assert.Equal(t, uint64(0), traffic.Download())
	assert.Equal(t, uint64(0), traffic.Total())
}
