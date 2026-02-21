package v3

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/lint/support"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
)

// ---------------------------------------------------------------------------
// New()
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	e := New()
	if e == nil {
		t.Fatal("New() returned nil")
	}
}

// ---------------------------------------------------------------------------
// Version()
// ---------------------------------------------------------------------------

func TestVersion(t *testing.T) {
	e := New()
	info, err := e.Version(context.Background())
	if err != nil {
		t.Fatalf("Version() error: %v", err)
	}
	if info == nil {
		t.Fatal("Version() returned nil info")
	}
	if info.Version != "v3 (SDK)" {
		t.Errorf("Version = %q, want %q", info.Version, "v3 (SDK)")
	}
	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %q, want %q", info.GoVersion, runtime.Version())
	}
}

// ---------------------------------------------------------------------------
// Env()
// ---------------------------------------------------------------------------

func TestEnv(t *testing.T) {
	e := New()
	env, err := e.Env(context.Background())
	if err != nil {
		t.Fatalf("Env() error: %v", err)
	}
	if len(env) == 0 {
		t.Fatal("Env() returned empty map")
	}
	// The Helm CLI environment always defines HELM_DATA_HOME (or similar).
	// Just verify we got some keys back.
	foundAny := false
	for k := range env {
		if k != "" {
			foundAny = true
			break
		}
	}
	if !foundAny {
		t.Error("Env() map has no non-empty keys")
	}
}

// ---------------------------------------------------------------------------
// newActionConfigNoCluster()
// ---------------------------------------------------------------------------

func TestNewActionConfigNoCluster_NilCfg(t *testing.T) {
	actionCfg, settings := newActionConfigNoCluster(nil)
	if actionCfg == nil {
		t.Fatal("newActionConfigNoCluster(nil) returned nil actionConfig")
	}
	if settings == nil {
		t.Fatal("newActionConfigNoCluster(nil) returned nil settings")
	}
}

func TestNewActionConfigNoCluster_WithCfg(t *testing.T) {
	cfg := &helmengine.GlobalConfig{
		Namespace:   "my-ns",
		KubeContext: "my-ctx",
		KubeConfig:  "/tmp/fake-kubeconfig",
		Debug:       true,
	}
	actionCfg, settings := newActionConfigNoCluster(cfg)
	if actionCfg == nil {
		t.Fatal("newActionConfigNoCluster returned nil actionConfig")
	}
	if settings == nil {
		t.Fatal("newActionConfigNoCluster returned nil settings")
	}
	if settings.KubeContext != "my-ctx" {
		t.Errorf("KubeContext = %q, want %q", settings.KubeContext, "my-ctx")
	}
	if settings.KubeConfig != "/tmp/fake-kubeconfig" {
		t.Errorf("KubeConfig = %q, want %q", settings.KubeConfig, "/tmp/fake-kubeconfig")
	}
	if !settings.Debug {
		t.Error("expected Debug=true on settings")
	}
}

func TestNewActionConfigNoCluster_EmptyCfg(t *testing.T) {
	cfg := &helmengine.GlobalConfig{}
	actionCfg, settings := newActionConfigNoCluster(cfg)
	if actionCfg == nil {
		t.Fatal("newActionConfigNoCluster returned nil actionConfig")
	}
	if settings == nil {
		t.Fatal("newActionConfigNoCluster returned nil settings")
	}
}

// ---------------------------------------------------------------------------
// parseDuration()
// ---------------------------------------------------------------------------

func TestParseDuration_EmptyString(t *testing.T) {
	d, err := parseDuration("")
	if err != nil {
		t.Fatalf("parseDuration(\"\") error: %v", err)
	}
	if d != helmengine.DefaultTimeout {
		t.Errorf("parseDuration(\"\") = %v, want %v", d, helmengine.DefaultTimeout)
	}
}

func TestParseDuration_ValidString(t *testing.T) {
	d, err := parseDuration("5m")
	if err != nil {
		t.Fatalf("parseDuration(\"5m\") error: %v", err)
	}
	if d != 5*time.Minute {
		t.Errorf("parseDuration(\"5m\") = %v, want %v", d, 5*time.Minute)
	}
}

