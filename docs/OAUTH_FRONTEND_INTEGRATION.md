# OAuth Frontend Integration Guide

## Overview

This document provides a secure implementation guide for integrating OAuth authentication with the Orris backend API.

## Security Model

The OAuth flow uses `postMessage` for secure cross-window communication:

1. Frontend opens OAuth in a popup window
2. Backend redirects to OAuth provider (Google/GitHub)
3. Provider redirects back to backend callback
4. Backend sends tokens via `postMessage` to the opener window
5. Frontend validates origin and processes tokens

## Frontend Implementation

### 1. Initiate OAuth Login (React/Vue/Vanilla JS)

```typescript
// OAuth utilities
const BACKEND_URL = 'http://localhost:8081';
const ALLOWED_BACKEND_ORIGINS = ['http://localhost:8081'];

function openOAuthPopup(provider: 'google' | 'github'): Window | null {
  const width = 500;
  const height = 600;
  const left = (screen.width - width) / 2;
  const top = (screen.height - height) / 2;

  const popup = window.open(
    `${BACKEND_URL}/auth/oauth/${provider}`,
    `oauth_${provider}`,
    `width=${width},height=${height},left=${left},top=${top},resizable,scrollbars`
  );

  if (!popup) {
    console.error('Popup blocked. Please allow popups for OAuth.');
    return null;
  }

  return popup;
}
```

### 2. Listen for OAuth Response (CRITICAL: Validate Origin)

```typescript
interface OAuthSuccessMessage {
  type: 'oauth_success';
  access_token: string;
  refresh_token: string;
  token_type: 'Bearer';
  expires_in: number;
  user: {
    id: number;
    email: string;
    name: string;
    // ... other user fields
  };
}

interface OAuthErrorMessage {
  type: 'oauth_error';
  error: string;
}

type OAuthMessage = OAuthSuccessMessage | OAuthErrorMessage;

function setupOAuthListener(
  onSuccess: (data: OAuthSuccessMessage) => void,
  onError: (error: string) => void
): () => void {
  const handleMessage = (event: MessageEvent<OAuthMessage>) => {
    // ⚠️ CRITICAL SECURITY CHECK: Validate origin
    if (!ALLOWED_BACKEND_ORIGINS.includes(event.origin)) {
      console.error('Rejected postMessage from untrusted origin:', event.origin);
      return;
    }

    // Validate message structure
    if (!event.data || typeof event.data !== 'object' || !('type' in event.data)) {
      console.warn('Invalid message format:', event.data);
      return;
    }

    // Handle success response
    if (event.data.type === 'oauth_success') {
      console.log('OAuth login successful');
      onSuccess(event.data);
    }
    // Handle error response
    else if (event.data.type === 'oauth_error') {
      console.error('OAuth login failed:', event.data.error);
      onError(event.data.error);
    }
  };

  window.addEventListener('message', handleMessage);

  // Return cleanup function
  return () => window.removeEventListener('message', handleMessage);
}
```

### 3. Complete OAuth Flow Example

```typescript
// Example: React Hook
import { useEffect, useState } from 'react';

interface OAuthUser {
  id: number;
  email: string;
  name: string;
}

interface OAuthResult {
  user: OAuthUser;
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

function useOAuth() {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const login = (provider: 'google' | 'github') => {
    setIsLoading(true);
    setError(null);

    // Open OAuth popup
    const popup = openOAuthPopup(provider);
    if (!popup) {
      setError('Failed to open OAuth popup. Please allow popups.');
      setIsLoading(false);
      return;
    }

    // Setup message listener
    const cleanup = setupOAuthListener(
      (data) => {
        // Store tokens
        localStorage.setItem('access_token', data.access_token);
        localStorage.setItem('refresh_token', data.refresh_token);
        localStorage.setItem('token_expiry', String(Date.now() + data.expires_in * 1000));
        localStorage.setItem('user', JSON.stringify(data.user));

        setIsLoading(false);
        setError(null);

        // Redirect or update app state
        window.location.href = '/dashboard';
      },
      (errorMsg) => {
        setError(errorMsg);
        setIsLoading(false);
      }
    );

    // Cleanup on unmount
    return cleanup;
  };

  return { login, isLoading, error };
}

// Usage in component
function LoginPage() {
  const { login, isLoading, error } = useOAuth();

  return (
    <div>
      <h1>Login</h1>
      {error && <div className="error">{error}</div>}
      <button onClick={() => login('google')} disabled={isLoading}>
        {isLoading ? 'Logging in...' : 'Login with Google'}
      </button>
      <button onClick={() => login('github')} disabled={isLoading}>
        {isLoading ? 'Logging in...' : 'Login with GitHub'}
      </button>
    </div>
  );
}
```

