package v4

import (
	"context"
	"fmt"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart"
	"helm.sh/helm/v4/pkg/chart/common"
	"helm.sh/helm/v4/pkg/chart/loader"
	chartv2 "helm.sh/helm/v4/pkg/chart/v2"
	chartv2util "helm.sh/helm/v4/pkg/chart/v2/util"
	"helm.sh/helm/v4/pkg/chart/v2/lint/support"
	"helm.sh/helm/v4/pkg/cli/values"
	"helm.sh/helm/v4/pkg/downloader"
	"helm.sh/helm/v4/pkg/getter"
	"helm.sh/helm/v4/pkg/release"
)

func severityToString(sev int) string {
	switch sev {
	case support.InfoSev:
		return "INFO"
	case support.WarningSev:
		return "WARNING"
	case support.ErrorSev:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func (e *V4Engine) Create(_ context.Context, opts *helmengine.CreateOptions) (string, error) {
	path, err := chartv2util.Create(opts.Name, ".")
	if err != nil {
		return "", fmt.Errorf("helm create failed: %w", err)
	}
	return path, nil
}

func (e *V4Engine) Lint(_ context.Context, opts *helmengine.LintOptions) (*helmengine.LintResult, error) {
	client := action.NewLint()
	client.Strict = opts.Strict
	client.WithSubcharts = opts.WithSubcharts
	client.Quiet = opts.Quiet
	client.Namespace = opts.Namespace

	paths := opts.Paths
	if len(paths) == 0 {
		paths = []string{"."}
	}

	vals := make(map[string]interface{})
	if opts.Values != nil {
		vals = opts.Values
	}

	result := client.Run(paths, vals)

	lintResult := &helmengine.LintResult{
		TotalCharts: result.TotalChartsLinted,
	}

	for _, msg := range result.Messages {
		errMsg := ""
		if msg.Err != nil {
			errMsg = msg.Err.Error()
		}
		lintResult.Messages = append(lintResult.Messages, helmengine.LintMessage{
			Severity: severityToString(msg.Severity),
			Path:     msg.Path,
			Message:  errMsg,
		})
	}

	lintResult.Passed = len(result.Errors) == 0

	return lintResult, nil
}

func (e *V4Engine) Template(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.TemplateOptions) (string, error) {
	actionConfig, settings, err := newActionConfig(cfg)
	if err != nil {
		actionConfig, settings = newActionConfigNoCluster(cfg)
	}

	client := action.NewInstall(actionConfig)
	client.ReleaseName = opts.ReleaseName
	client.Replace = true
	client.IncludeCRDs = opts.IncludeCRDs
	client.SkipCRDs = opts.SkipCRDs
	client.Namespace = cfg.Namespace

	if opts.Validate {
		client.DryRunStrategy = action.DryRunServer
	} else {
		client.DryRunStrategy = action.DryRunClient
	}

	if opts.KubeVersion != "" {
		parsedVersion, err := common.ParseKubeVersion(opts.KubeVersion)
		if err != nil {
			return "", fmt.Errorf("invalid kube_version: %w", err)
		}
		client.KubeVersion = parsedVersion
	}

	if opts.Version != "" {
		client.Version = opts.Version
	}

	chartPath, err := client.LocateChart(opts.Chart, settings)
	if err != nil {
		return "", fmt.Errorf("failed to locate chart %q: %w", opts.Chart, err)
	}

	chartObj, err := loader.Load(chartPath)
	if err != nil {
		return "", fmt.Errorf("failed to load chart %q: %w", chartPath, err)
	}

	providers := getter.All(settings)
	valueOpts := &values.Options{
		ValueFiles: opts.ValuesFiles,
	}
	vals, err := valueOpts.MergeValues(providers)
	if err != nil {
		return "", fmt.Errorf("failed to merge values: %w", err)
	}
	for k, v := range opts.Values {
		vals[k] = v
	}

	rel, err := client.RunWithContext(ctx, chartObj, vals)
	if err != nil {
		return "", fmt.Errorf("helm template failed: %w", err)
	}

	accessor, err := release.NewAccessor(rel)
	if err != nil {
		return "", fmt.Errorf("failed to access release: %w", err)
	}

	return accessor.Manifest(), nil
}

func (e *V4Engine) Package(_ context.Context, opts *helmengine.PackageOptions) (string, error) {
	client := action.NewPackage()
	client.Destination = opts.Destination
	if opts.Version != "" {
		client.Version = opts.Version
	}
	if opts.AppVersion != "" {
		client.AppVersion = opts.AppVersion
	}
	client.Sign = opts.Sign
	client.Key = opts.Key
	client.Keyring = opts.Keyring
	client.PassphraseFile = opts.PassphraseFile
	client.DependencyUpdate = opts.DependencyUpdate

	path, err := client.Run(opts.Path, nil)
	if err != nil {
		return "", fmt.Errorf("helm package failed: %w", err)
	}

	return path, nil
}

func (e *V4Engine) Pull(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.PullOptions) (string, error) {
	actionConfig, settings := newActionConfigNoCluster(cfg)

	client := action.NewPull(action.WithConfig(actionConfig))
	client.Settings = settings
	client.Untar = opts.Untar
	client.UntarDir = opts.UntarDir
	client.DestDir = opts.Destination
	client.VerifyLater = opts.Verify

	if opts.Version != "" {
		client.Version = opts.Version
	}
	if opts.Repo != "" {
		client.RepoURL = opts.Repo
	}
	if opts.Username != "" {
		client.Username = opts.Username
	}
	if opts.Password != "" {
		client.Password = opts.Password
	}
	if opts.Keyring != "" {
		client.Keyring = opts.Keyring
	}

	output, err := client.Run(opts.Chart)
	if err != nil {
		return "", fmt.Errorf("helm pull failed: %w", err)
	}

	return output, nil
}

func (e *V4Engine) Push(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.PushOptions) (string, error) {
	pushOpts := []action.PushOpt{
		action.WithPushConfig(action.NewConfiguration()),
		action.WithInsecureSkipTLSVerify(opts.InsecureSkipTLSVerify),
		action.WithPlainHTTP(opts.PlainHTTP),
	}

	if opts.CertFile != "" || opts.KeyFile != "" || opts.CAFile != "" {
		pushOpts = append(pushOpts, action.WithTLSClientConfig(opts.CertFile, opts.KeyFile, opts.CAFile))
	}

	client := action.NewPushWithOpts(pushOpts...)

	output, err := client.Run(opts.ChartRef, opts.Remote)
	if err != nil {
		return "", fmt.Errorf("helm push failed: %w", err)
	}

	return output, nil
}

func (e *V4Engine) Verify(_ context.Context, opts *helmengine.VerifyOptions) (string, error) {
	client := action.NewVerify()
	if opts.Keyring != "" {
		client.Keyring = opts.Keyring
	}

	output, err := client.Run(opts.ChartFile)
	if err != nil {
		return "", fmt.Errorf("helm verify failed: %w", err)
	}

	return output, nil
}

func (e *V4Engine) ShowAll(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions) (string, error) {
	return e.showChart(cfg, opts, action.ShowAll)
}

func (e *V4Engine) ShowChart(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions) (string, error) {
	return e.showChart(cfg, opts, action.ShowChart)
}

func (e *V4Engine) ShowValues(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions) (string, error) {
	return e.showChart(cfg, opts, action.ShowValues)
}

func (e *V4Engine) ShowReadme(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions) (string, error) {
	return e.showChart(cfg, opts, action.ShowReadme)
}

func (e *V4Engine) ShowCRDs(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions) (string, error) {
	return e.showChart(cfg, opts, action.ShowCRDs)
}

func (e *V4Engine) showChart(cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions, outputFormat action.ShowOutputFormat) (string, error) {
	actionConfig, settings := newActionConfigNoCluster(cfg)

	client := action.NewShow(outputFormat, actionConfig)
	client.Devel = opts.Devel

	if opts.JSONPath != "" {
		client.JSONPathTemplate = opts.JSONPath
	}

	if opts.Version != "" {
		client.Version = opts.Version
	}

	chartPath := opts.Chart
	if opts.Repo != "" {
		client.RepoURL = opts.Repo
	}

	cp, err := client.LocateChart(chartPath, settings)
	if err != nil {
		return "", fmt.Errorf("failed to locate chart %q: %w", chartPath, err)
	}

	output, err := client.Run(cp)
	if err != nil {
		return "", fmt.Errorf("helm show failed: %w", err)
	}

	return output, nil
}

func (e *V4Engine) DependencyBuild(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.DependencyOptions) error {
	_, settings := newActionConfigNoCluster(cfg)

	man := &downloader.Manager{
		ChartPath:        opts.ChartPath,
		Keyring:          opts.Keyring,
		SkipUpdate:       opts.SkipRefresh,
		Getters:          getter.All(settings),
		RepositoryConfig: settings.RepositoryConfig,
		RepositoryCache:  settings.RepositoryCache,
	}

	if opts.Verify {
		man.Verify = downloader.VerifyAlways
	}

	if err := man.Build(); err != nil {
		return fmt.Errorf("helm dependency build failed: %w", err)
	}

	return nil
}

func (e *V4Engine) DependencyList(_ context.Context, _ *helmengine.GlobalConfig, opts *helmengine.DependencyOptions) (string, error) {
	chartObj, err := loader.Load(opts.ChartPath)
	if err != nil {
		return "", fmt.Errorf("failed to load chart at %q: %w", opts.ChartPath, err)
	}

	chAcc, err := chart.NewAccessor(chartObj)
	if err != nil {
		return "", fmt.Errorf("failed to access chart: %w", err)
	}

	deps := chAcc.MetaDependencies()
	if len(deps) == 0 {
		return "No dependencies found.", nil
	}

	result := "NAME\tVERSION\tREPOSITORY\tSTATUS\n"
	for _, dep := range deps {
		if d, ok := dep.(*chartv2.Dependency); ok {
			result += fmt.Sprintf("%s\t%s\t%s\t\n", d.Name, d.Version, d.Repository)
		} else if d, ok := dep.(chartv2.Dependency); ok {
			result += fmt.Sprintf("%s\t%s\t%s\t\n", d.Name, d.Version, d.Repository)
		}
	}

	return result, nil
}

func (e *V4Engine) DependencyUpdate(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.DependencyOptions) error {
	_, settings := newActionConfigNoCluster(cfg)

	man := &downloader.Manager{
		ChartPath:        opts.ChartPath,
		Keyring:          opts.Keyring,
		SkipUpdate:       opts.SkipRefresh,
		Getters:          getter.All(settings),
		RepositoryConfig: settings.RepositoryConfig,
		RepositoryCache:  settings.RepositoryCache,
	}

	if opts.Verify {
		man.Verify = downloader.VerifyAlways
	}

	if err := man.Update(); err != nil {
		return fmt.Errorf("helm dependency update failed: %w", err)
	}

	return nil
}
