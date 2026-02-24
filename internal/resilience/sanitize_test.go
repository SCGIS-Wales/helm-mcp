package resilience

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSanitizeManifest_Empty(t *testing.T) {
	if got := SanitizeManifest(""); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestSanitizeManifest_StripsManagedFields(t *testing.T) {
	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  managedFields:
    - manager: kubectl
      operation: Apply
      apiVersion: v1
      fieldsType: FieldsV1
      fieldsV1:
        f:data:
          f:key: {}
data:
  key: value
`
	result := SanitizeManifest(manifest)
	if strings.Contains(result, "managedFields") {
		t.Error("expected managedFields to be stripped")
	}
	if !strings.Contains(result, "name: test") {
		t.Error("expected name to be preserved")
	}
	if !strings.Contains(result, "key: value") {
		t.Error("expected data to be preserved")
	}
}

func TestSanitizeManifest_StripsNoisyAnnotations(t *testing.T) {
	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"test"}}
    deployment.kubernetes.io/revision: "3"
    app.kubernetes.io/name: myapp
data:
  key: value
`
	result := SanitizeManifest(manifest)
	if strings.Contains(result, "last-applied-configuration") {
		t.Error("expected last-applied-configuration to be stripped")
	}
	if strings.Contains(result, "deployment.kubernetes.io/revision") {
		t.Error("expected deployment revision annotation to be stripped")
	}
	if !strings.Contains(result, "app.kubernetes.io/name: myapp") {
		t.Error("expected non-noisy annotation to be preserved")
	}
	if !strings.Contains(result, "key: value") {
		t.Error("expected data to be preserved")
	}
}

func TestSanitizeManifest_MultiDocument(t *testing.T) {
	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm1
  managedFields:
    - manager: kubectl
data:
  key1: val1
---
apiVersion: v1
kind: Secret
metadata:
  name: secret1
  managedFields:
    - manager: helm
data:
  password: cGFzcw==
`
	result := SanitizeManifest(manifest)
	if strings.Contains(result, "managedFields") {
		t.Error("expected managedFields to be stripped from all documents")
	}
	if !strings.Contains(result, "name: cm1") {
		t.Error("expected first document name preserved")
	}
	if !strings.Contains(result, "name: secret1") {
		t.Error("expected second document name preserved")
	}
	if !strings.Contains(result, "---") {
		t.Error("expected document separator preserved")
	}
}

func TestSanitizeManifest_PreservesCleanManifest(t *testing.T) {
	manifest := `apiVersion: v1
kind: Service
metadata:
  name: my-svc
  labels:
    app: web
spec:
  type: ClusterIP
  ports:
    - port: 80
`
	result := SanitizeManifest(manifest)
	if !strings.Contains(result, "name: my-svc") {
		t.Error("expected name preserved")
	}
	if !strings.Contains(result, "app: web") {
		t.Error("expected labels preserved")
	}
	if !strings.Contains(result, "port: 80") {
		t.Error("expected spec preserved")
	}
}

func TestSanitizeManifest_SizeReduction(t *testing.T) {
	// Build a realistic manifest with managedFields and last-applied-config.
	var sb strings.Builder
	sb.WriteString("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: app\n")
	sb.WriteString("  annotations:\n")
	sb.WriteString("    kubectl.kubernetes.io/last-applied-configuration: |\n")
	// Simulate a large last-applied-configuration (typical: 2-5KB).
	sb.WriteString("      ")
	sb.WriteString(strings.Repeat(`{"apiVersion":"apps/v1","kind":"Deployment"}`, 50))
	sb.WriteString("\n")
	sb.WriteString("  managedFields:\n")
	for i := 0; i < 20; i++ {
		sb.WriteString("    - manager: kubectl\n")
		sb.WriteString("      operation: Apply\n")
		sb.WriteString("      apiVersion: apps/v1\n")
		sb.WriteString("      fieldsType: FieldsV1\n")
	}
	sb.WriteString("spec:\n  replicas: 3\n")

	manifest := sb.String()
	result := SanitizeManifest(manifest)

	reduction := float64(len(manifest)-len(result)) / float64(len(manifest)) * 100
	t.Logf("size reduction: %.1f%% (from %d to %d bytes)", reduction, len(manifest), len(result))

	if len(result) >= len(manifest) {
		t.Error("expected sanitized manifest to be smaller")
	}
	if !strings.Contains(result, "replicas: 3") {
		t.Error("expected spec to be preserved")
	}
}

func TestSanitizeJSON_StripsManagedFields(t *testing.T) {
	obj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name": "test",
			"managedFields": []interface{}{
				map[string]interface{}{
					"manager":   "kubectl",
					"operation": "Apply",
				},
			},
			"annotations": map[string]interface{}{
				"kubectl.kubernetes.io/last-applied-configuration": `{"apiVersion":"v1"}`,
				"app.kubernetes.io/name":                          "myapp",
			},
		},
		"data": map[string]interface{}{
			"key": "value",
		},
	}

	data, _ := json.Marshal(obj)
	result := SanitizeJSON(data)

	var cleaned map[string]interface{}
	if err := json.Unmarshal(result, &cleaned); err != nil {
		t.Fatalf("expected valid JSON, got error: %v", err)
	}

	meta := cleaned["metadata"].(map[string]interface{})
	if _, ok := meta["managedFields"]; ok {
		t.Error("expected managedFields to be stripped")
	}

	ann := meta["annotations"].(map[string]interface{})
	if _, ok := ann["kubectl.kubernetes.io/last-applied-configuration"]; ok {
		t.Error("expected last-applied-configuration to be stripped")
	}
	if ann["app.kubernetes.io/name"] != "myapp" {
		t.Error("expected non-noisy annotation preserved")
	}
}

func TestSanitizeJSON_InvalidJSON(t *testing.T) {
	input := []byte("not json")
	result := SanitizeJSON(input)
	if string(result) != "not json" {
		t.Error("expected invalid JSON to be returned unchanged")
	}
}

func TestSanitizeJSON_RecursiveStripping(t *testing.T) {
	// Nested object (like Deployment spec.template.metadata).
	obj := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":          "app",
			"managedFields": []interface{}{"field1"},
		},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"kubectl.kubernetes.io/last-applied-configuration": "big",
						"prometheus.io/scrape":                            "true",
					},
				},
			},
		},
	}

	data, _ := json.Marshal(obj)
	result := SanitizeJSON(data)

	var cleaned map[string]interface{}
	if err := json.Unmarshal(result, &cleaned); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}

	// Check top-level metadata stripped.
	meta := cleaned["metadata"].(map[string]interface{})
	if _, ok := meta["managedFields"]; ok {
		t.Error("expected top-level managedFields stripped")
	}

	// Check nested template metadata stripped.
	spec := cleaned["spec"].(map[string]interface{})
	tmpl := spec["template"].(map[string]interface{})
	tmplMeta := tmpl["metadata"].(map[string]interface{})
	tmplAnn := tmplMeta["annotations"].(map[string]interface{})
	if _, ok := tmplAnn["kubectl.kubernetes.io/last-applied-configuration"]; ok {
		t.Error("expected nested last-applied-configuration stripped")
	}
	if tmplAnn["prometheus.io/scrape"] != "true" {
		t.Error("expected non-noisy nested annotation preserved")
	}
}

func TestSanitizeJSON_EmptyAnnotationsRemoved(t *testing.T) {
	obj := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "test",
			"annotations": map[string]interface{}{
				"kubectl.kubernetes.io/last-applied-configuration": "x",
			},
		},
	}

	data, _ := json.Marshal(obj)
	result := SanitizeJSON(data)

	var cleaned map[string]interface{}
	_ = json.Unmarshal(result, &cleaned)

	meta := cleaned["metadata"].(map[string]interface{})
	if _, ok := meta["annotations"]; ok {
		t.Error("expected empty annotations map to be removed")
	}
}
