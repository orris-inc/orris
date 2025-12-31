// Package crud provides HTTP handlers for forward agent CRUD management.
package crud

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/mod/semver"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// VersionHandler handles version-related operations for forward agents.
type VersionHandler struct {
	agentRepo      forward.AgentRepository
	releaseService *services.GitHubReleaseService
	agentHub       *services.AgentHub
	logger         logger.Interface
}

// NewVersionHandler creates a new VersionHandler.
func NewVersionHandler(
	agentRepo forward.AgentRepository,
	releaseService *services.GitHubReleaseService,
	agentHub *services.AgentHub,
	log logger.Interface,
) *VersionHandler {
	return &VersionHandler{
		agentRepo:      agentRepo,
		releaseService: releaseService,
		agentHub:       agentHub,
		logger:         log,
	}
}

// GetAgentVersion handles GET /forward-agents/:id/version
// Returns current version, latest version, and whether an update is available.
func (h *VersionHandler) GetAgentVersion(c *gin.Context) {
	shortID, err := parseAgentShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Get agent from database
	agent, err := h.agentRepo.GetBySID(c.Request.Context(), shortID)
	if err != nil {
		h.logger.Errorw("failed to get agent", "sid", shortID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}
	if agent == nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Forward agent not found")
		return
	}

	// Get latest release from GitHub
	releaseInfo, err := h.releaseService.GetLatestRelease(c.Request.Context())
	if err != nil {
		h.logger.Warnw("failed to get latest release", "error", err)
		// Return partial info without latest version
		info := &dto.AgentVersionInfo{
			AgentID:        agent.SID(),
			CurrentVersion: agent.AgentVersion(),
			Platform:       agent.Platform(),
			Arch:           agent.Arch(),
			HasUpdate:      false,
		}
		utils.SuccessResponse(c, http.StatusOK, "", info)
		return
	}

	// Get download URL for agent's platform/arch
	var downloadURL string
	if agent.Platform() != "" && agent.Arch() != "" {
		url, err := h.releaseService.GetDownloadURL(c.Request.Context(), agent.Platform(), agent.Arch())
		if err == nil {
			downloadURL = url
		}
	}

	// Compare versions using semver
	hasUpdate := hasNewerVersion(agent.AgentVersion(), releaseInfo.Version)

	info := &dto.AgentVersionInfo{
		AgentID:        agent.SID(),
		CurrentVersion: agent.AgentVersion(),
		LatestVersion:  releaseInfo.Version,
		HasUpdate:      hasUpdate,
		Platform:       agent.Platform(),
		Arch:           agent.Arch(),
		DownloadURL:    downloadURL,
		PublishedAt:    releaseInfo.PublishedAt.Format("2006-01-02T15:04:05Z"),
	}

	utils.SuccessResponse(c, http.StatusOK, "", info)
}

// TriggerUpdateResponse is the response for TriggerUpdate API.
type TriggerUpdateResponse struct {
	AgentID       string `json:"agent_id"`
	CommandID     string `json:"command_id"`
	TargetVersion string `json:"target_version"`
	Message       string `json:"message"`
}

// TriggerUpdate handles POST /forward-agents/:id/update
// Sends an update command to the agent to trigger self-update.
func (h *VersionHandler) TriggerUpdate(c *gin.Context) {
	shortID, err := parseAgentShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Get agent from database
	agent, err := h.agentRepo.GetBySID(c.Request.Context(), shortID)
	if err != nil {
		h.logger.Errorw("failed to get agent", "sid", shortID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}
	if agent == nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Forward agent not found")
		return
	}

	// Check if agent is online
	if !h.agentHub.IsAgentOnline(agent.ID()) {
		utils.ErrorResponse(c, http.StatusConflict, "Agent is offline, cannot send update command")
		return
	}

	// Check if platform and arch are set
	if agent.Platform() == "" || agent.Arch() == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Agent platform or architecture is unknown")
		return
	}

	// Get latest release from GitHub
	releaseInfo, err := h.releaseService.GetLatestRelease(c.Request.Context())
	if err != nil {
		h.logger.Errorw("failed to get latest release", "error", err)
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "Failed to get latest release information")
		return
	}

	// Check if there is an update available using semver comparison
	if !hasNewerVersion(agent.AgentVersion(), releaseInfo.Version) {
		utils.ErrorResponse(c, http.StatusConflict, "Agent is already at the latest version")
		return
	}

	// Get download URL for agent's platform/arch
	downloadURL, err := h.releaseService.GetDownloadURL(c.Request.Context(), agent.Platform(), agent.Arch())
	if err != nil {
		h.logger.Errorw("failed to get download URL", "platform", agent.Platform(), "arch", agent.Arch(), "error", err)
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "No download available for agent's platform and architecture")
		return
	}

	// Get checksum for agent's platform/arch
	var checksum string
	checksum, err = h.releaseService.GetChecksum(c.Request.Context(), agent.Platform(), agent.Arch())
	if err != nil {
		// Log warning but don't fail the update - checksum is optional for backward compatibility
		h.logger.Warnw("failed to get checksum, proceeding without verification",
			"platform", agent.Platform(),
			"arch", agent.Arch(),
			"error", err,
		)
	}

	// Build update payload
	updatePayload := &dto.UpdatePayload{
		Version:     releaseInfo.Version,
		DownloadURL: downloadURL,
		Checksum:    checksum,
	}

	// Build command data
	commandID := uuid.New().String()
	cmd := &dto.CommandData{
		CommandID: commandID,
		Action:    dto.CmdActionUpdate,
		Payload:   updatePayload,
	}

	// Send update command to agent
	if err := h.agentHub.SendCommandToAgent(agent.ID(), cmd); err != nil {
		h.logger.Errorw("failed to send update command to agent",
			"agent_id", agent.ID(),
			"sid", shortID,
			"error", err,
		)

		if errors.Is(err, services.ErrAgentNotConnected) {
			utils.ErrorResponse(c, http.StatusConflict, "Agent disconnected while processing request")
			return
		}
		if errors.Is(err, services.ErrSendChannelFull) {
			utils.ErrorResponse(c, http.StatusServiceUnavailable, "Agent command queue is full, please try again later")
			return
		}

		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to send update command")
		return
	}

	h.logger.Infow("update command sent to agent",
		"agent_id", agent.ID(),
		"sid", shortID,
		"command_id", commandID,
		"target_version", releaseInfo.Version,
		"checksum", checksum,
	)

	response := &TriggerUpdateResponse{
		AgentID:       agent.SID(),
		CommandID:     commandID,
		TargetVersion: releaseInfo.Version,
		Message:       "Update command sent successfully",
	}

	utils.SuccessResponse(c, http.StatusOK, "Update command sent", response)
}

