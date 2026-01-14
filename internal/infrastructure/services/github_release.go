// Package services provides infrastructure services.
package services

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/semver"
	"golang.org/x/sync/singleflight"

	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// Cache TTL for release info
	releaseCacheTTL = 1 * time.Hour

	// HTTP request timeout
	httpTimeout = 10 * time.Second

	// Cooldown period for force refresh to prevent DoS attacks
	forceRefreshCooldown = 10 * time.Second
)

// GitHubRepoConfig contains configuration for a GitHub repository.
type GitHubRepoConfig struct {
	Owner       string // Repository owner (e.g., "orris-inc")
	Repo        string // Repository name (e.g., "orris-client")
	AssetPrefix string // Asset name prefix (e.g., "orris-client")
}

// ReleaseInfo contains information about a GitHub release.
type ReleaseInfo struct {
	Version      string            `json:"version"`      // e.g., "v1.2.3"
	TagName      string            `json:"tag_name"`     // e.g., "v1.2.3"
	Assets       map[string]string `json:"assets"`       // platform-arch -> download_url
	ChecksumURL  string            `json:"checksum_url"` // URL to checksums.txt file
	ChecksumData map[string]string `json:"-"`            // Cached parsed checksums: platform-arch -> sha256
	PublishedAt  time.Time         `json:"published_at"` // Release publish time
}

// releaseCache holds cached release information.
type releaseCache struct {
	info      *ReleaseInfo
	expiresAt time.Time
}

// GitHubReleaseService fetches release information from GitHub.
type GitHubReleaseService struct {
	config           GitHubRepoConfig
	httpClient       *http.Client
	cache            *releaseCache
	cacheMu          sync.RWMutex
	lastForceRefresh time.Time          // Last force refresh time for cooldown
	fetchGroup       singleflight.Group // Prevents cache stampede on concurrent requests
	logger           logger.Interface
}

// NewGitHubReleaseService creates a new GitHubReleaseService with the given repository config.
func NewGitHubReleaseService(config GitHubRepoConfig, log logger.Interface) *GitHubReleaseService {
	return &GitHubReleaseService{
		config: config,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
		logger: log,
	}
}

// githubRelease represents the GitHub API response for a release.
type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

// githubAsset represents an asset in a GitHub release.
type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// GetLatestRelease fetches the latest release information from GitHub.
// Results are cached for 24 hours to avoid hitting GitHub API rate limits.
// Uses singleflight to prevent cache stampede when multiple goroutines request simultaneously.
func (s *GitHubReleaseService) GetLatestRelease(ctx context.Context) (*ReleaseInfo, error) {
	// Check cache first (fast path)
	s.cacheMu.RLock()
	if s.cache != nil && time.Now().Before(s.cache.expiresAt) {
		info := s.cache.info
		s.cacheMu.RUnlock()
		return info, nil
	}
	s.cacheMu.RUnlock()

	// Use singleflight to ensure only one goroutine fetches from GitHub
	// when cache is expired, preventing cache stampede
	result, err, _ := s.fetchGroup.Do("latest_release", func() (any, error) {
		// Double-check cache inside singleflight (another goroutine might have updated it)
		s.cacheMu.RLock()
		if s.cache != nil && time.Now().Before(s.cache.expiresAt) {
			info := s.cache.info
			s.cacheMu.RUnlock()
			return info, nil
		}
		s.cacheMu.RUnlock()

		return s.fetchFromGitHub(ctx)
	})

	if err != nil {
		return nil, err
	}

	return result.(*ReleaseInfo), nil
}

// fetchFromGitHub fetches release information from GitHub API.
func (s *GitHubReleaseService) fetchFromGitHub(ctx context.Context) (*ReleaseInfo, error) {
	// Build GitHub API URL from config
	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", s.config.Owner, s.config.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "orris-backend")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Parse assets to extract platform-arch -> download_url mapping and checksums.txt URL
	assets, checksumURL := s.parseAssets(release.Assets)

	// Remove leading 'v' prefix from version for consistency
	// GitHub tags are typically "v1.2.3", we normalize to "1.2.3"
	version := strings.TrimPrefix(release.TagName, "v")

	info := &ReleaseInfo{
		Version:     version,
		TagName:     release.TagName,
		Assets:      assets,
		ChecksumURL: checksumURL,
		PublishedAt: release.PublishedAt,
	}

	// Update cache
	s.cacheMu.Lock()
	s.cache = &releaseCache{
		info:      info,
		expiresAt: time.Now().Add(releaseCacheTTL),
	}
	s.cacheMu.Unlock()

	s.logger.Debugw("fetched latest release from GitHub",
		"version", info.Version,
		"assets_count", len(info.Assets),
	)

	return info, nil
}

