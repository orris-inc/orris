// Package testutil provides mock implementations for testing the forward application layer.
package testutil

import (
	"context"
	"testing"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
)

// TestMockForwardRuleRepository demonstrates basic usage of MockForwardRuleRepository.
func TestMockForwardRuleRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewMockForwardRuleRepository()

	// Create a test rule
	rule, err := forward.NewForwardRule(
		1,   // agentID
		nil, // userID
		vo.ForwardRuleTypeDirect,
		0,   // exitAgentID
		nil, // chainAgentIDs
		nil, // chainPortConfig
		"test-rule",
		8080,
		"192.168.1.1",
		80,
		nil, // targetNodeID
		"",  // bindIP
		vo.IPVersionAuto,
		vo.ForwardProtocolTCP,
		"test remark",
		nil, // trafficMultiplier
		func() (string, error) { return "fr_test123", nil },
	)
	if err != nil {
		t.Fatalf("failed to create rule: %v", err)
	}

	// Test Create
	err = repo.Create(ctx, rule)
	if err != nil {
		t.Fatalf("failed to create rule: %v", err)
	}

	// Test GetByID
	retrieved, err := repo.GetByID(ctx, rule.ID())
	if err != nil {
		t.Fatalf("failed to get rule: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected rule, got nil")
	}
	if retrieved.ID() != rule.ID() {
		t.Errorf("expected ID %d, got %d", rule.ID(), retrieved.ID())
	}

	// Test GetByShortID
	retrieved, err = repo.GetByShortID(ctx, rule.ShortID())
	if err != nil {
		t.Fatalf("failed to get rule by short ID: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected rule, got nil")
	}

	// Test ListEnabled
	err = rule.Enable()
	if err != nil {
		t.Fatalf("failed to enable rule: %v", err)
	}
	err = repo.Update(ctx, rule)
	if err != nil {
		t.Fatalf("failed to update rule: %v", err)
	}

	enabledRules, err := repo.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("failed to list enabled rules: %v", err)
	}
	if len(enabledRules) != 1 {
		t.Errorf("expected 1 enabled rule, got %d", len(enabledRules))
	}

	// Test Delete
	err = repo.Delete(ctx, rule.ID())
	if err != nil {
		t.Fatalf("failed to delete rule: %v", err)
	}

	retrieved, err = repo.GetByID(ctx, rule.ID())
	if err != nil {
		t.Fatalf("failed to get rule: %v", err)
	}
	if retrieved != nil {
		t.Error("expected rule to be deleted")
	}
}

// TestMockForwardRuleRepository_ErrorInjection demonstrates error injection.
func TestMockForwardRuleRepository_ErrorInjection(t *testing.T) {
	ctx := context.Background()
	repo := NewMockForwardRuleRepository()

	// Inject create error
	expectedErr := forward.ErrRuleNotFound
	repo.SetCreateError(expectedErr)

	rule, err := forward.NewForwardRule(
		1,
		nil, // userID
		vo.ForwardRuleTypeDirect,
		0,
		nil,
		nil,
		"test-rule",
		8080,
		"192.168.1.1",
		80,
		nil,
		"",
		vo.IPVersionAuto,
		vo.ForwardProtocolTCP,
		"test remark",
		nil, // trafficMultiplier
		func() (string, error) { return "fr_test123", nil },
	)
	if err != nil {
		t.Fatalf("failed to create rule: %v", err)
	}

	err = repo.Create(ctx, rule)
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	// Reset error
	repo.SetCreateError(nil)
	err = repo.Create(ctx, rule)
	if err != nil {
		t.Fatalf("expected no error after reset, got %v", err)
	}
}