// normalizeVersion ensures version string has "v" prefix for semver compatibility.
// Examples: "1.2.3" -> "v1.2.3", "v1.2.3" -> "v1.2.3"
func normalizeVersion(version string) string {
	if version == "" {
		return ""
	}
	version = strings.TrimSpace(version)
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

// hasNewerVersion checks if latestVersion is newer than currentVersion using semver.
// Returns true if an update is available.
func hasNewerVersion(currentVersion, latestVersion string) bool {
	// If latest version is unknown, no update available
	if latestVersion == "" {
		return false
	}

	// If current version is empty or "dev", always suggest update
	if currentVersion == "" || currentVersion == "dev" {
		return true
	}

	current := normalizeVersion(currentVersion)
	latest := normalizeVersion(latestVersion)

	// Validate both versions are valid semver
	if !semver.IsValid(current) {
		// Current version is not valid semver (e.g., "dev", "unknown")
		// Suggest update to get a proper release version
		return true
	}
	if !semver.IsValid(latest) {
		// Latest version is not valid semver, can't compare
		return false
	}

	// semver.Compare returns:
	// -1 if current < latest (update available)
	//  0 if current == latest (no update)
	// +1 if current > latest (current is newer, e.g., dev build)
	return semver.Compare(current, latest) < 0
}

// Maximum number of agents that can be updated in a single batch request.
const maxBatchUpdateAgents = 1000

// BatchTriggerUpdate handles POST /forward-agents/batch-update
// Triggers update for multiple agents at once.
func (h *VersionHandler) BatchTriggerUpdate(c *gin.Context) {
	var req dto.BatchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request - must specify exactly one of agent_ids or update_all
	hasAgentIDs := len(req.AgentIDs) > 0
	if hasAgentIDs && req.UpdateAll {
		utils.ErrorResponse(c, http.StatusBadRequest, "Cannot specify both agent_ids and update_all")
		return
	}
	if !hasAgentIDs && !req.UpdateAll {
		utils.ErrorResponse(c, http.StatusBadRequest, "Either agent_ids or update_all must be specified")
		return
	}

	ctx := c.Request.Context()

	// Get latest release info once for all agents
	releaseInfo, err := h.releaseService.GetLatestRelease(ctx)
	if err != nil {
		h.logger.Errorw("failed to get latest release for batch update", "error", err)
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "Failed to get latest release information")
		return
	}

	response := &dto.BatchUpdateResponse{
		Succeeded: []dto.BatchUpdateSuccess{},
		Failed:    []dto.BatchUpdateFailed{},
		Skipped:   []dto.BatchUpdateSkipped{},
	}

	// Get target agents
	var agents []*forward.ForwardAgent
	if hasAgentIDs {
		// Deduplicate agent IDs
		seen := make(map[string]bool)
		uniqueIDs := make([]string, 0, len(req.AgentIDs))
		for _, sid := range req.AgentIDs {
			if !seen[sid] {
				seen[sid] = true
				uniqueIDs = append(uniqueIDs, sid)
			}
		}

		// Get specified agents
		for _, sid := range uniqueIDs {
			agent, err := h.agentRepo.GetBySID(ctx, sid)
			if err != nil {
				h.logger.Warnw("failed to get agent for batch update", "sid", sid, "error", err)
				response.Failed = append(response.Failed, dto.BatchUpdateFailed{
					AgentID: sid,
					Reason:  "failed to retrieve agent",
				})
				continue
			}
			if agent == nil {
				response.Failed = append(response.Failed, dto.BatchUpdateFailed{
					AgentID: sid,
					Reason:  "agent not found",
				})
				continue
			}
			agents = append(agents, agent)
		}
	} else {
		// Get all agents for update_all mode with limit
		filter := forward.AgentListFilter{
			Page:     1,
			PageSize: maxBatchUpdateAgents,
		}
		allAgents, total, err := h.agentRepo.List(ctx, filter)
		if err != nil {
			h.logger.Errorw("failed to list agents for batch update", "error", err)
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to list agents")
			return
		}
		if total > int64(maxBatchUpdateAgents) {
			h.logger.Warnw("batch update truncated due to limit",
				"total", total,
				"limit", maxBatchUpdateAgents,
			)
			response.Truncated = true
		}
		agents = allAgents
	}

	// Process each agent
	for _, agent := range agents {
		result := h.processSingleAgentUpdate(ctx, agent, releaseInfo)
		switch result.status {
		case updateStatusSucceeded:
			response.Succeeded = append(response.Succeeded, dto.BatchUpdateSuccess{
				AgentID:       agent.SID(),
				CommandID:     result.commandID,
				TargetVersion: releaseInfo.Version,
			})
		case updateStatusFailed:
			response.Failed = append(response.Failed, dto.BatchUpdateFailed{
				AgentID: agent.SID(),
				Reason:  result.reason,
			})
		case updateStatusSkipped:
			response.Skipped = append(response.Skipped, dto.BatchUpdateSkipped{
				AgentID: agent.SID(),
				Reason:  result.reason,
			})
		}
	}

	// Set total count
	response.Total = len(response.Succeeded) + len(response.Failed) + len(response.Skipped)

	h.logger.Infow("batch update completed",
		"total", response.Total,
		"succeeded", len(response.Succeeded),
		"failed", len(response.Failed),
		"skipped", len(response.Skipped),
	)

	utils.SuccessResponse(c, http.StatusOK, "Batch update processed", response)
}

