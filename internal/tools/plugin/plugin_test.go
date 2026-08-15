package plugin

import (
	"context"
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

// --- Plugin Install ---

func TestHandleInstall_Success(t *testing.T) {
	setup(t)
	input := InstallInput{
		URLOrPath: "https://github.com/databus23/helm-diff",
		Version:   "3.8.1",
	}
	result, _, err := HandleInstall(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Plugin installed successfully") {
		t.Error("expected success message")
	}
}

func TestHandleInstall_FieldMapping(t *testing.T) {
	mock := setup(t)
	var capturedOpts *helmengine.PluginInstallOptions
	mock.PluginInstallFn = func(ctx context.Context, opts *helmengine.PluginInstallOptions) error {
		capturedOpts = opts
		return nil
	}
	input := InstallInput{
		URLOrPath: "https://github.com/example/plugin",
		Version:   "1.0.0",
	}
	_, _, _ = HandleInstall(context.Background(), nil, input)
	if capturedOpts.URLOrPath != "https://github.com/example/plugin" {
		t.Error("url_or_path not mapped")
	}
	if capturedOpts.Version != "1.0.0" {
		t.Error("version not mapped")
	}
}

func TestHandleInstall_Error(t *testing.T) {
	mock := setup(t)
	mock.PluginInstallFn = func(ctx context.Context, opts *helmengine.PluginInstallOptions) error {
		return errors.New("plugin already exists")
	}
	result, _, _ := HandleInstall(context.Background(), nil, InstallInput{URLOrPath: "http://example.com/plugin"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Plugin List ---

func TestHandleList_Success(t *testing.T) {
	setup(t)
	result, _, err := HandleList(context.Background(), nil, ListInput{})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "diff") {
		t.Error("expected plugin list in output")
	}
}

func TestHandleList_Error(t *testing.T) {
	mock := setup(t)
	mock.PluginListFn = func(ctx context.Context) ([]*helmengine.PluginInfo, error) {
		return nil, errors.New("helm home not found")
	}
	result, _, _ := HandleList(context.Background(), nil, ListInput{})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Plugin Uninstall ---

func TestHandleUninstall_Success(t *testing.T) {
	setup(t)
	input := UninstallInput{Name: "diff"}
	result, _, err := HandleUninstall(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Plugin uninstalled successfully") {
		t.Error("expected success message")
	}
}

func TestHandleUninstall_Error(t *testing.T) {
	mock := setup(t)
	mock.PluginUninstallFn = func(ctx context.Context, opts *helmengine.PluginUninstallOptions) error {
		return errors.New("plugin not found")
	}
	result, _, _ := HandleUninstall(context.Background(), nil, UninstallInput{Name: "missing"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Plugin Update ---

func TestHandleUpdate_Success(t *testing.T) {
	setup(t)
	input := UpdateInput{Name: "diff"}
	result, _, err := HandleUpdate(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Plugin updated successfully") {
		t.Error("expected success message")
	}
}

func TestHandleUpdate_Error(t *testing.T) {
	mock := setup(t)
	mock.PluginUpdateFn = func(ctx context.Context, opts *helmengine.PluginUpdateOptions) error {
		return errors.New("no updates available")
	}
	result, _, _ := HandleUpdate(context.Background(), nil, UpdateInput{Name: "diff"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Plugin Package (Helm v4 only) ---

func TestHandlePackage_Success(t *testing.T) {
	setup(t)
	result, _, err := HandlePackage(context.Background(), nil, PackageInput{
		PluginPath: "./example-plugin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %q", extractText(t, result))
	}
	if !strings.Contains(extractText(t, result), "Successfully packaged plugin") {
		t.Error("expected the packaging output to be returned")
	}
}

func TestHandlePackage_FieldMapping(t *testing.T) {
	mock := setup(t)
	var captured *helmengine.PluginPackageOptions
	mock.PluginPackageFn = func(_ context.Context, opts *helmengine.PluginPackageOptions) (string, error) {
		captured = opts
		return "ok", nil
	}

	// no_sign is the inverse of the engine's Sign field: signing is on by
	// default, and the tool argument is the opt-out.
	if _, _, err := HandlePackage(context.Background(), nil, PackageInput{
		PluginPath:  "./example-plugin",
		Destination: "/tmp/out",
		Key:         "signing-key",
		Keyring:     "/keys/pubring.gpg",
	}); err != nil {
		t.Fatal(err)
	}
	if captured == nil {
		t.Fatal("engine was not called")
	}
	if !captured.Sign {
		t.Error("Sign should default to true when no_sign is unset")
	}
	if captured.Destination != "/tmp/out" || captured.Key != "signing-key" || captured.Keyring != "/keys/pubring.gpg" {
		t.Errorf("options not forwarded: %+v", captured)
	}

	captured = nil
	if _, _, err := HandlePackage(context.Background(), nil, PackageInput{
		PluginPath: "./example-plugin",
		NoSign:     true,
	}); err != nil {
		t.Fatal(err)
	}
	if captured.Sign {
		t.Error("Sign should be false when no_sign is set")
	}
}

func TestHandlePackage_RequiresPluginPath(t *testing.T) {
	setup(t)
	result, _, err := HandlePackage(context.Background(), nil, PackageInput{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error when plugin_path is missing")
	}
	if !strings.Contains(extractText(t, result), "plugin_path is required") {
		t.Errorf("unexpected message: %q", extractText(t, result))
	}
}

func TestHandlePackage_Error(t *testing.T) {
	mock := setup(t)
	mock.PluginPackageFn = func(_ context.Context, _ *helmengine.PluginPackageOptions) (string, error) {
		return "", errors.New("signing key not found")
	}
	result, _, err := HandlePackage(context.Background(), nil, PackageInput{PluginPath: "./p"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error result")
	}
}

// --- Plugin Verify (Helm v4 only) ---

func TestHandleVerify_Success(t *testing.T) {
	mock := setup(t)
	var captured *helmengine.PluginVerifyOptions
	mock.PluginVerifyFn = func(_ context.Context, opts *helmengine.PluginVerifyOptions) (string, error) {
		captured = opts
		return "Signed by: Someone", nil
	}

	result, _, err := HandleVerify(context.Background(), nil, VerifyInput{
		PluginPath: "./example-plugin-0.1.0.tgz",
		Keyring:    "/keys/pubring.gpg",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %q", extractText(t, result))
	}
	if captured.PluginPath != "./example-plugin-0.1.0.tgz" || captured.Keyring != "/keys/pubring.gpg" {
		t.Errorf("options not forwarded: %+v", captured)
	}
	if !strings.Contains(extractText(t, result), "Signed by") {
		t.Error("expected verification output to be returned")
	}
}

func TestHandleVerify_RequiresPluginPath(t *testing.T) {
	setup(t)
	result, _, err := HandleVerify(context.Background(), nil, VerifyInput{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error when plugin_path is missing")
	}
}

func TestHandleVerify_Error(t *testing.T) {
	mock := setup(t)
	mock.PluginVerifyFn = func(_ context.Context, _ *helmengine.PluginVerifyOptions) (string, error) {
		return "", errors.New("provenance file not found")
	}
	result, _, err := HandleVerify(context.Background(), nil, VerifyInput{PluginPath: "./p.tgz"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error result")
	}
}
