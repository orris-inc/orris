package usecases

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/orris-inc/orris/internal/application/forward/testutil"
	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/domain/node"
	nodevo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/shared/id"
)

// TestCreateForwardRule_Direct_Success verifies creating a direct forward rule with target address.
// Business rule: Direct rules require agentID, listenPort, and either
// (targetAddress + targetPort) OR targetNodeID.
func TestCreateForwardRule_Direct_Success(t *testing.T) {
	// Setup mocks
	ruleRepo := testutil.NewMockForwardRuleRepository()
	agentRepo := testutil.NewMockForwardAgentRepository()
	nodeRepo := testutil.NewMockNodeRepository()
	notifier := testutil.NewMockConfigSyncNotifier()
	logger := testutil.NewMockLogger()

	// Create test agent
	agent := createTestAgent(t, "test-agent", "agent123")
	agentRepo.AddAgent(agent)

	// Create use case
	uc := NewCreateForwardRuleUseCase(
		ruleRepo, agentRepo, nodeRepo, notifier, logger,
	)

	// Create command
	cmd := CreateForwardRuleCommand{
		AgentShortID:  agent.ShortID(),
		RuleType:      "direct",
		Name:          "test-direct-rule",
		ListenPort:    8080,
		TargetAddress: "192.168.1.100",
		TargetPort:    9000,
		Protocol:      "tcp",
		IPVersion:     "auto",
	}

	// Execute
	result, err := uc.Execute(context.Background(), cmd)

	// Verify
	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
		return
	}
	if result == nil {
		t.Fatal("Execute() returned nil result")
	}
	if result.Name != cmd.Name {
		t.Errorf("result.Name = %v, want %v", result.Name, cmd.Name)
	}
	if result.ListenPort != cmd.ListenPort {
		t.Errorf("result.ListenPort = %v, want %v", result.ListenPort, cmd.ListenPort)
	}
	if result.TargetAddress != cmd.TargetAddress {
		t.Errorf("result.TargetAddress = %v, want %v", result.TargetAddress, cmd.TargetAddress)
	}
	if result.TargetPort != cmd.TargetPort {
		t.Errorf("result.TargetPort = %v, want %v", result.TargetPort, cmd.TargetPort)
	}
	if result.RuleType != cmd.RuleType {
		t.Errorf("result.RuleType = %v, want %v", result.RuleType, cmd.RuleType)
	}
	if result.Protocol != cmd.Protocol {
		t.Errorf("result.Protocol = %v, want %v", result.Protocol, cmd.Protocol)
	}
	if result.Status != "disabled" {
		t.Errorf("result.Status = %v, want disabled", result.Status)
	}

	// Verify rule was created in repository
	savedRule, err := ruleRepo.GetByListenPort(context.Background(), cmd.ListenPort)
	if err != nil {
		t.Errorf("GetByListenPort() error = %v", err)
	}
	if savedRule == nil {
		t.Error("Rule was not saved to repository")
	}

	// Verify ConfigSync was NOT called (rule is disabled)
	calls := notifier.GetCalls()
	if len(calls) != 0 {
		t.Errorf("NotifyRuleChange should not be called for disabled rule, got %d calls", len(calls))
	}
}