// parseAssets converts GitHub assets to platform-arch -> download_url mapping and finds checksums.txt URL.
// Expected asset name format: {assetPrefix}-{os}-{arch}
// Example: orris-client-linux-amd64, orrisp-linux-arm64
// Checksum file: checksums.txt (contains all platform checksums in one file)
func (s *GitHubReleaseService) parseAssets(assets []githubAsset) (binaries map[string]string, checksumURL string) {
	binaries = make(map[string]string)

	for _, asset := range assets {
		name := asset.Name

		// Check for checksums.txt file
		if name == "checksums.txt" {
			checksumURL = asset.BrowserDownloadURL
			continue
		}

		// Skip source code archives and other non-relevant files
		if strings.HasSuffix(name, ".md5") ||
			strings.HasSuffix(name, ".txt") ||
			strings.HasSuffix(name, ".sha256") ||
			strings.HasSuffix(name, ".zip") ||
			strings.HasSuffix(name, ".tar.gz") {
			continue
		}

		// Handle binary files
		// Extract os-arch from name using configured asset prefix
		// Format: {prefix}-{os}-{arch} -> {os}-{arch}
		prefix := s.config.AssetPrefix + "-"
		if strings.HasPrefix(name, prefix) {
			suffix := strings.TrimPrefix(name, prefix)
			parts := strings.Split(suffix, "-")
			if len(parts) >= 2 {
				os := parts[0]
				arch := parts[1]
				key := fmt.Sprintf("%s-%s", os, arch)
				binaries[key] = asset.BrowserDownloadURL
			}
		}
	}

	return binaries, checksumURL
}

// GetLatestReleaseWithVersionCheck fetches the latest release, automatically refreshing cache
// if currentVersion >= cached version (indicating cache may be stale).
// This provides smart cache invalidation without manual intervention.
// Includes cooldown protection to prevent DoS attacks via frequent refresh requests.
func (s *GitHubReleaseService) GetLatestReleaseWithVersionCheck(ctx context.Context, currentVersion string) (*ReleaseInfo, error) {
	info, err := s.GetLatestRelease(ctx)
	if err != nil {
		return nil, err
	}

	// If current version >= cached version, cache may be stale, consider force refresh
	if currentVersion != "" && !s.hasNewerVersion(currentVersion, info.Version) {
		// Atomically check and update cooldown to prevent race condition (TOCTOU)
		shouldRefresh := false
		s.cacheMu.Lock()
		if time.Since(s.lastForceRefresh) >= forceRefreshCooldown {
			s.lastForceRefresh = time.Now()
			shouldRefresh = true
		}
		s.cacheMu.Unlock()

		if !shouldRefresh {
			s.logger.Debugw("force refresh skipped due to cooldown",
				"current_version", currentVersion,
				"cached_version", info.Version,
			)
			return info, nil
		}

		s.logger.Debugw("cache may be stale, refreshing release info",
			"current_version", currentVersion,
			"cached_version", info.Version,
		)

		s.InvalidateCache()
		return s.GetLatestRelease(ctx)
	}

	return info, nil
}

// hasNewerVersion checks if latestVersion is newer than currentVersion using semver.
func (s *GitHubReleaseService) hasNewerVersion(currentVersion, latestVersion string) bool {
	if latestVersion == "" {
		return false
	}
	if currentVersion == "" || currentVersion == "dev" {
		return true
	}

	current := s.normalizeVersion(currentVersion)
	latest := s.normalizeVersion(latestVersion)

	if !semver.IsValid(current) {
		return true
	}
	if !semver.IsValid(latest) {
		return false
	}

	return semver.Compare(current, latest) < 0
}

