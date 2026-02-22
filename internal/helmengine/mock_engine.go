package helmengine

import (
	"context"
	"time"
)

// MockEngine implements Engine for testing. Each method field can be set to
// control the mock behavior. If a method field is nil, the mock returns
// sensible defaults.
type MockEngine struct {
	InstallFn           func(ctx context.Context, cfg *GlobalConfig, opts *InstallOptions) (*ReleaseInfo, error)
	UpgradeFn           func(ctx context.Context, cfg *GlobalConfig, opts *UpgradeOptions) (*ReleaseInfo, error)
	UninstallFn         func(ctx context.Context, cfg *GlobalConfig, opts *UninstallOptions) (*UninstallResult, error)
	RollbackFn          func(ctx context.Context, cfg *GlobalConfig, opts *RollbackOptions) error
	ListFn              func(ctx context.Context, cfg *GlobalConfig, opts *ListOptions) ([]*ReleaseInfo, error)
	StatusFn            func(ctx context.Context, cfg *GlobalConfig, opts *StatusOptions) (*ReleaseInfo, error)
	HistoryFn           func(ctx context.Context, cfg *GlobalConfig, opts *HistoryOptions) ([]*ReleaseInfo, error)
	TestFn              func(ctx context.Context, cfg *GlobalConfig, opts *TestOptions) (*ReleaseInfo, error)
	GetAllFn            func(ctx context.Context, cfg *GlobalConfig, opts *GetOptions) (*ReleaseDetail, error)
	GetValuesFn         func(ctx context.Context, cfg *GlobalConfig, opts *GetValuesOptions) (map[string]interface{}, error)
	GetMetadataFn       func(ctx context.Context, cfg *GlobalConfig, opts *GetOptions) (*MetadataInfo, error)
	GetManifestFn       func(ctx context.Context, cfg *GlobalConfig, opts *GetOptions) (string, error)
	GetHooksFn          func(ctx context.Context, cfg *GlobalConfig, opts *GetOptions) (string, error)
	GetNotesFn          func(ctx context.Context, cfg *GlobalConfig, opts *GetOptions) (string, error)
	CreateFn            func(ctx context.Context, opts *CreateOptions) (string, error)
	LintFn              func(ctx context.Context, opts *LintOptions) (*LintResult, error)
	TemplateFn          func(ctx context.Context, cfg *GlobalConfig, opts *TemplateOptions) (string, error)
	PackageFn           func(ctx context.Context, opts *PackageOptions) (string, error)
	PullFn              func(ctx context.Context, cfg *GlobalConfig, opts *PullOptions) (string, error)
	PushFn              func(ctx context.Context, cfg *GlobalConfig, opts *PushOptions) (string, error)
	VerifyFn            func(ctx context.Context, opts *VerifyOptions) (string, error)
	ShowAllFn           func(ctx context.Context, cfg *GlobalConfig, opts *ShowOptions) (string, error)
	ShowChartFn         func(ctx context.Context, cfg *GlobalConfig, opts *ShowOptions) (string, error)
	ShowValuesFn        func(ctx context.Context, cfg *GlobalConfig, opts *ShowOptions) (string, error)
	ShowReadmeFn        func(ctx context.Context, cfg *GlobalConfig, opts *ShowOptions) (string, error)
	ShowCRDsFn          func(ctx context.Context, cfg *GlobalConfig, opts *ShowOptions) (string, error)
	DependencyBuildFn   func(ctx context.Context, cfg *GlobalConfig, opts *DependencyOptions) error
	DependencyListFn    func(ctx context.Context, cfg *GlobalConfig, opts *DependencyOptions) (string, error)
	DependencyUpdateFn  func(ctx context.Context, cfg *GlobalConfig, opts *DependencyOptions) error
	RepoAddFn           func(ctx context.Context, opts *RepoAddOptions) error
	RepoListFn          func(ctx context.Context, opts *RepoListOptions) ([]*RepoEntry, error)
	RepoUpdateFn        func(ctx context.Context, opts *RepoUpdateOptions) (string, error)
	RepoRemoveFn        func(ctx context.Context, opts *RepoRemoveOptions) error
	RepoIndexFn         func(ctx context.Context, opts *RepoIndexOptions) error
	RegistryLoginFn     func(ctx context.Context, opts *RegistryLoginOptions) error
	RegistryLogoutFn    func(ctx context.Context, opts *RegistryLogoutOptions) error
	SearchHubFn         func(ctx context.Context, opts *SearchHubOptions) ([]*SearchResult, error)
	SearchRepoFn        func(ctx context.Context, opts *SearchRepoOptions) ([]*SearchResult, error)
	PluginInstallFn     func(ctx context.Context, opts *PluginInstallOptions) error
	PluginListFn        func(ctx context.Context) ([]*PluginInfo, error)
	PluginUninstallFn   func(ctx context.Context, opts *PluginUninstallOptions) error
	PluginUpdateFn      func(ctx context.Context, opts *PluginUpdateOptions) error
	EnvFn               func(ctx context.Context) (map[string]string, error)
	VersionFn           func(ctx context.Context) (*VersionInfo, error)

	// Call tracking
	LastInstallOpts     *InstallOptions
	LastUpgradeOpts     *UpgradeOptions
	LastUninstallOpts   *UninstallOptions
	LastRollbackOpts    *RollbackOptions
	LastListOpts        *ListOptions
	LastStatusOpts      *StatusOptions
	LastHistoryOpts     *HistoryOptions
	LastTestOpts        *TestOptions
	LastGetOpts         *GetOptions
	LastGetValuesOpts       *GetValuesOptions
	LastConfig              *GlobalConfig
	LastRepoAddOpts         *RepoAddOptions
	LastRegistryLoginOpts   *RegistryLoginOptions
	LastPullOpts            *PullOptions
}

