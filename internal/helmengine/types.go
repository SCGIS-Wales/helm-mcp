package helmengine

import "time"

// ReleaseInfo is the version-agnostic representation of a Helm release.
type ReleaseInfo struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	Revision     int               `json:"revision"`
	Status       string            `json:"status"`
	Chart        string            `json:"chart"`
	ChartVersion string            `json:"chart_version"`
	AppVersion   string            `json:"app_version"`
	Description  string            `json:"description,omitempty"`
	Updated      time.Time         `json:"updated"`
	Notes        string            `json:"notes,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
}

// ReleaseDetail contains full information for a release (used by get all).
type ReleaseDetail struct {
	Release  *ReleaseInfo           `json:"release"`
	Values   map[string]interface{} `json:"values,omitempty"`
	Manifest string                 `json:"manifest,omitempty"`
	Hooks    string                 `json:"hooks,omitempty"`
	Notes    string                 `json:"notes,omitempty"`
}

// MetadataInfo contains release metadata.
type MetadataInfo struct {
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	Revision     int       `json:"revision"`
	Status       string    `json:"status"`
	Chart        string    `json:"chart"`
	ChartVersion string    `json:"chart_version"`
	AppVersion   string    `json:"app_version"`
	DeployedAt   time.Time `json:"deployed_at"`
}

// UninstallResult contains the result of an uninstall operation.
type UninstallResult struct {
	ReleaseName string `json:"release_name"`
	Info        string `json:"info,omitempty"`
}

// LintResult contains the result of a lint operation.
type LintResult struct {
	TotalCharts int           `json:"total_charts"`
	Messages    []LintMessage `json:"messages"`
	Passed      bool          `json:"passed"`
}

// LintMessage represents a single lint message.
type LintMessage struct {
	Severity string `json:"severity"` // "ERROR", "WARNING", "INFO"
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

// RepoEntry represents a configured chart repository.
type RepoEntry struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// SearchResult represents a search result from hub or repo.
type SearchResult struct {
	Name         string `json:"name"`
	ChartVersion string `json:"chart_version"`
	AppVersion   string `json:"app_version"`
	Description  string `json:"description"`
	URL          string `json:"url,omitempty"`
}

// PluginInfo represents an installed Helm plugin.
type PluginInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// VersionInfo contains Helm version information.
type VersionInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit,omitempty"`
	GoVersion string `json:"go_version,omitempty"`
}

// InstallOptions contains options for helm install.
type InstallOptions struct {
	ReleaseName      string                 `json:"release_name"`
	Chart            string                 `json:"chart"`
	Version          string                 `json:"version,omitempty"`
	Values           map[string]interface{} `json:"values,omitempty"`
	ValuesFiles      []string               `json:"values_files,omitempty"`
	CreateNamespace  bool                   `json:"create_namespace,omitempty"`
	Wait             bool                   `json:"wait,omitempty"`
	WaitForJobs      bool                   `json:"wait_for_jobs,omitempty"`
	Timeout          string                 `json:"timeout,omitempty"`
	DryRun           string                 `json:"dry_run,omitempty"`
	Description      string                 `json:"description,omitempty"`
	DisableHooks     bool                   `json:"disable_hooks,omitempty"`
	Replace          bool                   `json:"replace,omitempty"`
	SkipCRDs         bool                   `json:"skip_crds,omitempty"`
	IncludeCRDs      bool                   `json:"include_crds,omitempty"`
	DependencyUpdate bool                   `json:"dependency_update,omitempty"`
	GenerateName     bool                   `json:"generate_name,omitempty"`
	NameTemplate     string                 `json:"name_template,omitempty"`
	Labels           map[string]string      `json:"labels,omitempty"`
	// v4-specific
	ServerSideApply   bool `json:"server_side_apply,omitempty"`
	TakeOwnership     bool `json:"take_ownership,omitempty"`
	RollbackOnFailure bool `json:"rollback_on_failure,omitempty"`
	HideSecret        bool `json:"hide_secret,omitempty"`
	ForceConflicts    bool `json:"force_conflicts,omitempty"`
}

