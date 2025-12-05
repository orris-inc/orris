package forward

import "errors"

var (
	// ErrRuleNotFound is returned when a forward rule is not found.
	ErrRuleNotFound = errors.New("forward rule not found")

	// ErrPortAlreadyUsed is returned when the listen port is already in use.
	ErrPortAlreadyUsed = errors.New("listen port is already in use")

	// ErrRuleAlreadyEnabled is returned when trying to enable an already enabled rule.
	ErrRuleAlreadyEnabled = errors.New("forward rule is already enabled")

	// ErrRuleAlreadyDisabled is returned when trying to disable an already disabled rule.
	ErrRuleAlreadyDisabled = errors.New("forward rule is already disabled")

	// ErrInvalidProtocol is returned when an invalid protocol is specified.
	ErrInvalidProtocol = errors.New("invalid protocol")

	// ErrInvalidTargetAddress is returned when the target address is invalid.
	ErrInvalidTargetAddress = errors.New("invalid target address")

	// ErrAgentNotConnected is returned when the agent is not connected.
	ErrAgentNotConnected = errors.New("agent not connected")

	// ErrProbeInProgress is returned when a probe session is already in progress.
	ErrProbeInProgress = errors.New("probe session already in progress")

	// ErrNoProbeTargets is returned when there are no targets to probe.
	ErrNoProbeTargets = errors.New("no probe targets available")
)
