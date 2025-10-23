package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockNodeTrafficRecorder struct {
	mock.Mock
}

func (m *mockNodeTrafficRecorder) RecordTraffic(ctx context.Context, nodeID uint, upload, download uint64) error {
	args := m.Called(ctx, nodeID, upload, download)
	return args.Error(0)
}

type mockNodeStatusUpdater struct {
	mock.Mock
}

func (m *mockNodeStatusUpdater) UpdateStatus(ctx context.Context, nodeID uint, status string, onlineUsers int, systemInfo *SystemInfo) error {
	args := m.Called(ctx, nodeID, status, onlineUsers, systemInfo)
	return args.Error(0)
}

type mockNodeLimitChecker struct {
	mock.Mock
}

func (m *mockNodeLimitChecker) CheckLimits(ctx context.Context, nodeID uint) (exceeded bool, remaining uint64, err error) {
	args := m.Called(ctx, nodeID)
	return args.Bool(0), args.Get(1).(uint64), args.Error(2)
}

func TestReportNodeDataUseCase_Execute_Success(t *testing.T) {
	trafficRecorder := new(mockNodeTrafficRecorder)
	statusUpdater := new(mockNodeStatusUpdater)
	limitChecker := new(mockNodeLimitChecker)
	logger := new(mockLogger)

	logger.On("Infow", "node data reported successfully", mock.Anything).Return()

	trafficRecorder.On("RecordTraffic", mock.Anything, uint(1), uint64(1000), uint64(2000)).Return(nil)
	statusUpdater.On("UpdateStatus", mock.Anything, uint(1), "active", 10, mock.Anything).Return(nil)
	limitChecker.On("CheckLimits", mock.Anything, uint(1)).Return(false, uint64(1000000), nil)

	uc := NewReportNodeDataUseCase(trafficRecorder, statusUpdater, limitChecker, logger)

	cmd := ReportNodeDataCommand{
		NodeID:      1,
		Upload:      1000,
		Download:    2000,
		OnlineUsers: 10,
		Status:      "active",
		SystemInfo: &SystemInfo{
			Load:        0.5,
			MemoryUsage: 0.6,
			DiskUsage:   0.3,
		},
		Timestamp: time.Now(),
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.TrafficExceeded)
	assert.Equal(t, uint64(1000000), result.TrafficRemaining)

	trafficRecorder.AssertExpectations(t)
	statusUpdater.AssertExpectations(t)
	limitChecker.AssertExpectations(t)
}

func TestReportNodeDataUseCase_Execute_MissingNodeID(t *testing.T) {
	trafficRecorder := new(mockNodeTrafficRecorder)
	statusUpdater := new(mockNodeStatusUpdater)
	limitChecker := new(mockNodeLimitChecker)
	logger := new(mockLogger)

	uc := NewReportNodeDataUseCase(trafficRecorder, statusUpdater, limitChecker, logger)

	cmd := ReportNodeDataCommand{
		NodeID:      0,
		Upload:      1000,
		Download:    2000,
		OnlineUsers: 10,
		Status:      "active",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "node_id is required")
}

