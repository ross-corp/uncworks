//go:build darwin

// update.go — Auto-update check for the UNCWORKS desktop app.
// Queries the GitHub Releases API for the latest stable or pre-release version.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Version is set via -ldflags at build time. Empty string means local/dev build.
var Version = ""

// UpdateInfo is returned by CheckForUpdate.
type UpdateInfo struct {
	// LocalBuild is true when the binary was not built from a release tag.
	LocalBuild bool `json:"localBuild"`
	// UpToDate is true when the installed version matches the latest available.
	UpToDate bool `json:"upToDate"`
	// CurrentVersion is the installed version (empty for local builds).
	CurrentVersion string `json:"currentVersion,omitempty"`
	// LatestVersion is the latest available version tag from GitHub Releases.
	LatestVersion string `json:"latestVersion,omitempty"`
	// ReleaseURL is the HTML URL of the latest GitHub Release.
	ReleaseURL string `json:"releaseURL,omitempty"`
	// Error is non-empty when the check failed (e.g. network error).
	Error string `json:"error,omitempty"`
}

var updateCache struct {
	mu   sync.Mutex
	info *UpdateInfo
}

// CheckForUpdate queries the GitHub Releases API for the latest version.
// Successful results are cached for the lifetime of the process; errors are
// not cached so that a transient network failure does not permanently suppress
// the update check.
// Exposed as a Wails binding.
func (a *App) CheckForUpdate() UpdateInfo {
	updateCache.mu.Lock()
	defer updateCache.mu.Unlock()

	if updateCache.info != nil {
		return *updateCache.info
	}

	result := checkForUpdate()
	if result.Error == "" {
		// Only cache successful checks.
		updateCache.info = &result
	}
	return result
}

func checkForUpdate() UpdateInfo {
	if Version == "" || Version == "dev" {
		return UpdateInfo{LocalBuild: true}
	}

	s, _ := loadAppSettings()
	channel := s.UpdateChannel
	if channel == "" {
		channel = "stable"
	}

	release, err := latestGitHubRelease("ross-corp", "uncworks", channel == "nightly")
	if err != nil {
		return UpdateInfo{LocalBuild: false, CurrentVersion: Version, Error: err.Error()}
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(Version, "v")

	return UpdateInfo{
		LocalBuild:     false,
		UpToDate:       current == latest,
		CurrentVersion: Version,
		LatestVersion:  release.TagName,
		ReleaseURL:     release.HTMLURL,
	}
}

type ghRelease struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
	HTMLURL    string `json:"html_url"`
}

func latestGitHubRelease(owner, repo string, includePrerelease bool) (*ghRelease, error) {
	client := &http.Client{Timeout: 8 * time.Second}

	// If we don't need pre-releases, use the /latest convenience endpoint.
	if !includePrerelease {
		url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Accept", "application/vnd.github+json")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		var rel ghRelease
		if err := json.Unmarshal(body, &rel); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}
		if rel.TagName == "" {
			return nil, fmt.Errorf("release has no tag_name")
		}
		return &rel, nil
	}

	// For nightly: fetch the first page of releases and return the newest one
	// (including pre-releases).
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=10", owner, repo)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var releases []ghRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}
	return &releases[0], nil
}
