package v3

import (
	"context"
	"fmt"
	"strings"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

func (e *V3Engine) SearchHub(_ context.Context, opts *helmengine.SearchHubOptions) ([]*helmengine.SearchResult, error) {
	// The Helm v3 SDK doesn't expose a direct API for Artifact Hub search.
	// The CLI uses an HTTP client to query https://hub.helm.sh/api/chartsvc/v1/charts/search
	// We'll make the HTTP call directly.
	return nil, fmt.Errorf("search hub is not directly supported via the Helm v3 SDK; use the Artifact Hub API at https://artifacthub.io/api/v1/packages/search")
}

func (e *V3Engine) SearchRepo(_ context.Context, opts *helmengine.SearchRepoOptions) ([]*helmengine.SearchResult, error) {
	settings := cli.New()

	f, err := repo.LoadFile(settings.RepositoryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load repository file: %w", err)
	}

	var results []*helmengine.SearchResult

	for _, r := range f.Repositories {
		indexPath := fmt.Sprintf("%s/%s-index.yaml", settings.RepositoryCache, r.Name)
		idx, err := repo.LoadIndexFile(indexPath)
		if err != nil {
			continue
		}

		for name, entries := range idx.Entries {
			if len(entries) == 0 {
				continue
			}

			// Match against keyword
			if opts.Keyword != "" {
				matched := false
				if containsIgnoreCase(name, opts.Keyword) ||
					containsIgnoreCase(entries[0].Description, opts.Keyword) {
					matched = true
				}
				if !matched {
					continue
				}
			}

			if opts.Versions {
				for _, entry := range entries {
					if opts.Devel || !isPrerelease(entry.Version) {
						results = append(results, &helmengine.SearchResult{
							Name:         fmt.Sprintf("%s/%s", r.Name, name),
							ChartVersion: entry.Version,
							AppVersion:   entry.AppVersion,
							Description:  entry.Description,
						})
					}
				}
			} else {
				entry := entries[0]
				if opts.Devel || !isPrerelease(entry.Version) {
					results = append(results, &helmengine.SearchResult{
						Name:         fmt.Sprintf("%s/%s", r.Name, name),
						ChartVersion: entry.Version,
						AppVersion:   entry.AppVersion,
						Description:  entry.Description,
					})
				}
			}
		}
	}

	return results, nil
}

// isPrerelease checks if a semver version string is a prerelease (contains a hyphen).
func isPrerelease(version string) bool {
	return strings.Contains(version, "-")
}

func containsIgnoreCase(s, substr string) bool {
	if s == "" || substr == "" {
		return substr == ""
	}
	// Simple case-insensitive contains
	sl := len(s)
	subl := len(substr)
	if subl > sl {
		return false
	}
	for i := 0; i <= sl-subl; i++ {
		match := true
		for j := 0; j < subl; j++ {
			sc := s[i+j]
			tc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if tc >= 'A' && tc <= 'Z' {
				tc += 32
			}
			if sc != tc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
