package forward

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v5"
)

// RunHubLoop connects to the hub and handles messages.
// This is a convenience method that manages the connection lifecycle without reconnection.
// For production use, prefer RunHubLoopWithReconnect which handles automatic reconnection.
func (c *Client) RunHubLoop(ctx context.Context, probeHandler ProbeTaskHandler) error {
	conn, err := c.ConnectHub(ctx)
	if err != nil {
		return fmt.Errorf("connect hub: %w", err)
	}
	defer conn.Close()

	// Set up message handler
	conn.SetMessageHandler(func(msg *HubMessage) {
		switch msg.Type {
		case MsgTypeProbeTask:
			if probeHandler != nil {
				go func() {
					task := parseProbeTask(msg.Data)
					if task != nil {
						result := probeHandler(task)
						if result != nil {
							conn.SendProbeResult(result)
						}
					}
				}()
			}
		case MsgTypeCommand:
			// Handle commands if needed
		}
	})

	// Run the connection
	return conn.Run(ctx)
}

// RunHubLoopWithReconnect connects to the hub with automatic reconnection.
// It uses exponential backoff strategy to retry failed connections.
// The probeHandler is called when a probe task is received.
// This method blocks until the context is canceled.
//
// Use the ReconnectConfig callbacks to handle connection events:
//   - OnConnected: called when successfully connected
//   - OnDisconnected: called when disconnected (with error)
//   - OnReconnecting: called before each reconnection attempt
func (c *Client) RunHubLoopWithReconnect(ctx context.Context, probeHandler ProbeTaskHandler, config *ReconnectConfig) error {
	if config == nil {
		config = DefaultReconnectConfig()
	}

	// Create exponential backoff strategy
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = config.InitialInterval
	expBackoff.MaxInterval = config.MaxInterval
	expBackoff.Multiplier = config.Multiplier
	expBackoff.RandomizationFactor = config.RandomizationFactor
	expBackoff.Reset()

	var attempt uint64
	startTime := time.Now()

	// Reconnection loop
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		attempt++

		// Attempt to connect and run
		err := c.runHubLoopOnce(ctx, probeHandler, config)

		// Connection ended, call disconnect callback
		if config.OnDisconnected != nil {
			config.OnDisconnected(err)
		}

		// If context was canceled, exit immediately
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Check max elapsed time
		if config.MaxElapsedTime > 0 && time.Since(startTime) >= config.MaxElapsedTime {
			return fmt.Errorf("reconnection failed after %v: %w", config.MaxElapsedTime, err)
		}

		// Calculate next backoff delay
		delay := expBackoff.NextBackOff()
		if delay == backoff.Stop {
			return fmt.Errorf("reconnection failed: %w", err)
		}

		// Call reconnecting callback with attempt info
		if config.OnReconnecting != nil {
			config.OnReconnecting(attempt, delay)
		}

		// Wait before reconnecting
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// Continue to next connection attempt
		}
	}
}

// runHubLoopOnce executes a single hub connection lifecycle.
func (c *Client) runHubLoopOnce(ctx context.Context, probeHandler ProbeTaskHandler, config *ReconnectConfig) error {
	conn, err := c.ConnectHub(ctx)
	if err != nil {
		return fmt.Errorf("connect hub: %w", err)
	}
	defer conn.Close()

	// Call connected callback
	if config.OnConnected != nil {
		config.OnConnected()
	}

	// Set up message handler
	conn.SetMessageHandler(func(msg *HubMessage) {
		switch msg.Type {
		case MsgTypeProbeTask:
			if probeHandler != nil {
				go func() {
					task := parseProbeTask(msg.Data)
					if task != nil {
						result := probeHandler(task)
						if result != nil {
							conn.SendProbeResult(result)
						}
					}
				}()
			}
		case MsgTypeConfigSync:
			if config.OnConfigSync != nil {
				go func() {
					configSync := parseConfigSync(msg.Data)
					if configSync != nil {
						err := config.OnConfigSync(configSync)
						// Send acknowledgment back to server
						ack := &ConfigAckData{
							Version: configSync.Version,
							Success: err == nil,
						}
						if err != nil {
							ack.Error = err.Error()
						}
						conn.SendConfigAck(ack)
					}
				}()
			}
		case MsgTypeCommand:
			// Handle commands if needed
		}
	})

	// Run the connection (blocks until disconnected)
	return conn.Run(ctx)
}
