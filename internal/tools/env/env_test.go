package env

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

// --- Env ---

func TestHandleEnv_Success(t *testing.T) {
	setup(t)
	result, _, err := HandleEnv(context.Background(), nil, EnvInput{})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "HELM_CACHE_HOME") {
		t.Error("expected env variables in output")
	}
	if !strings.Contains(text, "HELM_CONFIG_HOME") {
		t.Error("expected HELM_CONFIG_HOME in output")
	}
}

func TestHandleEnv_Error(t *testing.T) {
	mock := setup(t)
	mock.EnvFn = func(ctx context.Context) (map[string]string, error) {
		return nil, errors.New("helm not configured")
	}
	result, _, _ := HandleEnv(context.Background(), nil, EnvInput{})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Version ---

func TestHandleVersion_Success(t *testing.T) {
	setup(t)
	result, _, err := HandleVersion(context.Background(), nil, VersionInput{})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "v3.20.0") {
		t.Error("expected version in output")
	}
	if !strings.Contains(text, "git_commit") {
		t.Error("expected git commit in output")
	}
}

func TestHandleVersion_Short(t *testing.T) {
	setup(t)
	result, _, err := HandleVersion(context.Background(), nil, VersionInput{Short: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if text != "v3.20.0" {
		t.Errorf("expected just version string, got %q", text)
	}
}

func TestHandleVersion_Error(t *testing.T) {
	mock := setup(t)
	mock.VersionFn = func(ctx context.Context) (*helmengine.VersionInfo, error) {
		return nil, errors.New("version unavailable")
	}
	result, _, _ := HandleVersion(context.Background(), nil, VersionInput{})
	if !result.IsError {
		t.Fatal("expected error")
	}
}