// copyConfig creates a shallow copy of a GlobalConfig so that
// credential zeroing in handlers does not affect test assertions.
func copyConfig(cfg *GlobalConfig) *GlobalConfig {
	if cfg == nil {
		return nil
	}
	cp := *cfg
	return &cp
}

// copyRepoAddOptions creates a shallow copy so that ZeroPassword in
// handlers does not affect test assertions.
func copyRepoAddOptions(opts *RepoAddOptions) *RepoAddOptions {
	if opts == nil {
		return nil
	}
	cp := *opts
	return &cp
}

// copyRegistryLoginOptions creates a shallow copy so that ZeroPassword
// in handlers does not affect test assertions.
func copyRegistryLoginOptions(opts *RegistryLoginOptions) *RegistryLoginOptions {
	if opts == nil {
		return nil
	}
	cp := *opts
	return &cp
}

// copyPullOptions creates a shallow copy so that ZeroPassword in
// handlers does not affect test assertions.
func copyPullOptions(opts *PullOptions) *PullOptions {
	if opts == nil {
		return nil
	}
	cp := *opts
	return &cp
}

const defaultMockAppVersion = "1.25.0"

// DefaultRelease returns a standard test release.
func DefaultRelease() *ReleaseInfo {
	return &ReleaseInfo{
		Name:         "my-release",
		Namespace:    "default",
		Revision:     1,
		Status:       "deployed",
		Chart:        "nginx",
		ChartVersion: "1.0.0",
		AppVersion:   defaultMockAppVersion,
		Updated:      time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}
}

func (m *MockEngine) Install(ctx context.Context, cfg *GlobalConfig, opts *InstallOptions) (*ReleaseInfo, error) {
	m.LastInstallOpts = opts
	m.LastConfig = copyConfig(cfg)
	if m.InstallFn != nil {
		return m.InstallFn(ctx, cfg, opts)
	}
	return DefaultRelease(), nil
}

