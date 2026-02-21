package helmengine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const (
	artifactHubAPIBase = "https://artifacthub.io/api/v1/packages/search"
	defaultLimit       = 25
)

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
	limit := defaultLimit
	if opts.MaxColWidth > 0 {
		// MaxColWidth is a display hint, not a result limit, but we use a reasonable default.
		limit = defaultLimit
	}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", "0")

	reqURL := artifactHubAPIBase + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query Artifact Hub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Artifact Hub returned status %d: %s", resp.StatusCode, string(body))
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

	return results, nil
}
