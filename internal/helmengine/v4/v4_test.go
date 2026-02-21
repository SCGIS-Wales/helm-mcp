package v4

import (
	"context"
	"runtime"
	"testing"
	"time"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"

	chart "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/chart/v2/lint/support"
	repo "helm.sh/helm/v4/pkg/repo/v1"
)

// ---------------------------------------------------------------------------
// 1. New()
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	e := New()
	if e == nil {
		t.Fatal("New() returned nil")
	}
}

// ---------------------------------------------------------------------------
// 2. Version()
// ---------------------------------------------------------------------------

func TestVersion(t *testing.T) {
	e := New()
	info, err := e.Version(context.Background())
	if err != nil {
		t.Fatalf("Version() returned error: %v", err)
	}
	if info == nil {
		t.Fatal("Version() returned nil VersionInfo")
	}
	if info.Version != "v4 (SDK)" {
		t.Errorf("Version().Version = %q, want %q", info.Version, "v4 (SDK)")
	}
	if info.GoVersion != runtime.Version() {
		t.Errorf("Version().GoVersion = %q, want %q", info.GoVersion, runtime.Version())
	}
}

// ---------------------------------------------------------------------------
// 3. Env()
// ---------------------------------------------------------------------------

func TestEnv(t *testing.T) {
	e := New()
	env, err := e.Env(context.Background())
	if err != nil {
		t.Fatalf("Env() returned error: %v", err)
	}
	if len(env) == 0 {
		t.Fatal("Env() returned empty map")
	}
	// Helm CLI settings always populate HELM_CACHE_HOME etc.
	for _, key := range []string{"HELM_CACHE_HOME", "HELM_CONFIG_HOME", "HELM_DATA_HOME"} {
		if _, ok := env[key]; !ok {
			t.Errorf("Env() missing expected key %q", key)
		}
	}
}

// ---------------------------------------------------------------------------
// 4. newActionConfigNoCluster()
// ---------------------------------------------------------------------------

func TestNewActionConfigNoCluster_NilConfig(t *testing.T) {
	actionCfg, settings := newActionConfigNoCluster(nil)
	if actionCfg == nil {
		t.Fatal("newActionConfigNoCluster(nil) returned nil actionConfig")
	}
	if settings == nil {
		t.Fatal("newActionConfigNoCluster(nil) returned nil settings")
	}
}

func TestNewActionConfigNoCluster_WithConfig(t *testing.T) {
	cfg := &helmengine.GlobalConfig{
		Namespace:   "test-ns",
		KubeContext: "test-ctx",
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
	if ns := settings.Namespace(); ns != "test-ns" {
		t.Errorf("settings.Namespace() = %q, want %q", ns, "test-ns")
	}
	if settings.KubeContext != "test-ctx" {
		t.Errorf("settings.KubeContext = %q, want %q", settings.KubeContext, "test-ctx")
	}
	if settings.KubeConfig != "/tmp/fake-kubeconfig" {
		t.Errorf("settings.KubeConfig = %q, want %q", settings.KubeConfig, "/tmp/fake-kubeconfig")
	}
	if !settings.Debug {
		t.Error("settings.Debug should be true")
	}
}

func TestNewActionConfigNoCluster_EmptyConfig(t *testing.T) {
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
// 5. parseDuration()
// ---------------------------------------------------------------------------

func TestParseDuration_EmptyString(t *testing.T) {
	d, err := parseDuration("")
	if err != nil {
		t.Fatalf("parseDuration(\"\") returned error: %v", err)
	}
	if d != helmengine.DefaultTimeout {
		t.Errorf("parseDuration(\"\") = %v, want DefaultTimeout (%v)", d, helmengine.DefaultTimeout)
	}
}

func TestParseDuration_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"5m", 5 * time.Minute},
		{"30s", 30 * time.Second},
		{"1h", 1 * time.Hour},
		{"2h30m", 2*time.Hour + 30*time.Minute},
		{"100ms", 100 * time.Millisecond},
	}
	for _, tc := range tests {
		d, err := parseDuration(tc.input)
		if err != nil {
			t.Errorf("parseDuration(%q) returned error: %v", tc.input, err)
			continue
		}
		if d != tc.expected {
			t.Errorf("parseDuration(%q) = %v, want %v", tc.input, d, tc.expected)
		}
	}
}

