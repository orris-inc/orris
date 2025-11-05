package constants

// OAuthErrorCode represents OAuth error codes
type OAuthErrorCode string

const (
	// OAuth provider errors (from callback)
	OAuthErrorAccessDenied       OAuthErrorCode = "access_denied"
	OAuthErrorInvalidRequest     OAuthErrorCode = "invalid_request"
	OAuthErrorUnauthorizedClient OAuthErrorCode = "unauthorized_client"
	OAuthErrorServerError        OAuthErrorCode = "server_error"

	// Internal errors
	OAuthErrorMissingCode        OAuthErrorCode = "missing_code"
	OAuthErrorMissingState       OAuthErrorCode = "missing_state"
	OAuthErrorInvalidState       OAuthErrorCode = "invalid_state"
	OAuthErrorExpiredState       OAuthErrorCode = "expired_state"
	OAuthErrorExchangeFailed     OAuthErrorCode = "exchange_failed"
	OAuthErrorUserInfoFailed     OAuthErrorCode = "userinfo_failed"
)

// OAuthErrorMessages maps error codes to user-friendly messages
var OAuthErrorMessages = map[OAuthErrorCode]string{
	OAuthErrorAccessDenied:       "You denied the authorization request. Please try again if you wish to continue.",
	OAuthErrorInvalidRequest:     "Invalid OAuth request. Please contact support if this persists.",
	OAuthErrorUnauthorizedClient: "OAuth application is not authorized. Please contact support.",
	OAuthErrorServerError:        "OAuth provider encountered an error. Please try again later.",

	OAuthErrorMissingCode:        "Authorization code is missing. Please try logging in again.",
	OAuthErrorMissingState:       "Security validation failed. Please try logging in again.",
	OAuthErrorInvalidState:       "Invalid security token. This link may have expired.",
	OAuthErrorExpiredState:       "Login session expired (10 minutes). Please try again.",
	OAuthErrorExchangeFailed:     "Failed to complete authentication. Please try again.",
	OAuthErrorUserInfoFailed:     "Failed to retrieve your profile information. Please try again.",
}

// GetOAuthErrorMessage returns a user-friendly error message
func GetOAuthErrorMessage(code OAuthErrorCode) string {
	if msg, ok := OAuthErrorMessages[code]; ok {
		return msg
	}
	return "An unexpected error occurred during authentication. Please try again."
}

// GetOAuthErrorMessageFromString returns a user-friendly error message from string
func GetOAuthErrorMessageFromString(code string) string {
	return GetOAuthErrorMessage(OAuthErrorCode(code))
}