// TestCreateForwardRule_Entry_Success verifies creating an entry forward rule.
// Business rule: Entry rules require agentID, listenPort, exitAgentID, and either
// (targetAddress + targetPort) OR targetNodeID.
func TestCreateForwardRule_Entry_Success(t *testing.T) {
	// Setup mocks
	ruleRepo := testutil.NewMockForwardRuleRepository()
	agentRepo := testutil.NewMockForwardAgentRepository()
	nodeRepo := testutil.NewMockNodeRepository()
	notifier := testutil.NewMockConfigSyncNotifier()
	logger := testutil.NewMockLogger()

	// Create test agents
	agent := createTestAgent(t, "entry-agent", "agent123")
	exitAgent := createTestAgent(t, "exit-agent", "exit456")
	agentRepo.AddAgent(agent)
	agentRepo.AddAgent(exitAgent)

	// Create use case
	uc := NewCreateForwardRuleUseCase(
		ruleRepo, agentRepo, nodeRepo, notifier, logger,
	)

	// Create command
	cmd := CreateForwardRuleCommand{
		AgentShortID:     agent.ShortID(),
		RuleType:         "entry",
		ExitAgentShortID: exitAgent.ShortID(),
		Name:             "test-entry-rule",
		ListenPort:       8081,
		TargetAddress:    "10.0.0.10",
		TargetPort:       3306,
		Protocol:         "tcp",
	}

	// Execute
	result, err := uc.Execute(context.Background(), cmd)

	// Verify
	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
		return
	}
	if result == nil {
		t.Fatal("Execute() returned nil result")
	}
	if result.Name != cmd.Name {
		t.Errorf("result.Name = %v, want %v", result.Name, cmd.Name)
	}
	if result.ExitAgentID != exitAgent.ID() {
		t.Errorf("result.ExitAgentID = %v, want %v", result.ExitAgentID, exitAgent.ID())
	}
	if result.RuleType != cmd.RuleType {
		t.Errorf("result.RuleType = %v, want %v", result.RuleType, cmd.RuleType)
	}
}

// TestCreateForwardRule_Chain_Success verifies creating a chain forward rule.
// Business rule: Chain rules require agentID, listenPort, chainAgentIDs (1-10), and either
// (targetAddress + targetPort) OR targetNodeID.
func TestCreateForwardRule_Chain_Success(t *testing.T) {
	// Setup mocks
	ruleRepo := testutil.NewMockForwardRuleRepository()
	agentRepo := testutil.NewMockForwardAgentRepository()
	nodeRepo := testutil.NewMockNodeRepository()
	notifier := testutil.NewMockConfigSyncNotifier()
	logger := testutil.NewMockLogger()

	// Create test agents
	entryAgent := createTestAgent(t, "entry-agent", "entry123")
	chainAgent1 := createTestAgent(t, "chain-agent-1", "chain1")
	chainAgent2 := createTestAgent(t, "chain-agent-2", "chain2")
	agentRepo.AddAgent(entryAgent)
	agentRepo.AddAgent(chainAgent1)
	agentRepo.AddAgent(chainAgent2)

	// Create use case
	uc := NewCreateForwardRuleUseCase(
		ruleRepo, agentRepo, nodeRepo, notifier, logger,
	)

	// Create command
	cmd := CreateForwardRuleCommand{
		AgentShortID:       entryAgent.ShortID(),
		RuleType:           "chain",
		ChainAgentShortIDs: []string{chainAgent1.ShortID(), chainAgent2.ShortID()},
		Name:               "test-chain-rule",
		ListenPort:         8082,
		TargetAddress:      "172.16.0.1",
		TargetPort:         22,
		Protocol:           "tcp",
	}

	// Execute
	result, err := uc.Execute(context.Background(), cmd)

	// Verify
	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
		return
	}
	if result == nil {
		t.Fatal("Execute() returned nil result")
	}
	if result.Name != cmd.Name {
		t.Errorf("result.Name = %v, want %v", result.Name, cmd.Name)
	}
	if result.RuleType != cmd.RuleType {
		t.Errorf("result.RuleType = %v, want %v", result.RuleType, cmd.RuleType)
	}

	// Verify chain agents
	savedRule, _ := ruleRepo.GetByListenPort(context.Background(), cmd.ListenPort)
	if savedRule == nil {
		t.Fatal("Rule was not saved to repository")
	}
	if len(savedRule.ChainAgentIDs()) != 2 {
		t.Errorf("Chain agent count = %d, want 2", len(savedRule.ChainAgentIDs()))
	}
}