func (m *MockEngine) Upgrade(ctx context.Context, cfg *GlobalConfig, opts *UpgradeOptions) (*ReleaseInfo, error) {
	m.LastUpgradeOpts = opts
	m.LastConfig = copyConfig(cfg)
	if m.UpgradeFn != nil {
		return m.UpgradeFn(ctx, cfg, opts)
	}
	r := DefaultRelease()
	r.Revision = 2
	return r, nil
}

func (m *MockEngine) Uninstall(ctx context.Context, cfg *GlobalConfig, opts *UninstallOptions) (*UninstallResult, error) {
	m.LastUninstallOpts = opts
	m.LastConfig = copyConfig(cfg)
	if m.UninstallFn != nil {
		return m.UninstallFn(ctx, cfg, opts)
	}
	return &UninstallResult{ReleaseName: opts.ReleaseName, Info: "release uninstalled"}, nil
}

func (m *MockEngine) Rollback(ctx context.Context, cfg *GlobalConfig, opts *RollbackOptions) error {
	m.LastRollbackOpts = opts
	m.LastConfig = copyConfig(cfg)
	if m.RollbackFn != nil {
		return m.RollbackFn(ctx, cfg, opts)
	}
	return nil
}

func (m *MockEngine) List(ctx context.Context, cfg *GlobalConfig, opts *ListOptions) ([]*ReleaseInfo, error) {
	m.LastListOpts = opts
	m.LastConfig = copyConfig(cfg)
	if m.ListFn != nil {
		return m.ListFn(ctx, cfg, opts)
	}
	return []*ReleaseInfo{DefaultRelease()}, nil
}

func (m *MockEngine) Status(ctx context.Context, cfg *GlobalConfig, opts *StatusOptions) (*ReleaseInfo, error) {
	m.LastStatusOpts = opts
	m.LastConfig = copyConfig(cfg)
	if m.StatusFn != nil {
		return m.StatusFn(ctx, cfg, opts)
	}
	return DefaultRelease(), nil
}

func (m *MockEngine) History(ctx context.Context, cfg *GlobalConfig, opts *HistoryOptions) ([]*ReleaseInfo, error) {
	m.LastHistoryOpts = opts
	m.LastConfig = copyConfig(cfg)
	if m.HistoryFn != nil {
		return m.HistoryFn(ctx, cfg, opts)
	}
	r1 := DefaultRelease()
	r2 := DefaultRelease()
	r2.Revision = 2
	r2.Status = "superseded"
	return []*ReleaseInfo{r1, r2}, nil
}

func (m *MockEngine) Test(ctx context.Context, cfg *GlobalConfig, opts *TestOptions) (*ReleaseInfo, error) {
	m.LastTestOpts = opts
	m.LastConfig = copyConfig(cfg)
	if m.TestFn != nil {
		return m.TestFn(ctx, cfg, opts)
	}
	return DefaultRelease(), nil
}

func (m *MockEngine) GetAll(ctx context.Context, cfg *GlobalConfig, opts *GetOptions) (*ReleaseDetail, error) {
	m.LastGetOpts = opts
	m.LastConfig = copyConfig(cfg)
	if m.GetAllFn != nil {
		return m.GetAllFn(ctx, cfg, opts)
	}
	return &ReleaseDetail{
		Release:  DefaultRelease(),
		Values:   map[string]interface{}{"replicaCount": 1},
		Manifest: "---\napiVersion: v1\nkind: Service",
		Hooks:    "---\n# hook manifest",
		Notes:    "Release notes here",
	}, nil
}

func (m *MockEngine) GetValues(ctx context.Context, cfg *GlobalConfig, opts *GetValuesOptions) (map[string]interface{}, error) {
	m.LastGetValuesOpts = opts
	m.LastConfig = copyConfig(cfg)
	if m.GetValuesFn != nil {
		return m.GetValuesFn(ctx, cfg, opts)
	}
	return map[string]interface{}{"replicaCount": 1, "image": "nginx:latest"}, nil
}

