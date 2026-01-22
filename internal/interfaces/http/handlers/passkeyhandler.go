package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"

	"github.com/orris-inc/orris/internal/application/user/usecases"
	"github.com/orris-inc/orris/internal/shared/config"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// PasskeyHandler handles passkey (WebAuthn) related HTTP requests
type PasskeyHandler struct {
	startRegistrationUC    *usecases.StartPasskeyRegistrationUseCase
	finishRegistrationUC   *usecases.FinishPasskeyRegistrationUseCase
	startAuthenticationUC  *usecases.StartPasskeyAuthenticationUseCase
	finishAuthenticationUC *usecases.FinishPasskeyAuthenticationUseCase
	listPasskeysUC         *usecases.ListUserPasskeysUseCase
	deletePasskeyUC        *usecases.DeletePasskeyUseCase
	logger                 logger.Interface
	cookieConfig           config.CookieConfig
	jwtConfig              config.JWTConfig
}

// getUserID safely extracts user ID from gin context
func (h *PasskeyHandler) getUserID(c *gin.Context) (uint, bool) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type in context", "type", fmt.Sprintf("%T", userIDVal))
		return 0, false
	}
	return userID, true
}

// NewPasskeyHandler creates a new PasskeyHandler
func NewPasskeyHandler(
	startRegistrationUC *usecases.StartPasskeyRegistrationUseCase,
	finishRegistrationUC *usecases.FinishPasskeyRegistrationUseCase,
	startAuthenticationUC *usecases.StartPasskeyAuthenticationUseCase,
	finishAuthenticationUC *usecases.FinishPasskeyAuthenticationUseCase,
	listPasskeysUC *usecases.ListUserPasskeysUseCase,
	deletePasskeyUC *usecases.DeletePasskeyUseCase,
	logger logger.Interface,
	cookieConfig config.CookieConfig,
	jwtConfig config.JWTConfig,
) *PasskeyHandler {
	return &PasskeyHandler{
		startRegistrationUC:    startRegistrationUC,
		finishRegistrationUC:   finishRegistrationUC,
		startAuthenticationUC:  startAuthenticationUC,
		finishAuthenticationUC: finishAuthenticationUC,
		listPasskeysUC:         listPasskeysUC,
		deletePasskeyUC:        deletePasskeyUC,
		logger:                 logger,
		cookieConfig:           cookieConfig,
		jwtConfig:              jwtConfig,
	}
}

// StartRegistrationRequest is the request body for starting passkey registration
type StartRegistrationRequest struct {
	// DeviceName is optional - a friendly name for the passkey
	DeviceName string `json:"device_name"`
}

// StartRegistration starts the passkey registration ceremony
// @Summary Start passkey registration
// @Description Starts the WebAuthn registration ceremony for the authenticated user
// @Tags Passkey
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} protocol.CredentialCreation
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /auth/passkey/register/start [post]
func (h *PasskeyHandler) StartRegistration(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	cmd := usecases.StartPasskeyRegistrationCommand{
		UserID: userID,
	}

	result, err := h.startRegistrationUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("failed to start passkey registration", "user_id", userID, "error", err)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to start passkey registration")
		return
	}

	// Return the credential creation options
	// The challenge is included in the options and will be used as the key in Redis
	c.JSON(http.StatusOK, result.Options)
}

// FinishRegistrationRequest is the request body for finishing passkey registration
type FinishRegistrationRequest struct {
	// WebAuthn credential creation response from the browser
	ID                      string                               `json:"id" binding:"required"`
	RawID                   string                               `json:"rawId" binding:"required"`
	Type                    string                               `json:"type" binding:"required"`
	Response                AuthenticatorAttestationResponseJSON `json:"response" binding:"required"`
	AuthenticatorAttachment string                               `json:"authenticatorAttachment,omitempty"`
	ClientExtensionResults  map[string]interface{}               `json:"clientExtensionResults,omitempty"`
	// DeviceName is optional - a friendly name for the passkey
	DeviceName string `json:"device_name,omitempty"`
}