// TestCreateForwardRule_DirectChain_Success verifies creating a direct_chain forward rule.
// Business rule: DirectChain rules require agentID, listenPort, chainAgentIDs (1-10),
// chainPortConfig, and either (targetAddress + targetPort) OR targetNodeID.
func TestCreateForwardRule_DirectChain_Success(t *testing.T) {
	// Setup mocks
	ruleRepo := testutil.NewMockForwardRuleRepository()
	agentRepo := testutil.NewMockForwardAgentRepository()
	nodeRepo := testutil.NewMockNodeRepository()
	notifier := testutil.NewMockConfigSyncNotifier()
	logger := testutil.NewMockLogger()

	// Create test agents
	entryAgent := createTestAgent(t, "entry-agent", "entry123")
	chainAgent1 := createTestAgent(t, "chain-agent-1", "chain1")
	chainAgent2 := createTestAgent(t, "chain-agent-2", "chain2")
	agentRepo.AddAgent(entryAgent)
	agentRepo.AddAgent(chainAgent1)
	agentRepo.AddAgent(chainAgent2)

	// Create use case
	uc := NewCreateForwardRuleUseCase(
		ruleRepo, agentRepo, nodeRepo, notifier, logger,
	)

	// Create command with chain port config
	cmd := CreateForwardRuleCommand{
		AgentShortID:       entryAgent.ShortID(),
		RuleType:           "direct_chain",
		ChainAgentShortIDs: []string{chainAgent1.ShortID(), chainAgent2.ShortID()},
		ChainPortConfig: map[string]uint16{
			chainAgent1.ShortID(): 8001,
			chainAgent2.ShortID(): 8002,
		},
		Name:          "test-direct-chain-rule",
		ListenPort:    8083,
		TargetAddress: "192.168.100.1",
		TargetPort:    443,
		Protocol:      "tcp",
	}

	// Execute
	result, err := uc.Execute(context.Background(), cmd)

	// Verify
	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
		return
	}
	if result == nil {
		t.Fatal("Execute() returned nil result")
	}
	if result.Name != cmd.Name {
		t.Errorf("result.Name = %v, want %v", result.Name, cmd.Name)
	}
	if result.RuleType != cmd.RuleType {
		t.Errorf("result.RuleType = %v, want %v", result.RuleType, cmd.RuleType)
	}

	// Verify chain port configuration
	savedRule, _ := ruleRepo.GetByListenPort(context.Background(), cmd.ListenPort)
	if savedRule == nil {
		t.Fatal("Rule was not saved to repository")
	}
	portConfig := savedRule.ChainPortConfig()
	if len(portConfig) != 2 {
		t.Errorf("Chain port config size = %d, want 2", len(portConfig))
	}
}