func TestReportNodeDataUseCase_Execute_TrafficRecordingError(t *testing.T) {
	trafficRecorder := new(mockNodeTrafficRecorder)
	statusUpdater := new(mockNodeStatusUpdater)
	limitChecker := new(mockNodeLimitChecker)
	logger := new(mockLogger)

	logger.On("Errorw", "failed to record traffic", mock.Anything).Return()

	trafficRecorder.On("RecordTraffic", mock.Anything, uint(1), uint64(1000), uint64(2000)).
		Return(errors.New("database error"))

	uc := NewReportNodeDataUseCase(trafficRecorder, statusUpdater, limitChecker, logger)

	cmd := ReportNodeDataCommand{
		NodeID:      1,
		Upload:      1000,
		Download:    2000,
		OnlineUsers: 10,
		Status:      "active",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to record traffic")

	trafficRecorder.AssertExpectations(t)
}

func TestReportNodeDataUseCase_Execute_ZeroTraffic(t *testing.T) {
	trafficRecorder := new(mockNodeTrafficRecorder)
	statusUpdater := new(mockNodeStatusUpdater)
	limitChecker := new(mockNodeLimitChecker)
	logger := new(mockLogger)

	logger.On("Infow", "node data reported successfully", mock.Anything).Return()

	statusUpdater.On("UpdateStatus", mock.Anything, uint(1), "active", 5, mock.Anything).Return(nil)
	limitChecker.On("CheckLimits", mock.Anything, uint(1)).Return(false, uint64(1000000), nil)

	uc := NewReportNodeDataUseCase(trafficRecorder, statusUpdater, limitChecker, logger)

	cmd := ReportNodeDataCommand{
		NodeID:      1,
		Upload:      0,
		Download:    0,
		OnlineUsers: 5,
		Status:      "active",
		Timestamp:   time.Now(),
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.TrafficExceeded)

	statusUpdater.AssertExpectations(t)
	limitChecker.AssertExpectations(t)
}

func TestReportNodeDataUseCase_Execute_TrafficExceeded(t *testing.T) {
	trafficRecorder := new(mockNodeTrafficRecorder)
	statusUpdater := new(mockNodeStatusUpdater)
	limitChecker := new(mockNodeLimitChecker)
	logger := new(mockLogger)

	logger.On("Infow", "node data reported successfully", mock.Anything).Return()
	logger.On("Warnw", "node traffic limit exceeded", mock.Anything).Return()

	trafficRecorder.On("RecordTraffic", mock.Anything, uint(1), uint64(1000), uint64(2000)).Return(nil)
	statusUpdater.On("UpdateStatus", mock.Anything, uint(1), "active", 10, mock.Anything).Return(nil)
	limitChecker.On("CheckLimits", mock.Anything, uint(1)).Return(true, uint64(0), nil)

	uc := NewReportNodeDataUseCase(trafficRecorder, statusUpdater, limitChecker, logger)

	cmd := ReportNodeDataCommand{
		NodeID:      1,
		Upload:      1000,
		Download:    2000,
		OnlineUsers: 10,
		Status:      "active",
		Timestamp:   time.Now(),
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.TrafficExceeded)
	assert.Equal(t, uint64(0), result.TrafficRemaining)

	trafficRecorder.AssertExpectations(t)
	statusUpdater.AssertExpectations(t)
	limitChecker.AssertExpectations(t)
}

func TestReportNodeDataUseCase_Execute_StatusUpdateError(t *testing.T) {
	trafficRecorder := new(mockNodeTrafficRecorder)
	statusUpdater := new(mockNodeStatusUpdater)
	limitChecker := new(mockNodeLimitChecker)
	logger := new(mockLogger)

	logger.On("Infow", "node data reported successfully", mock.Anything).Return()
	logger.On("Warnw", "failed to update status", mock.Anything).Return()

	trafficRecorder.On("RecordTraffic", mock.Anything, uint(1), uint64(1000), uint64(2000)).Return(nil)
	statusUpdater.On("UpdateStatus", mock.Anything, uint(1), "active", 10, mock.Anything).
		Return(errors.New("update error"))
	limitChecker.On("CheckLimits", mock.Anything, uint(1)).Return(false, uint64(1000000), nil)

	uc := NewReportNodeDataUseCase(trafficRecorder, statusUpdater, limitChecker, logger)

	cmd := ReportNodeDataCommand{
		NodeID:      1,
		Upload:      1000,
		Download:    2000,
		OnlineUsers: 10,
		Status:      "active",
		Timestamp:   time.Now(),
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	trafficRecorder.AssertExpectations(t)
	statusUpdater.AssertExpectations(t)
	limitChecker.AssertExpectations(t)
}

func TestReportNodeDataUseCase_Execute_LimitCheckError(t *testing.T) {
	trafficRecorder := new(mockNodeTrafficRecorder)
	statusUpdater := new(mockNodeStatusUpdater)
	limitChecker := new(mockNodeLimitChecker)
	logger := new(mockLogger)

	logger.On("Infow", "node data reported successfully", mock.Anything).Return()
	logger.On("Warnw", "failed to check limits", mock.Anything).Return()

	trafficRecorder.On("RecordTraffic", mock.Anything, uint(1), uint64(1000), uint64(2000)).Return(nil)
	statusUpdater.On("UpdateStatus", mock.Anything, uint(1), "active", 10, mock.Anything).Return(nil)
	limitChecker.On("CheckLimits", mock.Anything, uint(1)).Return(false, uint64(0), errors.New("check error"))

	uc := NewReportNodeDataUseCase(trafficRecorder, statusUpdater, limitChecker, logger)

	cmd := ReportNodeDataCommand{
		NodeID:      1,
		Upload:      1000,
		Download:    2000,
		OnlineUsers: 10,
		Status:      "active",
		Timestamp:   time.Now(),
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	trafficRecorder.AssertExpectations(t)
	statusUpdater.AssertExpectations(t)
	limitChecker.AssertExpectations(t)
}

func TestReportNodeDataUseCase_Execute_WithSystemInfo(t *testing.T) {
	trafficRecorder := new(mockNodeTrafficRecorder)
	statusUpdater := new(mockNodeStatusUpdater)
	limitChecker := new(mockNodeLimitChecker)
	logger := new(mockLogger)

	logger.On("Infow", "node data reported successfully", mock.Anything).Return()

	systemInfo := &SystemInfo{
		Load:        0.75,
		MemoryUsage: 0.85,
		DiskUsage:   0.50,
	}

	trafficRecorder.On("RecordTraffic", mock.Anything, uint(1), uint64(1000), uint64(2000)).Return(nil)
	statusUpdater.On("UpdateStatus", mock.Anything, uint(1), "active", 10, systemInfo).Return(nil)
	limitChecker.On("CheckLimits", mock.Anything, uint(1)).Return(false, uint64(1000000), nil)

	uc := NewReportNodeDataUseCase(trafficRecorder, statusUpdater, limitChecker, logger)

	cmd := ReportNodeDataCommand{
		NodeID:      1,
		Upload:      1000,
		Download:    2000,
		OnlineUsers: 10,
		Status:      "active",
		SystemInfo:  systemInfo,
		Timestamp:   time.Now(),
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	trafficRecorder.AssertExpectations(t)
	statusUpdater.AssertExpectations(t)
	limitChecker.AssertExpectations(t)
}

func TestReportNodeDataUseCase_Execute_HighTraffic(t *testing.T) {
	trafficRecorder := new(mockNodeTrafficRecorder)
	statusUpdater := new(mockNodeStatusUpdater)
	limitChecker := new(mockNodeLimitChecker)
	logger := new(mockLogger)

	logger.On("Infow", "node data reported successfully", mock.Anything).Return()

	trafficRecorder.On("RecordTraffic", mock.Anything, uint(1), uint64(1000000000), uint64(2000000000)).Return(nil)
	statusUpdater.On("UpdateStatus", mock.Anything, uint(1), "active", 100, mock.Anything).Return(nil)
	limitChecker.On("CheckLimits", mock.Anything, uint(1)).Return(false, uint64(5000000000), nil)

	uc := NewReportNodeDataUseCase(trafficRecorder, statusUpdater, limitChecker, logger)

	cmd := ReportNodeDataCommand{
		NodeID:      1,
		Upload:      1000000000,
		Download:    2000000000,
		OnlineUsers: 100,
		Status:      "active",
		Timestamp:   time.Now(),
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.TrafficExceeded)
	assert.Equal(t, uint64(5000000000), result.TrafficRemaining)

	trafficRecorder.AssertExpectations(t)
	statusUpdater.AssertExpectations(t)
	limitChecker.AssertExpectations(t)
}

func TestReportNodeDataUseCase_Execute_MultipleStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{
			name:   "active status",
			status: "active",
		},
		{
			name:   "inactive status",
			status: "inactive",
		},
		{
			name:   "maintenance status",
			status: "maintenance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trafficRecorder := new(mockNodeTrafficRecorder)
			statusUpdater := new(mockNodeStatusUpdater)
			limitChecker := new(mockNodeLimitChecker)
			logger := new(mockLogger)

			logger.On("Infow", "node data reported successfully", mock.Anything).Return()

			trafficRecorder.On("RecordTraffic", mock.Anything, uint(1), uint64(1000), uint64(2000)).Return(nil)
			statusUpdater.On("UpdateStatus", mock.Anything, uint(1), tt.status, 10, mock.Anything).Return(nil)
			limitChecker.On("CheckLimits", mock.Anything, uint(1)).Return(false, uint64(1000000), nil)

			uc := NewReportNodeDataUseCase(trafficRecorder, statusUpdater, limitChecker, logger)

			cmd := ReportNodeDataCommand{
				NodeID:      1,
				Upload:      1000,
				Download:    2000,
				OnlineUsers: 10,
				Status:      tt.status,
				Timestamp:   time.Now(),
			}

			result, err := uc.Execute(context.Background(), cmd)

			assert.NoError(t, err)
			assert.NotNil(t, result)

			trafficRecorder.AssertExpectations(t)
			statusUpdater.AssertExpectations(t)
			limitChecker.AssertExpectations(t)
		})
	}
}

func TestReportNodeDataUseCase_Execute_NoOnlineUsers(t *testing.T) {
	trafficRecorder := new(mockNodeTrafficRecorder)
	statusUpdater := new(mockNodeStatusUpdater)
	limitChecker := new(mockNodeLimitChecker)
	logger := new(mockLogger)

	logger.On("Infow", "node data reported successfully", mock.Anything).Return()

	trafficRecorder.On("RecordTraffic", mock.Anything, uint(1), uint64(1000), uint64(2000)).Return(nil)
	statusUpdater.On("UpdateStatus", mock.Anything, uint(1), "active", 0, mock.Anything).Return(nil)
	limitChecker.On("CheckLimits", mock.Anything, uint(1)).Return(false, uint64(1000000), nil)

	uc := NewReportNodeDataUseCase(trafficRecorder, statusUpdater, limitChecker, logger)

	cmd := ReportNodeDataCommand{
		NodeID:      1,
		Upload:      1000,
		Download:    2000,
		OnlineUsers: 0,
		Status:      "active",
		Timestamp:   time.Now(),
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	trafficRecorder.AssertExpectations(t)
	statusUpdater.AssertExpectations(t)
	limitChecker.AssertExpectations(t)
}

func TestReportNodeDataUseCase_Execute_ResultFields(t *testing.T) {
	trafficRecorder := new(mockNodeTrafficRecorder)
	statusUpdater := new(mockNodeStatusUpdater)
	limitChecker := new(mockNodeLimitChecker)
	logger := new(mockLogger)

	logger.On("Infow", "node data reported successfully", mock.Anything).Return()

	trafficRecorder.On("RecordTraffic", mock.Anything, uint(1), uint64(1000), uint64(2000)).Return(nil)
	statusUpdater.On("UpdateStatus", mock.Anything, uint(1), "active", 10, mock.Anything).Return(nil)
	limitChecker.On("CheckLimits", mock.Anything, uint(1)).Return(false, uint64(1000000), nil)

	uc := NewReportNodeDataUseCase(trafficRecorder, statusUpdater, limitChecker, logger)

	cmd := ReportNodeDataCommand{
		NodeID:      1,
		Upload:      1000,
		Download:    2000,
		OnlineUsers: 10,
		Status:      "active",
		Timestamp:   time.Now(),
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.ShouldReload)
	assert.Equal(t, 1, result.ConfigVersion)
	assert.False(t, result.TrafficExceeded)
	assert.Equal(t, uint64(1000000), result.TrafficRemaining)

	trafficRecorder.AssertExpectations(t)
	statusUpdater.AssertExpectations(t)
	limitChecker.AssertExpectations(t)
}
