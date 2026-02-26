package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/user/usecases"
)

// detectDeviceType detects device type from User-Agent
func detectDeviceType(userAgent string) string {
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		return "mobile"
	}
	if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		return "tablet"
	}
	return "web"
}

// getAllowedOriginsJS generates JavaScript array string of allowed origins
// SECURITY: Never returns '*' - requires explicit origin configuration
func (h *AuthHandler) getAllowedOriginsJS() string {
	if len(h.allowedOrigins) == 0 {
		// Log warning but return empty array instead of '*'
		// This will prevent postMessage from sending tokens
		h.logger.Errorw("SECURITY: allowed_origins not configured, OAuth callback will fail")
		return "" // Empty - will cause postMessage to fail safely
	}

	quoted := make([]string, len(h.allowedOrigins))
	for i, origin := range h.allowedOrigins {
		quoted[i] = fmt.Sprintf("'%s'", origin)
	}
	return strings.Join(quoted, ", ")
}

// renderOAuthSuccess renders HTML success page with postMessage
// Tokens are already set as HttpOnly cookies before this page renders.
// The page notifies the opener via postMessage (best-effort) and always
// attempts to close itself. If auto-close fails (COOP restrictions),
// a manual close button is shown as fallback.
func (h *AuthHandler) renderOAuthSuccess(c *gin.Context, result *usecases.HandleOAuthCallbackResult) {
	userJSON, _ := json.Marshal(result.User.GetDisplayInfo())

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login Successful</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            background: #fafafa;
            color: #111;
        }
        @media (prefers-color-scheme: dark) {
            body { background: #0a0a0a; color: #fafafa; }
            .card { background: #18181b; border-color: #27272a; }
            .hint { color: #a1a1aa; }
            .close-btn { background: #27272a; color: #fafafa; border-color: #3f3f46; }
            .close-btn:hover { background: #3f3f46; }
        }
        .card {
            text-align: center;
            padding: 48px 40px;
            background: #fff;
            border: 1px solid #e4e4e7;
            border-radius: 16px;
            max-width: 380px;
            width: 90%%;
        }
        .icon-wrap {
            width: 64px;
            height: 64px;
            margin: 0 auto 24px;
            border-radius: 50%%;
            background: #dcfce7;
            display: flex;
            align-items: center;
            justify-content: center;
            animation: scaleIn 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275);
        }
        @media (prefers-color-scheme: dark) {
            .icon-wrap { background: #14532d; }
        }
        .icon-wrap svg {
            width: 32px;
            height: 32px;
            color: #16a34a;
            animation: checkDraw 0.5s ease-out 0.2s both;
        }
        @keyframes scaleIn {
            from { transform: scale(0); opacity: 0; }
            to { transform: scale(1); opacity: 1; }
        }
        @keyframes checkDraw {
            from { stroke-dashoffset: 24; opacity: 0; }
            to { stroke-dashoffset: 0; opacity: 1; }
        }
        h1 {
            font-size: 20px;
            font-weight: 600;
            margin-bottom: 8px;
            letter-spacing: -0.01em;
        }
        .hint {
            font-size: 14px;
            color: #71717a;
            line-height: 1.5;
        }
        .spinner-row {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
            margin-top: 24px;
        }
        .dot-spinner {
            display: flex;
            gap: 4px;
        }
        .dot-spinner span {
            width: 6px;
            height: 6px;
            border-radius: 50%%;
            background: #a1a1aa;
            animation: dotPulse 1.2s ease-in-out infinite;
        }
        .dot-spinner span:nth-child(2) { animation-delay: 0.15s; }
        .dot-spinner span:nth-child(3) { animation-delay: 0.3s; }
        @keyframes dotPulse {
            0%%, 80%%, 100%% { opacity: 0.3; transform: scale(0.8); }
            40%% { opacity: 1; transform: scale(1); }
        }
        .spinner-text {
            font-size: 13px;
            color: #a1a1aa;
        }
        .fallback {
            display: none;
            margin-top: 24px;
        }
        .close-btn {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            height: 40px;
            padding: 0 20px;
            font-size: 14px;
            font-weight: 500;
            border-radius: 8px;
            border: 1px solid #e4e4e7;
            background: #fff;
            color: #111;
            cursor: pointer;
            transition: background 0.15s;
        }
        .close-btn:hover { background: #f4f4f5; }
    </style>
</head>
<body>
    <div class="card">
        <div class="icon-wrap">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="20 6 9 17 4 12" stroke-dasharray="24" />
            </svg>
        </div>
        <h1>Login Successful</h1>
        <p class="hint" id="auto-hint">This window will close automatically.</p>
        <div class="spinner-row" id="auto-spinner">
            <div class="dot-spinner"><span></span><span></span><span></span></div>
            <span class="spinner-text">Closing...</span>
        </div>
        <div class="fallback" id="fallback">
            <p class="hint" style="margin-bottom: 16px;">Could not close this window automatically.</p>
            <button class="close-btn" onclick="window.close()">Close Window</button>
        </div>
    </div>
    <script>
        // Best-effort: notify opener via postMessage (may fail due to COOP)
        if (window.opener) {
            var allowedOrigins = [%s];
            allowedOrigins.forEach(function(origin) {
                try {
                    window.opener.postMessage({
                        type: 'oauth_success',
                        user: %s
                    }, origin);
                } catch (e) {
                    // Cross-origin errors are expected when COOP is active
                }
            });
        }

        // Always attempt to close popup after a short delay.
        // Cookies are already set; the opener polls /auth/me to detect login.
        setTimeout(function() {
            try { window.close(); } catch (e) {}
        }, 1500);

        // Fallback: if window is still open after 3 seconds, show manual close button
        setTimeout(function() {
            if (!window.closed) {
                document.getElementById('auto-hint').style.display = 'none';
                document.getElementById('auto-spinner').style.display = 'none';
                document.getElementById('fallback').style.display = 'block';
            }
        }, 3000);
    </script>
</body>
</html>
    `,
		h.getAllowedOriginsJS(),
		string(userJSON),
	)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// renderOAuthError renders HTML error page with postMessage
func (h *AuthHandler) renderOAuthError(c *gin.Context, errorMsg string) {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login Failed</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            background: #fafafa;
            color: #111;
        }
        @media (prefers-color-scheme: dark) {
            body { background: #0a0a0a; color: #fafafa; }
            .card { background: #18181b; border-color: #27272a; }
            .hint { color: #a1a1aa; }
            .close-btn { background: #27272a; color: #fafafa; border-color: #3f3f46; }
            .close-btn:hover { background: #3f3f46; }
        }
        .card {
            text-align: center;
            padding: 48px 40px;
            background: #fff;
            border: 1px solid #e4e4e7;
            border-radius: 16px;
            max-width: 380px;
            width: 90%%;
        }
        .icon-wrap {
            width: 64px;
            height: 64px;
            margin: 0 auto 24px;
            border-radius: 50%%;
            background: #fee2e2;
            display: flex;
            align-items: center;
            justify-content: center;
            animation: scaleIn 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275);
        }
        @media (prefers-color-scheme: dark) {
            .icon-wrap { background: #450a0a; }
        }
        .icon-wrap svg {
            width: 32px;
            height: 32px;
            color: #dc2626;
        }
        @keyframes scaleIn {
            from { transform: scale(0); opacity: 0; }
            to { transform: scale(1); opacity: 1; }
        }
        h1 {
            font-size: 20px;
            font-weight: 600;
            margin-bottom: 8px;
            letter-spacing: -0.01em;
        }
        .hint {
            font-size: 14px;
            color: #71717a;
            line-height: 1.5;
            margin-bottom: 24px;
        }
        .close-btn {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            height: 40px;
            padding: 0 20px;
            font-size: 14px;
            font-weight: 500;
            border-radius: 8px;
            border: 1px solid #e4e4e7;
            background: #fff;
            color: #111;
            cursor: pointer;
            transition: background 0.15s;
        }
        .close-btn:hover { background: #f4f4f5; }
    </style>
</head>
<body>
    <div class="card">
        <div class="icon-wrap">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
        </div>
        <h1>Login Failed</h1>
        <p class="hint">%s</p>
        <button class="close-btn" onclick="window.close()">Close Window</button>
    </div>
    <script>
        // Best-effort: notify opener via postMessage
        if (window.opener) {
            var allowedOrigins = [%s];
            allowedOrigins.forEach(function(origin) {
                try {
                    window.opener.postMessage({
                        type: 'oauth_error',
                        error: %q
                    }, origin);
                } catch (e) {
                    // Cross-origin errors are expected
                }
            });
        }
    </script>
</body>
</html>
    `, errorMsg, h.getAllowedOriginsJS(), errorMsg)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
