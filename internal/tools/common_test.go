package tools

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestSelectEngine(t *testing.T) {
	tests := []struct {
		version string
		wantV3  bool
	}{
		{"v3", true},
		{"3", true},
		{"v4", false},
		{"4", false},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		engine := SelectEngine(tt.version)
		if engine == nil {
			t.Errorf("SelectEngine(%q) returned nil", tt.version)
			continue
		}
		// v3Engine and v4Engine are package-level, check identity
		if tt.wantV3 && engine != v3Engine {
			t.Errorf("SelectEngine(%q) did not return v3 engine", tt.version)
		}
		if !tt.wantV3 && engine != v4Engine {
			t.Errorf("SelectEngine(%q) did not return v4 engine", tt.version)
		}
	}
}

func TestTextResult_String(t *testing.T) {
	result := TextResult("hello world")
	if result == nil {
		t.Fatal("TextResult returned nil")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	if result.IsError {
		t.Error("expected IsError=false")
	}
}

func TestTextResult_Struct(t *testing.T) {
	data := struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{Name: "test", Age: 42}

	result := TextResult(data)
	if result == nil {
		t.Fatal("TextResult returned nil")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	if result.IsError {
		t.Error("expected IsError=false")
	}
}

func TestTextResult_Nil(t *testing.T) {
	result := TextResult(nil)
	if result == nil {
		t.Fatal("TextResult returned nil")
	}
}

func TestErrorResult(t *testing.T) {
	err := errors.New("something went wrong")
	result := ErrorResult(err)
	if result == nil {
		t.Fatal("ErrorResult returned nil")
	}
	if !result.IsError {
		t.Error("expected IsError=true")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		field   string
		value   string
		wantErr bool
	}{
		{"release_name", "my-release", false},
		{"release_name", "", true},
		{"chart", "nginx", false},
		{"chart", "", true},
	}

	for _, tt := range tests {
		result := ValidateRequired(tt.field, tt.value)
		if tt.wantErr && result == nil {
			t.Errorf("ValidateRequired(%q, %q) expected error, got nil", tt.field, tt.value)
		}
		if !tt.wantErr && result != nil {
			t.Errorf("ValidateRequired(%q, %q) expected nil, got error", tt.field, tt.value)
		}
		if tt.wantErr && result != nil && !result.IsError {
			t.Errorf("ValidateRequired(%q, %q) result should have IsError=true", tt.field, tt.value)
		}
	}
}

func TestToGlobalConfig(t *testing.T) {
	input := &GlobalInput{
		HelmVersion:       "v4",
		Namespace:         "test-ns",
		KubeContext:       "test-ctx",
		KubeConfig:        "/path/to/kubeconfig",
		KubeAPIServer:     "https://api.example.com",
		KubeBearerToken:   "tok-123",
		KubeTLSServerName: "api.example.com",
		KubeInsecureTLS:   true,
		Debug:             true,
		BurstLimit:        100,
		QPS:               50.0,
	}

	cfg := input.ToGlobalConfig()

	if cfg.Namespace != "test-ns" {
		t.Errorf("Namespace = %q, want %q", cfg.Namespace, "test-ns")
	}
	if cfg.KubeContext != "test-ctx" {
		t.Errorf("KubeContext = %q, want %q", cfg.KubeContext, "test-ctx")
	}
	if cfg.KubeConfig != "/path/to/kubeconfig" {
		t.Errorf("KubeConfig = %q, want %q", cfg.KubeConfig, "/path/to/kubeconfig")
	}
	if cfg.KubeAPIServer != "https://api.example.com" {
		t.Errorf("KubeAPIServer = %q, want %q", cfg.KubeAPIServer, "https://api.example.com")
	}
	if cfg.KubeBearerToken != "tok-123" {
		t.Errorf("KubeBearerToken = %q, want %q", cfg.KubeBearerToken, "tok-123")
	}
	if cfg.KubeTLSServerName != "api.example.com" {
		t.Errorf("KubeTLSServerName = %q, want %q", cfg.KubeTLSServerName, "api.example.com")
	}
	if !cfg.KubeInsecureTLS {
		t.Error("expected KubeInsecureTLS=true")
	}
	if !cfg.Debug {
		t.Error("expected Debug=true")
	}
	if cfg.BurstLimit != 100 {
		t.Errorf("BurstLimit = %d, want 100", cfg.BurstLimit)
	}
	if cfg.QPS != 50.0 {
		t.Errorf("QPS = %f, want 50.0", cfg.QPS)
	}
}

func TestToGlobalConfig_Defaults(t *testing.T) {
	input := &GlobalInput{}
	cfg := input.ToGlobalConfig()

	if cfg.Namespace != "" {
		t.Error("expected empty Namespace")
	}
	if cfg.Debug {
		t.Error("expected Debug=false")
	}
}

func TestGlobalInputJSON(t *testing.T) {
	jsonStr := `{
		"helm_version": "v4",
		"namespace": "prod",
		"kube_context": "prod-cluster",
		"kubeconfig": "/home/user/.kube/config",
		"kube_apiserver": "https://k8s.example.com:6443",
		"kube_token": "my-token",
		"kube_insecure_tls": false,
		"debug": true
	}`

	var input GlobalInput
	if err := json.Unmarshal([]byte(jsonStr), &input); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if input.HelmVersion != "v4" {
		t.Errorf("HelmVersion = %q, want %q", input.HelmVersion, "v4")
	}
	if input.Namespace != "prod" {
		t.Errorf("Namespace = %q, want %q", input.Namespace, "prod")
	}
	if input.KubeContext != "prod-cluster" {
		t.Errorf("KubeContext = %q, want %q", input.KubeContext, "prod-cluster")
	}
	if input.KubeBearerToken != "my-token" {
		t.Errorf("KubeBearerToken = %q, want %q", input.KubeBearerToken, "my-token")
	}
}

func TestTextResult_LargeJSON(t *testing.T) {
	data := make(map[string]string, 100)
	for i := 0; i < 100; i++ {
		data[strings.Repeat("k", i+1)] = strings.Repeat("v", i+1)
	}

	result := TextResult(data)
	if result == nil {
		t.Fatal("TextResult returned nil for large map")
	}
	if result.IsError {
		t.Error("expected IsError=false for large map")
	}
}
