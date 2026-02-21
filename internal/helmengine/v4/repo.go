package v4

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"

	"helm.sh/helm/v4/pkg/cli"
	"helm.sh/helm/v4/pkg/getter"
	repo "helm.sh/helm/v4/pkg/repo/v1"
)

const errFailedLoadRepoFile = "failed to load repository file: %w"

func (e *V4Engine) RepoAdd(_ context.Context, opts *helmengine.RepoAddOptions) error {
	settings := cli.New()

	entry := &repo.Entry{
		Name:                  opts.Name,
		URL:                   opts.URL,
		Username:              opts.Username,
		Password:              opts.Password,
		CAFile:                opts.CAFile,
		CertFile:              opts.CertFile,
		KeyFile:               opts.KeyFile,
		InsecureSkipTLSVerify: opts.InsecureSkipTLSVerify,
		PassCredentialsAll:    opts.PassCredentialsAll,
	}

	repoFile := settings.RepositoryConfig
	if err := os.MkdirAll(filepath.Dir(repoFile), 0700); err != nil {
		return fmt.Errorf("failed to create repository config directory: %w", err)
	}

	var f *repo.File
	if _, err := os.Stat(repoFile); os.IsNotExist(err) {
		f = repo.NewFile()
	} else {
		var err error
		f, err = repo.LoadFile(repoFile)
		if err != nil {
			return fmt.Errorf(errFailedLoadRepoFile, err)
		}
	}

	if f.Has(opts.Name) && !opts.ForceUpdate {
		return fmt.Errorf("repository %q already exists; use force_update to overwrite", opts.Name)
	}

	r, err := repo.NewChartRepository(entry, getter.All(settings))
	if err != nil {
		return fmt.Errorf("failed to create chart repository: %w", err)
	}

	r.CachePath = settings.RepositoryCache
	if _, err := r.DownloadIndexFile(); err != nil {
		return fmt.Errorf("failed to download index file for %q: %w", opts.URL, err)
	}

	f.Update(entry)
	if err := f.WriteFile(repoFile, 0600); err != nil {
		return fmt.Errorf("failed to write repository file: %w", err)
	}

	return nil
}

func (e *V4Engine) RepoList(_ context.Context, _ *helmengine.RepoListOptions) ([]*helmengine.RepoEntry, error) {
	settings := cli.New()

	f, err := repo.LoadFile(settings.RepositoryConfig)
	if err != nil {
		return nil, fmt.Errorf(errFailedLoadRepoFile, err)
	}

	result := make([]*helmengine.RepoEntry, 0, len(f.Repositories))
	for _, r := range f.Repositories {
		result = append(result, &helmengine.RepoEntry{
			Name: r.Name,
			URL:  r.URL,
		})
	}

	return result, nil
}

func (e *V4Engine) RepoUpdate(_ context.Context, opts *helmengine.RepoUpdateOptions) (string, error) {
	settings := cli.New()

	f, err := repo.LoadFile(settings.RepositoryConfig)
	if err != nil {
		return "", fmt.Errorf(errFailedLoadRepoFile, err)
	}

	var repos []*repo.Entry
	if len(opts.Names) > 0 {
		nameSet := make(map[string]bool)
		for _, n := range opts.Names {
			nameSet[n] = true
		}
		for _, r := range f.Repositories {
			if nameSet[r.Name] {
				repos = append(repos, r)
			}
		}
	} else {
		repos = f.Repositories
	}

	var updated []string
	for _, entry := range repos {
		cr, err := repo.NewChartRepository(entry, getter.All(settings))
		if err != nil {
			return "", fmt.Errorf("failed to create chart repository for %q: %w", entry.Name, err)
		}
		cr.CachePath = settings.RepositoryCache
		if _, err := cr.DownloadIndexFile(); err != nil {
			return "", fmt.Errorf("failed to update repository %q: %w", entry.Name, err)
		}
		updated = append(updated, entry.Name)
	}

	return fmt.Sprintf("Successfully updated %d repositories: %s", len(updated), strings.Join(updated, ", ")), nil
}

func (e *V4Engine) RepoRemove(_ context.Context, opts *helmengine.RepoRemoveOptions) error {
	settings := cli.New()

	f, err := repo.LoadFile(settings.RepositoryConfig)
	if err != nil {
		return fmt.Errorf(errFailedLoadRepoFile, err)
	}

	for _, name := range opts.Names {
		if !f.Remove(name) {
			return fmt.Errorf("repository %q not found", name)
		}
	}

	if err := f.WriteFile(settings.RepositoryConfig, 0600); err != nil {
		return fmt.Errorf("failed to write repository file: %w", err)
	}

	return nil
}

func (e *V4Engine) RepoIndex(_ context.Context, opts *helmengine.RepoIndexOptions) error {
	i, err := repo.IndexDirectory(opts.Directory, opts.URL)
	if err != nil {
		return fmt.Errorf("failed to index directory: %w", err)
	}

	if opts.Merge != "" {
		existing, err := repo.LoadIndexFile(opts.Merge)
		if err != nil {
			return fmt.Errorf("failed to load existing index file: %w", err)
		}
		existing.Merge(i)
		i = existing
	}

	i.SortEntries()
	indexFile := filepath.Join(opts.Directory, "index.yaml")
	if err := i.WriteFile(indexFile, 0600); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}
