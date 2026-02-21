package v3

import (
	"context"
	"fmt"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/lint/support"
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

func (e *V3Engine) Create(_ context.Context, opts *helmengine.CreateOptions) (string, error) {
	path, err := chartutil.Create(opts.Name, ".")
	if err != nil {
		return "", fmt.Errorf("helm create failed: %w", err)
	}
	return path, nil
}

func (e *V3Engine) Lint(_ context.Context, opts *helmengine.LintOptions) (*helmengine.LintResult, error) {
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
		lintResult.Messages = append(lintResult.Messages, helmengine.LintMessage{
			Severity: severityToString(msg.Severity),
			Path:     msg.Path,
			Message:  msg.Err.Error(),
		})
	}
	lintResult.Passed = len(result.Errors) == 0
	return lintResult, nil
}

func (e *V3Engine) Template(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.TemplateOptions) (string, error) {
	actionConfig, settings, err := newActionConfig(cfg)
	if err != nil {
		actionConfig, settings = newActionConfigNoCluster(cfg)
	}

	client := action.NewInstall(actionConfig)
	client.DryRun = true
	client.ReleaseName = opts.ReleaseName
	client.Replace = true
	client.ClientOnly = !opts.Validate
	client.IncludeCRDs = opts.IncludeCRDs
	client.SkipCRDs = opts.SkipCRDs
	client.IsUpgrade = false
	client.Namespace = cfg.Namespace

	if opts.KubeVersion != "" {
		parsedVersion, err := chartutil.ParseKubeVersion(opts.KubeVersion)
		if err != nil {
			return "", fmt.Errorf("invalid kube_version: %w", err)
		}
		client.KubeVersion = parsedVersion
	}
	if opts.Version != "" {
		client.ChartPathOptions.Version = opts.Version
	}

	chartPath, err := client.ChartPathOptions.LocateChart(opts.Chart, settings)
	if err != nil {
		return "", fmt.Errorf("failed to locate chart %q: %w", opts.Chart, err)
	}
	chartObj, err := loader.Load(chartPath)
	if err != nil {
		return "", fmt.Errorf("failed to load chart %q: %w", chartPath, err)
	}

	providers := getter.All(settings)
	valueOpts := &values.Options{ValueFiles: opts.ValuesFiles}
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
	return rel.Manifest, nil
}

func (e *V3Engine) Package(_ context.Context, opts *helmengine.PackageOptions) (string, error) {
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

func (e *V3Engine) Pull(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.PullOptions) (string, error) {
	_, settings := newActionConfigNoCluster(cfg)
	client := action.NewPullWithOpts(action.WithConfig(new(action.Configuration)))
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

func (e *V3Engine) Push(_ context.Context, _ *helmengine.GlobalConfig, opts *helmengine.PushOptions) (string, error) {
	pushOpts := []action.PushOpt{
		action.WithPushConfig(new(action.Configuration)),
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

func (e *V3Engine) Verify(_ context.Context, opts *helmengine.VerifyOptions) (string, error) {
	client := action.NewVerify()
	if opts.Keyring != "" {
		client.Keyring = opts.Keyring
	}
	if err := client.Run(opts.ChartFile); err != nil {
		return "", fmt.Errorf("helm verify failed: %w", err)
	}
	return fmt.Sprintf("chart %s verified successfully", opts.ChartFile), nil
}

func (e *V3Engine) ShowAll(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions) (string, error) {
	return e.showChart(cfg, opts, action.ShowAll)
}
func (e *V3Engine) ShowChart(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions) (string, error) {
	return e.showChart(cfg, opts, action.ShowChart)
}
func (e *V3Engine) ShowValues(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions) (string, error) {
	return e.showChart(cfg, opts, action.ShowValues)
}
func (e *V3Engine) ShowReadme(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions) (string, error) {
	return e.showChart(cfg, opts, action.ShowReadme)
}
func (e *V3Engine) ShowCRDs(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions) (string, error) {
	return e.showChart(cfg, opts, action.ShowCRDs)
}

func (e *V3Engine) showChart(cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions, outputFormat action.ShowOutputFormat) (string, error) {
	if opts.JSONPath != "" {
		return "", fmt.Errorf("jsonpath is only supported in Helm v4")
	}
	_, settings := newActionConfigNoCluster(cfg)
	client := action.NewShow(outputFormat)
	client.Devel = opts.Devel
	if opts.Version != "" {
		client.Version = opts.Version
	}
	if opts.Repo != "" {
		client.RepoURL = opts.Repo
	}
	cp, err := client.ChartPathOptions.LocateChart(opts.Chart, settings)
	if err != nil {
		return "", fmt.Errorf("failed to locate chart %q: %w", opts.Chart, err)
	}
	output, err := client.Run(cp)
	if err != nil {
		return "", fmt.Errorf("helm show failed: %w", err)
	}
	return output, nil
}

func (e *V3Engine) DependencyBuild(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.DependencyOptions) error {
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

func (e *V3Engine) DependencyList(_ context.Context, _ *helmengine.GlobalConfig, opts *helmengine.DependencyOptions) (string, error) {
	chartObj, err := loader.Load(opts.ChartPath)
	if err != nil {
		return "", fmt.Errorf("failed to load chart at %q: %w", opts.ChartPath, err)
	}
	if chartObj.Metadata == nil || len(chartObj.Metadata.Dependencies) == 0 {
		return "No dependencies found.", nil
	}
	result := "NAME\tVERSION\tREPOSITORY\n"
	for _, dep := range chartObj.Metadata.Dependencies {
		result += fmt.Sprintf("%s\t%s\t%s\n", dep.Name, dep.Version, dep.Repository)
	}
	return result, nil
}

func (e *V3Engine) DependencyUpdate(_ context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.DependencyOptions) error {
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
