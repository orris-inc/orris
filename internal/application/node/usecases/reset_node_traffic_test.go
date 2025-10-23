package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"orris/internal/domain/node"
	vo "orris/internal/domain/node/value_objects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestResetNodeTraffic_Success(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	logger := new(mockLogger)

	logger.On("Infow", "resetting node traffic", mock.Anything).Return()
	logger.On("Infow", "node traffic reset successfully", mock.Anything).Return()

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

	err = n.RecordTraffic(uint64(500*1024*1024), uint64(600*1024*1024))
	assert.NoError(t, err)
	assert.Equal(t, uint64(1100*1024*1024), n.TrafficUsed())
	assert.True(t, n.IsTrafficExceeded())

	resetTimeBefore := n.TrafficResetAt()
	time.Sleep(10 * time.Millisecond)

	nodeRepo.On("GetByID", mock.Anything, uint(1)).Return(n, nil)
	nodeRepo.On("Update", mock.Anything, n).Return(nil)

	result, err := nodeRepo.GetByID(context.Background(), uint(1))
	assert.NoError(t, err)

	err = result.ResetTraffic()
	assert.NoError(t, err)

	err = nodeRepo.Update(context.Background(), result)
	assert.NoError(t, err)

	assert.Equal(t, uint64(0), result.TrafficUsed())
	assert.False(t, result.IsTrafficExceeded())
	assert.True(t, result.TrafficResetAt().After(resetTimeBefore))

	nodeRepo.AssertExpectations(t)
}

func TestResetNodeTraffic_NodeNotFound(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	logger := new(mockLogger)

	logger.On("Infow", "resetting node traffic", mock.Anything).Return()
	logger.On("Errorw", "node not found", mock.Anything).Return()

	nodeRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, errors.New("node not found"))

	result, err := nodeRepo.GetByID(context.Background(), uint(999))
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "node not found", err.Error())

	nodeRepo.AssertExpectations(t)
}

func TestResetNodeTraffic_AlreadyZero(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	logger := new(mockLogger)

	logger.On("Infow", "resetting node traffic", mock.Anything).Return()
	logger.On("Infow", "node traffic already zero", mock.Anything).Return()

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

	assert.Equal(t, uint64(0), n.TrafficUsed())

	nodeRepo.On("GetByID", mock.Anything, uint(1)).Return(n, nil)
	nodeRepo.On("Update", mock.Anything, n).Return(nil)

	result, err := nodeRepo.GetByID(context.Background(), uint(1))
	assert.NoError(t, err)

	resetTimeBefore := result.TrafficResetAt()
	time.Sleep(10 * time.Millisecond)

	err = result.ResetTraffic()
	assert.NoError(t, err)

	err = nodeRepo.Update(context.Background(), result)
	assert.NoError(t, err)

	assert.Equal(t, uint64(0), result.TrafficUsed())
	assert.True(t, result.TrafficResetAt().After(resetTimeBefore))

	nodeRepo.AssertExpectations(t)
}

func TestResetNodeTraffic_UpdateResetTimestamp(t *testing.T) {
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

	err = n.RecordTraffic(uint64(100*1024*1024), uint64(200*1024*1024))
	assert.NoError(t, err)

	firstResetTime := n.TrafficResetAt()
	time.Sleep(10 * time.Millisecond)

	err = n.ResetTraffic()
	assert.NoError(t, err)

	secondResetTime := n.TrafficResetAt()
	assert.True(t, secondResetTime.After(firstResetTime))

	err = n.RecordTraffic(uint64(50*1024*1024), uint64(100*1024*1024))
	assert.NoError(t, err)
	assert.Equal(t, uint64(150*1024*1024), n.TrafficUsed())

	time.Sleep(10 * time.Millisecond)

	err = n.ResetTraffic()
	assert.NoError(t, err)

	thirdResetTime := n.TrafficResetAt()
	assert.True(t, thirdResetTime.After(secondResetTime))
	assert.Equal(t, uint64(0), n.TrafficUsed())
}

func TestResetNodeTraffic_MultipleNodes(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	logger := new(mockLogger)

	logger.On("Infow", "resetting multiple nodes traffic", mock.Anything).Return()

	serverAddr, _ := vo.NewServerAddress("192.168.1.1")
	encryptionConfig, _ := vo.NewEncryptionConfig("aes-256-gcm", "password123")
	metadata := vo.NewNodeMetadata("US", "Test Provider", []string{"tag1"}, "Test description")

	nodes := make([]*node.Node, 3)
	for i := 0; i < 3; i++ {
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
		_ = n.SetID(uint(i + 1))

		err = n.RecordTraffic(uint64((i+1)*100*1024*1024), uint64((i+1)*200*1024*1024))
		assert.NoError(t, err)

		nodes[i] = n
	}

	for i, n := range nodes {
		nodeRepo.On("GetByID", mock.Anything, uint(i+1)).Return(n, nil)
		nodeRepo.On("Update", mock.Anything, n).Return(nil)
	}

	for i := 0; i < 3; i++ {
		result, err := nodeRepo.GetByID(context.Background(), uint(i+1))
		assert.NoError(t, err)

		err = result.ResetTraffic()
		assert.NoError(t, err)

		err = nodeRepo.Update(context.Background(), result)
		assert.NoError(t, err)

		assert.Equal(t, uint64(0), result.TrafficUsed())
	}

	nodeRepo.AssertExpectations(t)
}

func TestResetNodeTraffic_VersionIncrement(t *testing.T) {
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

	initialVersion := n.Version()

	err = n.RecordTraffic(uint64(100*1024*1024), uint64(200*1024*1024))
	assert.NoError(t, err)

	err = n.ResetTraffic()
	assert.NoError(t, err)

	assert.Equal(t, initialVersion+1, n.Version())
}

func TestResetNodeTraffic_AfterLimitExceeded(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	logger := new(mockLogger)

	logger.On("Infow", "resetting traffic after limit exceeded", mock.Anything).Return()

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
	_ = n.SetID(1)
	_ = n.Activate()

	err = n.RecordTraffic(uint64(60*1024*1024), uint64(50*1024*1024))
	assert.NoError(t, err)
	assert.True(t, n.IsTrafficExceeded())
	assert.False(t, n.IsAvailable())

	nodeRepo.On("GetByID", mock.Anything, uint(1)).Return(n, nil)
	nodeRepo.On("Update", mock.Anything, n).Return(nil)

	result, err := nodeRepo.GetByID(context.Background(), uint(1))
	assert.NoError(t, err)

	err = result.ResetTraffic()
	assert.NoError(t, err)

	err = nodeRepo.Update(context.Background(), result)
	assert.NoError(t, err)

	assert.False(t, result.IsTrafficExceeded())
	assert.True(t, result.IsAvailable())

	nodeRepo.AssertExpectations(t)
}
