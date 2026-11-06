// Package update implements the application ReleaseSource against GitHub's latest-release
// endpoint, ported from PigeonPost. The endpoint only ever reports a published, non-draft,
// non-prerelease release, so a tag pushed mid-development is structurally invisible: the guard is
// the endpoint's own contract, not a check here. The HTTP client is injected so tests never touch
// the network; the production constructor uses a short-timeout client because the check must never
// block for long.
//
// The repository is handed in as "owner/name" rather than written here: build.ps1 reads it from the
// build's own git remote, so the source names no particular account (CODE RULES 14). A build that
// carries none reports every check as unreachable.
package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"webvigil/internal/application"
)

// latestReleaseAPIFormat is GitHub's latest-release endpoint, completed with "owner/name".
const latestReleaseAPIFormat = "https://api.github.com/repos/%s/releases/latest"

// acceptHeader advertises a JSON client to the GitHub API.
const acceptHeader = "application/vnd.github+json"

// requestTimeout bounds the one network call the update check makes.
const requestTimeout = 5 * time.Second

// maxReleaseBytes caps what is read of the reply. A release description with every asset listed
// is a few tens of kilobytes; anything past this is not a release payload.
const maxReleaseBytes = 1 << 20

// ErrNoRepository reports a build that was not told which repository publishes it.
var ErrNoRepository = errors.New("this build names no release repository")

// Doer executes one HTTP request; *http.Client satisfies it and tests inject a fake.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// releasePayload is the subset of GitHub's latest-release JSON the check reads.
type releasePayload struct {
	TagName string         `json:"tag_name"`
	HTMLURL string         `json:"html_url"`
	Assets  []assetPayload `json:"assets"`
}

type assetPayload struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

// GitHubReleaseSource is an application.ReleaseSource backed by the GitHub API.
type GitHubReleaseSource struct {
	repo   string
	client Doer
}

// NewGitHubReleaseSource constructs the production source for "owner/name" with a short-timeout
// HTTP client.
func NewGitHubReleaseSource(repo string) *GitHubReleaseSource {
	return NewGitHubReleaseSourceWith(repo, &http.Client{Timeout: requestTimeout})
}

// NewGitHubReleaseSourceWith constructs a source for "owner/name" over the given client, for tests.
func NewGitHubReleaseSourceWith(repo string, client Doer) *GitHubReleaseSource {
	return &GitHubReleaseSource{repo: repo, client: client}
}

// APIURL is the endpoint this source asks.
func (s *GitHubReleaseSource) APIURL() string {
	return fmt.Sprintf(latestReleaseAPIFormat, s.repo)
}

// LatestRelease returns the latest published release; an error when it cannot be read. The caller
// (UpdateService.Check) treats any error as "no update", keeping the check silent.
func (s *GitHubReleaseSource) LatestRelease(ctx context.Context) (application.ReleaseInfo, error) {
	if s.repo == "" {
		return application.ReleaseInfo{}, ErrNoRepository
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.APIURL(), nil)
	if err != nil {
		return application.ReleaseInfo{}, fmt.Errorf("build release request: %w", err)
	}
	req.Header.Set("Accept", acceptHeader)
	resp, err := s.client.Do(req)
	if err != nil {
		return application.ReleaseInfo{}, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return application.ReleaseInfo{}, fmt.Errorf("fetch latest release: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxReleaseBytes))
	if err != nil {
		return application.ReleaseInfo{}, fmt.Errorf("read release body: %w", err)
	}
	var payload releasePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return application.ReleaseInfo{}, fmt.Errorf("parse release body: %w", err)
	}
	if payload.TagName == "" || payload.HTMLURL == "" {
		return application.ReleaseInfo{}, errors.New("release payload missing tag or page URL")
	}
	assets := make([]application.ReleaseAsset, 0, len(payload.Assets))
	for _, a := range payload.Assets {
		if a.Name != "" && a.DownloadURL != "" {
			assets = append(assets, application.ReleaseAsset{Name: a.Name, DownloadURL: a.DownloadURL})
		}
	}
	return application.ReleaseInfo{
		Version: payload.TagName,
		PageURL: payload.HTMLURL,
		Assets:  assets,
	}, nil
}