func (m *MockEngine) GetMetadata(ctx context.Context, cfg *GlobalConfig, opts *GetOptions) (*MetadataInfo, error) {
	m.LastGetOpts = opts
	m.LastConfig = copyConfig(cfg)
	if m.GetMetadataFn != nil {
		return m.GetMetadataFn(ctx, cfg, opts)
	}
	return &MetadataInfo{
		Name:         "my-release",
		Namespace:    "default",
		Revision:     1,
		Status:       "deployed",
		Chart:        "nginx",
		ChartVersion: "1.0.0",
		AppVersion:   defaultMockAppVersion,
		DeployedAt:   time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}, nil
}

func (m *MockEngine) GetManifest(ctx context.Context, cfg *GlobalConfig, opts *GetOptions) (string, error) {
	m.LastGetOpts = opts
	m.LastConfig = copyConfig(cfg)
	if m.GetManifestFn != nil {
		return m.GetManifestFn(ctx, cfg, opts)
	}
	return "---\napiVersion: v1\nkind: Service\nmetadata:\n  name: my-release-nginx", nil
}

func (m *MockEngine) GetHooks(ctx context.Context, cfg *GlobalConfig, opts *GetOptions) (string, error) {
	m.LastGetOpts = opts
	m.LastConfig = copyConfig(cfg)
	if m.GetHooksFn != nil {
		return m.GetHooksFn(ctx, cfg, opts)
	}
	return "---\n# Source: nginx/templates/tests/test-connection.yaml", nil
}

func (m *MockEngine) GetNotes(ctx context.Context, cfg *GlobalConfig, opts *GetOptions) (string, error) {
	m.LastGetOpts = opts
	m.LastConfig = copyConfig(cfg)
	if m.GetNotesFn != nil {
		return m.GetNotesFn(ctx, cfg, opts)
	}
	return "NOTES:\n1. Get the application URL", nil
}

func (m *MockEngine) Create(ctx context.Context, opts *CreateOptions) (string, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, opts)
	}
	return "Creating /tmp/" + opts.Name, nil
}

func (m *MockEngine) Lint(ctx context.Context, opts *LintOptions) (*LintResult, error) {
	if m.LintFn != nil {
		return m.LintFn(ctx, opts)
	}
	return &LintResult{TotalCharts: 1, Passed: true, Messages: []LintMessage{}}, nil
}

func (m *MockEngine) Template(ctx context.Context, cfg *GlobalConfig, opts *TemplateOptions) (string, error) {
	m.LastConfig = copyConfig(cfg)
	if m.TemplateFn != nil {
		return m.TemplateFn(ctx, cfg, opts)
	}
	return "---\napiVersion: v1\nkind: Service\nmetadata:\n  name: " + opts.ReleaseName, nil
}

func (m *MockEngine) Package(ctx context.Context, opts *PackageOptions) (string, error) {
	if m.PackageFn != nil {
		return m.PackageFn(ctx, opts)
	}
	return "Successfully packaged chart and saved to: /tmp/mychart-0.1.0.tgz", nil
}

func (m *MockEngine) Pull(ctx context.Context, cfg *GlobalConfig, opts *PullOptions) (string, error) {
	m.LastConfig = copyConfig(cfg)
	m.LastPullOpts = copyPullOptions(opts)
	if m.PullFn != nil {
		return m.PullFn(ctx, cfg, opts)
	}
	return "", nil
}

func (m *MockEngine) Push(ctx context.Context, cfg *GlobalConfig, opts *PushOptions) (string, error) {
	m.LastConfig = copyConfig(cfg)
	if m.PushFn != nil {
		return m.PushFn(ctx, cfg, opts)
	}
	return "", nil
}