// AuthenticatorAttestationResponseJSON represents the attestation response from the browser
type AuthenticatorAttestationResponseJSON struct {
	ClientDataJSON    string   `json:"clientDataJSON" binding:"required"`
	AttestationObject string   `json:"attestationObject" binding:"required"`
	Transports        []string `json:"transports,omitempty"`
}

// FinishRegistration completes the passkey registration ceremony
// @Summary Finish passkey registration
// @Description Completes the WebAuthn registration ceremony and stores the credential
// @Tags Passkey
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body FinishRegistrationRequest true "Credential creation response"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /auth/passkey/register/finish [post]
func (h *PasskeyHandler) FinishRegistration(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req FinishRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate and sanitize device name
	deviceName := h.sanitizeDeviceName(req.DeviceName)

	// Parse the credential creation response
	credResponse, err := h.parseCredentialCreationResponse(&req)
	if err != nil {
		h.logger.Errorw("failed to parse credential creation response", "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid credential response")
		return
	}

	// Parse to get the parsed data structure that includes challenge
	parsedResponse, err := credResponse.Parse()
	if err != nil {
		h.logger.Errorw("failed to parse credential response",
			"error", err,
			"attestation_object_len", len(credResponse.AttestationResponse.AttestationObject),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid client data")
		return
	}

	// Log attestation format for debugging
	h.logger.Infow("parsed attestation object",
		"user_id", userID,
		"format", parsedResponse.Response.AttestationObject.Format,
		"att_stmt_keys", getAttStmtKeys(parsedResponse.Response.AttestationObject.AttStatement),
	)

	cmd := usecases.FinishPasskeyRegistrationCommand{
		UserID:     userID,
		Challenge:  string(parsedResponse.Response.CollectedClientData.Challenge),
		Response:   parsedResponse,
		DeviceName: deviceName,
	}

	result, err := h.finishRegistrationUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("failed to finish passkey registration", "user_id", userID, "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, "failed to register passkey")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "passkey registered successfully", gin.H{
		"passkey": result.Credential.GetDisplayInfo(),
	})
}

// StartAuthenticationRequest is the request body for starting passkey authentication
type StartAuthenticationRequest struct {
	// Email is optional - if provided, only credentials for that user will be allowed
	// If empty, a discoverable credential flow will be used
	Email string `json:"email,omitempty"`
}

// StartAuthentication starts the passkey authentication ceremony
// @Summary Start passkey login
// @Description Starts the WebAuthn authentication ceremony
// @Tags Passkey
// @Accept json
// @Produce json
// @Param request body StartAuthenticationRequest false "Authentication options"
// @Success 200 {object} protocol.CredentialAssertion
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /auth/passkey/login/start [post]
func (h *PasskeyHandler) StartAuthentication(c *gin.Context) {
	var req StartAuthenticationRequest
	// Optional body, so ignore binding errors
	_ = c.ShouldBindJSON(&req)

	cmd := usecases.StartPasskeyAuthenticationCommand{
		Email: req.Email,
	}

	result, err := h.startAuthenticationUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("failed to start passkey authentication", "email", req.Email, "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, result.Options)
}

// FinishAuthenticationRequest is the request body for finishing passkey authentication
type FinishAuthenticationRequest struct {
	ID                      string                             `json:"id" binding:"required"`
	RawID                   string                             `json:"rawId" binding:"required"`
	Type                    string                             `json:"type" binding:"required"`
	Response                AuthenticatorAssertionResponseJSON `json:"response" binding:"required"`
	AuthenticatorAttachment string                             `json:"authenticatorAttachment,omitempty"`
	ClientExtensionResults  map[string]interface{}             `json:"clientExtensionResults,omitempty"`
}

// AuthenticatorAssertionResponseJSON represents the assertion response from the browser
type AuthenticatorAssertionResponseJSON struct {
	ClientDataJSON    string `json:"clientDataJSON" binding:"required"`
	AuthenticatorData string `json:"authenticatorData" binding:"required"`
	Signature         string `json:"signature" binding:"required"`
	UserHandle        string `json:"userHandle,omitempty"`
}

// FinishAuthentication completes the passkey authentication ceremony
// @Summary Finish passkey login
// @Description Completes the WebAuthn authentication ceremony and creates a session
// @Tags Passkey
// @Accept json
// @Produce json
// @Param request body FinishAuthenticationRequest true "Credential assertion response"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Router /auth/passkey/login/finish [post]
func (h *PasskeyHandler) FinishAuthentication(c *gin.Context) {
	var req FinishAuthenticationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Parse the credential assertion response
	credResponse, err := h.parseCredentialAssertionResponse(&req)
	if err != nil {
		h.logger.Errorw("failed to parse credential assertion response", "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid credential response")
		return
	}

	// Parse to get the parsed data structure
	parsedResponse, err := credResponse.Parse()
	if err != nil {
		h.logger.Errorw("failed to parse client data", "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid client data")
		return
	}

	cmd := usecases.FinishPasskeyAuthenticationCommand{
		Challenge:  string(parsedResponse.Response.CollectedClientData.Challenge),
		Response:   parsedResponse,
		DeviceName: c.GetHeader("User-Agent"),
		DeviceType: "web",
		IPAddress:  c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
	}

	result, err := h.finishAuthenticationUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("failed to finish passkey authentication", "error", err)
		utils.ErrorResponse(c, http.StatusUnauthorized, "authentication failed")
		return
	}

	// Set tokens in HttpOnly cookies
	accessMaxAge := h.jwtConfig.AccessExpMinutes * 60
	refreshMaxAge := h.jwtConfig.RefreshExpDays * 24 * 60 * 60
	utils.SetAuthCookies(c, h.cookieConfig, result.AccessToken, result.RefreshToken, accessMaxAge, refreshMaxAge)

	utils.SuccessResponse(c, http.StatusOK, "login successful", gin.H{
		"user":       result.User.GetDisplayInfo(),
		"expires_in": result.ExpiresIn,
	})
}

