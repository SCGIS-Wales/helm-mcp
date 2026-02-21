package helmengine

import "context"

// Engine defines the version-agnostic interface for all Helm operations.
// Two implementations exist: V3Engine (helm.sh/helm/v3) and V4Engine (helm.sh/helm/v4).
type Engine interface {
	// Release Management
	Install(ctx context.Context, cfg *GlobalConfig, opts *InstallOptions) (*ReleaseInfo, error)
	Upgrade(ctx context.Context, cfg *GlobalConfig, opts *UpgradeOptions) (*ReleaseInfo, error)
	Uninstall(ctx context.Context, cfg *GlobalConfig, opts *UninstallOptions) (*UninstallResult, error)
	Rollback(ctx context.Context, cfg *GlobalConfig, opts *RollbackOptions) error
	List(ctx context.Context, cfg *GlobalConfig, opts *ListOptions) ([]*ReleaseInfo, error)
	Status(ctx context.Context, cfg *GlobalConfig, opts *StatusOptions) (*ReleaseInfo, error)
	History(ctx context.Context, cfg *GlobalConfig, opts *HistoryOptions) ([]*ReleaseInfo, error)
	Test(ctx context.Context, cfg *GlobalConfig, opts *TestOptions) (*ReleaseInfo, error)
	GetAll(ctx context.Context, cfg *GlobalConfig, opts *GetOptions) (*ReleaseDetail, error)
	GetValues(ctx context.Context, cfg *GlobalConfig, opts *GetValuesOptions) (map[string]interface{}, error)
	GetMetadata(ctx context.Context, cfg *GlobalConfig, opts *GetOptions) (*MetadataInfo, error)
	GetManifest(ctx context.Context, cfg *GlobalConfig, opts *GetOptions) (string, error)
	GetHooks(ctx context.Context, cfg *GlobalConfig, opts *GetOptions) (string, error)
	GetNotes(ctx context.Context, cfg *GlobalConfig, opts *GetOptions) (string, error)

	// Chart Management
	Create(ctx context.Context, opts *CreateOptions) (string, error)
	Lint(ctx context.Context, opts *LintOptions) (*LintResult, error)
	Template(ctx context.Context, cfg *GlobalConfig, opts *TemplateOptions) (string, error)
	Package(ctx context.Context, opts *PackageOptions) (string, error)
	Pull(ctx context.Context, cfg *GlobalConfig, opts *PullOptions) (string, error)
	Push(ctx context.Context, cfg *GlobalConfig, opts *PushOptions) (string, error)
	Verify(ctx context.Context, opts *VerifyOptions) (string, error)
	ShowAll(ctx context.Context, cfg *GlobalConfig, opts *ShowOptions) (string, error)
	ShowChart(ctx context.Context, cfg *GlobalConfig, opts *ShowOptions) (string, error)
	ShowValues(ctx context.Context, cfg *GlobalConfig, opts *ShowOptions) (string, error)
	ShowReadme(ctx context.Context, cfg *GlobalConfig, opts *ShowOptions) (string, error)
	ShowCRDs(ctx context.Context, cfg *GlobalConfig, opts *ShowOptions) (string, error)
	DependencyBuild(ctx context.Context, cfg *GlobalConfig, opts *DependencyOptions) error
	DependencyList(ctx context.Context, cfg *GlobalConfig, opts *DependencyOptions) (string, error)
	DependencyUpdate(ctx context.Context, cfg *GlobalConfig, opts *DependencyOptions) error

	// Repository Management
	RepoAdd(ctx context.Context, opts *RepoAddOptions) error
	RepoList(ctx context.Context, opts *RepoListOptions) ([]*RepoEntry, error)
	RepoUpdate(ctx context.Context, opts *RepoUpdateOptions) (string, error)
	RepoRemove(ctx context.Context, opts *RepoRemoveOptions) error
	RepoIndex(ctx context.Context, opts *RepoIndexOptions) error

	// Registry (OCI)
	RegistryLogin(ctx context.Context, opts *RegistryLoginOptions) error
	RegistryLogout(ctx context.Context, opts *RegistryLogoutOptions) error

	// Search
	SearchHub(ctx context.Context, opts *SearchHubOptions) ([]*SearchResult, error)
	SearchRepo(ctx context.Context, opts *SearchRepoOptions) ([]*SearchResult, error)

	// Plugin Management
	PluginInstall(ctx context.Context, opts *PluginInstallOptions) error
	PluginList(ctx context.Context) ([]*PluginInfo, error)
	PluginUninstall(ctx context.Context, opts *PluginUninstallOptions) error
	PluginUpdate(ctx context.Context, opts *PluginUpdateOptions) error

	// Environment
	Env(ctx context.Context) (map[string]string, error)
	Version(ctx context.Context) (*VersionInfo, error)
}
