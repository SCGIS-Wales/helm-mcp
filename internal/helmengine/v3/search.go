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

			if opts.Keyword != "" && !matchesKeyword(name, entries[0].Description, opts.Keyword) {
				continue
			}

			fullName := fmt.Sprintf("%s/%s", r.Name, name)
			results = appendMatchingEntries(results, fullName, entries, opts.Versions, opts.Devel)
		}
	}

	return results, nil
}

// isPrerelease checks if a semver version string is a prerelease (contains a hyphen).
func isPrerelease(version string) bool {
	return strings.Contains(version, "-")
}

func matchesKeyword(name, description, keyword string) bool {
	return containsIgnoreCase(name, keyword) || containsIgnoreCase(description, keyword)
}

func appendMatchingEntries(results []*helmengine.SearchResult, fullName string, entries repo.ChartVersions, allVersions, devel bool) []*helmengine.SearchResult {
	if allVersions {
		for _, entry := range entries {
			if devel || !isPrerelease(entry.Version) {
				results = append(results, &helmengine.SearchResult{
					Name:         fullName,
					ChartVersion: entry.Version,
					AppVersion:   entry.AppVersion,
					Description:  entry.Description,
				})
			}
		}
		return results
	}
	entry := entries[0]
	if devel || !isPrerelease(entry.Version) {
		results = append(results, &helmengine.SearchResult{
			Name:         fullName,
			ChartVersion: entry.Version,
			AppVersion:   entry.AppVersion,
			Description:  entry.Description,
		})
	}
	return results
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
