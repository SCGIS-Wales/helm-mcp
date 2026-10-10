package helmengine

import (
	"strings"
	"testing"
)

const testManifest = `---
# Source: mychart/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: svc
---
# Source: mychart/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy
`

var testHooks = []HookManifest{
	{Path: "mychart/templates/pre-install-job.yaml", Manifest: "kind: Job\nmetadata:\n  name: pre"},
	{Path: "mychart/templates/tests/test-connection.yaml", Manifest: "kind: Pod\nmetadata:\n  name: test", IsTest: true},
}

func TestRenderTemplateOutput_IncludesHooksByDefault(t *testing.T) {
	out, err := RenderTemplateOutput(testManifest, testHooks, &TemplateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"name: svc", "name: deploy", "name: pre", "name: test"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "# Source: mychart/templates/pre-install-job.yaml") {
		t.Errorf("hook source comment missing:\n%s", out)
	}
}

func TestRenderTemplateOutput_NoHooks(t *testing.T) {
	out, err := RenderTemplateOutput(testManifest, testHooks, &TemplateOptions{NoHooks: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "name: pre") || strings.Contains(out, "name: test") {
		t.Errorf("no_hooks should drop all hooks:\n%s", out)
	}
	if !strings.Contains(out, "name: deploy") {
		t.Errorf("manifest should still be rendered:\n%s", out)
	}
}

func TestRenderTemplateOutput_SkipTests(t *testing.T) {
	out, err := RenderTemplateOutput(testManifest, testHooks, &TemplateOptions{SkipTests: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "name: test") {
		t.Errorf("skip_tests should drop test hooks:\n%s", out)
	}
	if !strings.Contains(out, "name: pre") {
		t.Errorf("skip_tests should keep non-test hooks:\n%s", out)
	}
}

func TestRenderTemplateOutput_ShowOnly(t *testing.T) {
	tests := []struct {
		name     string
		showOnly []string
		want     []string
		notWant  []string
	}{
		{
			name:     "exact path",
			showOnly: []string{"templates/service.yaml"},
			want:     []string{"name: svc"},
			notWant:  []string{"name: deploy", "name: pre"},
		},
		{
			name:     "glob matches a hook too",
			showOnly: []string{"templates/*job.yaml"},
			want:     []string{"name: pre"},
			notWant:  []string{"name: svc", "name: deploy"},
		},
		{
			name:     "several patterns keep request order",
			showOnly: []string{"templates/deployment.yaml", "templates/service.yaml"},
			want:     []string{"name: deploy", "name: svc"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := RenderTemplateOutput(testManifest, testHooks, &TemplateOptions{ShowOnly: tt.showOnly})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
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

	t.Run("order follows the patterns", func(t *testing.T) {
		out, _ := RenderTemplateOutput(testManifest, nil, &TemplateOptions{
			ShowOnly: []string{"templates/deployment.yaml", "templates/service.yaml"},
		})
		if strings.Index(out, "name: deploy") > strings.Index(out, "name: svc") {
			t.Errorf("deployment should come first:\n%s", out)
		}
	})
}

func TestRenderTemplateOutput_ShowOnlyMissingTemplate(t *testing.T) {
	_, err := RenderTemplateOutput(testManifest, nil, &TemplateOptions{ShowOnly: []string{"templates/nope.yaml"}})
	if err == nil || !strings.Contains(err.Error(), "could not find template templates/nope.yaml") {
		t.Fatalf("expected missing template error, got %v", err)
	}
}
