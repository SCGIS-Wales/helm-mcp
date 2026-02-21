package v4

import (
	"context"
	"fmt"
	"strings"
	"time"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart"
	"helm.sh/helm/v4/pkg/chart/loader"
	"helm.sh/helm/v4/pkg/cli"
	"helm.sh/helm/v4/pkg/cli/values"
	"helm.sh/helm/v4/pkg/getter"
	"helm.sh/helm/v4/pkg/kube"
	"helm.sh/helm/v4/pkg/release"
	v1release "helm.sh/helm/v4/pkg/release/v1"
)

// Error format strings used across multiple release operations.
const (
	errInvalidTimeout      = "invalid timeout: %w"
	errFailedAccessRelease = "failed to access release: %w"
)

func releaserToInfo(rel release.Releaser) *helmengine.ReleaseInfo {
	if rel == nil {
		return nil
	}

	accessor, err := release.NewAccessor(rel)
	if err != nil {
		return &helmengine.ReleaseInfo{}
	}

	info := &helmengine.ReleaseInfo{
		Name:      accessor.Name(),
		Namespace: accessor.Namespace(),
		Revision:  accessor.Version(),
		Status:    accessor.Status(),
		Notes:     accessor.Notes(),
		Labels:    accessor.Labels(),
		Updated:   accessor.DeployedAt(),
	}

	// Chart metadata via charter accessor
	ch := accessor.Chart()
	if ch != nil {
		if chAcc, err := chart.NewAccessor(ch); err == nil {
			info.Chart = chAcc.Name()
			mdMap := chAcc.MetadataAsMap()
			if v, ok := mdMap["version"]; ok {
				info.ChartVersion = fmt.Sprint(v)
			}
			if v, ok := mdMap["appVersion"]; ok {
				info.AppVersion = fmt.Sprint(v)
			}
		}
	}

	return info
}

func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return helmengine.DefaultTimeout, nil
	}
	return time.ParseDuration(s)
}

func (e *V4Engine) List(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ListOptions) ([]*helmengine.ReleaseInfo, error) {
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}

	client := action.NewList(actionConfig)
	client.AllNamespaces = opts.AllNamespaces
	client.Filter = opts.Filter
	client.Limit = opts.Limit
	client.Offset = opts.Offset
	client.Deployed = opts.Deployed
	client.Failed = opts.Failed
	client.Pending = opts.Pending
	client.Uninstalled = opts.Uninstalled
	client.Superseded = opts.Superseded
	client.SortReverse = opts.SortReverse
	client.Selector = opts.Selector

	switch strings.ToLower(opts.SortBy) {
	case "date":
		client.ByDate = true
	case "name", "":
		// default sort by name
	}

	if !opts.Deployed && !opts.Failed && !opts.Pending && !opts.Uninstalled && !opts.Superseded {
		client.Deployed = true
	}

	releases, err := client.Run()
	if err != nil {
		return nil, fmt.Errorf("helm list failed: %w", err)
	}

	result := make([]*helmengine.ReleaseInfo, 0, len(releases))
	for _, rel := range releases {
		result = append(result, releaserToInfo(rel))
	}

	return result, nil
}

