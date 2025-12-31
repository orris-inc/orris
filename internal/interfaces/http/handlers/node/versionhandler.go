// Package node provides HTTP handlers for node management.
package node

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/mod/semver"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// NodeVersionHandler handles version-related operations for nodes.
type NodeVersionHandler struct {
	nodeRepo       node.NodeRepository
	releaseService *services.GitHubReleaseService
	agentHub       *services.AgentHub
	logger         logger.Interface
}

// NewNodeVersionHandler creates a new NodeVersionHandler.
func NewNodeVersionHandler(
	nodeRepo node.NodeRepository,
	releaseService *services.GitHubReleaseService,
	agentHub *services.AgentHub,
	log logger.Interface,
) *NodeVersionHandler {
	return &NodeVersionHandler{
		nodeRepo:       nodeRepo,
		releaseService: releaseService,
		agentHub:       agentHub,
		logger:         log,
	}
}

// GetNodeVersion handles GET /nodes/:id/version
// Returns current version, latest version, and whether an update is available.
func (h *NodeVersionHandler) GetNodeVersion(c *gin.Context) {
	sid, err := parseNodeSID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Get node from database
	n, err := h.nodeRepo.GetBySID(c.Request.Context(), sid)
	if err != nil {
		h.logger.Errorw("failed to get node", "sid", sid, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}
	if n == nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Node not found")
		return
	}

	// Get current version info from node
	currentVersion := ""
	platform := ""
	arch := ""
	if n.AgentVersion() != nil {
		currentVersion = *n.AgentVersion()
	}
	if n.AgentPlatform() != nil {
		platform = *n.AgentPlatform()
	}
	if n.AgentArch() != nil {
		arch = *n.AgentArch()
	}

	// Get latest release from GitHub
	releaseInfo, err := h.releaseService.GetLatestRelease(c.Request.Context())
	if err != nil {
		h.logger.Warnw("failed to get latest release", "error", err)
		// Return partial info without latest version
		info := &dto.NodeVersionInfo{
			NodeID:         n.SID(),
			CurrentVersion: currentVersion,
			Platform:       platform,
			Arch:           arch,
			HasUpdate:      false,
		}
		utils.SuccessResponse(c, http.StatusOK, "", info)
		return
	}

	// Get download URL for node's platform/arch
	var downloadURL string
	if platform != "" && arch != "" {
		url, err := h.releaseService.GetDownloadURL(c.Request.Context(), platform, arch)
		if err == nil {
			downloadURL = url
		}
	}

	// Compare versions using semver
	hasUpdate := hasNewerVersion(currentVersion, releaseInfo.Version)

	info := &dto.NodeVersionInfo{
		NodeID:         n.SID(),
		CurrentVersion: currentVersion,
		LatestVersion:  releaseInfo.Version,
		HasUpdate:      hasUpdate,
		Platform:       platform,
		Arch:           arch,
		DownloadURL:    downloadURL,
		PublishedAt:    releaseInfo.PublishedAt.Format("2006-01-02T15:04:05Z"),
	}

	utils.SuccessResponse(c, http.StatusOK, "", info)
}

// TriggerUpdateResponse is the response for TriggerUpdate API.
type TriggerUpdateResponse struct {
	NodeID        string `json:"node_id"`
	CommandID     string `json:"command_id"`
	TargetVersion string `json:"target_version"`
	Message       string `json:"message"`
}

// TriggerUpdate handles POST /nodes/:id/update
// Sends an update command to the node agent to trigger self-update.
func (h *NodeVersionHandler) TriggerUpdate(c *gin.Context) {
	sid, err := parseNodeSID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Get node from database
	n, err := h.nodeRepo.GetBySID(c.Request.Context(), sid)
	if err != nil {
		h.logger.Errorw("failed to get node", "sid", sid, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}
	if n == nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Node not found")
		return
	}

	// Check if node agent is online
	if !h.agentHub.IsNodeOnline(n.ID()) {
		utils.ErrorResponse(c, http.StatusConflict, "Node agent is offline, cannot send update command")
		return
	}

	// Check if platform and arch are set
	if n.AgentPlatform() == nil || *n.AgentPlatform() == "" ||
		n.AgentArch() == nil || *n.AgentArch() == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Node agent platform or architecture is unknown")
		return
	}
	platform := *n.AgentPlatform()
	arch := *n.AgentArch()

	// Get latest release from GitHub
	releaseInfo, err := h.releaseService.GetLatestRelease(c.Request.Context())
	if err != nil {
		h.logger.Errorw("failed to get latest release", "error", err)
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "Failed to get latest release information")
		return
	}

	// Check if there is an update available using semver comparison
	currentVersion := ""
	if n.AgentVersion() != nil {
		currentVersion = *n.AgentVersion()
	}
	if !hasNewerVersion(currentVersion, releaseInfo.Version) {
		utils.ErrorResponse(c, http.StatusConflict, "Node agent is already at the latest version")
		return
	}

	// Get download URL for node's platform/arch
	downloadURL, err := h.releaseService.GetDownloadURL(c.Request.Context(), platform, arch)
	if err != nil {
		h.logger.Errorw("failed to get download URL", "platform", platform, "arch", arch, "error", err)
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "No download available for node's platform and architecture")
		return
	}

	// Get checksum for node's platform/arch
	var checksum string
	checksum, err = h.releaseService.GetChecksum(c.Request.Context(), platform, arch)
	if err != nil {
		// Log warning but don't fail the update - checksum is optional for backward compatibility
		h.logger.Warnw("failed to get checksum, proceeding without verification",
			"platform", platform,
			"arch", arch,
			"error", err,
		)
	}

	// Build update payload
	updatePayload := &dto.NodeUpdatePayload{
		Version:     releaseInfo.Version,
		DownloadURL: downloadURL,
		Checksum:    checksum,
	}

	// Build command data
	commandID := uuid.New().String()
	cmd := &dto.NodeCommandData{
		CommandID: commandID,
		Action:    dto.NodeCmdActionUpdate,
		Payload:   updatePayload,
	}

	// Send update command to node
	if err := h.agentHub.SendCommandToNode(n.ID(), cmd); err != nil {
		h.logger.Errorw("failed to send update command to node",
			"node_id", n.ID(),
			"sid", sid,
			"error", err,
		)

		if errors.Is(err, services.ErrNodeNotConnected) {
			utils.ErrorResponse(c, http.StatusConflict, "Node agent disconnected while processing request")
			return
		}
		if errors.Is(err, services.ErrSendChannelFull) {
			utils.ErrorResponse(c, http.StatusServiceUnavailable, "Node agent command queue is full, please try again later")
			return
		}

		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to send update command")
		return
	}

	h.logger.Infow("update command sent to node",
		"node_id", n.ID(),
		"sid", sid,
		"command_id", commandID,
		"target_version", releaseInfo.Version,
		"checksum", checksum,
	)

	response := &TriggerUpdateResponse{
		NodeID:        n.SID(),
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