// ListPasskeys lists all passkeys for the authenticated user
// @Summary List user's passkeys
// @Description Lists all registered passkeys for the authenticated user
// @Tags Passkey
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /users/me/passkeys [get]
func (h *PasskeyHandler) ListPasskeys(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	cmd := usecases.ListUserPasskeysCommand{
		UserID: userID,
	}

	result, err := h.listPasskeysUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("failed to list passkeys", "user_id", userID, "error", err)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to list passkeys")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "success", gin.H{
		"passkeys": result.Passkeys,
	})
}

// DeletePasskey deletes a passkey for the authenticated user
// @Summary Delete a passkey
// @Description Deletes a specific passkey for the authenticated user
// @Tags Passkey
// @Produce json
// @Security BearerAuth
// @Param id path string true "Passkey ID (pk_xxx)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Router /users/me/passkeys/{id} [delete]
func (h *PasskeyHandler) DeletePasskey(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	passkeySID := c.Param("id")
	if passkeySID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "passkey ID is required")
		return
	}

	// Validate passkey ID format (pk_xxx)
	if err := id.ValidatePrefix(passkeySID, id.PrefixPasskeyCredential); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid passkey ID format")
		return
	}

	cmd := usecases.DeletePasskeyCommand{
		UserID:     userID,
		PasskeySID: passkeySID,
	}

	if err := h.deletePasskeyUC.Execute(c.Request.Context(), cmd); err != nil {
		h.logger.Errorw("failed to delete passkey", "user_id", userID, "passkey_sid", passkeySID, "error", err)
		if strings.Contains(err.Error(), "not found") {
			utils.ErrorResponse(c, http.StatusNotFound, "passkey not found")
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to delete passkey")
		}
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "passkey deleted successfully", nil)
}

// Helper methods

// getAttStmtKeys returns the keys from an attestation statement for debugging
func getAttStmtKeys(attStmt map[string]interface{}) []string {
	keys := make([]string, 0, len(attStmt))
	for k := range attStmt {
		keys = append(keys, k)
	}
	return keys
}

// maxDeviceNameLength is the maximum allowed length for device names
const maxDeviceNameLength = 100

