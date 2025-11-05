package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type GitHubOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type GitHubOAuthClient struct {
	config *oauth2.Config
}

type githubUserInfo struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Login     string `json:"login"`
}

type githubEmail struct {
	Email      string `json:"email"`
	Primary    bool   `json:"primary"`
	Verified   bool   `json:"verified"`
	Visibility string `json:"visibility"`
}

func NewGitHubOAuthClient(cfg GitHubOAuthConfig) *GitHubOAuthClient {
	return &GitHubOAuthClient{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		},
	}
}

func (c *GitHubOAuthClient) GetAuthURL(state string) (string, string, error) {
	codeVerifier, codeChallenge, err := generatePKCEParams()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate PKCE parameters: %w", err)
	}

	authURL := c.config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	return authURL, codeVerifier, nil
}

func (c *GitHubOAuthClient) ExchangeCode(ctx context.Context, code string, codeVerifier string) (string, error) {
	token, err := c.config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		return "", fmt.Errorf("failed to exchange code: %w", err)
	}
	return token.AccessToken, nil
}

func (c *GitHubOAuthClient) GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	userInfo, err := c.fetchUserInfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	if userInfo.Email == "" {
		email, verified, err := c.fetchPrimaryEmail(ctx, accessToken)
		if err != nil {
			return nil, err
		}
		userInfo.Email = email
		return &OAuthUserInfo{
			Email:         userInfo.Email,
			Name:          userInfo.Name,
			Picture:       userInfo.AvatarURL,
			EmailVerified: verified,
			Provider:      "github",
			ProviderID:    strconv.Itoa(userInfo.ID),
		}, nil
	}

	return &OAuthUserInfo{
		Email:         userInfo.Email,
		Name:          userInfo.Name,
		Picture:       userInfo.AvatarURL,
		EmailVerified: true,
		Provider:      "github",
		ProviderID:    strconv.Itoa(userInfo.ID),
	}, nil
}

func (c *GitHubOAuthClient) fetchUserInfo(ctx context.Context, accessToken string) (*githubUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: status %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var gInfo githubUserInfo
	if err := json.Unmarshal(body, &gInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	return &gInfo, nil
}

func (c *GitHubOAuthClient) fetchPrimaryEmail(ctx context.Context, accessToken string) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("failed to get user emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", false, fmt.Errorf("failed to get user emails: status %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("failed to read response body: %w", err)
	}

	var emails []githubEmail
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", false, fmt.Errorf("failed to unmarshal emails: %w", err)
	}

	for _, email := range emails {
		if email.Primary {
			return email.Email, email.Verified, nil
		}
	}

	if len(emails) > 0 {
		return emails[0].Email, emails[0].Verified, nil
	}

	return "", false, fmt.Errorf("no email found")
}
