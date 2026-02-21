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

			if opts.Keyword != "" {
				matched := strings.Contains(strings.ToLower(name), strings.ToLower(opts.Keyword)) ||
					strings.Contains(strings.ToLower(entries[0].Description), strings.ToLower(opts.Keyword))
				if !matched {
					continue
				}
			}

			if opts.Versions {
				for _, entry := range entries {
					if opts.Devel || !strings.Contains(entry.Version, "-") {
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
				if opts.Devel || !strings.Contains(entry.Version, "-") {
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
