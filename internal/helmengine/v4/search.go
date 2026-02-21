package v4

import (
	"context"
	"fmt"
	"strings"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"

	"helm.sh/helm/v4/pkg/cli"
	repo "helm.sh/helm/v4/pkg/repo/v1"
)

func (e *V4Engine) SearchHub(_ context.Context, opts *helmengine.SearchHubOptions) ([]*helmengine.SearchResult, error) {
	return nil, fmt.Errorf("search hub is not directly supported via the Helm v4 SDK; use the Artifact Hub API at https://artifacthub.io/api/v1/packages/search")
}

func (e *V4Engine) SearchRepo(_ context.Context, opts *helmengine.SearchRepoOptions) ([]*helmengine.SearchResult, error) {
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

func matchesKeyword(name, description, keyword string) bool {
	kw := strings.ToLower(keyword)
	return strings.Contains(strings.ToLower(name), kw) ||
		strings.Contains(strings.ToLower(description), kw)
}

func appendMatchingEntries(results []*helmengine.SearchResult, fullName string, entries repo.ChartVersions, allVersions, devel bool) []*helmengine.SearchResult {
	if allVersions {
		for _, entry := range entries {
			if devel || !strings.Contains(entry.Version, "-") {
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
	if devel || !strings.Contains(entry.Version, "-") {
		results = append(results, &helmengine.SearchResult{
			Name:         fullName,
			ChartVersion: entry.Version,
			AppVersion:   entry.AppVersion,
			Description:  entry.Description,
		})
	}
	return results
}