// normalizeVersion ensures version string has "v" prefix for semver compatibility.
func (s *GitHubReleaseService) normalizeVersion(version string) string {
	if version == "" {
		return ""
	}
	version = strings.TrimSpace(version)
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

// GetDownloadURL returns the download URL for a specific platform and architecture.
func (s *GitHubReleaseService) GetDownloadURL(ctx context.Context, platform, arch string) (string, error) {
	info, err := s.GetLatestRelease(ctx)
	if err != nil {
		return "", err
	}

	key := fmt.Sprintf("%s-%s", platform, arch)
	url, ok := info.Assets[key]
	if !ok {
		return "", fmt.Errorf("no asset found for %s", key)
	}

	return url, nil
}

// GetVersion returns the latest version string.
func (s *GitHubReleaseService) GetVersion(ctx context.Context) (string, error) {
	info, err := s.GetLatestRelease(ctx)
	if err != nil {
		return "", err
	}
	return info.Version, nil
}

// GetChecksum fetches and returns the SHA256 checksum for a specific platform and architecture.
// It downloads the checksums.txt file and extracts the hash for the specified platform.
// Format of checksums.txt: "sha256hash  orris-client-{os}-{arch}" per line
func (s *GitHubReleaseService) GetChecksum(ctx context.Context, platform, arch string) (string, error) {
	info, err := s.GetLatestRelease(ctx)
	if err != nil {
		return "", err
	}

	if info.ChecksumURL == "" {
		return "", fmt.Errorf("no checksums.txt found in release")
	}

	key := fmt.Sprintf("%s-%s", platform, arch)

	// Check if we already have parsed checksum data cached
	if info.ChecksumData != nil {
		if checksum, ok := info.ChecksumData[key]; ok {
			return checksum, nil
		}
		return "", fmt.Errorf("no checksum found for %s", key)
	}

	// Fetch and parse the checksums.txt file
	checksumData, err := s.fetchAndParseChecksums(ctx, info.ChecksumURL)
	if err != nil {
		return "", err
	}

	// Cache the parsed data (update in cache)
	s.cacheMu.Lock()
	if s.cache != nil && s.cache.info == info {
		s.cache.info.ChecksumData = checksumData
	}
	s.cacheMu.Unlock()

	if checksum, ok := checksumData[key]; ok {
		return checksum, nil
	}

	return "", fmt.Errorf("no checksum found for %s", key)
}

// fetchAndParseChecksums downloads and parses the checksums.txt file.
// Returns a map of platform-arch -> sha256 checksum.
func (s *GitHubReleaseService) fetchAndParseChecksums(ctx context.Context, checksumURL string) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create checksum request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch checksum: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected checksum status code: %d", resp.StatusCode)
	}

	// Read checksums.txt content
	// Limit read to 8KB to prevent memory issues (should be plenty for a few checksums)
	content, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return nil, fmt.Errorf("read checksum: %w", err)
	}

	// Parse checksums.txt
	// Format: "sha256hash  orris-client-{os}-{arch}" per line
	checksumData := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: "sha256hash  filename" (two spaces between hash and filename)
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		checksum := parts[0]
		filename := parts[1]

		// Validate SHA256 format (64 hex characters)
		if len(checksum) != 64 {
			continue
		}
		if _, err := hex.DecodeString(checksum); err != nil {
			continue
		}

		// Extract platform-arch from filename using configured asset prefix
		// Format: {prefix}-{os}-{arch} -> {os}-{arch}
		prefix := s.config.AssetPrefix + "-"
		if strings.HasPrefix(filename, prefix) {
			suffix := strings.TrimPrefix(filename, prefix)
			parts := strings.Split(suffix, "-")
			if len(parts) >= 2 {
				os := parts[0]
				arch := parts[1]
				key := fmt.Sprintf("%s-%s", os, arch)
				checksumData[key] = checksum
			}
		}
	}

	if len(checksumData) == 0 {
		return nil, fmt.Errorf("no valid checksums found in checksums.txt")
	}

	return checksumData, nil
}

// InvalidateCache clears the cached release information.
func (s *GitHubReleaseService) InvalidateCache() {
	s.cacheMu.Lock()
	s.cache = nil
	s.cacheMu.Unlock()
}
