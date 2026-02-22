package v3

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
)

// Error format strings used across multiple release operations.
const (
	errInvalidTimeout        = "invalid timeout: %w"
	errServerSideApplyV4Only = "server_side_apply is only supported in Helm v4"
	errForceConflictsV4Only  = "force_conflicts is only supported in Helm v4"
)

func releaseToInfo(rel *release.Release) *helmengine.ReleaseInfo {
	if rel == nil {
		return nil
	}

	info := &helmengine.ReleaseInfo{
		Name:      rel.Name,
		Namespace: rel.Namespace,
		Revision:  rel.Version,
		Labels:    rel.Labels,
	}

	if rel.Info != nil {
		info.Status = string(rel.Info.Status)
		info.Description = rel.Info.Description
		info.Notes = rel.Info.Notes
		info.Updated = rel.Info.LastDeployed.Time
	}

	if rel.Chart != nil && rel.Chart.Metadata != nil {
		info.Chart = rel.Chart.Metadata.Name
		info.ChartVersion = rel.Chart.Metadata.Version
		info.AppVersion = rel.Chart.Metadata.AppVersion
	}

	return info
}

// parseDuration delegates to the shared helmengine.ParseDuration.
var parseDuration = helmengine.ParseDuration

func (e *V3Engine) List(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ListOptions) ([]*helmengine.ReleaseInfo, error) {
	actionConfig, settings, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}

	// When AllNamespaces is requested, re-init the action config with an empty
	// namespace so the underlying storage driver queries all namespaces.
	// Clearing cfg.Namespace before newActionConfig doesn't work because
	// settings.Namespace() falls back to "default", scoping the driver to only
	// the default namespace.
	if opts.AllNamespaces {
		if err := actionConfig.Init(
			settings.RESTClientGetter(),
			"",
			os.Getenv("HELM_DRIVER"),
			actionConfig.Log,
		); err != nil {
			return nil, fmt.Errorf("failed to reinitialize action config for all namespaces: %w", err)
		}
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

	if opts.Selector != "" {
		return nil, fmt.Errorf("selector is only supported in Helm v4; set helm_version to v4 or remove this field")
	}

	switch strings.ToLower(opts.SortBy) {
	case "date":
		client.ByDate = true
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
		result = append(result, releaseToInfo(rel))
	}
	return result, nil
}

func (e *V3Engine) Install(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.InstallOptions) (*helmengine.ReleaseInfo, error) {
	if opts.ServerSideApply {
		return nil, errors.New(errServerSideApplyV4Only)
	}
	if opts.TakeOwnership {
		return nil, fmt.Errorf("take_ownership is only supported in Helm v4")
	}
	if opts.RollbackOnFailure {
		return nil, fmt.Errorf("rollback_on_failure is only supported in Helm v4")
	}
	if opts.ForceConflicts {
		return nil, errors.New(errForceConflictsV4Only)
	}
	if opts.HideSecret {
		return nil, fmt.Errorf("hide_secret is only supported in Helm v4")
	}

	actionConfig, settings, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}

	client := action.NewInstall(actionConfig)
	client.ReleaseName = opts.ReleaseName
	client.Namespace = cfg.Namespace
	client.CreateNamespace = opts.CreateNamespace
	client.Wait = opts.Wait
	client.WaitForJobs = opts.WaitForJobs
	client.DisableHooks = opts.DisableHooks
	client.Replace = opts.Replace
	client.SkipCRDs = opts.SkipCRDs
	client.IncludeCRDs = opts.IncludeCRDs
	client.DependencyUpdate = opts.DependencyUpdate
	client.GenerateName = opts.GenerateName
	client.NameTemplate = opts.NameTemplate
	client.Description = opts.Description
	client.Labels = opts.Labels

	timeout, err := parseDuration(opts.Timeout)
	if err != nil {
		return nil, fmt.Errorf(errInvalidTimeout, err)
	}
	client.Timeout = timeout

	switch strings.ToLower(opts.DryRun) {
	case "client":
		client.DryRun = true
		client.ClientOnly = true
	case "server":
		client.DryRun = true
	case "none", "":
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
	return releaseToInfo(rel), nil
}