func TestParseDuration_Seconds(t *testing.T) {
	d, err := parseDuration("30s")
	if err != nil {
		t.Fatalf("parseDuration(\"30s\") error: %v", err)
	}
	if d != 30*time.Second {
		t.Errorf("parseDuration(\"30s\") = %v, want %v", d, 30*time.Second)
	}
}

func TestParseDuration_Hours(t *testing.T) {
	d, err := parseDuration("2h")
	if err != nil {
		t.Fatalf("parseDuration(\"2h\") error: %v", err)
	}
	if d != 2*time.Hour {
		t.Errorf("parseDuration(\"2h\") = %v, want %v", d, 2*time.Hour)
	}
}

func TestParseDuration_InvalidString(t *testing.T) {
	_, err := parseDuration("notaduration")
	if err == nil {
		t.Fatal("parseDuration(\"notaduration\") expected error, got nil")
	}
}

func TestParseDuration_InvalidUnit(t *testing.T) {
	_, err := parseDuration("5x")
	if err == nil {
		t.Fatal("parseDuration(\"5x\") expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// severityToString()
// ---------------------------------------------------------------------------

func TestSeverityToString_Info(t *testing.T) {
	got := severityToString(support.InfoSev)
	if got != "INFO" {
		t.Errorf("severityToString(InfoSev) = %q, want %q", got, "INFO")
	}
}

func TestSeverityToString_Warning(t *testing.T) {
	got := severityToString(support.WarningSev)
	if got != "WARNING" {
		t.Errorf("severityToString(WarningSev) = %q, want %q", got, "WARNING")
	}
}

func TestSeverityToString_Error(t *testing.T) {
	got := severityToString(support.ErrorSev)
	if got != "ERROR" {
		t.Errorf("severityToString(ErrorSev) = %q, want %q", got, "ERROR")
	}
}

func TestSeverityToString_Unknown(t *testing.T) {
	got := severityToString(999)
	if got != "UNKNOWN" {
		t.Errorf("severityToString(999) = %q, want %q", got, "UNKNOWN")
	}
}

func TestSeverityToString_NegativeValue(t *testing.T) {
	got := severityToString(-1)
	if got != "UNKNOWN" {
		t.Errorf("severityToString(-1) = %q, want %q", got, "UNKNOWN")
	}
}

// ---------------------------------------------------------------------------
// releaseToInfo()
// ---------------------------------------------------------------------------

func TestReleaseToInfo_Nil(t *testing.T) {
	info := releaseToInfo(nil)
	if info != nil {
		t.Errorf("releaseToInfo(nil) = %v, want nil", info)
	}
}

func TestReleaseToInfo_EmptyRelease(t *testing.T) {
	rel := &release.Release{}
	info := releaseToInfo(rel)
	if info == nil {
		t.Fatal("releaseToInfo(&Release{}) returned nil")
	}
	if info.Name != "" {
		t.Errorf("Name = %q, want empty", info.Name)
	}
	if info.Namespace != "" {
		t.Errorf("Namespace = %q, want empty", info.Namespace)
	}
	if info.Revision != 0 {
		t.Errorf("Revision = %d, want 0", info.Revision)
	}
	if info.Status != "" {
		t.Errorf("Status = %q, want empty", info.Status)
	}
	if info.Chart != "" {
		t.Errorf("Chart = %q, want empty", info.Chart)
	}
}

func TestReleaseToInfo_FullyPopulated(t *testing.T) {
	rel := &release.Release{
		Name:      "test",
		Namespace: "default",
		Version:   1,
		Info: &release.Info{
			Status:      release.StatusDeployed,
			Description: "test release",
		},
		Chart: &chart.Chart{
			Metadata: &chart.Metadata{
				Name:       "nginx",
				Version:    "1.0.0",
				AppVersion: "1.25.0",
			},
		},
	}

	info := releaseToInfo(rel)
	if info == nil {
		t.Fatal("releaseToInfo returned nil for fully populated release")
	}
	if info.Name != "test" {
		t.Errorf("Name = %q, want %q", info.Name, "test")
	}
	if info.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", info.Namespace, "default")
	}
	if info.Revision != 1 {
		t.Errorf("Revision = %d, want 1", info.Revision)
	}
	if info.Status != "deployed" {
		t.Errorf("Status = %q, want %q", info.Status, "deployed")
	}
	if info.Description != "test release" {
		t.Errorf("Description = %q, want %q", info.Description, "test release")
	}
	if info.Chart != "nginx" {
		t.Errorf("Chart = %q, want %q", info.Chart, "nginx")
	}
	if info.ChartVersion != "1.0.0" {
		t.Errorf("ChartVersion = %q, want %q", info.ChartVersion, "1.0.0")
	}
	if info.AppVersion != "1.25.0" {
		t.Errorf("AppVersion = %q, want %q", info.AppVersion, "1.25.0")
	}
}

func TestReleaseToInfo_NilInfo(t *testing.T) {
	rel := &release.Release{
		Name:      "test",
		Namespace: "default",
		Version:   2,
		Info:      nil,
		Chart: &chart.Chart{
			Metadata: &chart.Metadata{
				Name:       "redis",
				Version:    "2.0.0",
				AppVersion: "7.0.0",
			},
		},
	}
	info := releaseToInfo(rel)
	if info == nil {
		t.Fatal("releaseToInfo returned nil")
	}
	if info.Status != "" {
		t.Errorf("Status = %q, want empty (Info is nil)", info.Status)
	}
	if info.Chart != "redis" {
		t.Errorf("Chart = %q, want %q", info.Chart, "redis")
	}
}

func TestReleaseToInfo_NilChart(t *testing.T) {
	rel := &release.Release{
		Name:      "test",
		Namespace: "default",
		Version:   3,
		Info: &release.Info{
			Status: release.StatusFailed,
		},
		Chart: nil,
	}
	info := releaseToInfo(rel)
	if info == nil {
		t.Fatal("releaseToInfo returned nil")
	}
	if info.Status != "failed" {
		t.Errorf("Status = %q, want %q", info.Status, "failed")
	}
	if info.Chart != "" {
		t.Errorf("Chart = %q, want empty (Chart is nil)", info.Chart)
	}
}

func TestReleaseToInfo_NilChartMetadata(t *testing.T) {
	rel := &release.Release{
		Name:      "test",
		Namespace: "default",
		Version:   4,
		Chart:     &chart.Chart{Metadata: nil},
	}
	info := releaseToInfo(rel)
	if info == nil {
		t.Fatal("releaseToInfo returned nil")
	}
	if info.Chart != "" {
		t.Errorf("Chart = %q, want empty (Metadata is nil)", info.Chart)
	}
}

func TestReleaseToInfo_WithLabels(t *testing.T) {
	rel := &release.Release{
		Name:      "labeled",
		Namespace: "prod",
		Version:   1,
		Labels: map[string]string{
			"team": "platform",
			"env":  "production",
		},
	}
	info := releaseToInfo(rel)
	if info == nil {
		t.Fatal("releaseToInfo returned nil")
	}
	if len(info.Labels) != 2 {
		t.Fatalf("Labels count = %d, want 2", len(info.Labels))
	}
	if info.Labels["team"] != "platform" {
		t.Errorf("Labels[\"team\"] = %q, want %q", info.Labels["team"], "platform")
	}
	if info.Labels["env"] != "production" {
		t.Errorf("Labels[\"env\"] = %q, want %q", info.Labels["env"], "production")
	}
}

func TestReleaseToInfo_WithNotes(t *testing.T) {
	rel := &release.Release{
		Name:      "noted",
		Namespace: "default",
		Version:   1,
		Info: &release.Info{
			Status: release.StatusDeployed,
			Notes:  "Visit http://localhost:8080 to access your app",
		},
	}
	info := releaseToInfo(rel)
	if info == nil {
		t.Fatal("releaseToInfo returned nil")
	}
	if info.Notes != "Visit http://localhost:8080 to access your app" {
		t.Errorf("Notes = %q, want notes text", info.Notes)
	}
}

// ---------------------------------------------------------------------------
// SearchHub()
// ---------------------------------------------------------------------------

func TestSearchHub_ReturnsError(t *testing.T) {
	e := New()
	results, err := e.SearchHub(context.Background(), &helmengine.SearchHubOptions{
		Keyword: "nginx",
	})
	if err == nil {
		t.Fatal("SearchHub() expected error, got nil")
	}
	if results != nil {
		t.Errorf("SearchHub() results = %v, want nil", results)
	}
	if !strings.Contains(err.Error(), "search hub is not directly supported via the Helm v3 SDK") {
		t.Errorf("SearchHub() error = %q, want it to contain 'search hub is not directly supported via the Helm v3 SDK'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// isPrerelease()
// ---------------------------------------------------------------------------

func TestIsPrerelease(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"1.0.0", false},
		{"1.0.0-rc1", true},
		{"1.0.0-alpha", true},
		{"1.0.0-beta.1", true},
		{"2.3.4", false},
		{"0.1.0-dev", true},
		{"", false},
		{"1.0.0-0.3.7", true},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := isPrerelease(tt.version)
			if got != tt.want {
				t.Errorf("isPrerelease(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// containsIgnoreCase()
// ---------------------------------------------------------------------------

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{"exact match", "hello", "hello", true},
		{"case mismatch", "Hello", "hello", true},
		{"substr upper", "hello world", "WORLD", true},
		{"no match", "hello", "xyz", false},
		{"empty substr", "hello", "", true},
		{"empty s", "", "hello", false},
		{"both empty", "", "", true},
		{"partial match", "Nginx Ingress Controller", "nginx", true},
		{"mixed case substr", "TestValue", "testvalue", true},
		{"substr longer than s", "hi", "hello", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsIgnoreCase(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// matchesKeyword()
// ---------------------------------------------------------------------------

func TestMatchesKeyword(t *testing.T) {
	tests := []struct {
		name        string
		chartName   string
		description string
		keyword     string
		want        bool
	}{
		{"name match exact", "nginx", "", "nginx", true},
		{"name match case insensitive", "Nginx", "", "nginx", true},
		{"description match", "myapp", "An Nginx web server", "nginx", true},
		{"description match case insensitive", "myapp", "An NGINX Web Server", "nginx", true},
		{"no match", "redis", "In-memory data store", "nginx", false},
		{"empty keyword matches all", "anything", "any description", "", true},
		{"partial name match", "nginx-ingress", "", "nginx", true},
		{"partial description match", "app", "Built with Nginx for serving", "nginx", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesKeyword(tt.chartName, tt.description, tt.keyword)
			if got != tt.want {
				t.Errorf("matchesKeyword(%q, %q, %q) = %v, want %v",
					tt.chartName, tt.description, tt.keyword, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// appendMatchingEntries()
// ---------------------------------------------------------------------------

// makeChartVersions creates a repo.ChartVersions slice for testing.
func makeChartVersions(entries ...struct {
	version, appVersion, description string
}) repo.ChartVersions {
	var cvs repo.ChartVersions
	for _, e := range entries {
		cvs = append(cvs, &repo.ChartVersion{
			Metadata: &chart.Metadata{
				Version:     e.version,
				AppVersion:  e.appVersion,
				Description: e.description,
			},
		})
	}
	return cvs
}

func TestAppendMatchingEntries_AllVersions(t *testing.T) {
	entries := makeChartVersions(
		struct{ version, appVersion, description string }{"1.0.0", "1.25.0", "stable release"},
		struct{ version, appVersion, description string }{"1.0.0-rc1", "1.25.0", "release candidate"},
		struct{ version, appVersion, description string }{"0.9.0", "1.24.0", "previous stable"},
	)

	// allVersions=true, devel=true: should include all entries
	results := appendMatchingEntries(nil, "repo/nginx", entries, true, true)
	if len(results) != 3 {
		t.Fatalf("allVersions=true, devel=true: got %d results, want 3", len(results))
	}

	// allVersions=true, devel=false: should exclude prerelease entries
	results = appendMatchingEntries(nil, "repo/nginx", entries, true, false)
	if len(results) != 2 {
		t.Fatalf("allVersions=true, devel=false: got %d results, want 2", len(results))
	}
	for _, r := range results {
		if isPrerelease(r.ChartVersion) {
			t.Errorf("expected no prerelease entries, got %q", r.ChartVersion)
		}
	}
}

func TestAppendMatchingEntries_LatestOnly(t *testing.T) {
	entries := makeChartVersions(
		struct{ version, appVersion, description string }{"1.0.0", "1.25.0", "stable"},
		struct{ version, appVersion, description string }{"0.9.0", "1.24.0", "older"},
	)

	// allVersions=false: should only return the first entry
	results := appendMatchingEntries(nil, "repo/nginx", entries, false, false)
	if len(results) != 1 {
		t.Fatalf("latestOnly: got %d results, want 1", len(results))
	}
	if results[0].ChartVersion != "1.0.0" {
		t.Errorf("ChartVersion = %q, want %q", results[0].ChartVersion, "1.0.0")
	}
	if results[0].Name != "repo/nginx" {
		t.Errorf("Name = %q, want %q", results[0].Name, "repo/nginx")
	}
}

func TestAppendMatchingEntries_LatestOnlyPrerelease(t *testing.T) {
	entries := makeChartVersions(
		struct{ version, appVersion, description string }{"2.0.0-beta.1", "2.0.0", "beta"},
	)

	// Latest is prerelease, devel=false: should skip
	results := appendMatchingEntries(nil, "repo/app", entries, false, false)
	if len(results) != 0 {
		t.Fatalf("latestOnly prerelease, devel=false: got %d results, want 0", len(results))
	}

	// Latest is prerelease, devel=true: should include
	results = appendMatchingEntries(nil, "repo/app", entries, false, true)
	if len(results) != 1 {
		t.Fatalf("latestOnly prerelease, devel=true: got %d results, want 1", len(results))
	}
}

func TestAppendMatchingEntries_ResultFields(t *testing.T) {
	entries := makeChartVersions(
		struct{ version, appVersion, description string }{"3.1.0", "2.8.0", "A great chart"},
	)
	results := appendMatchingEntries(nil, "myrepo/mychart", entries, false, false)
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	r := results[0]
	if r.Name != "myrepo/mychart" {
		t.Errorf("Name = %q, want %q", r.Name, "myrepo/mychart")
	}
	if r.ChartVersion != "3.1.0" {
		t.Errorf("ChartVersion = %q, want %q", r.ChartVersion, "3.1.0")
	}
	if r.AppVersion != "2.8.0" {
		t.Errorf("AppVersion = %q, want %q", r.AppVersion, "2.8.0")
	}
	if r.Description != "A great chart" {
		t.Errorf("Description = %q, want %q", r.Description, "A great chart")
	}
}

func TestAppendMatchingEntries_AppendsToExisting(t *testing.T) {
	existing := []*helmengine.SearchResult{
		{Name: "existing/chart", ChartVersion: "0.1.0"},
	}
	entries := makeChartVersions(
		struct{ version, appVersion, description string }{"1.0.0", "1.0.0", "new chart"},
	)
	results := appendMatchingEntries(existing, "repo/new", entries, false, false)
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Name != "existing/chart" {
		t.Errorf("results[0].Name = %q, want %q", results[0].Name, "existing/chart")
	}
	if results[1].Name != "repo/new" {
		t.Errorf("results[1].Name = %q, want %q", results[1].Name, "repo/new")
	}
}

// ---------------------------------------------------------------------------
// Lint() - error handling with non-existent path
// ---------------------------------------------------------------------------

func TestLint_NonExistentPath(t *testing.T) {
	e := New()
	result, err := e.Lint(context.Background(), &helmengine.LintOptions{
		Paths: []string{"/tmp/nonexistent-chart-path-that-does-not-exist-12345"},
	})
	// Lint does not return a Go error; it returns a result.
	if err != nil {
		t.Fatalf("Lint() error: %v", err)
	}
	if result == nil {
		t.Fatal("Lint() returned nil result")
	}
	// Should not pass when path does not exist.
	// The Helm lint action reports the failure via result.Errors (mapped to
	// Passed=false) but may not produce individual Messages for a missing path.
	if result.Passed {
		t.Error("Lint() Passed = true for non-existent path, want false")
	}
}

func TestLint_DefaultPaths(t *testing.T) {
	e := New()
	// When Paths is empty, Lint uses "." as default.
	// In the test environment "." is unlikely to be a valid chart, so it should
	// return a result with errors but not a Go error.
	result, err := e.Lint(context.Background(), &helmengine.LintOptions{})
	if err != nil {
		t.Fatalf("Lint() error: %v", err)
	}
	if result == nil {
		t.Fatal("Lint() returned nil result")
	}
}