### 4. Fallback: Handle Direct Redirect (Non-Popup Mode)

If the user's browser blocks popups, the backend will redirect to `frontend_callback_url` with query parameters:

```typescript
// /oauth/callback route handler
import { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';

function OAuthCallbackPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();

  useEffect(() => {
    const accessToken = searchParams.get('access_token');
    const refreshToken = searchParams.get('refresh_token');
    const expiresIn = searchParams.get('expires_in');

    if (accessToken && refreshToken) {
      // Store tokens
      localStorage.setItem('access_token', accessToken);
      localStorage.setItem('refresh_token', refreshToken);
      if (expiresIn) {
        localStorage.setItem('token_expiry', String(Date.now() + parseInt(expiresIn) * 1000));
      }

      // Fetch user info
      fetch(`${BACKEND_URL}/auth/me`, {
        headers: {
          Authorization: `Bearer ${accessToken}`,
        },
      })
        .then((res) => res.json())
        .then((data) => {
          if (data.data) {
            localStorage.setItem('user', JSON.stringify(data.data));
          }
          navigate('/dashboard');
        })
        .catch((err) => {
          console.error('Failed to fetch user info:', err);
          navigate('/login?error=auth_failed');
        });
    } else {
      navigate('/login?error=missing_tokens');
    }
  }, [searchParams, navigate]);

  return <div>Processing OAuth callback...</div>;
}
```

## Configuration Requirements

### Backend (config.yaml)

```yaml
server:
  allowed_origins:
    - "http://localhost:3000"  # ⚠️ REQUIRED: Frontend origin
  frontend_callback_url: "http://localhost:3000/oauth/callback"  # Fallback redirect URL
```

### Frontend Environment Variables

```env
VITE_BACKEND_URL=http://localhost:8081
VITE_ALLOWED_BACKEND_ORIGINS=http://localhost:8081
```

## Security Checklist

### ✅ Backend (Already Implemented)

- [x] Specify exact `targetOrigin` (never `*`)
- [x] Send messages only to configured `allowed_origins`
- [x] Use HTTPS in production
- [x] Validate OAuth state parameter (CSRF protection)
- [x] Handle OAuth provider errors

### ✅ Frontend (Must Implement)

- [x] **Validate `event.origin`** before processing messages
- [x] Check message structure and type
- [x] Store tokens securely (consider httpOnly cookies for production)
- [x] Implement token refresh logic
- [x] Handle popup blockers gracefully
- [x] Use HTTPS in production

## Common Issues

### Issue: "Popup blocked"
**Solution**: Use a fallback button that opens OAuth in the same tab, or detect popup blocking and show instructions.

### Issue: "No message received"
**Checks**:
1. Verify `allowed_origins` in backend config matches frontend origin
2. Check browser console for CORS or CSP errors
3. Ensure popup is not blocked

### Issue: "Rejected postMessage from untrusted origin"
**Solution**: Update `ALLOWED_BACKEND_ORIGINS` in frontend to match backend URL

## Production Deployment

### Backend Configuration

```yaml
server:
  allowed_origins:
    - "https://app.yourdomain.com"
    - "https://yourdomain.com"
  frontend_callback_url: "https://app.yourdomain.com/oauth/callback"

oauth:
  google:
    redirect_url: "https://api.yourdomain.com/auth/oauth/google/callback"
  github:
    redirect_url: "https://api.yourdomain.com/auth/oauth/github/callback"
```

### Frontend Configuration

```typescript
const BACKEND_URL = 'https://api.yourdomain.com';
const ALLOWED_BACKEND_ORIGINS = ['https://api.yourdomain.com'];
```

## Testing

### Manual Testing Checklist

1. [ ] OAuth popup opens successfully
2. [ ] Redirects to OAuth provider (Google/GitHub)
3. [ ] After authorization, popup shows "Login Successful"
4. [ ] Popup closes automatically
5. [ ] Tokens received via postMessage
6. [ ] Tokens stored in localStorage
7. [ ] User redirected to dashboard
8. [ ] Token refresh works correctly

### Test with Popup Blocking

1. Block popups in browser settings
2. Click OAuth button
3. Verify fallback redirect to `/oauth/callback` works
4. Verify tokens are stored
5. Verify user is redirected to dashboard

## Additional Resources

- [MDN: Window.postMessage()](https://developer.mozilla.org/en-US/docs/Web/API/Window/postMessage)
- [OAuth 2.0 Security Best Practices](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-security-topics)
- [OWASP: Cross-Site Request Forgery Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
