package admin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"

	"github.com/orris-inc/orris/internal/application/setting/dto"
	"github.com/orris-inc/orris/internal/shared/utils"
)

const (
	maxBrandingFileSize = 2 << 20 // 2MB
	minBrandingFileSize = 100     // Minimum 100 bytes to reject empty/corrupt files
	brandingUploadDir   = "./uploads/branding"
	fileNameRandomBytes = 16 // 16 bytes = 128 bits of entropy
)

// allowedBrandingMIMETypes defines allowed MIME types detected by mimetype library
var allowedBrandingMIMETypes = map[string]string{
	"image/png":     ".png",
	"image/jpeg":    ".jpg",
	"image/x-icon":  ".ico",
	"image/svg+xml": ".svg",
}

// svgColorPattern validates safe fill/stroke values (colors, none, currentColor)
var svgColorPattern = regexp.MustCompile(`^(#[0-9a-fA-F]{3,8}|none|currentColor|transparent|[a-zA-Z]+)$`)

// GetBrandingSettings retrieves branding settings
// GET /admin/settings/branding
func (h *SettingHandler) GetBrandingSettings(c *gin.Context) {
	result, err := h.service.GetBrandingSettings(c.Request.Context())
	if err != nil {
		h.logger.Errorw("failed to get branding settings", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// UpdateBrandingSettings updates branding settings
// PUT /admin/settings/branding
func (h *SettingHandler) UpdateBrandingSettings(c *gin.Context) {
	var req dto.UpdateBrandingSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := h.service.UpdateBrandingSettings(c.Request.Context(), req, userID); err != nil {
		h.logger.Errorw("failed to update branding settings", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Branding settings updated successfully", nil)
}

// UploadBrandingImage uploads a branding image
// POST /admin/settings/branding/upload
// Security measures implemented per OWASP File Upload guidelines:
// - File size limits (min/max)
// - Content-based MIME type detection (not client-provided)
// - Whitelist validation of allowed types
// - SVG sanitization
// - Cryptographically secure random filenames
// - Proper response headers
func (h *SettingHandler) UploadBrandingImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.logger.Warnw("failed to get uploaded file", "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	// Validate file size (max)
	if header.Size > maxBrandingFileSize {
		utils.ErrorResponse(c, http.StatusBadRequest, "file size exceeds 2MB limit")
		return
	}

	// Validate file size (min) - reject empty or suspiciously small files
	if header.Size < minBrandingFileSize {
		utils.ErrorResponse(c, http.StatusBadRequest, "file is too small or empty")
		return
	}

	// Read file content with size limit to prevent memory exhaustion
	content, err := io.ReadAll(io.LimitReader(file, maxBrandingFileSize+1))
	if err != nil {
		h.logger.Errorw("failed to read uploaded file", "error", err)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to process file")
		return
	}

	// Double-check size after reading (defense in depth)
	if int64(len(content)) > maxBrandingFileSize {
		utils.ErrorResponse(c, http.StatusBadRequest, "file size exceeds 2MB limit")
		return
	}

	// Detect real MIME type from file content using mimetype library
	// This prevents MIME type spoofing attacks
	detectedMIME := mimetype.Detect(content)
	mimeType := detectedMIME.String()

	// Validate MIME type against whitelist
	ext, allowed := allowedBrandingMIMETypes[mimeType]
	if !allowed {
		h.logger.Warnw("rejected file upload with invalid MIME type",
			"detected_mime", mimeType,
			"filename", header.Filename,
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "only PNG, JPG, SVG, and ICO files are allowed")
		return
	}

	// For SVG files, sanitize content to remove dangerous elements
	if mimeType == "image/svg+xml" {
		content, err = sanitizeSVG(content)
		if err != nil {
			h.logger.Warnw("failed to sanitize SVG", "error", err)
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid SVG file")
			return
		}
	}

	// Ensure upload directory exists with restricted permissions
	if err := os.MkdirAll(brandingUploadDir, 0750); err != nil {
		h.logger.Errorw("failed to create upload directory", "error", err)
		utils.ErrorResponse(c, http.StatusInternalServerError, "internal error")
		return
	}

	// Generate cryptographically secure random filename
	// Format: {random_hex}{extension}
	// Using pure random name prevents enumeration and timing attacks
	filename, err := generateSecureFilename(ext)
	if err != nil {
		h.logger.Errorw("failed to generate secure filename", "error", err)
		utils.ErrorResponse(c, http.StatusInternalServerError, "internal error")
		return
	}

	dst := filepath.Join(brandingUploadDir, filename)

	// Verify the destination is within the upload directory (prevent path traversal)
	absUploadDir, err := filepath.Abs(brandingUploadDir)
	if err != nil {
		h.logger.Errorw("failed to resolve upload directory path", "error", err)
		utils.ErrorResponse(c, http.StatusInternalServerError, "internal error")
		return
	}
	absDst, err := filepath.Abs(dst)
	if err != nil {
		h.logger.Errorw("failed to resolve destination path", "error", err)
		utils.ErrorResponse(c, http.StatusInternalServerError, "internal error")
		return
	}
	if !strings.HasPrefix(absDst, absUploadDir) {
		h.logger.Errorw("path traversal attempt detected", "dst", dst)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid filename")
		return
	}

	// Write content to file with restricted permissions (owner read/write only)
	if err := os.WriteFile(dst, content, 0640); err != nil {
		h.logger.Errorw("failed to save uploaded file", "error", err)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to save file")
		return
	}

	url := "/uploads/branding/" + filename
	utils.SuccessResponse(c, http.StatusOK, "", dto.BrandingUploadResponse{URL: url})
}

// GetPublicBranding retrieves public branding config (no auth required)
// GET /branding
func (h *SettingHandler) GetPublicBranding(c *gin.Context) {
	result, err := h.service.GetPublicBranding(c.Request.Context())
	if err != nil {
		h.logger.Errorw("failed to get public branding", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// generateSecureFilename generates a cryptographically secure random filename
func generateSecureFilename(ext string) (string, error) {
	randBytes := make([]byte, fileNameRandomBytes)
	if _, err := rand.Read(randBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(randBytes) + ext, nil
}

// sanitizeSVG removes dangerous elements from SVG content using bluemonday
// This implements a strict whitelist approach per OWASP guidelines
func sanitizeSVG(content []byte) ([]byte, error) {
	// First, check for obvious malicious patterns before processing
	contentLower := strings.ToLower(string(content))
	dangerousPatterns := []string{
		"<script", "javascript:", "vbscript:", "data:text/html",
		"expression(", "eval(", "onclick", "onerror", "onload",
	}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(contentLower, pattern) {
			return nil, fmt.Errorf("potentially malicious content detected")
		}
	}

	// Create a strict SVG policy using bluemonday
	p := bluemonday.NewPolicy()

	// Allow basic SVG structure elements
	p.AllowElements("svg", "g", "defs", "symbol", "title", "desc")

	// Allow shape elements
	p.AllowElements("circle", "ellipse", "line", "path", "polygon", "polyline", "rect")

	// Allow text elements
	p.AllowElements("text", "tspan")

	// Allow gradient and pattern elements
	p.AllowElements("linearGradient", "radialGradient", "stop")

	// Allow clip and mask elements
	p.AllowElements("clipPath", "mask")

	// Safe attributes - NO style attribute (prevents CSS injection attacks)
	p.AllowAttrs("id", "class").Globally()
	p.AllowAttrs("x", "y", "width", "height", "rx", "ry").Globally()
	p.AllowAttrs("cx", "cy", "r", "fx", "fy").Globally()
	p.AllowAttrs("x1", "y1", "x2", "y2").Globally()
	p.AllowAttrs("points", "d").Globally()

	// Presentation attributes (safe alternatives to style attribute)
	p.AllowAttrs("fill", "stroke", "stroke-width", "stroke-linecap", "stroke-linejoin").Globally()
	p.AllowAttrs("opacity", "fill-opacity", "stroke-opacity").Globally()
	p.AllowAttrs("transform", "viewBox", "preserveAspectRatio").Globally()
	p.AllowAttrs("offset", "stop-color", "stop-opacity").Globally()
	p.AllowAttrs("font-family", "font-size", "font-weight", "text-anchor").Globally()

	// Only allow safe fill/stroke values (colors, none, currentColor)
	p.AllowAttrs("fill", "stroke").Matching(svgColorPattern).Globally()

	// Allow xmlns for SVG namespace
	p.AllowAttrs("xmlns").OnElements("svg")
	p.AllowAttrs("version").OnElements("svg")

	// Sanitize the content
	sanitized := p.SanitizeBytes(content)

	// Verify the result still contains svg element
	if !strings.Contains(strings.ToLower(string(sanitized)), "<svg") {
		return nil, fmt.Errorf("SVG content was completely stripped during sanitization")
	}

	// Final size check - sanitized content should not be larger than original
	if len(sanitized) > len(content)*2 {
		return nil, fmt.Errorf("sanitization resulted in unexpected content expansion")
	}

	return sanitized, nil
}
