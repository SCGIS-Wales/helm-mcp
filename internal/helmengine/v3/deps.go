package v3

import (
	"errors"
	"fmt"
	"io"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
)

// newDependencyManager returns a downloader.Manager wired the way the Helm
// CLI wires it. Out must never be nil: the manager writes progress with
// fmt.Fprintf, which panics on a nil writer as soon as a dependency is
// downloaded. The registry client lets oci:// dependencies resolve.
func newDependencyManager(chartPath, keyring string, skipUpdate bool, settings *cli.EnvSettings, rc *registry.Client) *downloader.Manager {
	if keyring == "" {
		keyring = defaultKeyring()
	}
	return &downloader.Manager{
		Out:              io.Discard,
		ChartPath:        chartPath,
		Keyring:          keyring,
		SkipUpdate:       skipUpdate,
		Getters:          getter.All(settings),
		RepositoryConfig: settings.RepositoryConfig,
		RepositoryCache:  settings.RepositoryCache,
		Debug:            settings.Debug,
		RegistryClient:   rc,
	}
}

// loadChartWithDependencies loads a chart and checks its declared
// dependencies are vendored, as `helm install/upgrade/template` do before
// rendering. Missing dependencies are fetched when update is set (the
// --dependency-update flag) and are an error otherwise. The action structs
// never read DependencyUpdate: the Helm CLI implements it.
func loadChartWithDependencies(chartPath string, update bool, keyring string, settings *cli.EnvSettings, rc *registry.Client) (*chart.Chart, error) {
	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart %q: %w", chartPath, err)
	}
	if ch.Metadata == nil || ch.Metadata.Dependencies == nil {
		return ch, nil
	}
	if err := action.CheckDependencies(ch, ch.Metadata.Dependencies); err != nil {
		if !update {
			return nil, fmt.Errorf("an error occurred while checking for chart dependencies. "+
				"Set dependency_update or run helm_dependency_build to fetch missing dependencies: %w", err)
		}
		if err := newDependencyManager(chartPath, keyring, false, settings, rc).Update(); err != nil {
			return nil, fmt.Errorf("failed to update chart dependencies: %w", err)
		}
		if ch, err = loader.Load(chartPath); err != nil {
			return nil, fmt.Errorf("failed reloading chart after dependency update: %w", err)
		}
	}
	return ch, nil
}

// errPostRendererV3 explains why post_renderer is refused on v3: a v3
// post-renderer is any executable path, so exposing it would let an MCP
// client run arbitrary programs on the server. Helm v4 post-renderers are
// installed plugins, referenced by name.
var errPostRendererV3 = errors.New("post_renderer is only supported with Helm v4, where post-renderers are installed plugins; " +
	"Helm v3 post-renderers execute an arbitrary binary and are not exposed")
