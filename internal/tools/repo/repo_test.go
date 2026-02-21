package repo

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

// --- Repo Add ---

func TestHandleAdd_Success(t *testing.T) {
	mock := setup(t)
	input := AddInput{
		Name:        "bitnami",
		URL:         "https://charts.bitnami.com/bitnami",
		Username:    "user",
		Password:    "pass",
		ForceUpdate: true,
	}
	result, _, err := HandleAdd(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Repository added successfully") {
		t.Error("expected success message")
	}
	// Verify mock was called (RepoAddFn is nil so default behavior)
	_ = mock
}

func TestHandleAdd_Error(t *testing.T) {
	mock := setup(t)
	mock.RepoAddFn = func(ctx context.Context, opts *helmengine.RepoAddOptions) error {
		return errors.New("repository already exists")
	}
	result, _, _ := HandleAdd(context.Background(), nil, AddInput{Name: "existing", URL: "https://example.com"})
	if !result.IsError {
		t.Fatal("expected error")
	}
	if !strings.Contains(extractText(t, result), "repository already exists") {
		t.Error("error message not propagated")
	}
}

func TestHandleAdd_FieldMapping(t *testing.T) {
	mock := setup(t)
	var capturedOpts *helmengine.RepoAddOptions
	mock.RepoAddFn = func(ctx context.Context, opts *helmengine.RepoAddOptions) error {
		capturedOpts = opts
		return nil
	}
	input := AddInput{
		Name:            "stable",
		URL:             "https://charts.helm.sh/stable",
		Username:        "admin",
		Password:        "secret",
		ForceUpdate:     true,
		CAFile:          "/path/ca.pem",
		InsecureSkipTLS: true,
	}
	_, _, _ = HandleAdd(context.Background(), nil, input)
	if capturedOpts.Name != "stable" {
		t.Error("name not mapped")
	}
	if capturedOpts.URL != "https://charts.helm.sh/stable" {
		t.Error("url not mapped")
	}
	if capturedOpts.Username != "admin" {
		t.Error("username not mapped")
	}
	if !capturedOpts.ForceUpdate {
		t.Error("force_update not mapped")
	}
	if capturedOpts.CAFile != "/path/ca.pem" {
		t.Error("ca_file not mapped")
	}
	if !capturedOpts.InsecureSkipTLSVerify {
		t.Error("insecure_skip_tls not mapped to InsecureSkipTLSVerify")
	}
}

// --- Repo List ---

func TestHandleList_Success(t *testing.T) {
	setup(t)
	input := ListInput{}
	result, _, err := HandleList(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "stable") {
		t.Error("expected repo list in output")
	}
	if !strings.Contains(text, "bitnami") {
		t.Error("expected bitnami in output")
	}
}

func TestHandleList_Error(t *testing.T) {
	mock := setup(t)
	mock.RepoListFn = func(ctx context.Context, opts *helmengine.RepoListOptions) ([]*helmengine.RepoEntry, error) {
		return nil, errors.New("no repositories configured")
	}
	result, _, _ := HandleList(context.Background(), nil, ListInput{})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Repo Update ---

func TestHandleUpdate_Success(t *testing.T) {
	setup(t)
	input := UpdateInput{Names: []string{"stable", "bitnami"}}
	result, _, err := HandleUpdate(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Happy Helming") {
		t.Error("expected update message")
	}
}

func TestHandleUpdate_AllRepos(t *testing.T) {
	mock := setup(t)
	var capturedOpts *helmengine.RepoUpdateOptions
	mock.RepoUpdateFn = func(ctx context.Context, opts *helmengine.RepoUpdateOptions) (string, error) {
		capturedOpts = opts
		return "Updated all repos", nil
	}
	_, _, _ = HandleUpdate(context.Background(), nil, UpdateInput{})
	if len(capturedOpts.Names) != 0 {
		t.Error("expected empty names (update all)")
	}
}

func TestHandleUpdate_Error(t *testing.T) {
	mock := setup(t)
	mock.RepoUpdateFn = func(ctx context.Context, opts *helmengine.RepoUpdateOptions) (string, error) {
		return "", errors.New("network error")
	}
	result, _, _ := HandleUpdate(context.Background(), nil, UpdateInput{})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Repo Remove ---

func TestHandleRemove_Success(t *testing.T) {
	setup(t)
	input := RemoveInput{Names: []string{"old-repo"}}
	result, _, err := HandleRemove(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Repositories removed successfully") {
		t.Error("expected success message")
	}
}

func TestHandleRemove_Error(t *testing.T) {
	mock := setup(t)
	mock.RepoRemoveFn = func(ctx context.Context, opts *helmengine.RepoRemoveOptions) error {
		return errors.New("repository not found")
	}
	result, _, _ := HandleRemove(context.Background(), nil, RemoveInput{Names: []string{"missing"}})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Repo Index ---

func TestHandleIndex_Success(t *testing.T) {
	setup(t)
	input := IndexInput{
		Directory: "/tmp/charts",
		URL:       "https://charts.example.com",
		Merge:     "/tmp/existing-index.yaml",
	}
	result, _, err := HandleIndex(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Index file generated successfully") {
		t.Error("expected success message")
	}
}

func TestHandleIndex_Error(t *testing.T) {
	mock := setup(t)
	mock.RepoIndexFn = func(ctx context.Context, opts *helmengine.RepoIndexOptions) error {
		return errors.New("directory not found")
	}
	result, _, _ := HandleIndex(context.Background(), nil, IndexInput{Directory: "/invalid"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}
