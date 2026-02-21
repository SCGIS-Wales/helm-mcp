package registry

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

// --- Login ---

func TestHandleLogin_Success(t *testing.T) {
	setup(t)
	input := LoginInput{
		Hostname: "registry.example.com",
		Username: "admin",
		Password: "password",
		Insecure: true,
	}
	result, _, err := HandleLogin(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Login successful") {
		t.Error("expected success message")
	}
}

func TestHandleLogin_FieldMapping(t *testing.T) {
	mock := setup(t)
	var capturedOpts *helmengine.RegistryLoginOptions
	mock.RegistryLoginFn = func(ctx context.Context, opts *helmengine.RegistryLoginOptions) error {
		capturedOpts = opts
		return nil
	}
	input := LoginInput{
		Hostname: "ghcr.io",
		Username: "user",
		Password: "token",
		Insecure: true,
		CAFile:   "/path/ca.pem",
	}
	_, _, _ = HandleLogin(context.Background(), nil, input)
	if capturedOpts.Hostname != "ghcr.io" {
		t.Error("hostname not mapped")
	}
	if capturedOpts.Username != "user" {
		t.Error("username not mapped")
	}
	if capturedOpts.Password != "token" {
		t.Error("password not mapped")
	}
	if !capturedOpts.Insecure {
		t.Error("insecure not mapped")
	}
	if capturedOpts.CAFile != "/path/ca.pem" {
		t.Error("ca_file not mapped")
	}
}

func TestHandleLogin_Error(t *testing.T) {
	mock := setup(t)
	mock.RegistryLoginFn = func(ctx context.Context, opts *helmengine.RegistryLoginOptions) error {
		return errors.New("authentication failed")
	}
	result, _, _ := HandleLogin(context.Background(), nil, LoginInput{Hostname: "registry.example.com"})
	if !result.IsError {
		t.Fatal("expected error")
	}
	if !strings.Contains(extractText(t, result), "authentication failed") {
		t.Error("error message not propagated")
	}
}

// --- Logout ---

func TestHandleLogout_Success(t *testing.T) {
	setup(t)
	input := LogoutInput{Hostname: "registry.example.com"}
	result, _, err := HandleLogout(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	text := extractText(t, result)
	if !strings.Contains(text, "Logout successful") {
		t.Error("expected success message")
	}
}

func TestHandleLogout_Error(t *testing.T) {
	mock := setup(t)
	mock.RegistryLogoutFn = func(ctx context.Context, opts *helmengine.RegistryLogoutOptions) error {
		return errors.New("not logged in")
	}
	result, _, _ := HandleLogout(context.Background(), nil, LogoutInput{Hostname: "registry.example.com"})
	if !result.IsError {
		t.Fatal("expected error")
	}
}
