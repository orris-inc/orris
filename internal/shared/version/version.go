// Package version provides utilities for semantic version comparison.
package version

import (
	"strings"

	"golang.org/x/mod/semver"
)

// Normalize ensures version string has "v" prefix for semver compatibility.
// Examples: "1.2.3" -> "v1.2.3", "v1.2.3" -> "v1.2.3"
func Normalize(version string) string {
	if version == "" {
		return ""
	}
	version = strings.TrimSpace(version)
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

// HasNewerVersion checks if latestVersion is newer than currentVersion using semver.
// Returns true if an update is available.
func HasNewerVersion(currentVersion, latestVersion string) bool {
	// If latest version is unknown, no update available
	if latestVersion == "" {
		return false
	}

	// If current version is empty or "dev", always suggest update
	if currentVersion == "" || currentVersion == "dev" {
		return true
	}

	current := Normalize(currentVersion)
	latest := Normalize(latestVersion)

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
