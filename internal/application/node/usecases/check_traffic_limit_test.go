package usecases

import (
	"context"
	"testing"
	"time"

	"orris/internal/domain/node"
	vo "orris/internal/domain/node/value_objects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCheckTrafficLimit_UnderLimit(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	logger := new(mockLogger)

	logger.On("Infow", "checking traffic limit", mock.Anything).Return()
	logger.On("Infow", "traffic is within limit", mock.Anything).Return()

	serverAddr, _ := vo.NewServerAddress("192.168.1.1")
	encryptionConfig, _ := vo.NewEncryptionConfig("aes-256-gcm", "password123")
	metadata := vo.NewNodeMetadata("US", "Test Provider", []string{"tag1"}, "Test description")

	n, err := node.NewNode(
		"Test Node",
		serverAddr,
		8388,
		encryptionConfig,
		nil,
		metadata,
		100,
		uint64(1024*1024*1024),
		1,
	)
	assert.NoError(t, err)
	_ = n.SetID(1)

	err = n.RecordTraffic(uint64(100*1024*1024), uint64(200*1024*1024))
	assert.NoError(t, err)

	nodeRepo.On("GetByID", mock.Anything, uint(1)).Return(n, nil)

	result, err := nodeRepo.GetByID(context.Background(), uint(1))
	assert.NoError(t, err)
	assert.NotNil(t, result)

	isExceeded := result.IsTrafficExceeded()
	assert.False(t, isExceeded)
	assert.Equal(t, uint64(300*1024*1024), result.TrafficUsed())
	assert.Equal(t, uint64(1024*1024*1024), result.TrafficLimit())

	nodeRepo.AssertExpectations(t)
}

func TestCheckTrafficLimit_Exceeded(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	logger := new(mockLogger)

	logger.On("Infow", "checking traffic limit", mock.Anything).Return()
	logger.On("Warnw", "traffic limit exceeded", mock.Anything).Return()

	serverAddr, _ := vo.NewServerAddress("192.168.1.1")
	encryptionConfig, _ := vo.NewEncryptionConfig("aes-256-gcm", "password123")
	metadata := vo.NewNodeMetadata("US", "Test Provider", []string{"tag1"}, "Test description")

	n, err := node.NewNode(
		"Test Node",
		serverAddr,
		8388,
		encryptionConfig,
		nil,
		metadata,
		100,
		uint64(1024*1024*1024),
		1,
	)
	assert.NoError(t, err)
	_ = n.SetID(1)

	err = n.RecordTraffic(uint64(600*1024*1024), uint64(500*1024*1024))
	assert.NoError(t, err)

	nodeRepo.On("GetByID", mock.Anything, uint(1)).Return(n, nil)

	result, err := nodeRepo.GetByID(context.Background(), uint(1))
	assert.NoError(t, err)
	assert.NotNil(t, result)

	isExceeded := result.IsTrafficExceeded()
	assert.True(t, isExceeded)
	assert.Equal(t, uint64(1100*1024*1024), result.TrafficUsed())
	assert.Equal(t, uint64(1024*1024*1024), result.TrafficLimit())

	nodeRepo.AssertExpectations(t)
}

func TestCheckTrafficLimit_ExactlyAtLimit(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	logger := new(mockLogger)

	logger.On("Infow", "checking traffic limit", mock.Anything).Return()
	logger.On("Warnw", "traffic limit reached", mock.Anything).Return()

	serverAddr, _ := vo.NewServerAddress("192.168.1.1")
	encryptionConfig, _ := vo.NewEncryptionConfig("aes-256-gcm", "password123")
	metadata := vo.NewNodeMetadata("US", "Test Provider", []string{"tag1"}, "Test description")

	n, err := node.NewNode(
		"Test Node",
		serverAddr,
		8388,
		encryptionConfig,
		nil,
		metadata,
		100,
		uint64(1024*1024*1024),
		1,
	)
	assert.NoError(t, err)
	_ = n.SetID(1)

	err = n.RecordTraffic(uint64(512*1024*1024), uint64(512*1024*1024))
	assert.NoError(t, err)

	nodeRepo.On("GetByID", mock.Anything, uint(1)).Return(n, nil)

	result, err := nodeRepo.GetByID(context.Background(), uint(1))
	assert.NoError(t, err)
	assert.NotNil(t, result)

	isExceeded := result.IsTrafficExceeded()
	assert.True(t, isExceeded)
	assert.Equal(t, uint64(1024*1024*1024), result.TrafficUsed())
	assert.Equal(t, uint64(1024*1024*1024), result.TrafficLimit())

	nodeRepo.AssertExpectations(t)
}

