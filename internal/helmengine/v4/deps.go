package v4

import (
	"fmt"
	"io"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart"
	"helm.sh/helm/v4/pkg/chart/loader"
	"helm.sh/helm/v4/pkg/cli"
	"helm.sh/helm/v4/pkg/downloader"
	"helm.sh/helm/v4/pkg/getter"
	"helm.sh/helm/v4/pkg/registry"
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
		ContentCache:     settings.ContentCache,
		Debug:            settings.Debug,
		RegistryClient:   rc,
	}
}

// loadChartWithDependencies loads a chart and checks its declared
// dependencies are vendored, as `helm install/upgrade/template` do before
// rendering. Missing dependencies are fetched when update is set (the
// --dependency-update flag) and are an error otherwise; without this check a
// chart with missing subcharts would render without them.
//
// The action structs carry a DependencyUpdate field but never read it: the
// Helm CLI implements the flag, so the engine has to as well.
func loadChartWithDependencies(chartPath string, update bool, keyring string, settings *cli.EnvSettings, rc *registry.Client) (chart.Charter, error) {
	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart %q: %w", chartPath, err)
	}

	ac, err := chart.NewAccessor(ch)
	if err != nil {
		return nil, fmt.Errorf("failed to access chart %q: %w", chartPath, err)
	}
	req := ac.MetaDependencies()
	if len(req) == 0 {
		return ch, nil
	}
	if err := action.CheckDependencies(ch, req); err != nil {
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
