package release

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

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
	if result == nil {
		t.Fatal("result is nil")
	}
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

// --- List ---

func TestHandleList_Success(t *testing.T) {
	mock := setup(t)
	input := ListInput{}
	result, _, err := HandleList(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "my-release") {
		t.Errorf("expected release name in output, got %s", text)
	}
	if mock.LastListOpts == nil {
		t.Error("expected list opts to be tracked")
	}
}

func TestHandleList_WithFilters(t *testing.T) {
	mock := setup(t)
	input := ListInput{
		AllNamespaces: true,
		Filter:        "nginx.*",
		Deployed:      true,
		Failed:        true,
		Limit:         10,
		Offset:        5,
		SortBy:        "date",
		SortReverse:   true,
	}
	_, _, _ = HandleList(context.Background(), nil, input)
	if !mock.LastListOpts.AllNamespaces {
		t.Error("expected AllNamespaces=true")
	}
	if mock.LastListOpts.Filter != "nginx.*" {
		t.Error("expected filter to be passed")
	}
	if mock.LastListOpts.Limit != 10 {
		t.Errorf("expected limit 10, got %d", mock.LastListOpts.Limit)
	}
	if mock.LastListOpts.Offset != 5 {
		t.Errorf("expected offset 5, got %d", mock.LastListOpts.Offset)
	}
	if !mock.LastListOpts.Deployed {
		t.Error("expected Deployed=true")
	}
	if !mock.LastListOpts.Failed {
		t.Error("expected Failed=true")
	}
	if mock.LastListOpts.SortBy != "date" {
		t.Error("expected SortBy=date")
	}
	if !mock.LastListOpts.SortReverse {
		t.Error("expected SortReverse=true")
	}
}

func TestHandleList_Error(t *testing.T) {
	mock := setup(t)
	mock.ListFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.ListOptions) ([]*helmengine.ReleaseInfo, error) {
		return nil, errors.New("connection refused")
	}
	result, _, err := HandleList(context.Background(), nil, ListInput{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "connection refused") {
		t.Errorf("expected error message, got %s", text)
	}
}

func TestHandleList_NamespacePassthrough(t *testing.T) {
	mock := setup(t)
	input := ListInput{
		GlobalInput: tools.GlobalInput{Namespace: "production"},
	}
	_, _, _ = HandleList(context.Background(), nil, input)
	if mock.LastConfig.Namespace != "production" {
		t.Errorf("expected namespace 'production', got %q", mock.LastConfig.Namespace)
	}
}

func TestHandleList_V3Selection(t *testing.T) {
	mock := setup(t)
	input := ListInput{
		GlobalInput: tools.GlobalInput{HelmVersion: "v3"},
	}
	_, _, _ = HandleList(context.Background(), nil, input)
	if mock.LastListOpts == nil {
		t.Error("expected v3 engine to be called (same mock for both)")
	}
}

// --- Install ---