func (e *V3Engine) Upgrade(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.UpgradeOptions) (*helmengine.ReleaseInfo, error) {
	if opts.ServerSideApply {
		return nil, errors.New(errServerSideApplyV4Only)
	}
	if opts.TakeOwnership {
		return nil, fmt.Errorf("take_ownership is only supported in Helm v4")
	}
	if opts.ForceConflicts {
		return nil, errors.New(errForceConflictsV4Only)
	}
	if opts.HideSecret {
		return nil, fmt.Errorf("hide_secret is only supported in Helm v4")
	}
	if opts.ResetThenReuseValues {
		return nil, fmt.Errorf("reset_then_reuse_values is only supported in Helm v4")
	}

	actionConfig, settings, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}

	client := action.NewUpgrade(actionConfig)
	client.Namespace = cfg.Namespace
	client.Install = opts.Install
	client.Force = opts.Force
	client.ResetValues = opts.ResetValues
	client.ReuseValues = opts.ReuseValues
	client.Wait = opts.Wait
	client.WaitForJobs = opts.WaitForJobs
	client.DisableHooks = opts.DisableHooks
	client.SkipCRDs = opts.SkipCRDs
	client.CleanupOnFail = opts.CleanupOnFail
	client.DependencyUpdate = opts.DependencyUpdate
	client.Description = opts.Description
	client.Labels = opts.Labels
	client.MaxHistory = opts.MaxHistory

	timeout, err := parseDuration(opts.Timeout)
	if err != nil {
		return nil, fmt.Errorf(errInvalidTimeout, err)
	}
	client.Timeout = timeout

	switch strings.ToLower(opts.DryRun) {
	case "client", "server":
		client.DryRun = true
	case "none", "":
		// no dry-run
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

	rel, err := client.RunWithContext(ctx, opts.ReleaseName, chartObj, vals)
	if err != nil {
		return nil, fmt.Errorf("helm upgrade failed: %w", err)
	}
	return releaseToInfo(rel), nil
}

func (e *V3Engine) Uninstall(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.UninstallOptions) (*helmengine.UninstallResult, error) {
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}

	client := action.NewUninstall(actionConfig)
	client.KeepHistory = opts.KeepHistory
	client.DryRun = opts.DryRun
	client.Wait = opts.Wait
	client.DisableHooks = opts.DisableHooks
	if opts.Timeout != "" {
		t, err := parseDuration(opts.Timeout)
		if err != nil {
			return nil, fmt.Errorf(errInvalidTimeout, err)
		}
		client.Timeout = t
	}
	if opts.Cascade != "" {
		client.DeletionPropagation = opts.Cascade
	}

	resp, err := client.Run(opts.ReleaseName)
	if err != nil {
		return nil, fmt.Errorf("helm uninstall failed: %w", err)
	}

	result := &helmengine.UninstallResult{ReleaseName: opts.ReleaseName}
	if resp != nil && resp.Info != "" {
		result.Info = resp.Info
	}
	return result, nil
}

func (e *V3Engine) Rollback(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.RollbackOptions) error {
	if opts.ServerSideApply {
		return errors.New(errServerSideApplyV4Only)
	}
	if opts.ForceConflicts {
		return errors.New(errForceConflictsV4Only)
	}

	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return err
	}

	client := action.NewRollback(actionConfig)
	client.Version = opts.Revision
	client.Wait = opts.Wait
	client.WaitForJobs = opts.WaitForJobs
	client.Force = opts.Force
	client.DryRun = opts.DryRun
	client.DisableHooks = opts.DisableHooks
	client.CleanupOnFail = opts.CleanupOnFail
	client.MaxHistory = opts.MaxHistory
	if opts.Timeout != "" {
		t, err := parseDuration(opts.Timeout)
		if err != nil {
			return fmt.Errorf(errInvalidTimeout, err)
		}
		client.Timeout = t
	}

	return client.Run(opts.ReleaseName)
}

func (e *V3Engine) Status(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.StatusOptions) (*helmengine.ReleaseInfo, error) {
	if opts.ShowResources {
		return nil, fmt.Errorf("show_resources is only supported in Helm v4")
	}
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}
	client := action.NewStatus(actionConfig)
	client.Version = opts.Revision
	rel, err := client.Run(opts.ReleaseName)
	if err != nil {
		return nil, fmt.Errorf("helm status failed: %w", err)
	}
	return releaseToInfo(rel), nil
}

