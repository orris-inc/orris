package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/id"
)

// HandleMessage processes probe-related messages from agents.
// Implements agent.MessageHandler interface.
func (s *ProbeService) HandleMessage(agentID uint, msgType string, data any) bool {
	switch msgType {
	case dto.MsgTypeProbeResult:
		s.handleProbeResult(agentID, data)
		return true
	default:
		return false
	}
}

// sendProbeTask sends a probe task to an agent and waits for the result.
func (s *ProbeService) sendProbeTask(
	ctx context.Context,
	agentID uint,
	ruleID string, // Stripe-style prefixed ID
	taskType dto.ProbeTaskType,
	target string,
	port uint16,
	protocol string,
) (int64, error) {
	taskID := uuid.New().String()

	// Create result channel
	resultChan := make(chan *dto.ProbeTaskResult, 1)
	s.pendingProbesMu.Lock()
	s.pendingProbes[taskID] = resultChan
	s.pendingProbesMu.Unlock()

	defer func() {
		s.pendingProbesMu.Lock()
		delete(s.pendingProbes, taskID)
		s.pendingProbesMu.Unlock()
	}()

	// Get agent short ID for Stripe-style prefixed ID
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return 0, err
	}
	if agent == nil {
		return 0, forward.ErrAgentNotFound
	}

	// Send probe task
	task := &dto.ProbeTask{
		ID:       taskID,
		Type:     taskType,
		RuleID:   ruleID,
		Target:   target,
		Port:     port,
		Protocol: protocol,
		Timeout:  int(probeTimeout.Milliseconds()),
	}

	msg := &dto.HubMessage{
		Type:      dto.MsgTypeProbeTask,
		AgentID:   id.FormatForwardAgentID(agent.ShortID()),
		Timestamp: time.Now().Unix(),
		Data:      task,
	}

	if err := s.hub.SendMessageToAgent(agentID, msg); err != nil {
		return 0, err
	}

	// Wait for result with timeout
	select {
	case result := <-resultChan:
		if !result.Success {
			return 0, &probeError{message: result.Error}
		}
		return result.LatencyMs, nil
	case <-time.After(probeTimeout):
		return 0, &probeError{message: "probe timeout"}
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// handleProbeResult handles probe result from agent.
func (s *ProbeService) handleProbeResult(agentID uint, data any) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return
	}

	var result dto.ProbeTaskResult
	if err := json.Unmarshal(dataBytes, &result); err != nil {
		s.logger.Warnw("failed to parse probe result",
			"error", err,
			"agent_id", agentID,
		)
		return
	}

	s.pendingProbesMu.RLock()
	resultChan, ok := s.pendingProbes[result.TaskID]
	s.pendingProbesMu.RUnlock()

	if ok {
		select {
		case resultChan <- &result:
		default:
			// Channel full or closed, ignore
		}
	} else {
		s.logger.Warnw("received probe result for unknown task",
			"task_id", result.TaskID,
			"agent_id", agentID,
		)
	}
}