func (e *V4Engine) Install(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.InstallOptions) (*helmengine.ReleaseInfo, error) {
	actionConfig, settings, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}

	client := action.NewInstall(actionConfig)
	client.ReleaseName = opts.ReleaseName
	client.Namespace = cfg.Namespace
	client.CreateNamespace = opts.CreateNamespace
	client.DisableHooks = opts.DisableHooks
	client.Replace = opts.Replace
	client.SkipCRDs = opts.SkipCRDs
	client.IncludeCRDs = opts.IncludeCRDs
	client.DependencyUpdate = opts.DependencyUpdate
	client.GenerateName = opts.GenerateName
	client.NameTemplate = opts.NameTemplate
	client.Description = opts.Description
	client.Labels = opts.Labels
	client.ServerSideApply = opts.ServerSideApply
	client.TakeOwnership = opts.TakeOwnership
	client.RollbackOnFailure = opts.RollbackOnFailure
	client.ForceConflicts = opts.ForceConflicts
	client.HideSecret = opts.HideSecret

	if opts.Wait {
		client.WaitStrategy = kube.StatusWatcherStrategy
	}
	if opts.WaitForJobs {
		client.WaitForJobs = true
	}

	timeout, err := parseDuration(opts.Timeout)
	if err != nil {
		return nil, fmt.Errorf(errInvalidTimeout, err)
	}
	client.Timeout = timeout

	switch strings.ToLower(opts.DryRun) {
	case "client":
		client.DryRunStrategy = action.DryRunClient
	case "server":
		client.DryRunStrategy = action.DryRunServer
	case "none", "":
		client.DryRunStrategy = action.DryRunNone
	default:
		return nil, fmt.Errorf("invalid dry_run value: %s (valid: none, client, server)", opts.DryRun)
	}

	if opts.Version != "" {
		client.Version = opts.Version
	}

	chartPath, err := client.LocateChart(opts.Chart, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart %q: %w", opts.Chart, err)
	}

	chartObj, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart %q: %w", chartPath, err)
	}

	vals, err := mergeValues(opts.Values, opts.ValuesFiles, settings)
	if err != nil {
		return nil, err
	}

	rel, err := client.RunWithContext(ctx, chartObj, vals)
	if err != nil {
		return nil, fmt.Errorf("helm install failed: %w", err)
	}

	return releaserToInfo(rel), nil
}

func (e *V4Engine) Upgrade(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.UpgradeOptions) (*helmengine.ReleaseInfo, error) {
	actionConfig, settings, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}

	client := action.NewUpgrade(actionConfig)
	client.Namespace = cfg.Namespace
	client.Install = opts.Install
	client.ForceReplace = opts.Force
	client.ResetValues = opts.ResetValues
	client.ReuseValues = opts.ReuseValues
	client.ResetThenReuseValues = opts.ResetThenReuseValues
	client.DisableHooks = opts.DisableHooks
	client.SkipCRDs = opts.SkipCRDs
	client.CleanupOnFail = opts.CleanupOnFail
	client.DependencyUpdate = opts.DependencyUpdate
	client.Description = opts.Description
	client.Labels = opts.Labels
	client.MaxHistory = opts.MaxHistory
	if opts.ServerSideApply {
		client.ServerSideApply = "true"
	}
	client.TakeOwnership = opts.TakeOwnership
	client.ForceConflicts = opts.ForceConflicts
	client.HideSecret = opts.HideSecret

	if opts.Wait {
		client.WaitStrategy = kube.StatusWatcherStrategy
	}
	if opts.WaitForJobs {
		client.WaitForJobs = true
	}

	timeout, err := parseDuration(opts.Timeout)
	if err != nil {
		return nil, fmt.Errorf(errInvalidTimeout, err)
	}
	client.Timeout = timeout

	switch strings.ToLower(opts.DryRun) {
	case "client":
		client.DryRunStrategy = action.DryRunClient
	case "server":
		client.DryRunStrategy = action.DryRunServer
	case "none", "":
		client.DryRunStrategy = action.DryRunNone
	}

	if opts.Version != "" {
		client.Version = opts.Version
	}

	chartPath, err := client.LocateChart(opts.Chart, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart %q: %w", opts.Chart, err)
	}

	chartObj, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart %q: %w", chartPath, err)
	}

	vals, err := mergeValues(opts.Values, opts.ValuesFiles, settings)
	if err != nil {
		return nil, err
	}

	rel, err := client.RunWithContext(ctx, opts.ReleaseName, chartObj, vals)
	if err != nil {
		return nil, fmt.Errorf("helm upgrade failed: %w", err)
	}

	return releaserToInfo(rel), nil
}