// Update result status constants.
const (
	updateStatusSucceeded = "succeeded"
	updateStatusFailed    = "failed"
	updateStatusSkipped   = "skipped"
)

// agentUpdateResult holds the result of a single agent update attempt.
type agentUpdateResult struct {
	status    string
	commandID string
	reason    string
}

// processSingleAgentUpdate processes update for a single agent.
func (h *VersionHandler) processSingleAgentUpdate(
	ctx context.Context,
	agent *forward.ForwardAgent,
	releaseInfo *services.ReleaseInfo,
) agentUpdateResult {
	// Check if agent is online
	if !h.agentHub.IsAgentOnline(agent.ID()) {
		return agentUpdateResult{status: updateStatusSkipped, reason: "agent is offline"}
	}

	// Check if platform and arch are set
	if agent.Platform() == "" || agent.Arch() == "" {
		return agentUpdateResult{status: updateStatusSkipped, reason: "platform or architecture is unknown"}
	}

	// Check if update is needed
	if !hasNewerVersion(agent.AgentVersion(), releaseInfo.Version) {
		return agentUpdateResult{status: updateStatusSkipped, reason: "already up to date"}
	}

	// Get download URL
	downloadURL, err := h.releaseService.GetDownloadURL(ctx, agent.Platform(), agent.Arch())
	if err != nil {
		return agentUpdateResult{status: updateStatusFailed, reason: "no download available for platform/arch"}
	}

	// Get checksum (optional)
	checksum, _ := h.releaseService.GetChecksum(ctx, agent.Platform(), agent.Arch())

	// Build update payload
	updatePayload := &dto.UpdatePayload{
		Version:     releaseInfo.Version,
		DownloadURL: downloadURL,
		Checksum:    checksum,
	}

	// Build command
	commandID := uuid.New().String()
	cmd := &dto.CommandData{
		CommandID: commandID,
		Action:    dto.CmdActionUpdate,
		Payload:   updatePayload,
	}

	// Send command
	if err := h.agentHub.SendCommandToAgent(agent.ID(), cmd); err != nil {
		if errors.Is(err, services.ErrAgentNotConnected) {
			return agentUpdateResult{status: updateStatusFailed, reason: "agent disconnected during processing"}
		}
		if errors.Is(err, services.ErrSendChannelFull) {
			return agentUpdateResult{status: updateStatusFailed, reason: "command queue is full"}
		}
		return agentUpdateResult{status: updateStatusFailed, reason: "failed to send command"}
	}

	h.logger.Infow("update command sent to agent",
		"agent_id", agent.ID(),
		"sid", agent.SID(),
		"command_id", commandID,
		"target_version", releaseInfo.Version,
	)

	return agentUpdateResult{status: updateStatusSucceeded, commandID: commandID}
}