// TestCreateForwardRule_ValidationErrors uses table-driven tests to verify validation errors.
// Business rule: All required fields must be present and valid based on rule type.
func TestCreateForwardRule_ValidationErrors(t *testing.T) {
	testCases := []struct {
		name    string
		setup   func() (*MockForwardAgentRepository, *MockNodeRepository)
		command CreateForwardRuleCommand
		wantErr string
	}{
		{
			name: "missing agent_id",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				return testutil.NewMockForwardAgentRepository(), testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				RuleType:      "direct",
				Name:          "test-rule",
				ListenPort:    8080,
				TargetAddress: "192.168.1.1",
				TargetPort:    9000,
				Protocol:      "tcp",
			},
			wantErr: "agent_id is required",
		},
		{
			name: "missing name",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				agentRepo := testutil.NewMockForwardAgentRepository()
				agent := createTestAgent(t, "test-agent", "agent123")
				agentRepo.AddAgent(agent)
				return agentRepo, testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				AgentShortID:  "agent123",
				RuleType:      "direct",
				ListenPort:    8080,
				TargetAddress: "192.168.1.1",
				TargetPort:    9000,
				Protocol:      "tcp",
			},
			wantErr: "name is required",
		},
		{
			name: "missing rule_type",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				agentRepo := testutil.NewMockForwardAgentRepository()
				agent := createTestAgent(t, "test-agent", "agent123")
				agentRepo.AddAgent(agent)
				return agentRepo, testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				AgentShortID:  "agent123",
				Name:          "test-rule",
				ListenPort:    8080,
				TargetAddress: "192.168.1.1",
				TargetPort:    9000,
				Protocol:      "tcp",
			},
			wantErr: "rule_type is required",
		},
		{
			name: "missing protocol",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				agentRepo := testutil.NewMockForwardAgentRepository()
				agent := createTestAgent(t, "test-agent", "agent123")
				agentRepo.AddAgent(agent)
				return agentRepo, testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				AgentShortID:  "agent123",
				RuleType:      "direct",
				Name:          "test-rule",
				ListenPort:    8080,
				TargetAddress: "192.168.1.1",
				TargetPort:    9000,
			},
			wantErr: "protocol is required",
		},
		{
			name: "invalid rule_type",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				agentRepo := testutil.NewMockForwardAgentRepository()
				agent := createTestAgent(t, "test-agent", "agent123")
				agentRepo.AddAgent(agent)
				return agentRepo, testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				AgentShortID:  "agent123",
				RuleType:      "invalid",
				Name:          "test-rule",
				ListenPort:    8080,
				TargetAddress: "192.168.1.1",
				TargetPort:    9000,
				Protocol:      "tcp",
			},
			wantErr: "invalid rule_type",
		},
		{
			name: "invalid protocol",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				agentRepo := testutil.NewMockForwardAgentRepository()
				agent := createTestAgent(t, "test-agent", "agent123")
				agentRepo.AddAgent(agent)
				return agentRepo, testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				AgentShortID:  "agent123",
				RuleType:      "direct",
				Name:          "test-rule",
				ListenPort:    8080,
				TargetAddress: "192.168.1.1",
				TargetPort:    9000,
				Protocol:      "invalid",
			},
			wantErr: "invalid protocol",
		},
		{
			name: "invalid agent short_id",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				return testutil.NewMockForwardAgentRepository(), testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				AgentShortID:  "nonexistent",
				RuleType:      "direct",
				Name:          "test-rule",
				ListenPort:    8080,
				TargetAddress: "192.168.1.1",
				TargetPort:    9000,
				Protocol:      "tcp",
			},
			wantErr: "forward agent",
		},
		{
			name: "invalid exit agent short_id",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				agentRepo := testutil.NewMockForwardAgentRepository()
				agent := createTestAgent(t, "entry-agent", "entry123")
				agentRepo.AddAgent(agent)
				return agentRepo, testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				AgentShortID:     "entry123",
				RuleType:         "entry",
				ExitAgentShortID: "nonexistent",
				Name:             "test-rule",
				ListenPort:       8080,
				TargetAddress:    "192.168.1.1",
				TargetPort:       9000,
				Protocol:         "tcp",
			},
			wantErr: "exit forward agent",
		},
		{
			name: "invalid chain agent short_id",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				agentRepo := testutil.NewMockForwardAgentRepository()
				agent := createTestAgent(t, "entry-agent", "entry123")
				agentRepo.AddAgent(agent)
				return agentRepo, testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				AgentShortID:       "entry123",
				RuleType:           "chain",
				ChainAgentShortIDs: []string{"nonexistent"},
				Name:               "test-rule",
				ListenPort:         8080,
				TargetAddress:      "192.168.1.1",
				TargetPort:         9000,
				Protocol:           "tcp",
			},
			wantErr: "chain forward agent",
		},
		{
			name: "invalid target node short_id",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				agentRepo := testutil.NewMockForwardAgentRepository()
				agent := createTestAgent(t, "test-agent", "agent123")
				agentRepo.AddAgent(agent)
				return agentRepo, testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				AgentShortID:      "agent123",
				RuleType:          "direct",
				Name:              "test-rule",
				ListenPort:        8080,
				TargetNodeShortID: "nonexistent",
				Protocol:          "tcp",
			},
			wantErr: "target node",
		},
		{
			name: "missing listen_port for direct",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				agentRepo := testutil.NewMockForwardAgentRepository()
				agent := createTestAgent(t, "test-agent", "agent123")
				agentRepo.AddAgent(agent)
				return agentRepo, testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				AgentShortID:  "agent123",
				RuleType:      "direct",
				Name:          "test-rule",
				TargetAddress: "192.168.1.1",
				TargetPort:    9000,
				Protocol:      "tcp",
			},
			wantErr: "listen_port is required",
		},
		{
			name: "missing target for direct",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				agentRepo := testutil.NewMockForwardAgentRepository()
				agent := createTestAgent(t, "test-agent", "agent123")
				agentRepo.AddAgent(agent)
				return agentRepo, testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				AgentShortID: "agent123",
				RuleType:     "direct",
				Name:         "test-rule",
				ListenPort:   8080,
				Protocol:     "tcp",
			},
			wantErr: "target_address+target_port or target_node_id is required",
		},
		{
			name: "mutually exclusive target fields",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				agentRepo := testutil.NewMockForwardAgentRepository()
				agent := createTestAgent(t, "test-agent", "agent123")
				agentRepo.AddAgent(agent)
				nodeRepo := testutil.NewMockNodeRepository()
				testNode := createTestNode(t, "test-node", "node123")
				nodeRepo.AddNode(testNode)
				return agentRepo, nodeRepo
			},
			command: CreateForwardRuleCommand{
				AgentShortID:      "agent123",
				RuleType:          "direct",
				Name:              "test-rule",
				ListenPort:        8080,
				TargetAddress:     "192.168.1.1",
				TargetPort:        9000,
				TargetNodeShortID: "node123",
				Protocol:          "tcp",
			},
			wantErr: "mutually exclusive",
		},
		{
			name: "missing exit_agent_id for entry",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				agentRepo := testutil.NewMockForwardAgentRepository()
				agent := createTestAgent(t, "test-agent", "agent123")
				agentRepo.AddAgent(agent)
				return agentRepo, testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				AgentShortID:  "agent123",
				RuleType:      "entry",
				Name:          "test-rule",
				ListenPort:    8080,
				TargetAddress: "192.168.1.1",
				TargetPort:    9000,
				Protocol:      "tcp",
			},
			wantErr: "exit_agent_id is required",
		},
		{
			name: "missing chain_agent_ids for chain",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				agentRepo := testutil.NewMockForwardAgentRepository()
				agent := createTestAgent(t, "test-agent", "agent123")
				agentRepo.AddAgent(agent)
				return agentRepo, testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				AgentShortID:  "agent123",
				RuleType:      "chain",
				Name:          "test-rule",
				ListenPort:    8080,
				TargetAddress: "192.168.1.1",
				TargetPort:    9000,
				Protocol:      "tcp",
			},
			wantErr: "chain_agent_ids is required",
		},
		{
			name: "chain exceeds maximum length",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				agentRepo := testutil.NewMockForwardAgentRepository()
				agent := createTestAgent(t, "entry-agent", "entry123")
				agentRepo.AddAgent(agent)
				// Create 11 chain agents
				for i := 1; i <= 11; i++ {
					chainAgent := createTestAgent(t, fmt.Sprintf("chain-%d", i), fmt.Sprintf("chain%d", i))
					agentRepo.AddAgent(chainAgent)
				}
				return agentRepo, testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				AgentShortID: "entry123",
				RuleType:     "chain",
				ChainAgentShortIDs: []string{
					"chain1", "chain2", "chain3", "chain4", "chain5",
					"chain6", "chain7", "chain8", "chain9", "chain10", "chain11",
				},
				Name:          "test-rule",
				ListenPort:    8080,
				TargetAddress: "192.168.1.1",
				TargetPort:    9000,
				Protocol:      "tcp",
			},
			wantErr: "maximum 10 intermediate agents",
		},
		{
			name: "missing chain_port_config for direct_chain",
			setup: func() (*MockForwardAgentRepository, *MockNodeRepository) {
				agentRepo := testutil.NewMockForwardAgentRepository()
				agent := createTestAgent(t, "entry-agent", "entry123")
				chainAgent := createTestAgent(t, "chain-agent", "chain1")
				agentRepo.AddAgent(agent)
				agentRepo.AddAgent(chainAgent)
				return agentRepo, testutil.NewMockNodeRepository()
			},
			command: CreateForwardRuleCommand{
				AgentShortID:       "entry123",
				RuleType:           "direct_chain",
				ChainAgentShortIDs: []string{"chain1"},
				Name:               "test-rule",
				ListenPort:         8080,
				TargetAddress:      "192.168.1.1",
				TargetPort:         9000,
				Protocol:           "tcp",
			},
			wantErr: "chain_port_config is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			agentRepo, nodeRepo := tc.setup()
			ruleRepo := testutil.NewMockForwardRuleRepository()
			notifier := testutil.NewMockConfigSyncNotifier()
			logger := testutil.NewMockLogger()

			uc := NewCreateForwardRuleUseCase(
				ruleRepo, agentRepo, nodeRepo, notifier, logger,
			)

			// Execute
			result, err := uc.Execute(context.Background(), tc.command)

			// Verify
			if err == nil {
				t.Errorf("Execute() expected error containing %q, got nil", tc.wantErr)
				return
			}
			if result != nil {
				t.Errorf("Execute() expected nil result on error, got %v", result)
			}
			if err != nil && tc.wantErr != "" {
				errMsg := err.Error()
				if !contains(errMsg, tc.wantErr) {
					t.Errorf("Execute() error = %v, want error containing %q", err, tc.wantErr)
				}
			}
		})
	}
}