func (e *V4Engine) Uninstall(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.UninstallOptions) (*helmengine.UninstallResult, error) {
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}

	client := action.NewUninstall(actionConfig)
	client.KeepHistory = opts.KeepHistory
	client.DryRun = opts.DryRun
	client.DisableHooks = opts.DisableHooks

	if opts.Wait {
		client.WaitStrategy = kube.StatusWatcherStrategy
	}

	if opts.Timeout != "" {
		timeout, err := parseDuration(opts.Timeout)
		if err != nil {
			return nil, fmt.Errorf(errInvalidTimeout, err)
		}
		client.Timeout = timeout
	}

	if opts.Cascade != "" {
		client.DeletionPropagation = opts.Cascade
	}

	resp, err := client.Run(opts.ReleaseName)
	if err != nil {
		return nil, fmt.Errorf("helm uninstall failed: %w", err)
	}

	result := &helmengine.UninstallResult{
		ReleaseName: opts.ReleaseName,
	}
	if resp != nil && resp.Info != "" {
		result.Info = resp.Info
	}

	return result, nil
}

func (e *V4Engine) Rollback(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.RollbackOptions) error {
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return err
	}

	client := action.NewRollback(actionConfig)
	client.Version = opts.Revision
	client.ForceReplace = opts.Force
	client.DisableHooks = opts.DisableHooks
	client.CleanupOnFail = opts.CleanupOnFail
	client.MaxHistory = opts.MaxHistory
	if opts.ServerSideApply {
		client.ServerSideApply = "true"
	}
	client.ForceConflicts = opts.ForceConflicts

	if opts.DryRun {
		client.DryRunStrategy = action.DryRunClient
	}

	if opts.Wait {
		client.WaitStrategy = kube.StatusWatcherStrategy
	}
	if opts.WaitForJobs {
		client.WaitForJobs = true
	}

	if opts.Timeout != "" {
		timeout, err := parseDuration(opts.Timeout)
		if err != nil {
			return fmt.Errorf(errInvalidTimeout, err)
		}
		client.Timeout = timeout
	}

	if err := client.Run(opts.ReleaseName); err != nil {
		return fmt.Errorf("helm rollback failed: %w", err)
	}

	return nil
}

func (e *V4Engine) Status(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.StatusOptions) (*helmengine.ReleaseInfo, error) {
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}

	client := action.NewStatus(actionConfig)
	client.Version = opts.Revision
	client.ShowResourcesTable = opts.ShowResources

	rel, err := client.Run(opts.ReleaseName)
	if err != nil {
		return nil, fmt.Errorf("helm status failed: %w", err)
	}

	return releaserToInfo(rel), nil
}

func (e *V4Engine) History(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.HistoryOptions) ([]*helmengine.ReleaseInfo, error) {
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}

	client := action.NewHistory(actionConfig)
	if opts.Max > 0 {
		client.Max = opts.Max
	}

	releases, err := client.Run(opts.ReleaseName)
	if err != nil {
		return nil, fmt.Errorf("helm history failed: %w", err)
	}

	result := make([]*helmengine.ReleaseInfo, 0, len(releases))
	for _, rel := range releases {
		result = append(result, releaserToInfo(rel))
	}

	return result, nil
}

func (e *V4Engine) Test(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.TestOptions) (*helmengine.ReleaseInfo, error) {
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}

	client := action.NewReleaseTesting(actionConfig)
	if opts.Timeout != "" {
		timeout, err := parseDuration(opts.Timeout)
		if err != nil {
			return nil, fmt.Errorf(errInvalidTimeout, err)
		}
		client.Timeout = timeout
	}
	client.Filters = map[string][]string{}
	if len(opts.Filters) > 0 {
		client.Filters["name"] = opts.Filters
	}

	rel, shutdownFn, err := client.Run(opts.ReleaseName)
	if shutdownFn != nil {
		defer func() { _ = shutdownFn() }()
	}
	if err != nil {
		return nil, fmt.Errorf("helm test failed: %w", err)
	}

	return releaserToInfo(rel), nil
}

