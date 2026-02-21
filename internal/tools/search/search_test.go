package search

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

// --- Search Hub ---

func TestHandleHub_Success(t *testing.T) {
	setup(t)
	input := HubInput{
		Keyword:     "nginx",
		MaxColWidth: 80,
		ListRepoURL: true,
	}
	result, _, err := HandleHub(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "nginx") {
		t.Error("expected search results in output")
	}
}

func TestHandleHub_FieldMapping(t *testing.T) {
	mock := setup(t)
	var capturedOpts *helmengine.SearchHubOptions
	mock.SearchHubFn = func(ctx context.Context, opts *helmengine.SearchHubOptions) ([]*helmengine.SearchResult, error) {
		capturedOpts = opts
		return []*helmengine.SearchResult{}, nil
	}
	input := HubInput{
		Keyword:     "redis",
		MaxColWidth: 100,
		ListRepoURL: true,
	}
	_, _, _ = HandleHub(context.Background(), nil, input)
	if capturedOpts.Keyword != "redis" {
		t.Error("keyword not mapped")
	}
	if capturedOpts.MaxColWidth != 100 {
		t.Error("max_col_width not mapped")
	}
	if !capturedOpts.ListRepoURL {
		t.Error("list_repo_url not mapped")
	}
}

func TestHandleHub_Error(t *testing.T) {
	mock := setup(t)
	mock.SearchHubFn = func(ctx context.Context, opts *helmengine.SearchHubOptions) ([]*helmengine.SearchResult, error) {
		return nil, errors.New("hub unavailable")
	}
	result, _, _ := HandleHub(context.Background(), nil, HubInput{Keyword: "nginx"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}

// --- Search Repo ---

func TestHandleRepo_Success(t *testing.T) {
	setup(t)
	input := RepoInput{
		Keyword:  "nginx",
		Versions: true,
		Devel:    true,
	}
	result, _, err := HandleRepo(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "nginx") {
		t.Error("expected search results")
	}
}

func TestHandleRepo_FieldMapping(t *testing.T) {
	mock := setup(t)
	var capturedOpts *helmengine.SearchRepoOptions
	mock.SearchRepoFn = func(ctx context.Context, opts *helmengine.SearchRepoOptions) ([]*helmengine.SearchResult, error) {
		capturedOpts = opts
		return []*helmengine.SearchResult{}, nil
	}
	input := RepoInput{
		Keyword:           "postgres",
		Regexp:            true,
		Versions:          true,
		Devel:             true,
		VersionConstraint: ">=1.0.0",
	}
	_, _, _ = HandleRepo(context.Background(), nil, input)
	if capturedOpts.Keyword != "postgres" {
		t.Error("keyword not mapped")
	}
	if !capturedOpts.Regexp {
		t.Error("regexp not mapped")
	}
	if !capturedOpts.Versions {
		t.Error("versions not mapped")
	}
	if !capturedOpts.Devel {
		t.Error("devel not mapped")
	}
	if capturedOpts.VersionConstraint != ">=1.0.0" {
		t.Error("version_constraint not mapped")
	}
}

func TestHandleRepo_Error(t *testing.T) {
	mock := setup(t)
	mock.SearchRepoFn = func(ctx context.Context, opts *helmengine.SearchRepoOptions) ([]*helmengine.SearchResult, error) {
		return nil, errors.New("no repositories configured")
	}
	result, _, _ := HandleRepo(context.Background(), nil, RepoInput{Keyword: "anything"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}
