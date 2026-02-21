package helmengine

import (
	"testing"
	"time"
)

func TestDefaultTimeout(t *testing.T) {
	if DefaultTimeout != 300*time.Second {
		t.Errorf("DefaultTimeout = %v, want 5m", DefaultTimeout)
	}
}

func TestGlobalConfigDefaults(t *testing.T) {
	cfg := &GlobalConfig{}

	if cfg.Namespace != "" {
		t.Error("expected empty Namespace")
	}
	if cfg.KubeContext != "" {
		t.Error("expected empty KubeContext")
	}
	if cfg.KubeConfig != "" {
		t.Error("expected empty KubeConfig")
	}
	if cfg.KubeAPIServer != "" {
		t.Error("expected empty KubeAPIServer")
	}
	if cfg.KubeBearerToken != "" {
		t.Error("expected empty KubeBearerToken")
	}
	if cfg.Debug {
		t.Error("expected Debug=false")
	}
	if cfg.BurstLimit != 0 {
		t.Error("expected BurstLimit=0")
	}
	if cfg.QPS != 0 {
		t.Error("expected QPS=0")
	}
	if cfg.KubeInsecureTLS {
		t.Error("expected KubeInsecureTLS=false")
	}
}

func TestGlobalConfigFieldAssignment(t *testing.T) {
	cfg := &GlobalConfig{
		Namespace:         "test-ns",
		KubeContext:       "test-ctx",
		KubeConfig:        "/path/to/kubeconfig",
		KubeAPIServer:     "https://api.example.com:6443",
		KubeBearerToken:   "token-123",
		KubeTLSServerName: "api.example.com",
		KubeInsecureTLS:   true,
		Debug:             true,
		BurstLimit:        100,
		QPS:               50.0,
	}

	if cfg.Namespace != "test-ns" {
		t.Errorf("Namespace = %q, want %q", cfg.Namespace, "test-ns")
	}
	if cfg.KubeContext != "test-ctx" {
		t.Errorf("KubeContext = %q, want %q", cfg.KubeContext, "test-ctx")
	}
	if cfg.KubeConfig != "/path/to/kubeconfig" {
		t.Errorf("KubeConfig = %q, want %q", cfg.KubeConfig, "/path/to/kubeconfig")
	}
	if cfg.KubeAPIServer != "https://api.example.com:6443" {
		t.Errorf("KubeAPIServer = %q, want %q", cfg.KubeAPIServer, "https://api.example.com:6443")
	}
	if cfg.KubeBearerToken != "token-123" {
		t.Errorf("KubeBearerToken = %q, want %q", cfg.KubeBearerToken, "token-123")
	}
	if !cfg.KubeInsecureTLS {
		t.Error("expected KubeInsecureTLS=true")
	}
	if cfg.BurstLimit != 100 {
		t.Errorf("BurstLimit = %d, want 100", cfg.BurstLimit)
	}
	if cfg.QPS != 50.0 {
		t.Errorf("QPS = %f, want 50.0", cfg.QPS)
	}
}
