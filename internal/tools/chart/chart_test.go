package chart

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setup(t *testing.T) *helmengine.MockEngine {
	t.Helper()
	mock := &helmengine.MockEngine{}
	cleanup := tools.SetEnginesForTest(mock, mock)
	t.Cleanup(cleanup)
	return mock
}

func extractText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil || len(result.Content) == 0 {
		t.Fatal("result is nil or empty")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

// --- Create ---

func TestHandleCreate_Success(t *testing.T) {
	setup(t)
	input := CreateInput{Name: "mychart"}
	result, _, err := HandleCreate(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "mychart") {
		t.Errorf("expected chart name in output, got %s", text)
	}
}

func TestHandleCreate_WithStarter(t *testing.T) {
	mock := setup(t)
	mock.CreateFn = func(ctx context.Context, opts *helmengine.CreateOptions) (string, error) {
		if opts.Starter != "starter-template" {
			t.Errorf("expected starter 'starter-template', got %q", opts.Starter)
		}
		return "Created chart", nil
	}
	input := CreateInput{Name: "mychart", Starter: "starter-template"}
	_, _, _ = HandleCreate(context.Background(), nil, input)
}

func TestHandleCreate_Error(t *testing.T) {
	mock := setup(t)
	mock.CreateFn = func(ctx context.Context, opts *helmengine.CreateOptions) (string, error) {
		return "", errors.New("directory already exists")
	}
	result, _, _ := HandleCreate(context.Background(), nil, CreateInput{Name: "existing"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Lint ---

func TestHandleLint_Success(t *testing.T) {
	setup(t)
	input := LintInput{
		Paths:  []string{"./mychart"},
		Strict: true,
		Quiet:  true,
	}
	result, _, err := HandleLint(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	var lintResult helmengine.LintResult
	if err := json.Unmarshal([]byte(text), &lintResult); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if !lintResult.Passed {
		t.Error("expected lint to pass")
	}
}

func TestHandleLint_WithWarnings(t *testing.T) {
	mock := setup(t)
	mock.LintFn = func(ctx context.Context, opts *helmengine.LintOptions) (*helmengine.LintResult, error) {
		return &helmengine.LintResult{
			TotalCharts: 1,
			Passed:      true,
			Messages: []helmengine.LintMessage{
				{Severity: "WARNING", Path: "templates/deployment.yaml", Message: "icon is recommended"},
			},
		}, nil
	}
	result, _, _ := HandleLint(context.Background(), nil, LintInput{Paths: []string{"./chart"}})
	text := extractText(t, result)
	if !strings.Contains(text, "WARNING") {
		t.Error("expected warning in output")
	}
}

func TestHandleLint_Error(t *testing.T) {
	mock := setup(t)
	mock.LintFn = func(ctx context.Context, opts *helmengine.LintOptions) (*helmengine.LintResult, error) {
		return nil, errors.New("chart not found")
	}
	result, _, _ := HandleLint(context.Background(), nil, LintInput{Paths: []string{"./missing"}})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Template ---

func TestHandleTemplate_Success(t *testing.T) {
	setup(t)
	input := TemplateInput{
		ReleaseName: "my-release",
		Chart:       "nginx",
		Version:     "1.0.0",
		Values:      map[string]interface{}{"replicaCount": 3},
		KubeVersion: "1.28",
		IncludeCRDs: true,
		Validate:    true,
	}
	result, _, err := HandleTemplate(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "my-release") {
		t.Error("expected release name in template output")
	}
}

func TestHandleTemplate_Error(t *testing.T) {
	mock := setup(t)
	mock.TemplateFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.TemplateOptions) (string, error) {
		return "", errors.New("template rendering failed")
	}
	result, _, _ := HandleTemplate(context.Background(), nil, TemplateInput{ReleaseName: "r", Chart: "c"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Package ---

func TestHandlePackage_Success(t *testing.T) {
	setup(t)
	input := PackageInput{
		Path:        "./mychart",
		Destination: "/tmp/output",
		Version:     "2.0.0",
		AppVersion:  "1.25.0",
		Sign:        true,
		Key:         "mykey",
	}
	result, _, err := HandlePackage(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
}

func TestHandlePackage_Error(t *testing.T) {
	mock := setup(t)
	mock.PackageFn = func(ctx context.Context, opts *helmengine.PackageOptions) (string, error) {
		return "", errors.New("Chart.yaml not found")
	}
	result, _, _ := HandlePackage(context.Background(), nil, PackageInput{Path: "./invalid"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Pull ---

func TestHandlePull_Success(t *testing.T) {
	setup(t)
	input := PullInput{
		Chart:       "nginx",
		Version:     "1.0.0",
		Repo:        "https://charts.example.com",
		Destination: "/tmp",
		Untar:       true,
	}
	result, _, err := HandlePull(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	// Mock returns empty string, handler converts to "Chart pulled successfully"
	if !strings.Contains(text, "Chart pulled successfully") {
		t.Errorf("expected success message, got %s", text)
	}
}

func TestHandlePull_Error(t *testing.T) {
	mock := setup(t)
	mock.PullFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.PullOptions) (string, error) {
		return "", errors.New("chart not found in repository")
	}
	result, _, _ := HandlePull(context.Background(), nil, PullInput{Chart: "missing"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Push ---

func TestHandlePush_Success(t *testing.T) {
	setup(t)
	input := PushInput{
		ChartRef:  "/tmp/mychart-1.0.0.tgz",
		Remote:    "oci://registry.example.com/charts",
		PlainHTTP: true,
	}
	result, _, err := HandlePush(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Chart pushed successfully") {
		t.Errorf("expected success message, got %s", text)
	}
}

func TestHandlePush_Error(t *testing.T) {
	mock := setup(t)
	mock.PushFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.PushOptions) (string, error) {
		return "", errors.New("authentication required")
	}
	result, _, _ := HandlePush(context.Background(), nil, PushInput{ChartRef: "c", Remote: "r"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Verify ---

func TestHandleVerify_Success(t *testing.T) {
	setup(t)
	input := VerifyInput{ChartFile: "/tmp/mychart-1.0.0.tgz"}
	result, _, err := HandleVerify(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Signed by") {
		t.Error("expected verification output")
	}
}

func TestHandleVerify_Error(t *testing.T) {
	mock := setup(t)
	mock.VerifyFn = func(ctx context.Context, opts *helmengine.VerifyOptions) (string, error) {
		return "", errors.New("no provenance file found")
	}
	result, _, _ := HandleVerify(context.Background(), nil, VerifyInput{ChartFile: "/tmp/unsigned.tgz"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Show ---

func TestHandleShowAll_Success(t *testing.T) {
	setup(t)
	input := ShowInput{Chart: "nginx"}
	result, _, err := HandleShowAll(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "nginx") {
		t.Error("expected chart name in output")
	}
}

func TestHandleShowChart_Success(t *testing.T) {
	setup(t)
	result, _, err := HandleShowChart(context.Background(), nil, ShowInput{Chart: "nginx"})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
}

func TestHandleShowValues_Success(t *testing.T) {
	setup(t)
	result, _, err := HandleShowValues(context.Background(), nil, ShowInput{Chart: "nginx"})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "replicaCount") {
		t.Error("expected values in output")
	}
}

func TestHandleShowReadme_Success(t *testing.T) {
	setup(t)
	result, _, err := HandleShowReadme(context.Background(), nil, ShowInput{Chart: "nginx"})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
}

func TestHandleShowCRDs_Success(t *testing.T) {
	setup(t)
	result, _, err := HandleShowCRDs(context.Background(), nil, ShowInput{Chart: "nginx"})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
}

func TestHandleShowAll_Error(t *testing.T) {
	mock := setup(t)
	mock.ShowAllFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions) (string, error) {
		return "", errors.New("chart not found")
	}
	result, _, _ := HandleShowAll(context.Background(), nil, ShowInput{Chart: "missing"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

func TestToShowOpts(t *testing.T) {
	input := &ShowInput{
		Chart:    "nginx",
		Version:  "1.0.0",
		Repo:     "https://charts.example.com",
		Devel:    true,
		JSONPath: "{.image}",
	}
	opts := toShowOpts(input)
	if opts.Chart != "nginx" {
		t.Error("chart not mapped")
	}
	if opts.Version != "1.0.0" {
		t.Error("version not mapped")
	}
	if opts.Repo != "https://charts.example.com" {
		t.Error("repo not mapped")
	}
	if !opts.Devel {
		t.Error("devel not mapped")
	}
	if opts.JSONPath != "{.image}" {
		t.Error("jsonpath not mapped")
	}
}

// --- Dependency ---

func TestHandleDependencyBuild_Success(t *testing.T) {
	setup(t)
	input := DependencyInput{
		ChartPath:   "./mychart",
		Verify:      true,
		SkipRefresh: true,
	}
	result, _, err := HandleDependencyBuild(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Dependency build successful") {
		t.Error("expected success message")
	}
}

func TestHandleDependencyList_Success(t *testing.T) {
	setup(t)
	input := DependencyListInput{ChartPath: "./mychart"}
	result, _, err := HandleDependencyList(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "redis") {
		t.Error("expected dependency list")
	}
}

func TestHandleDependencyUpdate_Success(t *testing.T) {
	setup(t)
	input := DependencyInput{ChartPath: "./mychart"}
	result, _, err := HandleDependencyUpdate(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Dependency update successful") {
		t.Error("expected success message")
	}
}

func TestHandleDependencyBuild_Error(t *testing.T) {
	mock := setup(t)
	mock.DependencyBuildFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.DependencyOptions) error {
		return errors.New("Chart.lock not found")
	}
	result, _, _ := HandleDependencyBuild(context.Background(), nil, DependencyInput{ChartPath: "./broken"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

func TestToDependencyOpts(t *testing.T) {
	input := &DependencyInput{
		ChartPath:   "./mychart",
		Verify:      true,
		Keyring:     "/home/.gnupg/pubring.gpg",
		SkipRefresh: true,
	}
	opts := toDependencyOpts(input)
	if opts.ChartPath != "./mychart" {
		t.Error("chart path not mapped")
	}
	if !opts.Verify {
		t.Error("verify not mapped")
	}
	if opts.Keyring != "/home/.gnupg/pubring.gpg" {
		t.Error("keyring not mapped")
	}
	if !opts.SkipRefresh {
		t.Error("skip refresh not mapped")
	}
}

// --- Show Error Tests ---

func TestHandleShowChart_Error(t *testing.T) {
	mock := setup(t)
	mock.ShowChartFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions) (string, error) {
		return "", errors.New("chart not found")
	}
	result, _, _ := HandleShowChart(context.Background(), nil, ShowInput{Chart: "missing"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

func TestHandleShowCRDs_Error(t *testing.T) {
	mock := setup(t)
	mock.ShowCRDsFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions) (string, error) {
		return "", errors.New("chart not found")
	}
	result, _, _ := HandleShowCRDs(context.Background(), nil, ShowInput{Chart: "missing"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

func TestHandleShowReadme_Error(t *testing.T) {
	mock := setup(t)
	mock.ShowReadmeFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions) (string, error) {
		return "", errors.New("chart not found")
	}
	result, _, _ := HandleShowReadme(context.Background(), nil, ShowInput{Chart: "missing"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

func TestHandleShowValues_Error(t *testing.T) {
	mock := setup(t)
	mock.ShowValuesFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ShowOptions) (string, error) {
		return "", errors.New("chart not found")
	}
	result, _, _ := HandleShowValues(context.Background(), nil, ShowInput{Chart: "missing"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Dependency Error Tests ---

func TestHandleDependencyList_Error(t *testing.T) {
	mock := setup(t)
	mock.DependencyListFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.DependencyOptions) (string, error) {
		return "", errors.New("Chart.yaml not found")
	}
	result, _, _ := HandleDependencyList(context.Background(), nil, DependencyListInput{ChartPath: "./broken"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

func TestHandleDependencyUpdate_Error(t *testing.T) {
	mock := setup(t)
	mock.DependencyUpdateFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.DependencyOptions) error {
		return errors.New("repository not found")
	}
	result, _, _ := HandleDependencyUpdate(context.Background(), nil, DependencyInput{ChartPath: "./broken"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}