func TestParseDuration_Invalid(t *testing.T) {
	invalids := []string{"abc", "5", "not-a-duration", "---"}
	for _, s := range invalids {
		_, err := parseDuration(s)
		if err == nil {
			t.Errorf("parseDuration(%q) expected error, got nil", s)
		}
	}
}

// ---------------------------------------------------------------------------
// 6. severityToString()
// ---------------------------------------------------------------------------

func TestSeverityToString(t *testing.T) {
	tests := []struct {
		severity int
		expected string
	}{
		{support.InfoSev, "INFO"},
		{support.WarningSev, "WARNING"},
		{support.ErrorSev, "ERROR"},
		{support.UnknownSev, "UNKNOWN"},
		{-1, "UNKNOWN"},
		{999, "UNKNOWN"},
	}
	for _, tc := range tests {
		got := severityToString(tc.severity)
		if got != tc.expected {
			t.Errorf("severityToString(%d) = %q, want %q", tc.severity, got, tc.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// 7. SearchHub()
// ---------------------------------------------------------------------------

func TestSearchHub(t *testing.T) {
	e := New()
	results, err := e.SearchHub(context.Background(), &helmengine.SearchHubOptions{
		Keyword: "nginx",
	})
	if err == nil {
		t.Fatal("SearchHub() expected error, got nil")
	}
	if results != nil {
		t.Errorf("SearchHub() expected nil results, got %v", results)
	}
	expected := "search hub is not directly supported via the Helm v4 SDK"
	if got := err.Error(); got[:len(expected)] != expected {
		t.Errorf("SearchHub() error = %q, want prefix %q", got, expected)
	}
}

// ---------------------------------------------------------------------------
// 8. matchesKeyword()
// ---------------------------------------------------------------------------

func TestMatchesKeyword(t *testing.T) {
	tests := []struct {
		name        string
		chartName   string
		description string
		keyword     string
		expected    bool
	}{
		{
			name:        "match by name",
			chartName:   "nginx-ingress",
			description: "An NGINX controller",
			keyword:     "nginx",
			expected:    true,
		},
		{
			name:        "match by description",
			chartName:   "my-chart",
			description: "A web server for production",
			keyword:     "production",
			expected:    true,
		},
		{
			name:        "case insensitive name match",
			chartName:   "MyApp",
			description: "",
			keyword:     "myapp",
			expected:    true,
		},
		{
			name:        "case insensitive description match",
			chartName:   "chart",
			description: "FANCY Database",
			keyword:     "fancy",
			expected:    true,
		},
		{
			name:        "no match",
			chartName:   "redis",
			description: "A key-value store",
			keyword:     "postgresql",
			expected:    false,
		},
		{
			name:        "empty keyword matches everything via Contains",
			chartName:   "anything",
			description: "anything",
			keyword:     "",
			expected:    true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchesKeyword(tc.chartName, tc.description, tc.keyword)
			if got != tc.expected {
				t.Errorf("matchesKeyword(%q, %q, %q) = %v, want %v",
					tc.chartName, tc.description, tc.keyword, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 9. appendMatchingEntries()
// ---------------------------------------------------------------------------

func makeEntries(versions ...string) repo.ChartVersions {
	var entries repo.ChartVersions
	for _, v := range versions {
		entries = append(entries, &repo.ChartVersion{
			Metadata: &chart.Metadata{
				Version:     v,
				AppVersion:  "1.0.0",
				Description: "test chart",
			},
		})
	}
	return entries
}

func TestAppendMatchingEntries_AllVersions_NoDevel(t *testing.T) {
	entries := makeEntries("1.0.0", "1.1.0-alpha", "2.0.0")
	results := appendMatchingEntries(nil, "repo/mychart", entries, true, false)

	// Should include 1.0.0 and 2.0.0, but NOT 1.1.0-alpha (prerelease has "-")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(results), results)
	}
	if results[0].ChartVersion != "1.0.0" {
		t.Errorf("results[0].ChartVersion = %q, want %q", results[0].ChartVersion, "1.0.0")
	}
	if results[1].ChartVersion != "2.0.0" {
		t.Errorf("results[1].ChartVersion = %q, want %q", results[1].ChartVersion, "2.0.0")
	}
}

func TestAppendMatchingEntries_AllVersions_WithDevel(t *testing.T) {
	entries := makeEntries("1.0.0", "1.1.0-alpha", "2.0.0")
	results := appendMatchingEntries(nil, "repo/mychart", entries, true, true)

	// With devel=true, all versions should be included
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[1].ChartVersion != "1.1.0-alpha" {
		t.Errorf("results[1].ChartVersion = %q, want %q", results[1].ChartVersion, "1.1.0-alpha")
	}
}

func TestAppendMatchingEntries_LatestOnly_NoDevel(t *testing.T) {
	entries := makeEntries("1.0.0", "0.9.0")
	results := appendMatchingEntries(nil, "repo/mychart", entries, false, false)

	// allVersions=false: only the first entry (latest) is used
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ChartVersion != "1.0.0" {
		t.Errorf("results[0].ChartVersion = %q, want %q", results[0].ChartVersion, "1.0.0")
	}
	if results[0].Name != "repo/mychart" {
		t.Errorf("results[0].Name = %q, want %q", results[0].Name, "repo/mychart")
	}
}

func TestAppendMatchingEntries_LatestOnly_PrereleaseSkipped(t *testing.T) {
	// When the latest version is a prerelease and devel=false, it should be skipped
	entries := makeEntries("2.0.0-beta.1")
	results := appendMatchingEntries(nil, "repo/mychart", entries, false, false)

	if len(results) != 0 {
		t.Fatalf("expected 0 results (prerelease skipped), got %d", len(results))
	}
}

func TestAppendMatchingEntries_LatestOnly_PrereleaseIncludedWithDevel(t *testing.T) {
	entries := makeEntries("2.0.0-beta.1")
	results := appendMatchingEntries(nil, "repo/mychart", entries, false, true)

	if len(results) != 1 {
		t.Fatalf("expected 1 result (devel=true), got %d", len(results))
	}
	if results[0].ChartVersion != "2.0.0-beta.1" {
		t.Errorf("results[0].ChartVersion = %q, want %q", results[0].ChartVersion, "2.0.0-beta.1")
	}
}

func TestAppendMatchingEntries_FieldPopulation(t *testing.T) {
	entries := repo.ChartVersions{
		&repo.ChartVersion{
			Metadata: &chart.Metadata{
				Version:     "3.2.1",
				AppVersion:  "2.4.0",
				Description: "A great chart",
			},
		},
	}
	results := appendMatchingEntries(nil, "stable/awesome", entries, false, false)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Name != "stable/awesome" {
		t.Errorf("Name = %q, want %q", r.Name, "stable/awesome")
	}
	if r.ChartVersion != "3.2.1" {
		t.Errorf("ChartVersion = %q, want %q", r.ChartVersion, "3.2.1")
	}
	if r.AppVersion != "2.4.0" {
		t.Errorf("AppVersion = %q, want %q", r.AppVersion, "2.4.0")
	}
	if r.Description != "A great chart" {
		t.Errorf("Description = %q, want %q", r.Description, "A great chart")
	}
}

func TestAppendMatchingEntries_AppendsToExisting(t *testing.T) {
	existing := []*helmengine.SearchResult{
		{Name: "existing/chart", ChartVersion: "0.1.0"},
	}
	entries := makeEntries("1.0.0")
	results := appendMatchingEntries(existing, "repo/new", entries, false, false)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Name != "existing/chart" {
		t.Errorf("results[0].Name = %q, want %q", results[0].Name, "existing/chart")
	}
	if results[1].Name != "repo/new" {
		t.Errorf("results[1].Name = %q, want %q", results[1].Name, "repo/new")
	}
}

// ---------------------------------------------------------------------------
// 10. Lint() with non-existent path
// ---------------------------------------------------------------------------

func TestLint_NonExistentPath(t *testing.T) {
	e := New()
	opts := &helmengine.LintOptions{
		Paths: []string{"/nonexistent/path/to/chart"},
	}
	result, err := e.Lint(context.Background(), opts)
	// Lint does not return an error itself; it returns a LintResult with messages.
	// The result should indicate failure.
	if err != nil {
		t.Fatalf("Lint() returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Lint() returned nil result")
	}
	if result.Passed {
		t.Error("Lint() with nonexistent path should not pass")
	}
}

func TestLint_EmptyPaths(t *testing.T) {
	// When Paths is empty, Lint defaults to ["."] which is likely not a chart dir.
	e := New()
	opts := &helmengine.LintOptions{}
	result, err := e.Lint(context.Background(), opts)
	if err != nil {
		t.Fatalf("Lint() returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Lint() returned nil result")
	}
	// Current directory is not a chart, so it should fail
	if result.Passed {
		t.Error("Lint() with cwd (not a chart) should not pass")
	}
}

// ---------------------------------------------------------------------------
// Additional edge case tests
// ---------------------------------------------------------------------------

func TestParseDuration_DefaultTimeoutValue(t *testing.T) {
	// Verify DefaultTimeout is 300 seconds
	if helmengine.DefaultTimeout != 300*time.Second {
		t.Errorf("DefaultTimeout = %v, want %v", helmengine.DefaultTimeout, 300*time.Second)
	}
}

func TestSeverityToString_BoundaryValues(t *testing.T) {
	// UnknownSev=0 is handled by the default case in severityToString
	// since there is no explicit case for 0
	got := severityToString(0)
	if got != "UNKNOWN" {
		t.Errorf("severityToString(0) = %q, want %q", got, "UNKNOWN")
	}

	// Verify the actual iota values
	if support.UnknownSev != 0 {
		t.Errorf("support.UnknownSev = %d, want 0", support.UnknownSev)
	}
	if support.InfoSev != 1 {
		t.Errorf("support.InfoSev = %d, want 1", support.InfoSev)
	}
	if support.WarningSev != 2 {
		t.Errorf("support.WarningSev = %d, want 2", support.WarningSev)
	}
	if support.ErrorSev != 3 {
		t.Errorf("support.ErrorSev = %d, want 3", support.ErrorSev)
	}
}

func TestMatchesKeyword_PartialMatch(t *testing.T) {
	// Partial substring matches should work
	if !matchesKeyword("nginx-ingress-controller", "", "ingress") {
		t.Error("expected partial name match for 'ingress' in 'nginx-ingress-controller'")
	}
	if !matchesKeyword("x", "The best database around", "database") {
		t.Error("expected partial description match for 'database'")
	}
}

func TestVersion_ReturnsNoGitCommit(t *testing.T) {
	e := New()
	info, err := e.Version(context.Background())
	if err != nil {
		t.Fatalf("Version() returned error: %v", err)
	}
	// v4 SDK-based Version does not set GitCommit
	if info.GitCommit != "" {
		t.Errorf("Version().GitCommit = %q, expected empty string", info.GitCommit)
	}
}

func TestAppendMatchingEntries_AllVersions_AllPrerelease(t *testing.T) {
	// All prerelease versions with devel=false should return nothing
	entries := makeEntries("1.0.0-alpha", "1.0.0-beta", "1.0.0-rc.1")
	results := appendMatchingEntries(nil, "repo/chart", entries, true, false)

	if len(results) != 0 {
		t.Errorf("expected 0 results for all-prerelease with devel=false, got %d", len(results))
	}
}

func TestAppendMatchingEntries_MixedVersions(t *testing.T) {
	entries := makeEntries("3.0.0", "2.1.0-rc.1", "2.0.0", "1.0.0-alpha")
	results := appendMatchingEntries(nil, "repo/chart", entries, true, false)

	// Only non-prerelease: 3.0.0 and 2.0.0
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ChartVersion != "3.0.0" {
		t.Errorf("results[0].ChartVersion = %q, want %q", results[0].ChartVersion, "3.0.0")
	}
	if results[1].ChartVersion != "2.0.0" {
		t.Errorf("results[1].ChartVersion = %q, want %q", results[1].ChartVersion, "2.0.0")
	}
}