func TestHandleInstall_Success(t *testing.T) {
	mock := setup(t)
	input := InstallInput{
		ReleaseName: "my-release",
		Chart:       "nginx",
		Version:     "1.0.0",
		Values:      map[string]interface{}{"replicaCount": 3},
		CreateNamespace: true,
		Wait:         true,
		Timeout:      "5m",
		DryRun:       "client",
	}
	result, _, err := HandleInstall(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if mock.LastInstallOpts.ReleaseName != "my-release" {
		t.Errorf("expected release name 'my-release', got %q", mock.LastInstallOpts.ReleaseName)
	}
	if mock.LastInstallOpts.Chart != "nginx" {
		t.Errorf("expected chart 'nginx', got %q", mock.LastInstallOpts.Chart)
	}
	if mock.LastInstallOpts.Version != "1.0.0" {
		t.Error("version not passed")
	}
	if !mock.LastInstallOpts.CreateNamespace {
		t.Error("expected CreateNamespace=true")
	}
	if !mock.LastInstallOpts.Wait {
		t.Error("expected Wait=true")
	}
	if mock.LastInstallOpts.Timeout != "5m" {
		t.Error("timeout not passed")
	}
	if mock.LastInstallOpts.DryRun != "client" {
		t.Error("dry run not passed")
	}
}

func TestHandleInstall_V4Options(t *testing.T) {
	mock := setup(t)
	input := InstallInput{
		ReleaseName:       "test",
		Chart:             "mychart",
		ServerSideApply:   true,
		TakeOwnership:     true,
		RollbackOnFailure: true,
		HideSecret:        true,
		ForceConflicts:    true,
		Labels:            map[string]string{"team": "platform"},
	}
	_, _, _ = HandleInstall(context.Background(), nil, input)
	if !mock.LastInstallOpts.ServerSideApply {
		t.Error("expected ServerSideApply=true")
	}
	if !mock.LastInstallOpts.TakeOwnership {
		t.Error("expected TakeOwnership=true")
	}
	if !mock.LastInstallOpts.RollbackOnFailure {
		t.Error("expected RollbackOnFailure=true")
	}
	if !mock.LastInstallOpts.HideSecret {
		t.Error("expected HideSecret=true")
	}
	if !mock.LastInstallOpts.ForceConflicts {
		t.Error("expected ForceConflicts=true")
	}
	if mock.LastInstallOpts.Labels["team"] != "platform" {
		t.Error("labels not passed")
	}
}

func TestHandleInstall_Error(t *testing.T) {
	mock := setup(t)
	mock.InstallFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.InstallOptions) (*helmengine.ReleaseInfo, error) {
		return nil, errors.New("chart not found")
	}
	result, _, _ := HandleInstall(context.Background(), nil, InstallInput{ReleaseName: "r", Chart: "missing"})
	if !result.IsError {
		t.Fatal("expected error")
	}
	if !strings.Contains(extractText(t, result), "chart not found") {
		t.Error("error message not propagated")
	}
}

func TestHandleInstall_ValuesFiles(t *testing.T) {
	mock := setup(t)
	input := InstallInput{
		ReleaseName: "r",
		Chart:       "c",
		ValuesFiles: []string{"values.yaml", "production.yaml"},
	}
	_, _, _ = HandleInstall(context.Background(), nil, input)
	if len(mock.LastInstallOpts.ValuesFiles) != 2 {
		t.Errorf("expected 2 values files, got %d", len(mock.LastInstallOpts.ValuesFiles))
	}
}

// --- Upgrade ---

