package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"orris/internal/application/user/usecases"
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
func (h *AuthHandler) getAllowedOriginsJS() string {
	if len(h.allowedOrigins) == 0 {
		return "'*'" // fallback
	}

	quoted := make([]string, len(h.allowedOrigins))
	for i, origin := range h.allowedOrigins {
		quoted[i] = fmt.Sprintf("'%s'", origin)
	}
	return strings.Join(quoted, ", ")
}

// renderOAuthSuccess renders HTML success page with postMessage
func (h *AuthHandler) renderOAuthSuccess(c *gin.Context, result *usecases.HandleOAuthCallbackResult) {
	userJSON, _ := json.Marshal(result.User.GetDisplayInfo())

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>OAuth Login Successful</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
        }
        .container {
            text-align: center;
            color: white;
        }
        .success-icon {
            font-size: 48px;
            margin-bottom: 20px;
        }
        h1 {
            margin: 0 0 20px 0;
            font-size: 24px;
        }
        .spinner {
            border: 4px solid rgba(255,255,255,0.3);
            border-top: 4px solid white;
            border-radius: 50%%;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 20px auto;
        }
        @keyframes spin {
            0%% { transform: rotate(0deg); }
            100%% { transform: rotate(360deg); }
        }
        p {
            margin: 10px 0;
            font-size: 16px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="success-icon">✅</div>
        <h1>Login Successful</h1>
        <div class="spinner"></div>
        <p>Redirecting...</p>
    </div>
    <script>
        const data = {
            type: 'oauth_success',
            access_token: %q,
            refresh_token: %q,
            token_type: 'Bearer',
            expires_in: %d,
            user: %s
        };

        if (window.opener) {
            // Send message to all allowed origins
            const allowedOrigins = [%s];
            allowedOrigins.forEach(origin => {
                try {
                    window.opener.postMessage(data, origin);
                } catch (e) {
                    // Ignore cross-origin errors
                }
            });

            // Close popup after delay
            setTimeout(() => window.close(), 1000);
        } else {
            // Fallback: redirect to frontend callback URL
            const params = new URLSearchParams({
                access_token: data.access_token,
                refresh_token: data.refresh_token,
                token_type: data.token_type,
                expires_in: data.expires_in.toString()
            });
            window.location.href = '%s?' + params.toString();
        }
    </script>
</body>
</html>
    `,
		result.AccessToken,
		result.RefreshToken,
		result.ExpiresIn,
		string(userJSON),
		h.getAllowedOriginsJS(),
		h.frontendCallbackURL,
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
    <title>OAuth Login Failed</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: #f44336;
        }
        .container {
            text-align: center;
            color: white;
            padding: 40px;
            background: rgba(0,0,0,0.2);
            border-radius: 8px;
            max-width: 500px;
        }
        .error-icon {
            font-size: 48px;
            margin-bottom: 20px;
        }
        h1 {
            margin: 0 0 20px 0;
            font-size: 24px;
        }
        p {
            margin: 0 0 30px 0;
            line-height: 1.5;
            font-size: 16px;
        }
        button {
            margin-top: 20px;
            padding: 12px 24px;
            cursor: pointer;
            background: white;
            border: none;
            border-radius: 4px;
            font-size: 14px;
            font-weight: 500;
            color: #f44336;
            transition: background-color 0.2s;
        }
        button:hover {
            background: #f0f0f0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="error-icon">❌</div>
        <h1>Login Failed</h1>
        <p>%s</p>
        <button onclick="window.close()">Close Window</button>
    </div>
    <script>
        if (window.opener) {
            const allowedOrigins = [%s];
            allowedOrigins.forEach(origin => {
                try {
                    window.opener.postMessage({
                        type: 'oauth_error',
                        error: %q
                    }, origin);
                } catch (e) {
                    // Ignore cross-origin errors
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
