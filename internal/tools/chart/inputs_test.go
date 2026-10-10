package chart

import (
	"context"
	"strings"
	"testing"

	"github.com/ssddgreg/helm-mcp/internal/tools"
)

func TestHandleTemplate_PassesNewInputs(t *testing.T) {
	mock := setup(t)
	input := TemplateInput{
		ChartSourceInput: tools.ChartSourceInput{Repo: "https://charts.example.com", Password: "p"},
		ValueSetInput:    tools.ValueSetInput{SetString: []string{"a=1"}},
		PostRenderInput:  tools.PostRenderInput{PostRenderer: "kustomize"},
		ReleaseName:      "test",
		Chart:            "mychart",
		ShowOnly:         []string{"templates/svc.yaml"},
		APIVersions:      []string{"example.com/v1"},
		NoHooks:          true,
		SkipTests:        true,
		IsUpgrade:        true,
	}
	result, _, _ := HandleTemplate(context.Background(), nil, input)
	if result.IsError {
		t.Fatalf("unexpected error: %s", extractText(t, result))
	}
	got := mock.LastTemplateOpts
	if got.Repo == "" || len(got.SetString) != 1 || got.PostRenderer != "kustomize" {
		t.Errorf("shared inputs not passed: %+v", got)
	}
	if len(got.ShowOnly) != 1 || len(got.APIVersions) != 1 || !got.NoHooks || !got.SkipTests || !got.IsUpgrade {
		t.Errorf("template inputs not passed: %+v", got)
	}
	if got.Password != "" {
		t.Error("the password must be zeroed once the call returns")
	}
}

func TestHandleLint_PassesNewInputs(t *testing.T) {
	mock := setup(t)
	_, _, _ = HandleLint(context.Background(), nil, LintInput{
		Paths:                []string{"mychart"},
		KubeVersion:          "1.34.0",
		SkipSchemaValidation: true,
	})
	if mock.LastLintOpts.KubeVersion != "1.34.0" || !mock.LastLintOpts.SkipSchemaValidation {
		t.Errorf("lint inputs not passed: %+v", mock.LastLintOpts)
	}
}

func TestPathInputsAreValidated(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		call func() (bool, string)
	}{
		{"lint paths", func() (bool, string) {
			r, _, _ := HandleLint(ctx, nil, LintInput{Paths: []string{"../../etc"}})
			return r.IsError, extractText(t, r)
		}},
		{"create name", func() (bool, string) {
			r, _, _ := HandleCreate(ctx, nil, CreateInput{Name: "../escape"})
			return r.IsError, extractText(t, r)
		}},
		{"create starter", func() (bool, string) {
			r, _, _ := HandleCreate(ctx, nil, CreateInput{Name: "ok", Starter: "../../starter"})
			return r.IsError, extractText(t, r)
		}},
		{"package path", func() (bool, string) {
			r, _, _ := HandlePackage(ctx, nil, PackageInput{Path: "../chart"})
			return r.IsError, extractText(t, r)
		}},
		{"package passphrase_file", func() (bool, string) {
			r, _, _ := HandlePackage(ctx, nil, PackageInput{Path: "chart", PassphraseFile: "../pass"})
			return r.IsError, extractText(t, r)
		}},
		{"pull untar_dir", func() (bool, string) {
			r, _, _ := HandlePull(ctx, nil, PullInput{Chart: "repo/chart", UntarDir: "../out"})
			return r.IsError, extractText(t, r)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)
			isErr, text := tt.call()
			if !isErr || !strings.Contains(text, "..") {
				t.Errorf("expected a traversal rejection, got error=%v %q", isErr, text)
			}
		})
	}
}

func TestHandleCreate_RequiresName(t *testing.T) {
	setup(t)
	r, _, _ := HandleCreate(context.Background(), nil, CreateInput{})
	if !r.IsError || !strings.Contains(extractText(t, r), "name is required") {
		t.Errorf("expected name to be required, got %q", extractText(t, r))
	}
}

func TestHandlePull_VerifyAndProv(t *testing.T) {
	mock := setup(t)
	_, _, _ = HandlePull(context.Background(), nil, PullInput{
		ChartSourceInput: tools.ChartSourceInput{Verify: true, Keyring: "keyring.gpg"},
		Chart:            "repo/chart",
		Prov:             true,
	})
	got := mock.LastPullOpts
	if !got.Verify || !got.Prov || got.Keyring != "keyring.gpg" {
		t.Errorf("verify and prov must be passed separately: %+v", got)
	}
}

func TestHandleShow_ChartSource(t *testing.T) {
	mock := setup(t)
	_, _, _ = HandleShowValues(context.Background(), nil, ShowInput{
		ChartSourceInput: tools.ChartSourceInput{Repo: "https://charts.example.com", InsecureSkipTLSVerify: true},
		Chart:            "nginx",
		JSONPath:         "{.image}",
	})
	got := mock.LastShowOpts
	if got.Repo != "https://charts.example.com" || !got.InsecureSkipTLSVerify || got.JSONPath != "{.image}" {
		t.Errorf("show inputs not passed: %+v", got)
	}
}

func TestHandlePackage_PassphraseFile(t *testing.T) {
	mock := setup(t)
	_, _, _ = HandlePackage(context.Background(), nil, PackageInput{Path: "chart", Sign: true, PassphraseFile: "pass.txt"})
	if mock.LastPackageOpts.PassphraseFile != "pass.txt" {
		t.Errorf("passphrase_file not passed: %+v", mock.LastPackageOpts)
	}
}