func TestHandleUpgrade_Success(t *testing.T) {
	mock := setup(t)
	input := UpgradeInput{
		ReleaseName:  "my-release",
		Chart:        "nginx",
		Version:      "2.0.0",
		Install:      true,
		Force:        true,
		ResetValues:  true,
		CleanupOnFail: true,
		MaxHistory:   10,
	}
	result, _, err := HandleUpgrade(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if mock.LastUpgradeOpts.ReleaseName != "my-release" {
		t.Error("release name not passed")
	}
	if !mock.LastUpgradeOpts.Install {
		t.Error("expected Install=true")
	}
	if !mock.LastUpgradeOpts.Force {
		t.Error("expected Force=true")
	}
	if !mock.LastUpgradeOpts.ResetValues {
		t.Error("expected ResetValues=true")
	}
	if !mock.LastUpgradeOpts.CleanupOnFail {
		t.Error("expected CleanupOnFail=true")
	}
	if mock.LastUpgradeOpts.MaxHistory != 10 {
		t.Errorf("expected MaxHistory=10, got %d", mock.LastUpgradeOpts.MaxHistory)
	}
}

func TestHandleUpgrade_V4Options(t *testing.T) {
	mock := setup(t)
	input := UpgradeInput{
		ReleaseName:          "test",
		Chart:                "mychart",
		ServerSideApply:      true,
		TakeOwnership:        true,
		HideSecret:           true,
		ForceConflicts:       true,
		ResetThenReuseValues: true,
	}
	_, _, _ = HandleUpgrade(context.Background(), nil, input)
	if !mock.LastUpgradeOpts.ServerSideApply {
		t.Error("expected ServerSideApply=true")
	}
	if !mock.LastUpgradeOpts.ResetThenReuseValues {
		t.Error("expected ResetThenReuseValues=true")
	}
}

func TestHandleUpgrade_Error(t *testing.T) {
	mock := setup(t)
	mock.UpgradeFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.UpgradeOptions) (*helmengine.ReleaseInfo, error) {
		return nil, errors.New("upgrade failed: incompatible chart")
	}
	result, _, _ := HandleUpgrade(context.Background(), nil, UpgradeInput{ReleaseName: "r", Chart: "c"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Uninstall ---

func TestHandleUninstall_Success(t *testing.T) {
	mock := setup(t)
	input := UninstallInput{
		ReleaseName: "my-release",
		KeepHistory: true,
		DryRun:      true,
		Wait:        true,
		Cascade:     "background",
	}
	result, _, err := HandleUninstall(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if mock.LastUninstallOpts.ReleaseName != "my-release" {
		t.Error("release name not passed")
	}
	if !mock.LastUninstallOpts.KeepHistory {
		t.Error("expected KeepHistory=true")
	}
	if !mock.LastUninstallOpts.DryRun {
		t.Error("expected DryRun=true")
	}
	if mock.LastUninstallOpts.Cascade != "background" {
		t.Error("expected Cascade=background")
	}
}

func TestHandleUninstall_Error(t *testing.T) {
	mock := setup(t)
	mock.UninstallFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.UninstallOptions) (*helmengine.UninstallResult, error) {
		return nil, errors.New("release not found")
	}
	result, _, _ := HandleUninstall(context.Background(), nil, UninstallInput{ReleaseName: "missing"})
	if !result.IsError {
		t.Fatal("expected error")
	}
	if !strings.Contains(extractText(t, result), "release not found") {
		t.Error("error message not propagated")
	}
}

// --- Rollback ---

func TestHandleRollback_Success(t *testing.T) {
	mock := setup(t)
	input := RollbackInput{
		ReleaseName:     "my-release",
		Revision:        2,
		Wait:            true,
		Force:           true,
		DryRun:          true,
		CleanupOnFail:   true,
		MaxHistory:      5,
		ServerSideApply: true,
		ForceConflicts:  true,
	}
	result, _, err := HandleRollback(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Rollback successful") {
		t.Errorf("expected success message, got %s", text)
	}
	if mock.LastRollbackOpts.Revision != 2 {
		t.Errorf("expected revision 2, got %d", mock.LastRollbackOpts.Revision)
	}
	if !mock.LastRollbackOpts.ServerSideApply {
		t.Error("expected ServerSideApply=true")
	}
	if !mock.LastRollbackOpts.ForceConflicts {
		t.Error("expected ForceConflicts=true")
	}
}

func TestHandleRollback_Error(t *testing.T) {
	mock := setup(t)
	mock.RollbackFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.RollbackOptions) error {
		return errors.New("revision 99 not found")
	}
	result, _, _ := HandleRollback(context.Background(), nil, RollbackInput{ReleaseName: "r", Revision: 99})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Status ---

func TestHandleStatus_Success(t *testing.T) {
	mock := setup(t)
	input := StatusInput{
		ReleaseName:   "my-release",
		Revision:      1,
		ShowResources: true,
	}
	result, _, err := HandleStatus(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "my-release") {
		t.Error("expected release info in output")
	}
	if mock.LastStatusOpts.Revision != 1 {
		t.Error("revision not passed")
	}
	if !mock.LastStatusOpts.ShowResources {
		t.Error("expected ShowResources=true")
	}
}

func TestHandleStatus_Error(t *testing.T) {
	mock := setup(t)
	mock.StatusFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.StatusOptions) (*helmengine.ReleaseInfo, error) {
		return nil, errors.New("release not found")
	}
	result, _, _ := HandleStatus(context.Background(), nil, StatusInput{ReleaseName: "missing"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- History ---

func TestHandleHistory_Success(t *testing.T) {
	mock := setup(t)
	input := HistoryInput{
		ReleaseName: "my-release",
		Max:         10,
	}
	result, _, err := HandleHistory(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	// Mock returns 2 revisions
	if !strings.Contains(text, "my-release") {
		t.Error("expected release info in output")
	}
	if mock.LastHistoryOpts.Max != 10 {
		t.Errorf("expected max=10, got %d", mock.LastHistoryOpts.Max)
	}
}

func TestHandleHistory_Error(t *testing.T) {
	mock := setup(t)
	mock.HistoryFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.HistoryOptions) ([]*helmengine.ReleaseInfo, error) {
		return nil, errors.New("release not found")
	}
	result, _, _ := HandleHistory(context.Background(), nil, HistoryInput{ReleaseName: "missing"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Test ---

func TestHandleTest_Success(t *testing.T) {
	mock := setup(t)
	input := TestInput{
		ReleaseName: "my-release",
		Timeout:     "2m",
		Filters:     []string{"test-connection"},
	}
	result, _, err := HandleTest(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if mock.LastTestOpts.Timeout != "2m" {
		t.Error("timeout not passed")
	}
	if len(mock.LastTestOpts.Filters) != 1 || mock.LastTestOpts.Filters[0] != "test-connection" {
		t.Error("filters not passed")
	}
}

func TestHandleTest_Error(t *testing.T) {
	mock := setup(t)
	mock.TestFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.TestOptions) (*helmengine.ReleaseInfo, error) {
		return nil, errors.New("test failed")
	}
	result, _, _ := HandleTest(context.Background(), nil, TestInput{ReleaseName: "r"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- GetAll ---

func TestHandleGetAll_Success(t *testing.T) {
	mock := setup(t)
	input := GetAllInput{
		ReleaseName: "my-release",
		Revision:    2,
	}
	result, _, err := HandleGetAll(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "manifest") {
		t.Error("expected manifest in output")
	}
	if mock.LastGetOpts.Revision != 2 {
		t.Error("revision not passed")
	}
}

// --- GetValues ---

func TestHandleGetValues_Success(t *testing.T) {
	mock := setup(t)
	input := GetValuesInput{
		ReleaseName: "my-release",
		All:         true,
	}
	result, _, err := HandleGetValues(context.Background(), nil, input)
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
	if !mock.LastGetValuesOpts.All {
		t.Error("expected All=true")
	}
}

// --- GetManifest ---

func TestHandleGetManifest_Success(t *testing.T) {
	setup(t)
	input := GetManifestInput{ReleaseName: "my-release"}
	result, _, err := HandleGetManifest(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "apiVersion") {
		t.Error("expected manifest in output")
	}
}

// --- GetHooks ---

func TestHandleGetHooks_Success(t *testing.T) {
	setup(t)
	input := GetHooksInput{ReleaseName: "my-release"}
	result, _, err := HandleGetHooks(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
}

// --- GetMetadata ---

func TestHandleGetMetadata_Success(t *testing.T) {
	setup(t)
	input := GetMetadataInput{ReleaseName: "my-release"}
	result, _, err := HandleGetMetadata(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "my-release") {
		t.Error("expected metadata in output")
	}
}

// --- GetNotes ---

func TestHandleGetNotes_Success(t *testing.T) {
	setup(t)
	input := GetNotesInput{ReleaseName: "my-release"}
	result, _, err := HandleGetNotes(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "NOTES") {
		t.Error("expected notes in output")
	}
}

// --- Config Passthrough ---

func TestKubeAuthPassthrough(t *testing.T) {
	mock := setup(t)
	input := ListInput{
		GlobalInput: tools.GlobalInput{
			KubeContext:       "eks-prod",
			KubeConfig:        "/home/user/.kube/config",
			KubeAPIServer:     "https://ABCDEF.gr7.us-east-1.eks.amazonaws.com",
			KubeBearerToken:   "eyJhbG...",
			KubeTLSServerName: "kubernetes",
			KubeInsecureTLS:   true,
			Debug:             true,
			BurstLimit:        100,
			QPS:               50.0,
		},
	}
	_, _, _ = HandleList(context.Background(), nil, input)
	cfg := mock.LastConfig
	if cfg.KubeContext != "eks-prod" {
		t.Error("KubeContext not passed through")
	}
	if cfg.KubeConfig != "/home/user/.kube/config" {
		t.Error("KubeConfig not passed through")
	}
	if cfg.KubeAPIServer != "https://ABCDEF.gr7.us-east-1.eks.amazonaws.com" {
		t.Error("KubeAPIServer not passed through")
	}
	if cfg.KubeBearerToken != "eyJhbG..." {
		t.Error("KubeBearerToken not passed through")
	}
	if cfg.KubeTLSServerName != "kubernetes" {
		t.Error("KubeTLSServerName not passed through")
	}
	if !cfg.KubeInsecureTLS {
		t.Error("KubeInsecureTLS not passed through")
	}
	if !cfg.Debug {
		t.Error("Debug not passed through")
	}
	if cfg.BurstLimit != 100 {
		t.Error("BurstLimit not passed through")
	}
	if cfg.QPS != 50.0 {
		t.Error("QPS not passed through")
	}
}

// --- Output JSON format ---

func TestHandleInstall_OutputFormat(t *testing.T) {
	setup(t)
	result, _, _ := HandleInstall(context.Background(), nil, InstallInput{ReleaseName: "r", Chart: "c"})
	text := extractText(t, result)
	var rel helmengine.ReleaseInfo
	if err := json.Unmarshal([]byte(text), &rel); err != nil {
		t.Fatalf("output should be valid JSON: %v", err)
	}
	if rel.Name != "my-release" {
		t.Errorf("expected 'my-release', got %q", rel.Name)
	}
	if rel.Updated.IsZero() {
		t.Error("expected non-zero updated time")
	}
}

func TestHandleHistory_MultipleRevisions(t *testing.T) {
	mock := setup(t)
	mock.HistoryFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.HistoryOptions) ([]*helmengine.ReleaseInfo, error) {
		return []*helmengine.ReleaseInfo{
			{Name: "r", Revision: 1, Status: "superseded", Updated: time.Now()},
			{Name: "r", Revision: 2, Status: "deployed", Updated: time.Now()},
			{Name: "r", Revision: 3, Status: "deployed", Updated: time.Now()},
		}, nil
	}
	result, _, _ := HandleHistory(context.Background(), nil, HistoryInput{ReleaseName: "r"})
	text := extractText(t, result)
	var releases []helmengine.ReleaseInfo
	if err := json.Unmarshal([]byte(text), &releases); err != nil {
		t.Fatalf("output should be valid JSON array: %v", err)
	}
	if len(releases) != 3 {
		t.Errorf("expected 3 revisions, got %d", len(releases))
	}
}

// --- Get Error Paths ---

func TestHandleGetAll_Error(t *testing.T) {
	mock := setup(t)
	mock.GetAllFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetOptions) (*helmengine.ReleaseDetail, error) {
		return nil, errors.New("get all failed")
	}
	result, _, err := HandleGetAll(context.Background(), nil, GetAllInput{ReleaseName: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestHandleGetHooks_Error(t *testing.T) {
	mock := setup(t)
	mock.GetHooksFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetOptions) (string, error) {
		return "", errors.New("get hooks failed")
	}
	result, _, err := HandleGetHooks(context.Background(), nil, GetHooksInput{ReleaseName: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestHandleGetManifest_Error(t *testing.T) {
	mock := setup(t)
	mock.GetManifestFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetOptions) (string, error) {
		return "", errors.New("get manifest failed")
	}
	result, _, err := HandleGetManifest(context.Background(), nil, GetManifestInput{ReleaseName: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestHandleGetMetadata_Error(t *testing.T) {
	mock := setup(t)
	mock.GetMetadataFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetOptions) (*helmengine.MetadataInfo, error) {
		return nil, errors.New("get metadata failed")
	}
	result, _, err := HandleGetMetadata(context.Background(), nil, GetMetadataInput{ReleaseName: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestHandleGetNotes_Error(t *testing.T) {
	mock := setup(t)
	mock.GetNotesFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetOptions) (string, error) {
		return "", errors.New("get notes failed")
	}
	result, _, err := HandleGetNotes(context.Background(), nil, GetNotesInput{ReleaseName: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestHandleGetValues_Error(t *testing.T) {
	mock := setup(t)
	mock.GetValuesFn = func(ctx context.Context, cfg *helmengine.GlobalConfig, opts *helmengine.GetValuesOptions) (map[string]interface{}, error) {
		return nil, errors.New("get values failed")
	}
	result, _, err := HandleGetValues(context.Background(), nil, GetValuesInput{ReleaseName: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}