func TestCheckTrafficLimit_Unlimited(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	logger := new(mockLogger)

	logger.On("Infow", "checking traffic limit", mock.Anything).Return()
	logger.On("Infow", "node has unlimited traffic", mock.Anything).Return()

	serverAddr, _ := vo.NewServerAddress("192.168.1.1")
	encryptionConfig, _ := vo.NewEncryptionConfig("aes-256-gcm", "password123")
	metadata := vo.NewNodeMetadata("US", "Test Provider", []string{"tag1"}, "Test description")

	n, err := node.NewNode(
		"Test Node",
		serverAddr,
		8388,
		encryptionConfig,
		nil,
		metadata,
		100,
		uint64(0),
		1,
	)
	assert.NoError(t, err)
	_ = n.SetID(1)

	err = n.RecordTraffic(uint64(10*1024*1024*1024), uint64(20*1024*1024*1024))
	assert.NoError(t, err)

	nodeRepo.On("GetByID", mock.Anything, uint(1)).Return(n, nil)

	result, err := nodeRepo.GetByID(context.Background(), uint(1))
	assert.NoError(t, err)
	assert.NotNil(t, result)

	isExceeded := result.IsTrafficExceeded()
	assert.False(t, isExceeded)
	assert.Equal(t, uint64(30*1024*1024*1024), result.TrafficUsed())
	assert.Equal(t, uint64(0), result.TrafficLimit())

	nodeRepo.AssertExpectations(t)
}

func TestCheckTrafficLimit_NodeAvailability(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	logger := new(mockLogger)

	logger.On("Infow", "checking node availability", mock.Anything).Return()

	serverAddr, _ := vo.NewServerAddress("192.168.1.1")
	encryptionConfig, _ := vo.NewEncryptionConfig("aes-256-gcm", "password123")
	metadata := vo.NewNodeMetadata("US", "Test Provider", []string{"tag1"}, "Test description")

	n, err := node.NewNode(
		"Test Node",
		serverAddr,
		8388,
		encryptionConfig,
		nil,
		metadata,
		100,
		uint64(1024*1024*1024),
		1,
	)
	assert.NoError(t, err)
	_ = n.SetID(1)
	_ = n.Activate()

	err = n.RecordTraffic(uint64(100*1024*1024), uint64(200*1024*1024))
	assert.NoError(t, err)

	nodeRepo.On("GetByID", mock.Anything, uint(1)).Return(n, nil)

	result, err := nodeRepo.GetByID(context.Background(), uint(1))
	assert.NoError(t, err)

	isAvailable := result.IsAvailable()
	assert.True(t, isAvailable)

	err = result.RecordTraffic(uint64(500*1024*1024), uint64(300*1024*1024))
	assert.NoError(t, err)

	isAvailableAfterExceeded := result.IsAvailable()
	assert.False(t, isAvailableAfterExceeded)

	nodeRepo.AssertExpectations(t)
}

func TestCheckTrafficLimit_MultipleRecordings(t *testing.T) {
	serverAddr, _ := vo.NewServerAddress("192.168.1.1")
	encryptionConfig, _ := vo.NewEncryptionConfig("aes-256-gcm", "password123")
	metadata := vo.NewNodeMetadata("US", "Test Provider", []string{"tag1"}, "Test description")

	n, err := node.NewNode(
		"Test Node",
		serverAddr,
		8388,
		encryptionConfig,
		nil,
		metadata,
		100,
		uint64(1024*1024*100),
		1,
	)
	assert.NoError(t, err)

	recordings := []struct {
		upload   uint64
		download uint64
	}{
		{uint64(10 * 1024 * 1024), uint64(20 * 1024 * 1024)},
		{uint64(15 * 1024 * 1024), uint64(25 * 1024 * 1024)},
		{uint64(20 * 1024 * 1024), uint64(30 * 1024 * 1024)},
	}

	for _, rec := range recordings {
		err = n.RecordTraffic(rec.upload, rec.download)
		assert.NoError(t, err)
	}

	expectedTotal := uint64((10 + 15 + 20 + 20 + 25 + 30) * 1024 * 1024)
	assert.Equal(t, expectedTotal, n.TrafficUsed())
	assert.True(t, n.IsTrafficExceeded())
}

func TestCheckTrafficLimit_WithResetTracking(t *testing.T) {
	serverAddr, _ := vo.NewServerAddress("192.168.1.1")
	encryptionConfig, _ := vo.NewEncryptionConfig("aes-256-gcm", "password123")
	metadata := vo.NewNodeMetadata("US", "Test Provider", []string{"tag1"}, "Test description")

	n, err := node.NewNode(
		"Test Node",
		serverAddr,
		8388,
		encryptionConfig,
		nil,
		metadata,
		100,
		uint64(1024*1024*1024),
		1,
	)
	assert.NoError(t, err)

	err = n.RecordTraffic(uint64(600*1024*1024), uint64(500*1024*1024))
	assert.NoError(t, err)
	assert.True(t, n.IsTrafficExceeded())

	resetTimeBefore := n.TrafficResetAt()
	time.Sleep(10 * time.Millisecond)

	err = n.ResetTraffic()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), n.TrafficUsed())
	assert.False(t, n.IsTrafficExceeded())
	assert.True(t, n.TrafficResetAt().After(resetTimeBefore))
}