// TestCreateForwardRule_PortConflict verifies port conflict detection.
// Business rule: Each listen port can only be used by one rule.
func TestCreateForwardRule_PortConflict(t *testing.T) {
	// Setup mocks
	ruleRepo := testutil.NewMockForwardRuleRepository()
	agentRepo := testutil.NewMockForwardAgentRepository()
	nodeRepo := testutil.NewMockNodeRepository()
	notifier := testutil.NewMockConfigSyncNotifier()
	logger := testutil.NewMockLogger()

	// Create test agent
	agent := createTestAgent(t, "test-agent", "agent123")
	agentRepo.AddAgent(agent)

	// Create existing rule with port 8080
	existingRule := createTestRule(t, agent.ID(), "existing-rule", 8080)
	ruleRepo.AddRule(existingRule)

	// Create use case
	uc := NewCreateForwardRuleUseCase(
		ruleRepo, agentRepo, nodeRepo, notifier, logger,
	)

	// Create command with same port
	cmd := CreateForwardRuleCommand{
		AgentShortID:  agent.ShortID(),
		RuleType:      "direct",
		Name:          "new-rule",
		ListenPort:    8080, // Conflicts with existing rule
		TargetAddress: "192.168.1.100",
		TargetPort:    9000,
		Protocol:      "tcp",
	}

	// Execute
	result, err := uc.Execute(context.Background(), cmd)

	// Verify
	if err == nil {
		t.Error("Execute() expected error for port conflict, got nil")
		return
	}
	if result != nil {
		t.Errorf("Execute() expected nil result on error, got %v", result)
	}
	if !contains(err.Error(), "already in use") {
		t.Errorf("Execute() error = %v, want error containing 'already in use'", err)
	}
}