// UpgradeOptions contains options for helm upgrade.
type UpgradeOptions struct {
	ReleaseName          string                 `json:"release_name"`
	Chart                string                 `json:"chart"`
	Version              string                 `json:"version,omitempty"`
	Values               map[string]interface{} `json:"values,omitempty"`
	ValuesFiles          []string               `json:"values_files,omitempty"`
	Install              bool                   `json:"install,omitempty"`
	Force                bool                   `json:"force,omitempty"`
	ResetValues          bool                   `json:"reset_values,omitempty"`
	ReuseValues          bool                   `json:"reuse_values,omitempty"`
	Wait                 bool                   `json:"wait,omitempty"`
	WaitForJobs          bool                   `json:"wait_for_jobs,omitempty"`
	Timeout              string                 `json:"timeout,omitempty"`
	DryRun               string                 `json:"dry_run,omitempty"`
	Description          string                 `json:"description,omitempty"`
	DisableHooks         bool                   `json:"disable_hooks,omitempty"`
	SkipCRDs             bool                   `json:"skip_crds,omitempty"`
	CleanupOnFail        bool                   `json:"cleanup_on_fail,omitempty"`
	DependencyUpdate     bool                   `json:"dependency_update,omitempty"`
	Labels               map[string]string      `json:"labels,omitempty"`
	MaxHistory           int                    `json:"max_history,omitempty"`
	ResetThenReuseValues bool                   `json:"reset_then_reuse_values,omitempty"`
	// v4-specific
	ServerSideApply bool `json:"server_side_apply,omitempty"`
	TakeOwnership   bool `json:"take_ownership,omitempty"`
	HideSecret      bool `json:"hide_secret,omitempty"`
	ForceConflicts  bool `json:"force_conflicts,omitempty"`
}

// UninstallOptions contains options for helm uninstall.
type UninstallOptions struct {
	ReleaseName  string `json:"release_name"`
	KeepHistory  bool   `json:"keep_history,omitempty"`
	DryRun       bool   `json:"dry_run,omitempty"`
	Wait         bool   `json:"wait,omitempty"`
	Timeout      string `json:"timeout,omitempty"`
	DisableHooks bool   `json:"disable_hooks,omitempty"`
	Cascade      string `json:"cascade,omitempty"`
}

// RollbackOptions contains options for helm rollback.
type RollbackOptions struct {
	ReleaseName   string `json:"release_name"`
	Revision      int    `json:"revision"`
	Wait          bool   `json:"wait,omitempty"`
	WaitForJobs   bool   `json:"wait_for_jobs,omitempty"`
	Timeout       string `json:"timeout,omitempty"`
	Force         bool   `json:"force,omitempty"`
	DryRun        bool   `json:"dry_run,omitempty"`
	DisableHooks  bool   `json:"disable_hooks,omitempty"`
	CleanupOnFail bool   `json:"cleanup_on_fail,omitempty"`
	MaxHistory    int    `json:"max_history,omitempty"`
	// v4-specific
	ServerSideApply bool `json:"server_side_apply,omitempty"`
	ForceConflicts  bool `json:"force_conflicts,omitempty"`
}

