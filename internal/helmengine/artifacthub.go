package helmengine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

//nolint:gochecknoglobals // test override
var artifactHubAPIBase = "https://artifacthub.io/api/v1/packages/search"

//nolint:gochecknoglobals // dedicated client with timeout instead of http.DefaultClient
var artifactHubClient = &http.Client{Timeout: 30 * time.Second}

const defaultLimit = 25

// setArtifactHubAPIBase overrides the API base URL (used in tests).
func setArtifactHubAPIBase(u string) {
	artifactHubAPIBase = u
}

// artifactHubPackage represents the relevant fields from the Artifact Hub API response.
type artifactHubPackage struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	AppVersion  string `json:"app_version"`
	Description string `json:"description"`
	Repository  struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"repository"`
}

// artifactHubResponse wraps the API response.
type artifactHubResponse struct {
	Packages []artifactHubPackage `json:"packages"`
}

// SearchArtifactHub queries the Artifact Hub API for Helm charts matching the keyword.
// This is used by both v3 and v4 engines since the Helm SDK does not expose this functionality.
func SearchArtifactHub(ctx context.Context, opts *SearchHubOptions) ([]*SearchResult, error) {
	if opts.Keyword == "" {
		return nil, fmt.Errorf("keyword is required for search hub")
	}

	params := url.Values{}
	params.Set("ts_query_web", opts.Keyword)
	params.Set("kind", "0") // 0 = Helm charts
	params.Set("limit", strconv.Itoa(defaultLimit))
	params.Set("offset", "0")

	// Build a fixed URL from the constant base and user-controlled query params.
	// The base URL is hardcoded (not from user input), so this is safe from SSRF.
	reqURL := artifactHubAPIBase + "?" + params.Encode()

	log.Printf("searching Artifact Hub for %q", opts.Keyword)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := artifactHubClient.Do(req) //nolint:gosec // URL base is a hardcoded constant, not user-controlled
	if err != nil {
		return nil, fmt.Errorf("failed to query Artifact Hub: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("artifact Hub returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp artifactHubResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode Artifact Hub response: %w", err)
	}

	results := make([]*SearchResult, 0, len(apiResp.Packages))
	for _, pkg := range apiResp.Packages {
		name := pkg.Name
		if pkg.Repository.Name != "" {
			name = pkg.Repository.Name + "/" + pkg.Name
		}
		sr := &SearchResult{
			Name:         name,
			ChartVersion: pkg.Version,
			AppVersion:   pkg.AppVersion,
			Description:  pkg.Description,
		}
		if opts.ListRepoURL && pkg.Repository.URL != "" {
			sr.URL = pkg.Repository.URL
		}
		results = append(results, sr)
	}

	log.Printf("Artifact Hub returned %d results for %q", len(results), opts.Keyword)

	return results, nil
}
