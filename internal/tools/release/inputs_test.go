package release

import (
	"context"
	"strings"
	"testing"

	"github.com/ssddgreg/helm-mcp/internal/tools"
)

func TestHandleInstall_PassesNewInputs(t *testing.T) {
	mock := setup(t)
	f := false
	input := InstallInput{
		ChartSourceInput: tools.ChartSourceInput{
			Repo:     "https://charts.example.com",
			Username: "user",
			Password: "s3cret",
			Verify:   true,
		},
		ValueSetInput:   tools.ValueSetInput{SetString: []string{"a=007"}, SetLiteral: []string{"b=x,y"}},
		PostRenderInput: tools.PostRenderInput{PostRenderer: "kustomize", PostRendererArgs: []string{"--x"}},
		ReleaseName:     "test",
		Chart:           "mychart",
		ForceReplace:    true,
		ServerSideApply: &f,
	}
	result, _, _ := HandleInstall(context.Background(), nil, input)
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}

	got := mock.LastInstallOpts
	if got.Repo != "https://charts.example.com" || got.Username != "user" || !got.Verify {
		t.Errorf("chart source not passed: %+v", got.ChartSourceOptions)
	}
	if got.Password != "" {
		t.Error("the password must be zeroed once the call returns")
	}
	if len(got.SetString) != 1 || len(got.SetLiteral) != 1 {
		t.Errorf("value sets not passed: %+v", got.ValueSetOptions)
	}
	if got.PostRenderer != "kustomize" || len(got.PostRendererArgs) != 1 {
		t.Errorf("post renderer not passed: %+v", got.PostRenderOptions)
	}
	if !got.ForceReplace {
		t.Error("force_replace not passed")
	}
	if got.ServerSideApply == nil || *got.ServerSideApply {
		t.Error("server_side_apply=false must reach the engine as an explicit false")
	}
}

func TestHandleInstall_ServerSideApplyUnsetStaysNil(t *testing.T) {
	mock := setup(t)
	_, _, _ = HandleInstall(context.Background(), nil, InstallInput{ReleaseName: "test", Chart: "mychart"})
	if mock.LastInstallOpts.ServerSideApply != nil {
		t.Error("unset server_side_apply must stay nil so Helm's own default applies")
	}
}

func TestReleaseTools_RejectInvalidNewInputs(t *testing.T) {
	tests := []struct {
		name  string
		input InstallInput
		want  string
	}{
		{
			name:  "set_file URL",
			input: InstallInput{ValueSetInput: tools.ValueSetInput{SetFile: []string{"k=https://example.com/x"}}},
			want:  "set_file only reads local files",
		},
		{
			name:  "set_file traversal",
			input: InstallInput{ValueSetInput: tools.ValueSetInput{SetFile: []string{"k=../../etc/passwd"}}},
			want:  "must not contain '..'",
		},
		{
			name:  "post_renderer name",
			input: InstallInput{PostRenderInput: tools.PostRenderInput{PostRenderer: "../bin/sh"}},
			want:  "plugin name",
		},
		{
			name:  "repo URL",
			input: InstallInput{ChartSourceInput: tools.ChartSourceInput{Repo: "file:///etc"}},
			want:  "must start with https://",
		},
		{
			name:  "ca_file traversal",
			input: InstallInput{ChartSourceInput: tools.ChartSourceInput{CAFile: "../ca.pem"}},
			want:  "must not contain '..'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := setup(t)
			in := tt.input
			in.ReleaseName = "test"
			in.Chart = "mychart"
			result, _, _ := HandleInstall(context.Background(), nil, in)
			if !result.IsError {
				t.Fatal("expected a validation error")
			}
			if text := extractText(t, result); !strings.Contains(strings.ToLower(text), strings.ToLower(tt.want)) {
				t.Errorf("error %q should mention %q", text, tt.want)
			}
			if mock.LastInstallOpts != nil {
				t.Error("the engine must not be called on invalid input")
			}
		})
	}
}

func TestHandleUpgrade_PassesNewInputs(t *testing.T) {
	mock := setup(t)
	input := UpgradeInput{
		ChartSourceInput: tools.ChartSourceInput{PlainHTTP: true, Password: "p"},
		ValueSetInput:    tools.ValueSetInput{SetString: []string{"a=1"}},
		ReleaseName:      "test",
		Chart:            "oci://registry.example.com/charts/app",
	}
	result, _, _ := HandleUpgrade(context.Background(), nil, input)
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}
	got := mock.LastUpgradeOpts
	if !got.PlainHTTP || len(got.SetString) != 1 {
		t.Errorf("new inputs not passed: %+v", got)
	}
	if got.ServerSideApply != nil {
		t.Error("unset server_side_apply must stay nil (Helm's auto)")
	}
	if got.Password != "" {
		t.Error("the password must be zeroed once the call returns")
	}
}

func TestHandleRollback_Description(t *testing.T) {
	mock := setup(t)
	_, _, _ = HandleRollback(context.Background(), nil, RollbackInput{ReleaseName: "test", Revision: 1, Description: "revert bad deploy"})
	if mock.LastRollbackOpts.Description != "revert bad deploy" {
		t.Errorf("description not passed: %+v", mock.LastRollbackOpts)
	}
	if mock.LastRollbackOpts.ServerSideApply != nil {
		t.Error("unset server_side_apply must stay nil (Helm's auto)")
	}
}

func TestHandleTest_Logs(t *testing.T) {
	mock := setup(t)
	_, _, _ = HandleTest(context.Background(), nil, TestInput{ReleaseName: "test", Logs: true})
	if !mock.LastTestOpts.Logs {
		t.Error("logs not passed")
	}
}