// ListOptions contains options for helm list.
type ListOptions struct {
	AllNamespaces bool   `json:"all_namespaces,omitempty"`
	Filter        string `json:"filter,omitempty"`
	Selector      string `json:"selector,omitempty"` // v4 only
	SortBy        string `json:"sort_by,omitempty"`
	SortReverse   bool   `json:"sort_reverse,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Offset        int    `json:"offset,omitempty"`
	Deployed      bool   `json:"deployed,omitempty"`
	Failed        bool   `json:"failed,omitempty"`
	Pending       bool   `json:"pending,omitempty"`
	Uninstalled   bool   `json:"uninstalled,omitempty"`
	Superseded    bool   `json:"superseded,omitempty"`
}

// StatusOptions contains options for helm status.
type StatusOptions struct {
	ReleaseName   string `json:"release_name"`
	Revision      int    `json:"revision,omitempty"`
	ShowResources bool   `json:"show_resources,omitempty"` // v4 only
}

// HistoryOptions contains options for helm history.
type HistoryOptions struct {
	ReleaseName string `json:"release_name"`
	Max         int    `json:"max,omitempty"`
}

// TestOptions contains options for helm test.
type TestOptions struct {
	ReleaseName string   `json:"release_name"`
	Timeout     string   `json:"timeout,omitempty"`
	Filters     []string `json:"filters,omitempty"`
}

// GetOptions contains options for helm get subcommands.
type GetOptions struct {
	ReleaseName string `json:"release_name"`
	Revision    int    `json:"revision,omitempty"`
}

// GetValuesOptions extends GetOptions with a flag for all values.
type GetValuesOptions struct {
	ReleaseName string `json:"release_name"`
	Revision    int    `json:"revision,omitempty"`
	All         bool   `json:"all,omitempty"`
}

// CreateOptions contains options for helm create.
type CreateOptions struct {
	Name    string `json:"name"`
	Starter string `json:"starter,omitempty"`
}

// LintOptions contains options for helm lint.
type LintOptions struct {
	Paths         []string               `json:"paths"`
	Values        map[string]interface{} `json:"values,omitempty"`
	ValuesFiles   []string               `json:"values_files,omitempty"`
	Strict        bool                   `json:"strict,omitempty"`
	WithSubcharts bool                   `json:"with_subcharts,omitempty"`
	Quiet         bool                   `json:"quiet,omitempty"`
	Namespace     string                 `json:"namespace,omitempty"`
	KubeVersion   string                 `json:"kube_version,omitempty"`
}

// TemplateOptions contains options for helm template.
type TemplateOptions struct {
	ReleaseName      string                 `json:"release_name"`
	Chart            string                 `json:"chart"`
	Version          string                 `json:"version,omitempty"`
	Values           map[string]interface{} `json:"values,omitempty"`
	ValuesFiles      []string               `json:"values_files,omitempty"`
	ShowOnly         []string               `json:"show_only,omitempty"`
	Validate         bool                   `json:"validate,omitempty"`
	KubeVersion      string                 `json:"kube_version,omitempty"`
	APIVersions      []string               `json:"api_versions,omitempty"`
	IncludeCRDs      bool                   `json:"include_crds,omitempty"`
	SkipCRDs         bool                   `json:"skip_crds,omitempty"`
	NoHooks          bool                   `json:"no_hooks,omitempty"`
	DependencyUpdate bool                   `json:"dependency_update,omitempty"`
}

// PackageOptions contains options for helm package.
type PackageOptions struct {
	Path             string `json:"path"`
	Destination      string `json:"destination,omitempty"`
	Version          string `json:"version,omitempty"`
	AppVersion       string `json:"app_version,omitempty"`
	Sign             bool   `json:"sign,omitempty"`
	Key              string `json:"key,omitempty"`
	Keyring          string `json:"keyring,omitempty"`
	PassphraseFile   string `json:"passphrase_file,omitempty"`
	DependencyUpdate bool   `json:"dependency_update,omitempty"`
}

// PullOptions contains options for helm pull.
type PullOptions struct {
	Chart                string `json:"chart"`
	Version              string `json:"version,omitempty"`
	Repo                 string `json:"repo,omitempty"`
	Destination          string `json:"destination,omitempty"`
	Untar                bool   `json:"untar,omitempty"`
	UntarDir             string `json:"untar_dir,omitempty"`
	Verify               bool   `json:"verify,omitempty"`
	Keyring              string `json:"keyring,omitempty"`
	Username             string `json:"username,omitempty"`
	Password             string `json:"password,omitempty"`
	PlainHTTP            bool   `json:"plain_http,omitempty"`
	InsecureSkipTLSVerify bool  `json:"insecure_skip_tls_verify,omitempty"`
}

// PushOptions contains options for helm push.
type PushOptions struct {
	ChartRef              string `json:"chart_ref"`
	Remote                string `json:"remote"`
	PlainHTTP             bool   `json:"plain_http,omitempty"`
	InsecureSkipTLSVerify bool   `json:"insecure_skip_tls_verify,omitempty"`
	CAFile                string `json:"ca_file,omitempty"`
	CertFile              string `json:"cert_file,omitempty"`
	KeyFile               string `json:"key_file,omitempty"`
}

// VerifyOptions contains options for helm verify.
type VerifyOptions struct {
	ChartFile string `json:"chart_file"`
	Keyring   string `json:"keyring,omitempty"`
}

// ShowOptions contains options for helm show subcommands.
type ShowOptions struct {
	Chart    string `json:"chart"`
	Version  string `json:"version,omitempty"`
	Repo     string `json:"repo,omitempty"`
	Devel    bool   `json:"devel,omitempty"`
	JSONPath string `json:"jsonpath,omitempty"` // v4 only
}

// DependencyOptions contains options for helm dependency subcommands.
type DependencyOptions struct {
	ChartPath   string `json:"chart_path"`
	Verify      bool   `json:"verify,omitempty"`
	Keyring     string `json:"keyring,omitempty"`
	SkipRefresh bool   `json:"skip_refresh,omitempty"`
}

// RepoAddOptions contains options for helm repo add.
type RepoAddOptions struct {
	Name                  string `json:"name"`
	URL                   string `json:"url"`
	Username              string `json:"username,omitempty"`
	Password              string `json:"password,omitempty"`
	ForceUpdate           bool   `json:"force_update,omitempty"`
	CAFile                string `json:"ca_file,omitempty"`
	CertFile              string `json:"cert_file,omitempty"`
	KeyFile               string `json:"key_file,omitempty"`
	InsecureSkipTLSVerify bool   `json:"insecure_skip_tls_verify,omitempty"`
	PassCredentialsAll    bool   `json:"pass_credentials_all,omitempty"`
}

// RepoListOptions contains options for helm repo list (currently none).
type RepoListOptions struct{}

// RepoUpdateOptions contains options for helm repo update.
type RepoUpdateOptions struct {
	Names []string `json:"names,omitempty"`
}

// RepoRemoveOptions contains options for helm repo remove.
type RepoRemoveOptions struct {
	Names []string `json:"names"`
}

// RepoIndexOptions contains options for helm repo index.
type RepoIndexOptions struct {
	Directory string `json:"directory"`
	URL       string `json:"url,omitempty"`
	Merge     string `json:"merge,omitempty"`
}

// RegistryLoginOptions contains options for helm registry login.
type RegistryLoginOptions struct {
	Hostname              string `json:"hostname"`
	Username              string `json:"username,omitempty"`
	Password              string `json:"password,omitempty"`
	Insecure              bool   `json:"insecure,omitempty"`
	PlainHTTP             bool   `json:"plain_http,omitempty"`
	CAFile                string `json:"ca_file,omitempty"`
	CertFile              string `json:"cert_file,omitempty"`
	KeyFile               string `json:"key_file,omitempty"`
}

// RegistryLogoutOptions contains options for helm registry logout.
type RegistryLogoutOptions struct {
	Hostname string `json:"hostname"`
}

// SearchHubOptions contains options for helm search hub.
type SearchHubOptions struct {
	Keyword      string `json:"keyword"`
	MaxColWidth  int    `json:"max_col_width,omitempty"`
	ListRepoURL  bool   `json:"list_repo_url,omitempty"`
}

// SearchRepoOptions contains options for helm search repo.
type SearchRepoOptions struct {
	Keyword           string `json:"keyword"`
	Regexp            bool   `json:"regexp,omitempty"`
	Versions          bool   `json:"versions,omitempty"`
	Devel             bool   `json:"devel,omitempty"`
	VersionConstraint string `json:"version_constraint,omitempty"`
}

// PluginInstallOptions contains options for helm plugin install.
type PluginInstallOptions struct {
	URLOrPath string `json:"url_or_path"`
	Version   string `json:"version,omitempty"`
}

// PluginUninstallOptions contains options for helm plugin uninstall.
type PluginUninstallOptions struct {
	Name string `json:"name"`
}

// PluginUpdateOptions contains options for helm plugin update.
type PluginUpdateOptions struct {
	Name string `json:"name"`
}

// ZeroPassword zeroes the Password field after use.
func (o *PullOptions) ZeroPassword() {
	if o != nil {
		zeroString(&o.Password)
	}
}

// ZeroPassword zeroes the Password field after use.
func (o *RepoAddOptions) ZeroPassword() {
	if o != nil {
		zeroString(&o.Password)
	}
}

// ZeroPassword zeroes the Password field after use.
func (o *RegistryLoginOptions) ZeroPassword() {
	if o != nil {
		zeroString(&o.Password)
	}
}
