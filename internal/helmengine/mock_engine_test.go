package helmengine

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Original tests (kept as-is)
// ---------------------------------------------------------------------------

func TestMockEngineImplementsInterface(t *testing.T) {
	var _ Engine = &MockEngine{}
}

func TestDefaultRelease(t *testing.T) {
	r := DefaultRelease()
	if r.Name != "my-release" {
		t.Errorf("expected name 'my-release', got %q", r.Name)
	}
	if r.Status != "deployed" {
		t.Errorf("expected status 'deployed', got %q", r.Status)
	}
	if r.Revision != 1 {
		t.Errorf("expected revision 1, got %d", r.Revision)
	}
}

func TestMockEngineCallTracking(t *testing.T) {
	m := &MockEngine{}
	ctx := context.Background()

	cfg := &GlobalConfig{Namespace: "test-ns"}
	_, _ = m.Install(ctx, cfg, &InstallOptions{ReleaseName: "test", Chart: "nginx"})
	if m.LastInstallOpts == nil || m.LastInstallOpts.ReleaseName != "test" {
		t.Error("expected install opts to be tracked")
	}
	if m.LastConfig == nil || m.LastConfig.Namespace != "test-ns" {
		t.Error("expected config to be tracked")
	}
}

func TestMockEngineCustomFunctions(t *testing.T) {
	m := &MockEngine{
		ListFn: func(ctx context.Context, cfg *GlobalConfig, opts *ListOptions) ([]*ReleaseInfo, error) {
			return []*ReleaseInfo{
				{Name: "custom-release", Status: "failed"},
			}, nil
		},
	}

	result, err := m.List(context.Background(), &GlobalConfig{}, &ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Name != "custom-release" {
		t.Errorf("expected custom result, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// DefaultRelease: verify AppVersion uses constant
// ---------------------------------------------------------------------------

func TestDefaultReleaseAppVersion(t *testing.T) {
	r := DefaultRelease()
	if r.AppVersion != defaultMockAppVersion {
		t.Errorf("expected AppVersion %q, got %q", defaultMockAppVersion, r.AppVersion)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func cfg() *GlobalConfig          { return &GlobalConfig{Namespace: "ns"} }
func bg() context.Context         { return context.Background() }
var errMock = errors.New("mock-err")

// ---------------------------------------------------------------------------
// Default (nil Fn) path tests -- exercises every method and its call tracking
// ---------------------------------------------------------------------------

func TestInstallDefault(t *testing.T) {
	m := &MockEngine{}
	r, err := m.Install(bg(), cfg(), &InstallOptions{ReleaseName: "r1", Chart: "c"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "my-release" {
		t.Errorf("unexpected name %q", r.Name)
	}
	if m.LastInstallOpts == nil || m.LastInstallOpts.ReleaseName != "r1" {
		t.Error("install opts not tracked")
	}
	if m.LastConfig == nil || m.LastConfig.Namespace != "ns" {
		t.Error("config not tracked")
	}
}

func TestUpgradeDefault(t *testing.T) {
	m := &MockEngine{}
	r, err := m.Upgrade(bg(), cfg(), &UpgradeOptions{ReleaseName: "r1", Chart: "c"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Revision != 2 {
		t.Errorf("expected revision 2, got %d", r.Revision)
	}
	if m.LastUpgradeOpts == nil || m.LastUpgradeOpts.ReleaseName != "r1" {
		t.Error("upgrade opts not tracked")
	}
}

func TestUninstallDefault(t *testing.T) {
	m := &MockEngine{}
	res, err := m.Uninstall(bg(), cfg(), &UninstallOptions{ReleaseName: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if res.ReleaseName != "r1" {
		t.Errorf("expected release name 'r1', got %q", res.ReleaseName)
	}
	if res.Info != "release uninstalled" {
		t.Errorf("unexpected info %q", res.Info)
	}
	if m.LastUninstallOpts == nil || m.LastUninstallOpts.ReleaseName != "r1" {
		t.Error("uninstall opts not tracked")
	}
}

func TestRollbackDefault(t *testing.T) {
	m := &MockEngine{}
	err := m.Rollback(bg(), cfg(), &RollbackOptions{ReleaseName: "r1", Revision: 1})
	if err != nil {
		t.Fatal(err)
	}
	if m.LastRollbackOpts == nil || m.LastRollbackOpts.ReleaseName != "r1" {
		t.Error("rollback opts not tracked")
	}
}

func TestListDefault(t *testing.T) {
	m := &MockEngine{}
	list, err := m.List(bg(), cfg(), &ListOptions{Filter: "f"})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "my-release" {
		t.Error("unexpected list result")
	}
	if m.LastListOpts == nil || m.LastListOpts.Filter != "f" {
		t.Error("list opts not tracked")
	}
}

func TestStatusDefault(t *testing.T) {
	m := &MockEngine{}
	r, err := m.Status(bg(), cfg(), &StatusOptions{ReleaseName: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "my-release" {
		t.Errorf("unexpected name %q", r.Name)
	}
	if m.LastStatusOpts == nil || m.LastStatusOpts.ReleaseName != "r1" {
		t.Error("status opts not tracked")
	}
}

func TestHistoryDefault(t *testing.T) {
	m := &MockEngine{}
	list, err := m.History(bg(), cfg(), &HistoryOptions{ReleaseName: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(list))
	}
	if list[0].Revision != 1 || list[1].Revision != 2 {
		t.Error("unexpected revisions")
	}
	if list[1].Status != "superseded" {
		t.Errorf("expected superseded, got %q", list[1].Status)
	}
	if m.LastHistoryOpts == nil || m.LastHistoryOpts.ReleaseName != "r1" {
		t.Error("history opts not tracked")
	}
}

func TestTestDefault(t *testing.T) {
	m := &MockEngine{}
	r, err := m.Test(bg(), cfg(), &TestOptions{ReleaseName: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "my-release" {
		t.Errorf("unexpected name %q", r.Name)
	}
	if m.LastTestOpts == nil || m.LastTestOpts.ReleaseName != "r1" {
		t.Error("test opts not tracked")
	}
}

func TestGetAllDefault(t *testing.T) {
	m := &MockEngine{}
	d, err := m.GetAll(bg(), cfg(), &GetOptions{ReleaseName: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Release == nil || d.Release.Name != "my-release" {
		t.Error("unexpected release in detail")
	}
	if d.Values == nil {
		t.Error("expected non-nil values")
	}
	if d.Manifest == "" {
		t.Error("expected non-empty manifest")
	}
	if d.Hooks == "" {
		t.Error("expected non-empty hooks")
	}
	if d.Notes == "" {
		t.Error("expected non-empty notes")
	}
	if m.LastGetOpts == nil || m.LastGetOpts.ReleaseName != "r1" {
		t.Error("get opts not tracked")
	}
}

func TestGetValuesDefault(t *testing.T) {
	m := &MockEngine{}
	vals, err := m.GetValues(bg(), cfg(), &GetValuesOptions{ReleaseName: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if vals["replicaCount"] != 1 {
		t.Error("unexpected values")
	}
	if m.LastGetValuesOpts == nil || m.LastGetValuesOpts.ReleaseName != "r1" {
		t.Error("get values opts not tracked")
	}
}

func TestGetMetadataDefault(t *testing.T) {
	m := &MockEngine{}
	md, err := m.GetMetadata(bg(), cfg(), &GetOptions{ReleaseName: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if md.Name != "my-release" {
		t.Errorf("unexpected name %q", md.Name)
	}
	if md.AppVersion != defaultMockAppVersion {
		t.Errorf("expected AppVersion %q, got %q", defaultMockAppVersion, md.AppVersion)
	}
	if m.LastGetOpts == nil || m.LastGetOpts.ReleaseName != "r1" {
		t.Error("get opts not tracked")
	}
}

func TestGetManifestDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.GetManifest(bg(), cfg(), &GetOptions{ReleaseName: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "apiVersion") {
		t.Error("expected manifest content")
	}
	if m.LastGetOpts == nil || m.LastGetOpts.ReleaseName != "r1" {
		t.Error("get opts not tracked")
	}
}

func TestGetHooksDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.GetHooks(bg(), cfg(), &GetOptions{ReleaseName: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if s == "" {
		t.Error("expected non-empty hooks")
	}
	if m.LastGetOpts == nil || m.LastGetOpts.ReleaseName != "r1" {
		t.Error("get opts not tracked")
	}
}

func TestGetNotesDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.GetNotes(bg(), cfg(), &GetOptions{ReleaseName: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "NOTES") {
		t.Error("expected NOTES content")
	}
	if m.LastGetOpts == nil || m.LastGetOpts.ReleaseName != "r1" {
		t.Error("get opts not tracked")
	}
}

func TestCreateDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.Create(bg(), &CreateOptions{Name: "mychart"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "mychart") {
		t.Errorf("expected chart name in output, got %q", s)
	}
}

func TestLintDefault(t *testing.T) {
	m := &MockEngine{}
	r, err := m.Lint(bg(), &LintOptions{Paths: []string{"./chart"}})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Passed {
		t.Error("expected lint to pass")
	}
	if r.TotalCharts != 1 {
		t.Errorf("expected 1 total chart, got %d", r.TotalCharts)
	}
}

func TestTemplateDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.Template(bg(), cfg(), &TemplateOptions{ReleaseName: "myrel", Chart: "c"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "myrel") {
		t.Errorf("expected release name in output, got %q", s)
	}
	if m.LastConfig == nil {
		t.Error("config not tracked")
	}
}

func TestPackageDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.Package(bg(), &PackageOptions{Path: "./chart"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "Successfully packaged") {
		t.Errorf("unexpected output %q", s)
	}
}

func TestPullDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.Pull(bg(), cfg(), &PullOptions{Chart: "nginx"})
	if err != nil {
		t.Fatal(err)
	}
	if s != "" {
		t.Errorf("expected empty string, got %q", s)
	}
	if m.LastConfig == nil {
		t.Error("config not tracked")
	}
}

func TestPushDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.Push(bg(), cfg(), &PushOptions{ChartRef: "mychart-0.1.0.tgz", Remote: "oci://reg"})
	if err != nil {
		t.Fatal(err)
	}
	if s != "" {
		t.Errorf("expected empty string, got %q", s)
	}
	if m.LastConfig == nil {
		t.Error("config not tracked")
	}
}

func TestVerifyDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.Verify(bg(), &VerifyOptions{ChartFile: "mychart-0.1.0.tgz"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "Signed by") {
		t.Errorf("unexpected output %q", s)
	}
}

func TestShowAllDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.ShowAll(bg(), cfg(), &ShowOptions{Chart: "nginx"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "nginx") {
		t.Errorf("expected chart name in output, got %q", s)
	}
}

func TestShowChartDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.ShowChart(bg(), cfg(), &ShowOptions{Chart: "nginx"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "nginx") {
		t.Errorf("expected chart name in output, got %q", s)
	}
}

func TestShowValuesDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.ShowValues(bg(), cfg(), &ShowOptions{Chart: "nginx"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "replicaCount") {
		t.Errorf("unexpected output %q", s)
	}
}

func TestShowReadmeDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.ShowReadme(bg(), cfg(), &ShowOptions{Chart: "nginx"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "nginx") {
		t.Errorf("expected chart name in output, got %q", s)
	}
}

func TestShowCRDsDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.ShowCRDs(bg(), cfg(), &ShowOptions{Chart: "nginx"})
	if err != nil {
		t.Fatal(err)
	}
	if s != "" {
		t.Errorf("expected empty string, got %q", s)
	}
}

func TestDependencyBuildDefault(t *testing.T) {
	m := &MockEngine{}
	err := m.DependencyBuild(bg(), cfg(), &DependencyOptions{ChartPath: "./chart"})
	if err != nil {
		t.Fatal(err)
	}
	if m.LastConfig == nil {
		t.Error("config not tracked")
	}
}

func TestDependencyListDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.DependencyList(bg(), cfg(), &DependencyOptions{ChartPath: "./chart"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "redis") {
		t.Errorf("expected dependency list content, got %q", s)
	}
}

func TestDependencyUpdateDefault(t *testing.T) {
	m := &MockEngine{}
	err := m.DependencyUpdate(bg(), cfg(), &DependencyOptions{ChartPath: "./chart"})
	if err != nil {
		t.Fatal(err)
	}
	if m.LastConfig == nil {
		t.Error("config not tracked")
	}
}

func TestRepoAddDefault(t *testing.T) {
	m := &MockEngine{}
	err := m.RepoAdd(bg(), &RepoAddOptions{Name: "stable", URL: "https://charts.helm.sh/stable"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRepoListDefault(t *testing.T) {
	m := &MockEngine{}
	repos, err := m.RepoList(bg(), &RepoListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].Name != "stable" {
		t.Errorf("expected 'stable', got %q", repos[0].Name)
	}
}

func TestRepoUpdateDefault(t *testing.T) {
	m := &MockEngine{}
	s, err := m.RepoUpdate(bg(), &RepoUpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "Update Complete") {
		t.Errorf("unexpected output %q", s)
	}
}

func TestRepoRemoveDefault(t *testing.T) {
	m := &MockEngine{}
	err := m.RepoRemove(bg(), &RepoRemoveOptions{Names: []string{"stable"}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRepoIndexDefault(t *testing.T) {
	m := &MockEngine{}
	err := m.RepoIndex(bg(), &RepoIndexOptions{Directory: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegistryLoginDefault(t *testing.T) {
	m := &MockEngine{}
	err := m.RegistryLogin(bg(), &RegistryLoginOptions{Hostname: "registry.example.com"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegistryLogoutDefault(t *testing.T) {
	m := &MockEngine{}
	err := m.RegistryLogout(bg(), &RegistryLogoutOptions{Hostname: "registry.example.com"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSearchHubDefault(t *testing.T) {
	m := &MockEngine{}
	results, err := m.SearchHub(bg(), &SearchHubOptions{Keyword: "nginx"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Name != "nginx" {
		t.Error("unexpected search hub results")
	}
}

func TestSearchRepoDefault(t *testing.T) {
	m := &MockEngine{}
	results, err := m.SearchRepo(bg(), &SearchRepoOptions{Keyword: "nginx"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Name != "stable/nginx" {
		t.Error("unexpected search repo results")
	}
}

func TestPluginInstallDefault(t *testing.T) {
	m := &MockEngine{}
	err := m.PluginInstall(bg(), &PluginInstallOptions{URLOrPath: "https://example.com/plugin"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPluginListDefault(t *testing.T) {
	m := &MockEngine{}
	plugins, err := m.PluginList(bg())
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 1 || plugins[0].Name != "diff" {
		t.Error("unexpected plugin list")
	}
}

func TestPluginUninstallDefault(t *testing.T) {
	m := &MockEngine{}
	err := m.PluginUninstall(bg(), &PluginUninstallOptions{Name: "diff"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPluginUpdateDefault(t *testing.T) {
	m := &MockEngine{}
	err := m.PluginUpdate(bg(), &PluginUpdateOptions{Name: "diff"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEnvDefault(t *testing.T) {
	m := &MockEngine{}
	env, err := m.Env(bg())
	if err != nil {
		t.Fatal(err)
	}
	if env["HELM_DRIVER"] != "secret" {
		t.Errorf("unexpected HELM_DRIVER %q", env["HELM_DRIVER"])
	}
	if _, ok := env["HELM_CACHE_HOME"]; !ok {
		t.Error("expected HELM_CACHE_HOME")
	}
}

func TestVersionDefault(t *testing.T) {
	m := &MockEngine{}
	v, err := m.Version(bg())
	if err != nil {
		t.Fatal(err)
	}
	if v.Version != "v3.20.0" {
		t.Errorf("unexpected version %q", v.Version)
	}
	if v.GitCommit != "abc123" {
		t.Errorf("unexpected git commit %q", v.GitCommit)
	}
}

// ---------------------------------------------------------------------------
// Call tracking for all methods that record opts
// ---------------------------------------------------------------------------

func TestCallTrackingUpgrade(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.Upgrade(bg(), cfg(), &UpgradeOptions{ReleaseName: "u1", Chart: "c"})
	if m.LastUpgradeOpts == nil || m.LastUpgradeOpts.ReleaseName != "u1" {
		t.Error("upgrade opts not tracked")
	}
	if m.LastConfig == nil {
		t.Error("config not tracked")
	}
}

func TestCallTrackingUninstall(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.Uninstall(bg(), cfg(), &UninstallOptions{ReleaseName: "u1"})
	if m.LastUninstallOpts == nil || m.LastUninstallOpts.ReleaseName != "u1" {
		t.Error("uninstall opts not tracked")
	}
}

func TestCallTrackingRollback(t *testing.T) {
	m := &MockEngine{}
	_ = m.Rollback(bg(), cfg(), &RollbackOptions{ReleaseName: "rb1", Revision: 3})
	if m.LastRollbackOpts == nil || m.LastRollbackOpts.Revision != 3 {
		t.Error("rollback opts not tracked")
	}
}

func TestCallTrackingStatus(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.Status(bg(), cfg(), &StatusOptions{ReleaseName: "s1"})
	if m.LastStatusOpts == nil || m.LastStatusOpts.ReleaseName != "s1" {
		t.Error("status opts not tracked")
	}
}

func TestCallTrackingHistory(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.History(bg(), cfg(), &HistoryOptions{ReleaseName: "h1", Max: 5})
	if m.LastHistoryOpts == nil || m.LastHistoryOpts.Max != 5 {
		t.Error("history opts not tracked")
	}
}

func TestCallTrackingTest(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.Test(bg(), cfg(), &TestOptions{ReleaseName: "t1"})
	if m.LastTestOpts == nil || m.LastTestOpts.ReleaseName != "t1" {
		t.Error("test opts not tracked")
	}
}

func TestCallTrackingGetAll(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.GetAll(bg(), cfg(), &GetOptions{ReleaseName: "ga1"})
	if m.LastGetOpts == nil || m.LastGetOpts.ReleaseName != "ga1" {
		t.Error("get opts not tracked")
	}
}

func TestCallTrackingGetValues(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.GetValues(bg(), cfg(), &GetValuesOptions{ReleaseName: "gv1", All: true})
	if m.LastGetValuesOpts == nil || m.LastGetValuesOpts.ReleaseName != "gv1" {
		t.Error("get values opts not tracked")
	}
	if !m.LastGetValuesOpts.All {
		t.Error("expected All=true")
	}
}

func TestCallTrackingGetMetadata(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.GetMetadata(bg(), cfg(), &GetOptions{ReleaseName: "gm1", Revision: 2})
	if m.LastGetOpts == nil || m.LastGetOpts.Revision != 2 {
		t.Error("get opts not tracked")
	}
}

func TestCallTrackingGetManifest(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.GetManifest(bg(), cfg(), &GetOptions{ReleaseName: "gman1"})
	if m.LastGetOpts == nil || m.LastGetOpts.ReleaseName != "gman1" {
		t.Error("get opts not tracked")
	}
}

func TestCallTrackingGetHooks(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.GetHooks(bg(), cfg(), &GetOptions{ReleaseName: "gh1"})
	if m.LastGetOpts == nil || m.LastGetOpts.ReleaseName != "gh1" {
		t.Error("get opts not tracked")
	}
}

func TestCallTrackingGetNotes(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.GetNotes(bg(), cfg(), &GetOptions{ReleaseName: "gn1"})
	if m.LastGetOpts == nil || m.LastGetOpts.ReleaseName != "gn1" {
		t.Error("get opts not tracked")
	}
}

func TestCallTrackingList(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.List(bg(), cfg(), &ListOptions{AllNamespaces: true})
	if m.LastListOpts == nil || !m.LastListOpts.AllNamespaces {
		t.Error("list opts not tracked")
	}
}

// ---------------------------------------------------------------------------
// Custom Fn override tests -- exercises the non-nil Fn branches
// ---------------------------------------------------------------------------

func TestInstallCustomFn(t *testing.T) {
	m := &MockEngine{
		InstallFn: func(_ context.Context, _ *GlobalConfig, opts *InstallOptions) (*ReleaseInfo, error) {
			return &ReleaseInfo{Name: opts.ReleaseName, Status: "pending-install"}, nil
		},
	}
	r, err := m.Install(bg(), cfg(), &InstallOptions{ReleaseName: "custom"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "custom" || r.Status != "pending-install" {
		t.Errorf("unexpected result %+v", r)
	}
}

func TestUpgradeCustomFn(t *testing.T) {
	m := &MockEngine{
		UpgradeFn: func(_ context.Context, _ *GlobalConfig, _ *UpgradeOptions) (*ReleaseInfo, error) {
			return nil, errMock
		},
	}
	_, err := m.Upgrade(bg(), cfg(), &UpgradeOptions{ReleaseName: "r", Chart: "c"})
	if !errors.Is(err, errMock) {
		t.Errorf("expected errMock, got %v", err)
	}
}

func TestUninstallCustomFn(t *testing.T) {
	m := &MockEngine{
		UninstallFn: func(_ context.Context, _ *GlobalConfig, opts *UninstallOptions) (*UninstallResult, error) {
			return &UninstallResult{ReleaseName: opts.ReleaseName, Info: "custom"}, nil
		},
	}
	res, err := m.Uninstall(bg(), cfg(), &UninstallOptions{ReleaseName: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Info != "custom" {
		t.Errorf("unexpected info %q", res.Info)
	}
}

func TestRollbackCustomFn(t *testing.T) {
	m := &MockEngine{
		RollbackFn: func(_ context.Context, _ *GlobalConfig, _ *RollbackOptions) error {
			return errMock
		},
	}
	err := m.Rollback(bg(), cfg(), &RollbackOptions{ReleaseName: "r"})
	if !errors.Is(err, errMock) {
		t.Errorf("expected errMock, got %v", err)
	}
}

func TestStatusCustomFn(t *testing.T) {
	m := &MockEngine{
		StatusFn: func(_ context.Context, _ *GlobalConfig, opts *StatusOptions) (*ReleaseInfo, error) {
			return &ReleaseInfo{Name: opts.ReleaseName, Status: "failed"}, nil
		},
	}
	r, _ := m.Status(bg(), cfg(), &StatusOptions{ReleaseName: "s"})
	if r.Status != "failed" {
		t.Errorf("expected failed, got %q", r.Status)
	}
}

func TestHistoryCustomFn(t *testing.T) {
	m := &MockEngine{
		HistoryFn: func(_ context.Context, _ *GlobalConfig, _ *HistoryOptions) ([]*ReleaseInfo, error) {
			return nil, errMock
		},
	}
	_, err := m.History(bg(), cfg(), &HistoryOptions{ReleaseName: "h"})
	if !errors.Is(err, errMock) {
		t.Errorf("expected errMock, got %v", err)
	}
}

func TestTestCustomFn(t *testing.T) {
	m := &MockEngine{
		TestFn: func(_ context.Context, _ *GlobalConfig, _ *TestOptions) (*ReleaseInfo, error) {
			return &ReleaseInfo{Name: "tested"}, nil
		},
	}
	r, _ := m.Test(bg(), cfg(), &TestOptions{ReleaseName: "r"})
	if r.Name != "tested" {
		t.Errorf("expected 'tested', got %q", r.Name)
	}
}

func TestGetAllCustomFn(t *testing.T) {
	m := &MockEngine{
		GetAllFn: func(_ context.Context, _ *GlobalConfig, _ *GetOptions) (*ReleaseDetail, error) {
			return &ReleaseDetail{Notes: "custom notes"}, nil
		},
	}
	d, _ := m.GetAll(bg(), cfg(), &GetOptions{ReleaseName: "r"})
	if d.Notes != "custom notes" {
		t.Errorf("unexpected notes %q", d.Notes)
	}
}

func TestGetValuesCustomFn(t *testing.T) {
	m := &MockEngine{
		GetValuesFn: func(_ context.Context, _ *GlobalConfig, _ *GetValuesOptions) (map[string]interface{}, error) {
			return map[string]interface{}{"custom": true}, nil
		},
	}
	v, _ := m.GetValues(bg(), cfg(), &GetValuesOptions{ReleaseName: "r"})
	if v["custom"] != true {
		t.Error("expected custom value")
	}
}

func TestGetMetadataCustomFn(t *testing.T) {
	m := &MockEngine{
		GetMetadataFn: func(_ context.Context, _ *GlobalConfig, _ *GetOptions) (*MetadataInfo, error) {
			return &MetadataInfo{Name: "custom-md"}, nil
		},
	}
	md, _ := m.GetMetadata(bg(), cfg(), &GetOptions{ReleaseName: "r"})
	if md.Name != "custom-md" {
		t.Errorf("unexpected name %q", md.Name)
	}
}

func TestGetManifestCustomFn(t *testing.T) {
	m := &MockEngine{
		GetManifestFn: func(_ context.Context, _ *GlobalConfig, _ *GetOptions) (string, error) {
			return "custom-manifest", nil
		},
	}
	s, _ := m.GetManifest(bg(), cfg(), &GetOptions{ReleaseName: "r"})
	if s != "custom-manifest" {
		t.Errorf("unexpected manifest %q", s)
	}
}

func TestGetHooksCustomFn(t *testing.T) {
	m := &MockEngine{
		GetHooksFn: func(_ context.Context, _ *GlobalConfig, _ *GetOptions) (string, error) {
			return "custom-hooks", nil
		},
	}
	s, _ := m.GetHooks(bg(), cfg(), &GetOptions{ReleaseName: "r"})
	if s != "custom-hooks" {
		t.Errorf("unexpected hooks %q", s)
	}
}

func TestGetNotesCustomFn(t *testing.T) {
	m := &MockEngine{
		GetNotesFn: func(_ context.Context, _ *GlobalConfig, _ *GetOptions) (string, error) {
			return "custom-notes", nil
		},
	}
	s, _ := m.GetNotes(bg(), cfg(), &GetOptions{ReleaseName: "r"})
	if s != "custom-notes" {
		t.Errorf("unexpected notes %q", s)
	}
}

func TestCreateCustomFn(t *testing.T) {
	m := &MockEngine{
		CreateFn: func(_ context.Context, opts *CreateOptions) (string, error) {
			return "created:" + opts.Name, nil
		},
	}
	s, _ := m.Create(bg(), &CreateOptions{Name: "x"})
	if s != "created:x" {
		t.Errorf("unexpected output %q", s)
	}
}

func TestLintCustomFn(t *testing.T) {
	m := &MockEngine{
		LintFn: func(_ context.Context, _ *LintOptions) (*LintResult, error) {
			return &LintResult{Passed: false, TotalCharts: 2}, nil
		},
	}
	r, _ := m.Lint(bg(), &LintOptions{Paths: []string{"./a"}})
	if r.Passed {
		t.Error("expected lint to fail")
	}
}

func TestTemplateCustomFn(t *testing.T) {
	m := &MockEngine{
		TemplateFn: func(_ context.Context, _ *GlobalConfig, opts *TemplateOptions) (string, error) {
			return "tpl:" + opts.ReleaseName, nil
		},
	}
	s, _ := m.Template(bg(), cfg(), &TemplateOptions{ReleaseName: "x", Chart: "c"})
	if s != "tpl:x" {
		t.Errorf("unexpected output %q", s)
	}
}

func TestPackageCustomFn(t *testing.T) {
	m := &MockEngine{
		PackageFn: func(_ context.Context, _ *PackageOptions) (string, error) {
			return "pkg-done", nil
		},
	}
	s, _ := m.Package(bg(), &PackageOptions{Path: "."})
	if s != "pkg-done" {
		t.Errorf("unexpected output %q", s)
	}
}

func TestPullCustomFn(t *testing.T) {
	m := &MockEngine{
		PullFn: func(_ context.Context, _ *GlobalConfig, _ *PullOptions) (string, error) {
			return "pulled", nil
		},
	}
	s, _ := m.Pull(bg(), cfg(), &PullOptions{Chart: "c"})
	if s != "pulled" {
		t.Errorf("unexpected output %q", s)
	}
}

func TestPushCustomFn(t *testing.T) {
	m := &MockEngine{
		PushFn: func(_ context.Context, _ *GlobalConfig, _ *PushOptions) (string, error) {
			return "pushed", nil
		},
	}
	s, _ := m.Push(bg(), cfg(), &PushOptions{ChartRef: "c", Remote: "r"})
	if s != "pushed" {
		t.Errorf("unexpected output %q", s)
	}
}

func TestVerifyCustomFn(t *testing.T) {
	m := &MockEngine{
		VerifyFn: func(_ context.Context, _ *VerifyOptions) (string, error) {
			return "verified", nil
		},
	}
	s, _ := m.Verify(bg(), &VerifyOptions{ChartFile: "c.tgz"})
	if s != "verified" {
		t.Errorf("unexpected output %q", s)
	}
}

func TestShowAllCustomFn(t *testing.T) {
	m := &MockEngine{
		ShowAllFn: func(_ context.Context, _ *GlobalConfig, _ *ShowOptions) (string, error) {
			return "all-info", nil
		},
	}
	s, _ := m.ShowAll(bg(), cfg(), &ShowOptions{Chart: "c"})
	if s != "all-info" {
		t.Errorf("unexpected output %q", s)
	}
}

func TestShowChartCustomFn(t *testing.T) {
	m := &MockEngine{
		ShowChartFn: func(_ context.Context, _ *GlobalConfig, _ *ShowOptions) (string, error) {
			return "chart-info", nil
		},
	}
	s, _ := m.ShowChart(bg(), cfg(), &ShowOptions{Chart: "c"})
	if s != "chart-info" {
		t.Errorf("unexpected output %q", s)
	}
}

func TestShowValuesCustomFn(t *testing.T) {
	m := &MockEngine{
		ShowValuesFn: func(_ context.Context, _ *GlobalConfig, _ *ShowOptions) (string, error) {
			return "vals", nil
		},
	}
	s, _ := m.ShowValues(bg(), cfg(), &ShowOptions{Chart: "c"})
	if s != "vals" {
		t.Errorf("unexpected output %q", s)
	}
}

func TestShowReadmeCustomFn(t *testing.T) {
	m := &MockEngine{
		ShowReadmeFn: func(_ context.Context, _ *GlobalConfig, _ *ShowOptions) (string, error) {
			return "readme", nil
		},
	}
	s, _ := m.ShowReadme(bg(), cfg(), &ShowOptions{Chart: "c"})
	if s != "readme" {
		t.Errorf("unexpected output %q", s)
	}
}

func TestShowCRDsCustomFn(t *testing.T) {
	m := &MockEngine{
		ShowCRDsFn: func(_ context.Context, _ *GlobalConfig, _ *ShowOptions) (string, error) {
			return "crds", nil
		},
	}
	s, _ := m.ShowCRDs(bg(), cfg(), &ShowOptions{Chart: "c"})
	if s != "crds" {
		t.Errorf("unexpected output %q", s)
	}
}

func TestDependencyBuildCustomFn(t *testing.T) {
	m := &MockEngine{
		DependencyBuildFn: func(_ context.Context, _ *GlobalConfig, _ *DependencyOptions) error {
			return errMock
		},
	}
	err := m.DependencyBuild(bg(), cfg(), &DependencyOptions{ChartPath: "."})
	if !errors.Is(err, errMock) {
		t.Errorf("expected errMock, got %v", err)
	}
}

func TestDependencyListCustomFn(t *testing.T) {
	m := &MockEngine{
		DependencyListFn: func(_ context.Context, _ *GlobalConfig, _ *DependencyOptions) (string, error) {
			return "custom-deps", nil
		},
	}
	s, _ := m.DependencyList(bg(), cfg(), &DependencyOptions{ChartPath: "."})
	if s != "custom-deps" {
		t.Errorf("unexpected output %q", s)
	}
}

func TestDependencyUpdateCustomFn(t *testing.T) {
	m := &MockEngine{
		DependencyUpdateFn: func(_ context.Context, _ *GlobalConfig, _ *DependencyOptions) error {
			return errMock
		},
	}
	err := m.DependencyUpdate(bg(), cfg(), &DependencyOptions{ChartPath: "."})
	if !errors.Is(err, errMock) {
		t.Errorf("expected errMock, got %v", err)
	}
}

func TestRepoAddCustomFn(t *testing.T) {
	m := &MockEngine{
		RepoAddFn: func(_ context.Context, _ *RepoAddOptions) error {
			return errMock
		},
	}
	err := m.RepoAdd(bg(), &RepoAddOptions{Name: "n", URL: "u"})
	if !errors.Is(err, errMock) {
		t.Errorf("expected errMock, got %v", err)
	}
}

func TestRepoListCustomFn(t *testing.T) {
	m := &MockEngine{
		RepoListFn: func(_ context.Context, _ *RepoListOptions) ([]*RepoEntry, error) {
			return []*RepoEntry{{Name: "custom"}}, nil
		},
	}
	repos, _ := m.RepoList(bg(), &RepoListOptions{})
	if len(repos) != 1 || repos[0].Name != "custom" {
		t.Error("unexpected repos")
	}
}

func TestRepoUpdateCustomFn(t *testing.T) {
	m := &MockEngine{
		RepoUpdateFn: func(_ context.Context, _ *RepoUpdateOptions) (string, error) {
			return "done", nil
		},
	}
	s, _ := m.RepoUpdate(bg(), &RepoUpdateOptions{})
	if s != "done" {
		t.Errorf("unexpected output %q", s)
	}
}

func TestRepoRemoveCustomFn(t *testing.T) {
	m := &MockEngine{
		RepoRemoveFn: func(_ context.Context, _ *RepoRemoveOptions) error {
			return errMock
		},
	}
	err := m.RepoRemove(bg(), &RepoRemoveOptions{Names: []string{"x"}})
	if !errors.Is(err, errMock) {
		t.Errorf("expected errMock, got %v", err)
	}
}

func TestRepoIndexCustomFn(t *testing.T) {
	m := &MockEngine{
		RepoIndexFn: func(_ context.Context, _ *RepoIndexOptions) error {
			return errMock
		},
	}
	err := m.RepoIndex(bg(), &RepoIndexOptions{Directory: "/tmp"})
	if !errors.Is(err, errMock) {
		t.Errorf("expected errMock, got %v", err)
	}
}

func TestRegistryLoginCustomFn(t *testing.T) {
	m := &MockEngine{
		RegistryLoginFn: func(_ context.Context, _ *RegistryLoginOptions) error {
			return errMock
		},
	}
	err := m.RegistryLogin(bg(), &RegistryLoginOptions{Hostname: "h"})
	if !errors.Is(err, errMock) {
		t.Errorf("expected errMock, got %v", err)
	}
}

func TestRegistryLogoutCustomFn(t *testing.T) {
	m := &MockEngine{
		RegistryLogoutFn: func(_ context.Context, _ *RegistryLogoutOptions) error {
			return errMock
		},
	}
	err := m.RegistryLogout(bg(), &RegistryLogoutOptions{Hostname: "h"})
	if !errors.Is(err, errMock) {
		t.Errorf("expected errMock, got %v", err)
	}
}

func TestSearchHubCustomFn(t *testing.T) {
	m := &MockEngine{
		SearchHubFn: func(_ context.Context, _ *SearchHubOptions) ([]*SearchResult, error) {
			return nil, errMock
		},
	}
	_, err := m.SearchHub(bg(), &SearchHubOptions{Keyword: "k"})
	if !errors.Is(err, errMock) {
		t.Errorf("expected errMock, got %v", err)
	}
}

func TestSearchRepoCustomFn(t *testing.T) {
	m := &MockEngine{
		SearchRepoFn: func(_ context.Context, _ *SearchRepoOptions) ([]*SearchResult, error) {
			return nil, errMock
		},
	}
	_, err := m.SearchRepo(bg(), &SearchRepoOptions{Keyword: "k"})
	if !errors.Is(err, errMock) {
		t.Errorf("expected errMock, got %v", err)
	}
}

func TestPluginInstallCustomFn(t *testing.T) {
	m := &MockEngine{
		PluginInstallFn: func(_ context.Context, _ *PluginInstallOptions) error {
			return errMock
		},
	}
	err := m.PluginInstall(bg(), &PluginInstallOptions{URLOrPath: "u"})
	if !errors.Is(err, errMock) {
		t.Errorf("expected errMock, got %v", err)
	}
}

func TestPluginListCustomFn(t *testing.T) {
	m := &MockEngine{
		PluginListFn: func(_ context.Context) ([]*PluginInfo, error) {
			return []*PluginInfo{{Name: "custom-plugin"}}, nil
		},
	}
	plugins, _ := m.PluginList(bg())
	if len(plugins) != 1 || plugins[0].Name != "custom-plugin" {
		t.Error("unexpected plugins")
	}
}

func TestPluginUninstallCustomFn(t *testing.T) {
	m := &MockEngine{
		PluginUninstallFn: func(_ context.Context, _ *PluginUninstallOptions) error {
			return errMock
		},
	}
	err := m.PluginUninstall(bg(), &PluginUninstallOptions{Name: "p"})
	if !errors.Is(err, errMock) {
		t.Errorf("expected errMock, got %v", err)
	}
}

func TestPluginUpdateCustomFn(t *testing.T) {
	m := &MockEngine{
		PluginUpdateFn: func(_ context.Context, _ *PluginUpdateOptions) error {
			return errMock
		},
	}
	err := m.PluginUpdate(bg(), &PluginUpdateOptions{Name: "p"})
	if !errors.Is(err, errMock) {
		t.Errorf("expected errMock, got %v", err)
	}
}

func TestEnvCustomFn(t *testing.T) {
	m := &MockEngine{
		EnvFn: func(_ context.Context) (map[string]string, error) {
			return map[string]string{"CUSTOM": "val"}, nil
		},
	}
	env, _ := m.Env(bg())
	if env["CUSTOM"] != "val" {
		t.Error("expected custom env")
	}
}

func TestVersionCustomFn(t *testing.T) {
	m := &MockEngine{
		VersionFn: func(_ context.Context) (*VersionInfo, error) {
			return &VersionInfo{Version: "v4.0.0"}, nil
		},
	}
	v, _ := m.Version(bg())
	if v.Version != "v4.0.0" {
		t.Errorf("unexpected version %q", v.Version)
	}
}

// ---------------------------------------------------------------------------
// Config tracking for methods that set LastConfig
// ---------------------------------------------------------------------------

func TestConfigTrackingTemplate(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.Template(bg(), &GlobalConfig{Namespace: "tpl-ns"}, &TemplateOptions{ReleaseName: "r", Chart: "c"})
	if m.LastConfig == nil || m.LastConfig.Namespace != "tpl-ns" {
		t.Error("config not tracked for Template")
	}
}

func TestConfigTrackingPull(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.Pull(bg(), &GlobalConfig{Namespace: "pull-ns"}, &PullOptions{Chart: "c"})
	if m.LastConfig == nil || m.LastConfig.Namespace != "pull-ns" {
		t.Error("config not tracked for Pull")
	}
}

func TestConfigTrackingPush(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.Push(bg(), &GlobalConfig{Namespace: "push-ns"}, &PushOptions{ChartRef: "c", Remote: "r"})
	if m.LastConfig == nil || m.LastConfig.Namespace != "push-ns" {
		t.Error("config not tracked for Push")
	}
}

func TestConfigTrackingShowAll(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.ShowAll(bg(), &GlobalConfig{Namespace: "show-ns"}, &ShowOptions{Chart: "c"})
	if m.LastConfig == nil || m.LastConfig.Namespace != "show-ns" {
		t.Error("config not tracked for ShowAll")
	}
}

func TestConfigTrackingDependencyBuild(t *testing.T) {
	m := &MockEngine{}
	_ = m.DependencyBuild(bg(), &GlobalConfig{Namespace: "dep-ns"}, &DependencyOptions{ChartPath: "."})
	if m.LastConfig == nil || m.LastConfig.Namespace != "dep-ns" {
		t.Error("config not tracked for DependencyBuild")
	}
}

func TestConfigTrackingDependencyList(t *testing.T) {
	m := &MockEngine{}
	_, _ = m.DependencyList(bg(), &GlobalConfig{Namespace: "dep-ns"}, &DependencyOptions{ChartPath: "."})
	if m.LastConfig == nil || m.LastConfig.Namespace != "dep-ns" {
		t.Error("config not tracked for DependencyList")
	}
}

func TestConfigTrackingDependencyUpdate(t *testing.T) {
	m := &MockEngine{}
	_ = m.DependencyUpdate(bg(), &GlobalConfig{Namespace: "dep-ns"}, &DependencyOptions{ChartPath: "."})
	if m.LastConfig == nil || m.LastConfig.Namespace != "dep-ns" {
		t.Error("config not tracked for DependencyUpdate")
	}
}