func (m *MockEngine) Verify(ctx context.Context, opts *VerifyOptions) (string, error) {
	if m.VerifyFn != nil {
		return m.VerifyFn(ctx, opts)
	}
	return "Signed by: Test User <test@example.com>", nil
}

func (m *MockEngine) ShowAll(ctx context.Context, cfg *GlobalConfig, opts *ShowOptions) (string, error) {
	m.LastConfig = copyConfig(cfg)
	if m.ShowAllFn != nil {
		return m.ShowAllFn(ctx, cfg, opts)
	}
	return "apiVersion: v2\nname: " + opts.Chart + "\nversion: 1.0.0", nil
}

func (m *MockEngine) ShowChart(ctx context.Context, cfg *GlobalConfig, opts *ShowOptions) (string, error) {
	m.LastConfig = copyConfig(cfg)
	if m.ShowChartFn != nil {
		return m.ShowChartFn(ctx, cfg, opts)
	}
	return "apiVersion: v2\nname: " + opts.Chart, nil
}

func (m *MockEngine) ShowValues(ctx context.Context, cfg *GlobalConfig, opts *ShowOptions) (string, error) {
	m.LastConfig = copyConfig(cfg)
	if m.ShowValuesFn != nil {
		return m.ShowValuesFn(ctx, cfg, opts)
	}
	return "replicaCount: 1\nimage:\n  repository: nginx", nil
}

func (m *MockEngine) ShowReadme(ctx context.Context, cfg *GlobalConfig, opts *ShowOptions) (string, error) {
	m.LastConfig = copyConfig(cfg)
	if m.ShowReadmeFn != nil {
		return m.ShowReadmeFn(ctx, cfg, opts)
	}
	return "# " + opts.Chart + "\n\nA Helm chart", nil
}

func (m *MockEngine) ShowCRDs(ctx context.Context, cfg *GlobalConfig, opts *ShowOptions) (string, error) {
	m.LastConfig = copyConfig(cfg)
	if m.ShowCRDsFn != nil {
		return m.ShowCRDsFn(ctx, cfg, opts)
	}
	return "", nil
}

func (m *MockEngine) DependencyBuild(ctx context.Context, cfg *GlobalConfig, opts *DependencyOptions) error {
	m.LastConfig = copyConfig(cfg)
	if m.DependencyBuildFn != nil {
		return m.DependencyBuildFn(ctx, cfg, opts)
	}
	return nil
}

func (m *MockEngine) DependencyList(ctx context.Context, cfg *GlobalConfig, opts *DependencyOptions) (string, error) {
	m.LastConfig = copyConfig(cfg)
	if m.DependencyListFn != nil {
		return m.DependencyListFn(ctx, cfg, opts)
	}
	return "NAME\tVERSION\tREPOSITORY\tSTATUS\nredis\t17.0.0\thttps://charts.bitnami.com/bitnami\tok", nil
}

func (m *MockEngine) DependencyUpdate(ctx context.Context, cfg *GlobalConfig, opts *DependencyOptions) error {
	m.LastConfig = copyConfig(cfg)
	if m.DependencyUpdateFn != nil {
		return m.DependencyUpdateFn(ctx, cfg, opts)
	}
	return nil
}

func (m *MockEngine) RepoAdd(ctx context.Context, opts *RepoAddOptions) error {
	m.LastRepoAddOpts = copyRepoAddOptions(opts)
	if m.RepoAddFn != nil {
		return m.RepoAddFn(ctx, opts)
	}
	return nil
}

func (m *MockEngine) RepoList(ctx context.Context, opts *RepoListOptions) ([]*RepoEntry, error) {
	if m.RepoListFn != nil {
		return m.RepoListFn(ctx, opts)
	}
	return []*RepoEntry{
		{Name: "stable", URL: "https://charts.helm.sh/stable"},
		{Name: "bitnami", URL: "https://charts.bitnami.com/bitnami"},
	}, nil
}