// sanitizeDeviceName validates and sanitizes the device name input
func (h *PasskeyHandler) sanitizeDeviceName(name string) string {
	// Trim whitespace
	name = strings.TrimSpace(name)

	// Default name if empty
	if name == "" {
		return "Passkey"
	}

	// Ensure valid UTF-8
	if !utf8.ValidString(name) {
		return "Passkey"
	}

	// Remove control characters and null bytes, and truncate by rune count
	var result strings.Builder
	runeCount := 0
	for _, r := range name {
		if r >= 32 && r != 127 { // printable characters only
			if runeCount >= maxDeviceNameLength {
				break
			}
			result.WriteRune(r)
			runeCount++
		}
	}

	sanitized := result.String()
	if sanitized == "" {
		return "Passkey"
	}

	return sanitized
}

// decodeBase64 decodes base64 data supporting both standard and URL-safe encoding.
// WebAuthn spec requires base64url, but some browsers/libraries may use standard base64.
func decodeBase64(data string) ([]byte, error) {
	// Try base64url first (WebAuthn standard)
	if decoded, err := base64.RawURLEncoding.DecodeString(data); err == nil {
		return decoded, nil
	}
	// Try standard base64url with padding
	if decoded, err := base64.URLEncoding.DecodeString(data); err == nil {
		return decoded, nil
	}
	// Try standard base64 (with +/ instead of -_)
	if decoded, err := base64.StdEncoding.DecodeString(data); err == nil {
		return decoded, nil
	}
	// Try standard base64 without padding
	return base64.RawStdEncoding.DecodeString(data)
}

func (h *PasskeyHandler) parseCredentialCreationResponse(req *FinishRegistrationRequest) (*protocol.CredentialCreationResponse, error) {
	rawID, err := decodeBase64(req.RawID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode rawId: %w", err)
	}

	clientDataJSON, err := decodeBase64(req.Response.ClientDataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to decode clientDataJSON: %w", err)
	}

	attestationObject, err := decodeBase64(req.Response.AttestationObject)
	if err != nil {
		return nil, fmt.Errorf("failed to decode attestationObject: %w", err)
	}

	return &protocol.CredentialCreationResponse{
		PublicKeyCredential: protocol.PublicKeyCredential{
			Credential: protocol.Credential{
				ID:   req.ID,
				Type: req.Type,
			},
			RawID:                   rawID,
			ClientExtensionResults:  protocol.AuthenticationExtensionsClientOutputs(req.ClientExtensionResults),
			AuthenticatorAttachment: req.AuthenticatorAttachment,
		},
		AttestationResponse: protocol.AuthenticatorAttestationResponse{
			AuthenticatorResponse: protocol.AuthenticatorResponse{
				ClientDataJSON: clientDataJSON,
			},
			AttestationObject: attestationObject,
			Transports:        req.Response.Transports,
		},
	}, nil
}

func (h *PasskeyHandler) parseCredentialAssertionResponse(req *FinishAuthenticationRequest) (*protocol.CredentialAssertionResponse, error) {
	rawID, err := decodeBase64(req.RawID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode rawId: %w", err)
	}

	clientDataJSON, err := decodeBase64(req.Response.ClientDataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to decode clientDataJSON: %w", err)
	}

	authenticatorData, err := decodeBase64(req.Response.AuthenticatorData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode authenticatorData: %w", err)
	}

	signature, err := decodeBase64(req.Response.Signature)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	var userHandle []byte
	if req.Response.UserHandle != "" {
		userHandle, err = decodeBase64(req.Response.UserHandle)
		if err != nil {
			return nil, fmt.Errorf("failed to decode userHandle: %w", err)
		}
	}

	return &protocol.CredentialAssertionResponse{
		PublicKeyCredential: protocol.PublicKeyCredential{
			Credential: protocol.Credential{
				ID:   req.ID,
				Type: req.Type,
			},
			RawID:                   rawID,
			ClientExtensionResults:  protocol.AuthenticationExtensionsClientOutputs(req.ClientExtensionResults),
			AuthenticatorAttachment: req.AuthenticatorAttachment,
		},
		AssertionResponse: protocol.AuthenticatorAssertionResponse{
			AuthenticatorResponse: protocol.AuthenticatorResponse{
				ClientDataJSON: clientDataJSON,
			},
			AuthenticatorData: authenticatorData,
			Signature:         signature,
			UserHandle:        userHandle,
		},
	}, nil
}
