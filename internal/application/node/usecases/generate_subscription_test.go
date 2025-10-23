package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func (m *mockNodeRepository) GetBySubscriptionToken(ctx context.Context, token string) ([]*Node, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Node), args.Error(1)
}

func (m *mockNodeRepository) GetByTokenHash(ctx context.Context, tokenHash string) (NodeData, error) {
	args := m.Called(ctx, tokenHash)
	return args.Get(0).(NodeData), args.Error(1)
}

type mockSubscriptionTokenValidator struct {
	mock.Mock
}

func (m *mockSubscriptionTokenValidator) Validate(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

type mockSubscriptionFormatter struct {
	mock.Mock
}

func (m *mockSubscriptionFormatter) Format(nodes []*Node) (string, error) {
	args := m.Called(nodes)
	return args.String(0), args.Error(1)
}

func (m *mockSubscriptionFormatter) ContentType() string {
	args := m.Called()
	return args.String(0)
}

func TestGenerateSubscriptionUseCase_Execute_Base64Format(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	tokenValidator := new(mockSubscriptionTokenValidator)
	logger := new(mockLogger)

	logger.On("Infow", mock.Anything, mock.Anything).Return()

	nodes := []*Node{
		{
			ID:               1,
			Name:             "Test Node",
			ServerAddress:    "192.168.1.1",
			ServerPort:       8388,
			EncryptionMethod: "aes-256-gcm",
			Password:         "testpassword",
		},
	}

	tokenValidator.On("Validate", mock.Anything, "valid_token").Return(nil)
	nodeRepo.On("GetBySubscriptionToken", mock.Anything, "valid_token").Return(nodes, nil)

	uc := NewGenerateSubscriptionUseCase(nodeRepo, tokenValidator, logger)

	cmd := GenerateSubscriptionCommand{
		SubscriptionToken: "valid_token",
		Format:            "base64",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "base64", result.Format)
	assert.NotEmpty(t, result.Content)
	assert.NotEmpty(t, result.ContentType)

	tokenValidator.AssertExpectations(t)
	nodeRepo.AssertExpectations(t)
}

func TestGenerateSubscriptionUseCase_Execute_ClashFormat(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	tokenValidator := new(mockSubscriptionTokenValidator)
	logger := new(mockLogger)

	logger.On("Infow", mock.Anything, mock.Anything).Return()

	nodes := []*Node{
		{
			ID:               1,
			Name:             "Test Node",
			ServerAddress:    "192.168.1.1",
			ServerPort:       8388,
			EncryptionMethod: "aes-256-gcm",
			Password:         "testpassword",
		},
	}

	tokenValidator.On("Validate", mock.Anything, "valid_token").Return(nil)
	nodeRepo.On("GetBySubscriptionToken", mock.Anything, "valid_token").Return(nodes, nil)

	uc := NewGenerateSubscriptionUseCase(nodeRepo, tokenValidator, logger)

	cmd := GenerateSubscriptionCommand{
		SubscriptionToken: "valid_token",
		Format:            "clash",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "clash", result.Format)
	assert.NotEmpty(t, result.Content)
	assert.NotEmpty(t, result.ContentType)

	tokenValidator.AssertExpectations(t)
	nodeRepo.AssertExpectations(t)
}

func TestGenerateSubscriptionUseCase_Execute_InvalidToken(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	tokenValidator := new(mockSubscriptionTokenValidator)
	logger := new(mockLogger)

	logger.On("Warnw", "invalid subscription token", mock.Anything).Return()

	tokenValidator.On("Validate", mock.Anything, "invalid_token").Return(errors.New("invalid token"))

	uc := NewGenerateSubscriptionUseCase(nodeRepo, tokenValidator, logger)

	cmd := GenerateSubscriptionCommand{
		SubscriptionToken: "invalid_token",
		Format:            "base64",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid subscription token")

	tokenValidator.AssertExpectations(t)
}

func TestGenerateSubscriptionUseCase_Execute_NoAvailableNodes(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	tokenValidator := new(mockSubscriptionTokenValidator)
	logger := new(mockLogger)

	logger.On("Warnw", "no available nodes found", mock.Anything).Return()

	tokenValidator.On("Validate", mock.Anything, "valid_token").Return(nil)
	nodeRepo.On("GetBySubscriptionToken", mock.Anything, "valid_token").Return([]*Node{}, nil)

	uc := NewGenerateSubscriptionUseCase(nodeRepo, tokenValidator, logger)

	cmd := GenerateSubscriptionCommand{
		SubscriptionToken: "valid_token",
		Format:            "base64",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no available nodes")

	tokenValidator.AssertExpectations(t)
	nodeRepo.AssertExpectations(t)
}

func TestGenerateSubscriptionUseCase_Execute_UnsupportedFormat(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	tokenValidator := new(mockSubscriptionTokenValidator)
	logger := new(mockLogger)

	logger.On("Warnw", "unsupported format", mock.Anything).Return()

	nodes := []*Node{
		{
			ID:               1,
			Name:             "Test Node",
			ServerAddress:    "192.168.1.1",
			ServerPort:       8388,
			EncryptionMethod: "aes-256-gcm",
			Password:         "testpassword",
		},
	}

	tokenValidator.On("Validate", mock.Anything, "valid_token").Return(nil)
	nodeRepo.On("GetBySubscriptionToken", mock.Anything, "valid_token").Return(nodes, nil)

	uc := NewGenerateSubscriptionUseCase(nodeRepo, tokenValidator, logger)

	cmd := GenerateSubscriptionCommand{
		SubscriptionToken: "valid_token",
		Format:            "unsupported",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported format")

	tokenValidator.AssertExpectations(t)
	nodeRepo.AssertExpectations(t)
}

func TestGenerateSubscriptionUseCase_Execute_NodeRepositoryError(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	tokenValidator := new(mockSubscriptionTokenValidator)
	logger := new(mockLogger)

	logger.On("Errorw", "failed to get nodes", mock.Anything).Return()

	tokenValidator.On("Validate", mock.Anything, "valid_token").Return(nil)
	nodeRepo.On("GetBySubscriptionToken", mock.Anything, "valid_token").Return(nil, errors.New("database error"))

	uc := NewGenerateSubscriptionUseCase(nodeRepo, tokenValidator, logger)

	cmd := GenerateSubscriptionCommand{
		SubscriptionToken: "valid_token",
		Format:            "base64",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get nodes")

	tokenValidator.AssertExpectations(t)
	nodeRepo.AssertExpectations(t)
}

func TestGenerateSubscriptionUseCase_Execute_MultipleNodes(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	tokenValidator := new(mockSubscriptionTokenValidator)
	logger := new(mockLogger)

	logger.On("Infow", mock.Anything, mock.Anything).Return()

	nodes := []*Node{
		{
			ID:               1,
			Name:             "Node 1",
			ServerAddress:    "192.168.1.1",
			ServerPort:       8388,
			EncryptionMethod: "aes-256-gcm",
			Password:         "password1",
		},
		{
			ID:               2,
			Name:             "Node 2",
			ServerAddress:    "192.168.1.2",
			ServerPort:       8389,
			EncryptionMethod: "chacha20-ietf-poly1305",
			Password:         "password2",
		},
		{
			ID:               3,
			Name:             "Node 3",
			ServerAddress:    "192.168.1.3",
			ServerPort:       8390,
			EncryptionMethod: "aes-128-gcm",
			Password:         "password3",
		},
	}

	tokenValidator.On("Validate", mock.Anything, "valid_token").Return(nil)
	nodeRepo.On("GetBySubscriptionToken", mock.Anything, "valid_token").Return(nodes, nil)

	uc := NewGenerateSubscriptionUseCase(nodeRepo, tokenValidator, logger)

	cmd := GenerateSubscriptionCommand{
		SubscriptionToken: "valid_token",
		Format:            "base64",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "base64", result.Format)
	assert.NotEmpty(t, result.Content)

	tokenValidator.AssertExpectations(t)
	nodeRepo.AssertExpectations(t)
}

func TestGenerateSubscriptionUseCase_Execute_V2RayFormat(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	tokenValidator := new(mockSubscriptionTokenValidator)
	logger := new(mockLogger)

	logger.On("Infow", mock.Anything, mock.Anything).Return()

	nodes := []*Node{
		{
			ID:               1,
			Name:             "Test Node",
			ServerAddress:    "192.168.1.1",
			ServerPort:       8388,
			EncryptionMethod: "aes-256-gcm",
			Password:         "testpassword",
		},
	}

	tokenValidator.On("Validate", mock.Anything, "valid_token").Return(nil)
	nodeRepo.On("GetBySubscriptionToken", mock.Anything, "valid_token").Return(nodes, nil)

	uc := NewGenerateSubscriptionUseCase(nodeRepo, tokenValidator, logger)

	cmd := GenerateSubscriptionCommand{
		SubscriptionToken: "valid_token",
		Format:            "v2ray",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "v2ray", result.Format)
	assert.NotEmpty(t, result.Content)

	tokenValidator.AssertExpectations(t)
	nodeRepo.AssertExpectations(t)
}

func TestGenerateSubscriptionUseCase_Execute_SIP008Format(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	tokenValidator := new(mockSubscriptionTokenValidator)
	logger := new(mockLogger)

	logger.On("Infow", mock.Anything, mock.Anything).Return()

	nodes := []*Node{
		{
			ID:               1,
			Name:             "Test Node",
			ServerAddress:    "192.168.1.1",
			ServerPort:       8388,
			EncryptionMethod: "aes-256-gcm",
			Password:         "testpassword",
		},
	}

	tokenValidator.On("Validate", mock.Anything, "valid_token").Return(nil)
	nodeRepo.On("GetBySubscriptionToken", mock.Anything, "valid_token").Return(nodes, nil)

	uc := NewGenerateSubscriptionUseCase(nodeRepo, tokenValidator, logger)

	cmd := GenerateSubscriptionCommand{
		SubscriptionToken: "valid_token",
		Format:            "sip008",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "sip008", result.Format)
	assert.NotEmpty(t, result.Content)

	tokenValidator.AssertExpectations(t)
	nodeRepo.AssertExpectations(t)
}

func TestGenerateSubscriptionUseCase_Execute_SurgeFormat(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	tokenValidator := new(mockSubscriptionTokenValidator)
	logger := new(mockLogger)

	logger.On("Infow", mock.Anything, mock.Anything).Return()

	nodes := []*Node{
		{
			ID:               1,
			Name:             "Test Node",
			ServerAddress:    "192.168.1.1",
			ServerPort:       8388,
			EncryptionMethod: "aes-256-gcm",
			Password:         "testpassword",
		},
	}

	tokenValidator.On("Validate", mock.Anything, "valid_token").Return(nil)
	nodeRepo.On("GetBySubscriptionToken", mock.Anything, "valid_token").Return(nodes, nil)

	uc := NewGenerateSubscriptionUseCase(nodeRepo, tokenValidator, logger)

	cmd := GenerateSubscriptionCommand{
		SubscriptionToken: "valid_token",
		Format:            "surge",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "surge", result.Format)
	assert.NotEmpty(t, result.Content)

	tokenValidator.AssertExpectations(t)
	nodeRepo.AssertExpectations(t)
}

func TestGenerateSubscriptionUseCase_Execute_NodeWithPlugin(t *testing.T) {
	nodeRepo := new(mockNodeRepository)
	tokenValidator := new(mockSubscriptionTokenValidator)
	logger := new(mockLogger)

	logger.On("Infow", mock.Anything, mock.Anything).Return()

	nodes := []*Node{
		{
			ID:               1,
			Name:             "Test Node with Plugin",
			ServerAddress:    "192.168.1.1",
			ServerPort:       8388,
			EncryptionMethod: "aes-256-gcm",
			Password:         "testpassword",
			Plugin:           "obfs-local",
			PluginOpts: map[string]string{
				"obfs": "http",
				"host": "www.bing.com",
			},
		},
	}

	tokenValidator.On("Validate", mock.Anything, "valid_token").Return(nil)
	nodeRepo.On("GetBySubscriptionToken", mock.Anything, "valid_token").Return(nodes, nil)

	uc := NewGenerateSubscriptionUseCase(nodeRepo, tokenValidator, logger)

	cmd := GenerateSubscriptionCommand{
		SubscriptionToken: "valid_token",
		Format:            "base64",
	}

	result, err := uc.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "base64", result.Format)
	assert.NotEmpty(t, result.Content)

	tokenValidator.AssertExpectations(t)
	nodeRepo.AssertExpectations(t)
}