func (e *V4Engine) GetAll(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetOptions) (*helmengine.ReleaseDetail, error) {
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}

	client := action.NewGet(actionConfig)
	client.Version = opts.Revision

	rel, err := client.Run(opts.ReleaseName)
	if err != nil {
		return nil, fmt.Errorf("helm get all failed: %w", err)
	}

	accessor, err := release.NewAccessor(rel)
	if err != nil {
		return nil, fmt.Errorf(errFailedAccessRelease, err)
	}

	detail := &helmengine.ReleaseDetail{
		Release:  releaserToInfo(rel),
		Manifest: accessor.Manifest(),
		Notes:    accessor.Notes(),
	}

	// Config via type assertion to concrete release type
	if r, ok := rel.(*v1release.Release); ok && r.Config != nil {
		detail.Values = r.Config
	}

	// Hooks via type assertion to concrete release type
	if r, ok := rel.(*v1release.Release); ok && len(r.Hooks) > 0 {
		var hookStrs []string
		for _, h := range r.Hooks {
			hookStrs = append(hookStrs, fmt.Sprintf("---\n# Source: %s\n%s", h.Path, h.Manifest))
		}
		detail.Hooks = strings.Join(hookStrs, "\n")
	}

	return detail, nil
}

func (e *V4Engine) GetValues(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetValuesOptions) (map[string]interface{}, error) {
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}

	client := action.NewGetValues(actionConfig)
	client.Version = opts.Revision
	client.AllValues = opts.All

	vals, err := client.Run(opts.ReleaseName)
	if err != nil {
		return nil, fmt.Errorf("helm get values failed: %w", err)
	}

	return vals, nil
}

func (e *V4Engine) GetMetadata(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetOptions) (*helmengine.MetadataInfo, error) {
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}

	client := action.NewGetMetadata(actionConfig)
	client.Version = opts.Revision

	md, err := client.Run(opts.ReleaseName)
	if err != nil {
		return nil, fmt.Errorf("helm get metadata failed: %w", err)
	}

	var deployedAt time.Time
	if md.DeployedAt != "" {
		deployedAt, _ = time.Parse(time.RFC3339, md.DeployedAt)
	}

	return &helmengine.MetadataInfo{
		Name:         md.Name,
		Namespace:    md.Namespace,
		Revision:     md.Revision,
		Status:       md.Status,
		Chart:        md.Chart,
		ChartVersion: md.Version,
		AppVersion:   md.AppVersion,
		DeployedAt:   deployedAt,
	}, nil
}

func (e *V4Engine) GetManifest(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetOptions) (string, error) {
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return "", err
	}

	client := action.NewGet(actionConfig)
	client.Version = opts.Revision

	rel, err := client.Run(opts.ReleaseName)
	if err != nil {
		return "", fmt.Errorf("helm get manifest failed: %w", err)
	}

	accessor, err := release.NewAccessor(rel)
	if err != nil {
		return "", fmt.Errorf(errFailedAccessRelease, err)
	}

	return accessor.Manifest(), nil
}

func (e *V4Engine) GetHooks(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetOptions) (string, error) {
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return "", err
	}

	client := action.NewGet(actionConfig)
	client.Version = opts.Revision

	rel, err := client.Run(opts.ReleaseName)
	if err != nil {
		return "", fmt.Errorf("helm get hooks failed: %w", err)
	}

	// Use type assertion to concrete release type for hook access
	if r, ok := rel.(*v1release.Release); ok {
		var hookStrs []string
		for _, h := range r.Hooks {
			hookStrs = append(hookStrs, fmt.Sprintf("---\n# Source: %s\n%s", h.Path, h.Manifest))
		}
		return strings.Join(hookStrs, "\n"), nil
	}

	return "", nil
}

func (e *V4Engine) GetNotes(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetOptions) (string, error) {
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return "", err
	}

	client := action.NewGet(actionConfig)
	client.Version = opts.Revision

	rel, err := client.Run(opts.ReleaseName)
	if err != nil {
		return "", fmt.Errorf("helm get notes failed: %w", err)
	}

	accessor, err := release.NewAccessor(rel)
	if err != nil {
		return "", fmt.Errorf(errFailedAccessRelease, err)
	}

	return accessor.Notes(), nil
}

func mergeValues(inline map[string]interface{}, valuesFiles []string, settings *cli.EnvSettings) (map[string]interface{}, error) {
	providers := getter.All(settings)
	valueOpts := &values.Options{
		ValueFiles: valuesFiles,
	}
	merged, err := valueOpts.MergeValues(providers)
	if err != nil {
		return nil, fmt.Errorf("failed to merge values files: %w", err)
	}

	for k, v := range inline {
		merged[k] = v
	}

	return merged, nil
}
