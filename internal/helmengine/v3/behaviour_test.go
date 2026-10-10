package v3

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"
)

// isolateHelm points every Helm home and the kubeconfig at a temporary
// directory so tests never read or write the developer's real Helm state.
func isolateHelm(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HELM_CACHE_HOME", filepath.Join(dir, "cache"))
	t.Setenv("HELM_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("HELM_DATA_HOME", filepath.Join(dir, "data"))
	t.Setenv("KUBECONFIG", filepath.Join(dir, "no-kubeconfig"))
	return dir
}

// writeFiles writes files (relative path -> content) under root.
func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// writeHookChart writes a chart with a ConfigMap, a pre-install hook and a
// test hook, whose ConfigMap echoes the values and capabilities under test.
func writeHookChart(t *testing.T, root string) string {
	t.Helper()
	chartDir := filepath.Join(root, "hooks")
	writeFiles(t, chartDir, map[string]string{
		"Chart.yaml":  "apiVersion: v2\nname: hooks\nversion: 0.1.0\n",
		"values.yaml": "image:\n  repository: nginx\n  tag: \"1.0\"\n",
		"templates/cm.yaml": `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-cm
data:
  repository: {{ .Values.image.repository | quote }}
  tag: {{ .Values.image.tag | quote }}
  tagKind: {{ kindOf .Values.image.tag | quote }}
  api: {{ .Capabilities.APIVersions.Has "example.com/v1" | quote }}
  upgrade: {{ .Release.IsUpgrade | quote }}
`,
		"templates/hook.yaml": `apiVersion: batch/v1
kind: Job
metadata:
  name: pre-install-job
  annotations:
    "helm.sh/hook": pre-install
spec:
  template:
    spec:
      restartPolicy: Never
      containers: [{name: c, image: busybox}]
`,
		"templates/tests/test.yaml": `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  annotations:
    "helm.sh/hook": test
spec:
  containers: [{name: c, image: busybox}]
`,
	})
	return chartDir
}

func TestTemplate_HooksAndOptions(t *testing.T) {
	dir := isolateHelm(t)
	chartDir := writeHookChart(t, dir)
	valuesFile := filepath.Join(dir, "set-file.txt")
	writeFiles(t, dir, map[string]string{"set-file.txt": "from-file"})

	tests := []struct {
		name    string
		opts    helmengine.TemplateOptions
		want    []string
		notWant []string
	}{
		{
			name: "hooks are rendered by default",
			want: []string{"kind: ConfigMap", "kind: Job", "kind: Pod"},
		},
		{
			name:    "no_hooks drops every hook",
			opts:    helmengine.TemplateOptions{NoHooks: true},
			want:    []string{"kind: ConfigMap"},
			notWant: []string{"kind: Job", "kind: Pod"},
		},
		{
			name:    "skip_tests drops only test hooks",
			opts:    helmengine.TemplateOptions{SkipTests: true},
			want:    []string{"kind: Job"},
			notWant: []string{"kind: Pod"},
		},
		{
			name:    "show_only filters by template path",
			opts:    helmengine.TemplateOptions{ShowOnly: []string{"templates/cm.yaml"}},
			want:    []string{"kind: ConfigMap"},
			notWant: []string{"kind: Job", "kind: Pod"},
		},
		{
			name: "api_versions reach the capabilities",
			opts: helmengine.TemplateOptions{APIVersions: []string{"example.com/v1"}},
			want: []string{`api: "true"`},
		},
		{
			name: "is_upgrade sets .Release.IsUpgrade",
			opts: helmengine.TemplateOptions{IsUpgrade: true},
			want: []string{`upgrade: "true"`},
		},
		{
			name: "inline values deep-merge over chart values",
			opts: helmengine.TemplateOptions{Values: map[string]interface{}{
				"image": map[string]interface{}{"tag": "2.0"},
			}},
			want: []string{`repository: "nginx"`, `tag: "2.0"`},
		},
		{
			name: "set_string keeps the string form",
			opts: helmengine.TemplateOptions{ValueSetOptions: helmengine.ValueSetOptions{
				SetString: []string{"image.tag=007"},
			}},
			want: []string{`tag: "007"`, `tagKind: "string"`},
		},
		{
			name: "set_file reads the file",
			opts: helmengine.TemplateOptions{ValueSetOptions: helmengine.ValueSetOptions{
				SetFile: []string{"image.tag=" + valuesFile},
			}},
			want: []string{`tag: "from-file"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.opts
			opts.ReleaseName = "demo"
			opts.Chart = chartDir
			out, err := New().Template(context.Background(), &helmengine.GlobalConfig{Namespace: "default"}, &opts)
			if err != nil {
				t.Fatalf("Template: %v", err)
			}
			for _, w := range tt.want {
				if !strings.Contains(out, w) {
					t.Errorf("output missing %q:\n%s", w, out)
				}
			}
			for _, nw := range tt.notWant {
				if strings.Contains(out, nw) {
					t.Errorf("output should not contain %q:\n%s", nw, out)
				}
			}
		})
	}
}

func TestTemplate_ShowOnlyUnknownTemplate(t *testing.T) {
	dir := isolateHelm(t)
	_, err := New().Template(context.Background(), &helmengine.GlobalConfig{}, &helmengine.TemplateOptions{
		ReleaseName: "demo",
		Chart:       writeHookChart(t, dir),
		ShowOnly:    []string{"templates/missing.yaml"},
	})
	if err == nil || !strings.Contains(err.Error(), "could not find template") {
		t.Fatalf("expected a missing template error, got %v", err)
	}
}

// writeDependentChart writes a parent chart that depends on a sibling chart
// through a file:// repository, without vendoring it.
func writeDependentChart(t *testing.T, root string) string {
	t.Helper()
	writeFiles(t, root, map[string]string{
		"sub/Chart.yaml":          "apiVersion: v2\nname: sub\nversion: 0.1.0\n",
		"sub/templates/cm.yaml":   "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: from-subchart\n",
		"parent/Chart.yaml":       "apiVersion: v2\nname: parent\nversion: 0.1.0\ndependencies:\n- name: sub\n  version: 0.1.0\n  repository: file://../sub\n",
		"parent/templates/x.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: from-parent\n",
	})
	return filepath.Join(root, "parent")
}

func TestTemplate_DependencyUpdate(t *testing.T) {
	dir := isolateHelm(t)
	parent := writeDependentChart(t, dir)
	cfg := &helmengine.GlobalConfig{}

	_, err := New().Template(context.Background(), cfg, &helmengine.TemplateOptions{ReleaseName: "demo", Chart: parent})
	if err == nil || !strings.Contains(err.Error(), "dependency_update") {
		t.Fatalf("missing dependencies should be reported, got %v", err)
	}

	out, err := New().Template(context.Background(), cfg, &helmengine.TemplateOptions{
		ReleaseName: "demo", Chart: parent, DependencyUpdate: true,
	})
	if err != nil {
		t.Fatalf("dependency_update should fetch the subchart: %v", err)
	}
	if !strings.Contains(out, "from-subchart") {
		t.Errorf("subchart should be rendered:\n%s", out)
	}
}

// TestDependencyUpdate_DoesNotPanic is a regression test: the downloader
// manager was built without an Out writer and panicked on the first
// dependency it downloaded.
func TestDependencyUpdate_DoesNotPanic(t *testing.T) {
	dir := isolateHelm(t)
	parent := writeDependentChart(t, dir)
	e := New()

	if err := e.DependencyUpdate(context.Background(), &helmengine.GlobalConfig{}, &helmengine.DependencyOptions{ChartPath: parent}); err != nil {
		t.Fatalf("DependencyUpdate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(parent, "charts", "sub-0.1.0.tgz")); err != nil {
		t.Errorf("subchart should be vendored: %v", err)
	}
	if err := e.DependencyBuild(context.Background(), &helmengine.GlobalConfig{}, &helmengine.DependencyOptions{ChartPath: parent}); err != nil {
		t.Fatalf("DependencyBuild: %v", err)
	}
}

func TestCreate_PathAndStarter(t *testing.T) {
	dir := isolateHelm(t)
	e := New()

	path, err := e.Create(context.Background(), &helmengine.CreateOptions{Name: filepath.Join(dir, "plain")})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if path != filepath.Join(dir, "plain") {
		t.Errorf("chart should be created at the given path, got %s", path)
	}

	writeFiles(t, filepath.Join(dir, "data", "starters", "mystarter"), map[string]string{
		"Chart.yaml":          "apiVersion: v2\nname: mystarter\nversion: 9.9.9\n",
		"templates/only.yaml": "# from the starter\n",
	})
	path, err = e.Create(context.Background(), &helmengine.CreateOptions{
		Name:    filepath.Join(dir, "fromstarter"),
		Starter: "mystarter",
	})
	if err != nil {
		t.Fatalf("Create from starter: %v", err)
	}
	if _, err := os.Stat(filepath.Join(path, "templates", "only.yaml")); err != nil {
		t.Errorf("starter templates should be copied: %v", err)
	}
	chartYAML, err := os.ReadFile(filepath.Clean(filepath.Join(path, "Chart.yaml")))
	if err != nil || !strings.Contains(string(chartYAML), "name: fromstarter") {
		t.Errorf("Chart.yaml should carry the new name, got %q (%v)", chartYAML, err)
	}
}

func TestLint_ValuesFilesAndSubcharts(t *testing.T) {
	dir := isolateHelm(t)
	writeFiles(t, dir, map[string]string{
		"needs/Chart.yaml":                    "apiVersion: v2\nname: needs\nversion: 0.1.0\ndependencies:\n- name: sub\n  version: 0.1.0\n",
		"needs/templates/cm.yaml":             "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cm\ndata:\n  foo: {{ .Values.foo | quote }}\n",
		"needs/values.schema.json":            `{"type": "object", "required": ["foo"], "properties": {"foo": {"type": "string"}}}`,
		"needs/charts/sub/Chart.yaml":         "apiVersion: v2\nname: sub\nversion: 0.1.0\n",
		"needs/charts/sub/templates/.gitkeep": "",
		"values.yaml":                         "foo: bar\n",
	})
	chartDir := filepath.Join(dir, "needs")
	e := New()

	res, err := e.Lint(context.Background(), &helmengine.LintOptions{Paths: []string{chartDir}})
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}
	if res.Passed {
		t.Fatal("lint should fail without the required value")
	}

	res, err = e.Lint(context.Background(), &helmengine.LintOptions{Paths: []string{chartDir}, SkipSchemaValidation: true})
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}
	if !res.Passed {
		t.Errorf("skip_schema_validation should ignore values.schema.json: %+v", res.Messages)
	}

	res, err = e.Lint(context.Background(), &helmengine.LintOptions{
		Paths:       []string{chartDir},
		ValuesFiles: []string{filepath.Join(dir, "values.yaml")},
	})
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}
	if !res.Passed {
		t.Errorf("values_files should satisfy the required value: %+v", res.Messages)
	}

	res, err = e.Lint(context.Background(), &helmengine.LintOptions{
		Paths:         []string{chartDir},
		ValuesFiles:   []string{filepath.Join(dir, "values.yaml")},
		WithSubcharts: true,
	})
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}
	if res.TotalCharts != 2 {
		t.Errorf("with_subcharts should lint the subchart too, linted %d", res.TotalCharts)
	}
}

// TestParityFieldsAccepted pins down that fields Helm v3.22 supports are
// passed through rather than rejected as "v4 only", which they used to be.
func TestParityFieldsAccepted(t *testing.T) {
	dir := isolateHelm(t)
	chartDir := writeHookChart(t, dir)
	f := false
	e := New()
	ctx := context.Background()
	cfg := &helmengine.GlobalConfig{Namespace: "default"}

	_, err := e.Install(ctx, cfg, &helmengine.InstallOptions{
		ReleaseName:     "demo",
		Chart:           chartDir,
		DryRun:          "client",
		TakeOwnership:   true,
		HideSecret:      true,
		ForceReplace:    true,
		ServerSideApply: &f, // explicit client-side apply is what v3 does anyway
	})
	if err != nil {
		t.Fatalf("Install should accept v3-supported fields: %v", err)
	}

	checks := map[string]error{}
	_, checks["upgrade"] = e.Upgrade(ctx, cfg, &helmengine.UpgradeOptions{
		ReleaseName: "demo", Chart: chartDir, DryRun: "client",
		TakeOwnership: true, HideSecret: true, ResetThenReuseValues: true,
	})
	_, checks["list"] = e.List(ctx, cfg, &helmengine.ListOptions{Selector: "team=a"})
	_, checks["status"] = e.Status(ctx, cfg, &helmengine.StatusOptions{ReleaseName: "demo", ShowResources: true})
	_, checks["show"] = e.ShowValues(ctx, cfg, &helmengine.ShowOptions{Chart: chartDir, JSONPath: "{.image.tag}"})
	for name, err := range checks {
		if err != nil && strings.Contains(err.Error(), "only supported in Helm v4") {
			t.Errorf("%s: field wrongly rejected as v4-only: %v", name, err)
		}
	}

	out, err := e.ShowValues(ctx, cfg, &helmengine.ShowOptions{Chart: chartDir, JSONPath: "{.image.tag}"})
	if err != nil || strings.TrimSpace(out) != "1.0" {
		t.Errorf("jsonpath should apply on v3, got %q, %v", out, err)
	}
}

func TestRejectsV4OnlyArguments(t *testing.T) {
	isolateHelm(t)
	tr := true
	e := New()
	ctx := context.Background()
	cfg := &helmengine.GlobalConfig{Namespace: "default"}
	pr := helmengine.PostRenderOptions{PostRenderer: "kustomize"}

	checks := map[string]struct {
		err  error
		want string
	}{}
	add := func(name, want string, err error) {
		checks[name] = struct {
			err  error
			want string
		}{err, want}
	}

	_, err := e.Install(ctx, cfg, &helmengine.InstallOptions{ServerSideApply: &tr})
	add("install ssa", "server_side_apply", err)
	_, err = e.Install(ctx, cfg, &helmengine.InstallOptions{PostRenderOptions: pr})
	add("install post_renderer", "post_renderer", err)
	_, err = e.Upgrade(ctx, cfg, &helmengine.UpgradeOptions{PostRenderOptions: pr})
	add("upgrade post_renderer", "post_renderer", err)
	_, err = e.Template(ctx, cfg, &helmengine.TemplateOptions{PostRenderOptions: pr})
	add("template post_renderer", "post_renderer", err)
	add("rollback description", "description", e.Rollback(ctx, cfg, &helmengine.RollbackOptions{Description: "x"}))
	add("plugin verify", "verify", e.PluginInstall(ctx, &helmengine.PluginInstallOptions{URLOrPath: "x", Verify: &tr}))

	for name, c := range checks {
		if c.err == nil || !strings.Contains(c.err.Error(), c.want) {
			t.Errorf("%s: expected a %q rejection, got %v", name, c.want, c.err)
		}
	}
}

func TestRegistryLogin_BadTLSFileIsAnError(t *testing.T) {
	isolateHelm(t)
	err := New().RegistryLogin(context.Background(), &helmengine.RegistryLoginOptions{
		Hostname: "registry.example.com",
		CAFile:   "/nonexistent/ca.pem",
	})
	if err == nil || !strings.Contains(err.Error(), "CA file") {
		t.Fatalf("expected a CA file error rather than a panic, got %v", err)
	}
}