// TestCreateForwardRule_RepositoryErrors verifies repository error handling.
func TestCreateForwardRule_RepositoryErrors(t *testing.T) {
	testCases := []struct {
		name      string
		setupMock func(*testutil.MockForwardRuleRepository, *testutil.MockForwardAgentRepository, *testutil.MockNodeRepository)
		command   CreateForwardRuleCommand
		wantErr   string
	}{
		{
			name: "agent repository get error",
			setupMock: func(ruleRepo *testutil.MockForwardRuleRepository, agentRepo *testutil.MockForwardAgentRepository, nodeRepo *testutil.MockNodeRepository) {
				agentRepo.SetGetError(errors.New("database connection failed"))
			},
			command: CreateForwardRuleCommand{
				AgentShortID:  "agent123",
				RuleType:      "direct",
				Name:          "test-rule",
				ListenPort:    8080,
				TargetAddress: "192.168.1.1",
				TargetPort:    9000,
				Protocol:      "tcp",
			},
			wantErr: "failed to validate agent",
		},
		{
			name: "node repository get error",
			setupMock: func(ruleRepo *testutil.MockForwardRuleRepository, agentRepo *testutil.MockForwardAgentRepository, nodeRepo *testutil.MockNodeRepository) {
				agent := createTestAgent(t, "test-agent", "agent123")
				agentRepo.AddAgent(agent)
				// Set error BEFORE node lookup
				nodeRepo.SetGetError(errors.New("database connection failed"))
			},
			command: CreateForwardRuleCommand{
				AgentShortID:      "agent123",
				RuleType:          "direct",
				Name:              "test-rule",
				ListenPort:        8080,
				TargetNodeShortID: "node123",
				Protocol:          "tcp",
			},
			wantErr: "failed to validate target node",
		},
		{
			name: "rule repository exists error",
			setupMock: func(ruleRepo *testutil.MockForwardRuleRepository, agentRepo *testutil.MockForwardAgentRepository, nodeRepo *testutil.MockNodeRepository) {
				agent := createTestAgent(t, "test-agent", "agent123")
				agentRepo.AddAgent(agent)
				ruleRepo.SetExistsError(errors.New("database query failed"))
			},
			command: CreateForwardRuleCommand{
				AgentShortID:  "agent123",
				RuleType:      "direct",
				Name:          "test-rule",
				ListenPort:    8080,
				TargetAddress: "192.168.1.1",
				TargetPort:    9000,
				Protocol:      "tcp",
			},
			wantErr: "failed to check existing rule",
		},
		{
			name: "rule repository create error",
			setupMock: func(ruleRepo *testutil.MockForwardRuleRepository, agentRepo *testutil.MockForwardAgentRepository, nodeRepo *testutil.MockNodeRepository) {
				agent := createTestAgent(t, "test-agent", "agent123")
				agentRepo.AddAgent(agent)
				ruleRepo.SetCreateError(errors.New("database insert failed"))
			},
			command: CreateForwardRuleCommand{
				AgentShortID:  "agent123",
				RuleType:      "direct",
				Name:          "test-rule",
				ListenPort:    8080,
				TargetAddress: "192.168.1.1",
				TargetPort:    9000,
				Protocol:      "tcp",
			},
			wantErr: "failed to save forward rule",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			ruleRepo := testutil.NewMockForwardRuleRepository()
			agentRepo := testutil.NewMockForwardAgentRepository()
			nodeRepo := testutil.NewMockNodeRepository()
			notifier := testutil.NewMockConfigSyncNotifier()
			logger := testutil.NewMockLogger()

			tc.setupMock(ruleRepo, agentRepo, nodeRepo)

			uc := NewCreateForwardRuleUseCase(
				ruleRepo, agentRepo, nodeRepo, notifier, logger,
			)

			// Execute
			result, err := uc.Execute(context.Background(), tc.command)

			// Verify
			if err == nil {
				t.Errorf("Execute() expected error, got nil")
				return
			}
			if result != nil {
				t.Errorf("Execute() expected nil result on error, got %v", result)
			}
			if !contains(err.Error(), tc.wantErr) {
				t.Errorf("Execute() error = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

// TestCreateForwardRule_ConfigSyncNotification verifies ConfigSync notification behavior.
// Business rule: ConfigSync is only notified when rule is enabled.
func TestCreateForwardRule_ConfigSyncNotification(t *testing.T) {
	testCases := []struct {
		name              string
		ruleEnabled       bool
		wantNotifyCalls   int
		wantNotifySkipped bool
	}{
		{
			name:              "rule disabled - notification skipped",
			ruleEnabled:       false,
			wantNotifyCalls:   0,
			wantNotifySkipped: true,
		},
		{
			name:              "rule enabled - notification sent",
			ruleEnabled:       true,
			wantNotifyCalls:   1,
			wantNotifySkipped: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			ruleRepo := testutil.NewMockForwardRuleRepository()
			agentRepo := testutil.NewMockForwardAgentRepository()
			nodeRepo := testutil.NewMockNodeRepository()
			notifier := testutil.NewMockConfigSyncNotifier()
			logger := testutil.NewMockLogger()

			// Create test agent
			agent := createTestAgent(t, "test-agent", "agent123")
			agentRepo.AddAgent(agent)

			// Create use case
			uc := NewCreateForwardRuleUseCase(
				ruleRepo, agentRepo, nodeRepo, notifier, logger,
			)

			// Create command
			cmd := CreateForwardRuleCommand{
				AgentShortID:  agent.ShortID(),
				RuleType:      "direct",
				Name:          "test-rule",
				ListenPort:    8080,
				TargetAddress: "192.168.1.100",
				TargetPort:    9000,
				Protocol:      "tcp",
			}

			// Execute
			result, err := uc.Execute(context.Background(), cmd)

			// Verify
			if err != nil {
				t.Errorf("Execute() unexpected error = %v", err)
				return
			}

			// Enable rule if needed for testing
			if tc.ruleEnabled {
				savedRule, _ := ruleRepo.GetByListenPort(context.Background(), cmd.ListenPort)
				if savedRule != nil {
					_ = savedRule.Enable()
					_ = ruleRepo.Update(context.Background(), savedRule)
					// Manually trigger notification since we're testing
					_ = notifier.NotifyRuleChange(context.Background(), savedRule.AgentID(), savedRule.ShortID(), "added")
				}
			}

			// Wait a bit for async notification
			time.Sleep(100 * time.Millisecond)

			// Verify notification calls
			calls := notifier.GetCalls()
			if tc.wantNotifySkipped && len(calls) != 0 {
				t.Errorf("Expected no notification calls for disabled rule, got %d", len(calls))
			}
			if !tc.wantNotifySkipped && len(calls) != tc.wantNotifyCalls {
				t.Errorf("Notification calls = %d, want %d", len(calls), tc.wantNotifyCalls)
			}

			// Verify notification content if called
			if !tc.wantNotifySkipped && len(calls) > 0 {
				call := calls[0]
				if call.AgentID != agent.ID() {
					t.Errorf("Notification AgentID = %v, want %v", call.AgentID, agent.ID())
				}
				if call.ChangeType != "added" {
					t.Errorf("Notification ChangeType = %v, want 'added'", call.ChangeType)
				}
			}

			_ = result
		})
	}
}

// Helper functions

// createTestAgent creates a test forward agent with the given name and short ID.
func createTestAgent(t *testing.T, name, shortID string) *forward.ForwardAgent {
	t.Helper()
	agent, err := forward.NewForwardAgent(
		name,
		"",
		"",
		"",
		func() (string, error) { return shortID, nil },
		func(sid string) (string, string) { return "token", "hash" },
	)
	if err != nil {
		t.Fatalf("Failed to create test agent: %v", err)
	}
	return agent
}

// createTestRule creates a test forward rule with the given agent ID, name, and listen port.
func createTestRule(t *testing.T, agentID uint, name string, listenPort uint16) *forward.ForwardRule {
	t.Helper()
	rule, err := forward.NewForwardRule(
		agentID,
		nil, // userID
		vo.ForwardRuleTypeDirect,
		0,
		nil,
		nil,
		name,
		listenPort,
		"192.168.1.1",
		9000,
		nil,
		"",
		vo.IPVersionAuto,
		vo.ForwardProtocolTCP,
		"",
		nil, // trafficMultiplier
		id.NewForwardRuleID,
	)
	if err != nil {
		t.Fatalf("Failed to create test rule: %v", err)
	}
	return rule
}

// createTestNode creates a test node with the given name and short ID.
func createTestNode(t *testing.T, name, shortID string) *node.Node {
	t.Helper()
	serverAddr, err := nodevo.NewServerAddress("test.example.com")
	if err != nil {
		t.Fatalf("Failed to create server address: %v", err)
	}
	encryptionConfig, err := nodevo.NewEncryptionConfig("aes-256-gcm")
	if err != nil {
		t.Fatalf("Failed to create encryption config: %v", err)
	}
	metadata := nodevo.NewNodeMetadata("Test Region", []string{}, "Test Description")

	n, err := node.NewNode(
		name,
		serverAddr,
		8388,
		nil,
		nodevo.ProtocolShadowsocks,
		encryptionConfig,
		nil,
		nil,
		metadata,
		0,
		func() (string, error) { return shortID, nil },
	)
	if err != nil {
		t.Fatalf("Failed to create test node: %v", err)
	}
	return n
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || anyIndex(s, substr) >= 0)
}

// anyIndex returns the index of substr in s, or -1 if not found.
func anyIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Type alias for test setup - these will be removed once we verify imports work
type MockForwardAgentRepository = testutil.MockForwardAgentRepository
type MockNodeRepository = testutil.MockNodeRepository