// TestMockForwardAgentRepository demonstrates basic usage of MockForwardAgentRepository.
func TestMockForwardAgentRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewMockForwardAgentRepository()

	// Create a test agent
	tokenGen := func(shortID string) (string, string) {
		return "test_token", "test_hash"
	}

	agent, err := forward.NewForwardAgent(
		"test-agent",
		"192.168.1.1",
		"tunnel.example.com",
		"test remark",
		func() (string, error) { return "fa_test123", nil },
		tokenGen,
	)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Test Create
	err = repo.Create(ctx, agent)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Test GetByID
	retrieved, err := repo.GetByID(ctx, agent.ID())
	if err != nil {
		t.Fatalf("failed to get agent: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected agent, got nil")
	}
	if retrieved.ID() != agent.ID() {
		t.Errorf("expected ID %d, got %d", agent.ID(), retrieved.ID())
	}

	// Test GetByTokenHash
	retrieved, err = repo.GetByTokenHash(ctx, agent.TokenHash())
	if err != nil {
		t.Fatalf("failed to get agent by token hash: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected agent, got nil")
	}

	// Test ExistsByName
	exists, err := repo.ExistsByName(ctx, "test-agent")
	if err != nil {
		t.Fatalf("failed to check existence: %v", err)
	}
	if !exists {
		t.Error("expected agent to exist")
	}

	// Test GetShortIDsByIDs
	shortIDs, err := repo.GetShortIDsByIDs(ctx, []uint{agent.ID()})
	if err != nil {
		t.Fatalf("failed to get short IDs: %v", err)
	}
	if shortIDs[agent.ID()] != agent.ShortID() {
		t.Errorf("expected short ID %s, got %s", agent.ShortID(), shortIDs[agent.ID()])
	}
}

// TestMockConfigSyncNotifier demonstrates basic usage of MockConfigSyncNotifier.
func TestMockConfigSyncNotifier(t *testing.T) {
	ctx := context.Background()
	notifier := NewMockConfigSyncNotifier()

	// Test NotifyRuleChange
	err := notifier.NotifyRuleChange(ctx, 1, "fr_test123", "created")
	if err != nil {
		t.Fatalf("failed to notify rule change: %v", err)
	}

	// Verify call was recorded
	calls := notifier.GetCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}

	if calls[0].AgentID != 1 {
		t.Errorf("expected agent ID 1, got %d", calls[0].AgentID)
	}
	if calls[0].RuleShortID != "fr_test123" {
		t.Errorf("expected rule short ID 'fr_test123', got '%s'", calls[0].RuleShortID)
	}
	if calls[0].ChangeType != "created" {
		t.Errorf("expected change type 'created', got '%s'", calls[0].ChangeType)
	}

	// Test error injection
	notifier.Reset()
	expectedErr := context.Canceled
	notifier.SetError(expectedErr)

	err = notifier.NotifyRuleChange(ctx, 1, "fr_test123", "created")
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

// TestMockAgentStatusQuerier demonstrates basic usage of MockAgentStatusQuerier.
func TestMockAgentStatusQuerier(t *testing.T) {
	ctx := context.Background()
	querier := NewMockAgentStatusQuerier()

	// Set up test status
	status := &dto.AgentStatusDTO{
		CPUPercent:        50.0,
		MemoryPercent:     60.0,
		ActiveRules:       5,
		ActiveConnections: 10,
	}
	querier.SetStatus(1, status)

	// Test GetStatus
	retrieved, err := querier.GetStatus(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected status, got nil")
	}
	if retrieved.CPUPercent != 50.0 {
		t.Errorf("expected CPU percent 50.0, got %f", retrieved.CPUPercent)
	}

	// Test GetMultipleStatus
	statuses, err := querier.GetMultipleStatus(ctx, []uint{1})
	if err != nil {
		t.Fatalf("failed to get multiple statuses: %v", err)
	}
	if len(statuses) != 1 {
		t.Errorf("expected 1 status, got %d", len(statuses))
	}

	// Test non-existent agent
	retrieved, err = querier.GetStatus(ctx, 999)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}
	if retrieved != nil {
		t.Error("expected nil status for non-existent agent")
	}
}

// TestMockLogger demonstrates basic usage of MockLogger.
func TestMockLogger(t *testing.T) {
	logger := NewMockLogger()

	// Test logging
	logger.Info("test message", "key1", "value1", "key2", 123)
	logger.Error("error message", "error", "test error")

	// Verify entries
	entries := logger.GetEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Check first entry
	if entries[0].Level != "INFO" {
		t.Errorf("expected level INFO, got %s", entries[0].Level)
	}
	if entries[0].Message != "test message" {
		t.Errorf("expected message 'test message', got '%s'", entries[0].Message)
	}
	if entries[0].Fields["key1"] != "value1" {
		t.Errorf("expected field key1 to be 'value1', got '%v'", entries[0].Fields["key1"])
	}

	// Check second entry
	if entries[1].Level != "ERROR" {
		t.Errorf("expected level ERROR, got %s", entries[1].Level)
	}

	// Test reset
	logger.Reset()
	entries = logger.GetEntries()
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after reset, got %d", len(entries))
	}
}