func (e *V3Engine) History(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.HistoryOptions) ([]*helmengine.ReleaseInfo, error) {
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
		result = append(result, releaseToInfo(rel))
	}
	return result, nil
}

func (e *V3Engine) Test(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.TestOptions) (*helmengine.ReleaseInfo, error) {
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}
	client := action.NewReleaseTesting(actionConfig)
	if opts.Timeout != "" {
		t, err := parseDuration(opts.Timeout)
		if err != nil {
			return nil, fmt.Errorf(errInvalidTimeout, err)
		}
		client.Timeout = t
	}
	client.Filters = map[string][]string{}
	if len(opts.Filters) > 0 {
		client.Filters["name"] = opts.Filters
	}
	rel, err := client.Run(opts.ReleaseName)
	if err != nil {
		return nil, fmt.Errorf("helm test failed: %w", err)
	}
	return releaseToInfo(rel), nil
}

func (e *V3Engine) GetAll(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetOptions) (*helmengine.ReleaseDetail, error) {
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
	detail := &helmengine.ReleaseDetail{
		Release:  releaseToInfo(rel),
		Manifest: rel.Manifest,
	}
	if rel.Info != nil {
		detail.Notes = rel.Info.Notes
	}
	if rel.Config != nil {
		detail.Values = rel.Config
	}
	if len(rel.Hooks) > 0 {
		var hookStrs []string
		for _, h := range rel.Hooks {
			hookStrs = append(hookStrs, fmt.Sprintf("---\n# Source: %s\n%s", h.Path, h.Manifest))
		}
		detail.Hooks = strings.Join(hookStrs, "\n")
	}
	return detail, nil
}

func (e *V3Engine) GetValues(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetValuesOptions) (map[string]interface{}, error) {
	actionConfig, _, err := newActionConfig(cfg)
	if err != nil {
		return nil, err
	}
	client := action.NewGetValues(actionConfig)
	client.Version = opts.Revision
	client.AllValues = opts.All
	return client.Run(opts.ReleaseName)
}

func (e *V3Engine) GetMetadata(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetOptions) (*helmengine.MetadataInfo, error) {
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
	revision, err := strconv.Atoi(md.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse revision %q: %w", md.Version, err)
	}
	var deployedAt time.Time
	if md.DeployedAt != "" {
		deployedAt, err = time.Parse(time.RFC3339, md.DeployedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse deployed_at %q: %w", md.DeployedAt, err)
		}
	}
	return &helmengine.MetadataInfo{
		Name:         md.Name,
		Namespace:    md.Namespace,
		Revision:     revision,
		Status:       md.Status,
		Chart:        md.Chart,
		ChartVersion: md.Version,
		AppVersion:   md.AppVersion,
		DeployedAt:   deployedAt,
	}, nil
}

func (e *V3Engine) GetManifest(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetOptions) (string, error) {
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
	return rel.Manifest, nil
}

func (e *V3Engine) GetHooks(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetOptions) (string, error) {
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
	var hookStrs []string
	for _, h := range rel.Hooks {
		hookStrs = append(hookStrs, fmt.Sprintf("---\n# Source: %s\n%s", h.Path, h.Manifest))
	}
	return strings.Join(hookStrs, "\n"), nil
}

func (e *V3Engine) GetNotes(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetOptions) (string, error) {
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
	if rel.Info != nil {
		return rel.Info.Notes, nil
	}
	return "", nil
}

func mergeValues(inline map[string]interface{}, valuesFiles []string, settings *cli.EnvSettings) (map[string]interface{}, error) {
	// Validate that values files actually exist before passing them to the
	// Helm SDK, which produces opaque errors for missing files.
	for _, f := range valuesFiles {
		if _, err := os.Stat(f); err != nil {
			return nil, fmt.Errorf("values file %q: %w", f, err)
		}
	}

	providers := getter.All(settings)
	valueOpts := &values.Options{ValueFiles: valuesFiles}
	merged, err := valueOpts.MergeValues(providers)
	if err != nil {
		return nil, fmt.Errorf("failed to merge values files: %w", err)
	}
	for k, v := range inline {
		merged[k] = v
	}
	return merged, nil
}