func (m *MockEngine) RepoUpdate(ctx context.Context, opts *RepoUpdateOptions) (string, error) {
	if m.RepoUpdateFn != nil {
		return m.RepoUpdateFn(ctx, opts)
	}
	return "Update Complete. Happy Helming!", nil
}

func (m *MockEngine) RepoRemove(ctx context.Context, opts *RepoRemoveOptions) error {
	if m.RepoRemoveFn != nil {
		return m.RepoRemoveFn(ctx, opts)
	}
	return nil
}

func (m *MockEngine) RepoIndex(ctx context.Context, opts *RepoIndexOptions) error {
	if m.RepoIndexFn != nil {
		return m.RepoIndexFn(ctx, opts)
	}
	return nil
}

func (m *MockEngine) RegistryLogin(ctx context.Context, opts *RegistryLoginOptions) error {
	m.LastRegistryLoginOpts = copyRegistryLoginOptions(opts)
	if m.RegistryLoginFn != nil {
		return m.RegistryLoginFn(ctx, opts)
	}
	return nil
}

func (m *MockEngine) RegistryLogout(ctx context.Context, opts *RegistryLogoutOptions) error {
	if m.RegistryLogoutFn != nil {
		return m.RegistryLogoutFn(ctx, opts)
	}
	return nil
}

func (m *MockEngine) SearchHub(ctx context.Context, opts *SearchHubOptions) ([]*SearchResult, error) {
	if m.SearchHubFn != nil {
		return m.SearchHubFn(ctx, opts)
	}
	return []*SearchResult{
		{Name: "nginx", ChartVersion: "1.0.0", AppVersion: defaultMockAppVersion, Description: "An nginx chart"},
	}, nil
}

func (m *MockEngine) SearchRepo(ctx context.Context, opts *SearchRepoOptions) ([]*SearchResult, error) {
	if m.SearchRepoFn != nil {
		return m.SearchRepoFn(ctx, opts)
	}
	return []*SearchResult{
		{Name: "stable/nginx", ChartVersion: "1.0.0", AppVersion: defaultMockAppVersion, Description: "An nginx chart"},
	}, nil
}

func (m *MockEngine) PluginInstall(ctx context.Context, opts *PluginInstallOptions) error {
	if m.PluginInstallFn != nil {
		return m.PluginInstallFn(ctx, opts)
	}
	return nil
}

func (m *MockEngine) PluginList(ctx context.Context) ([]*PluginInfo, error) {
	if m.PluginListFn != nil {
		return m.PluginListFn(ctx)
	}
	return []*PluginInfo{
		{Name: "diff", Version: "3.8.1", Description: "Preview helm upgrade changes"},
	}, nil
}

func (m *MockEngine) PluginUninstall(ctx context.Context, opts *PluginUninstallOptions) error {
	if m.PluginUninstallFn != nil {
		return m.PluginUninstallFn(ctx, opts)
	}
	return nil
}

func (m *MockEngine) PluginUpdate(ctx context.Context, opts *PluginUpdateOptions) error {
	if m.PluginUpdateFn != nil {
		return m.PluginUpdateFn(ctx, opts)
	}
	return nil
}

func (m *MockEngine) Env(ctx context.Context) (map[string]string, error) {
	if m.EnvFn != nil {
		return m.EnvFn(ctx)
	}
	return map[string]string{
		"HELM_CACHE_HOME":  "/home/user/.cache/helm",
		"HELM_CONFIG_HOME": "/home/user/.config/helm",
		"HELM_DATA_HOME":   "/home/user/.local/share/helm",
		"HELM_DRIVER":      "secret",
	}, nil
}

func (m *MockEngine) Version(ctx context.Context) (*VersionInfo, error) {
	if m.VersionFn != nil {
		return m.VersionFn(ctx)
	}
	return &VersionInfo{
		Version:   "v3.20.0",
		GitCommit: "abc123",
		GoVersion: "go1.26",
	}, nil
}
