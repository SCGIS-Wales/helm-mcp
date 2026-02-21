package security

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReleaseName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"my-release", false},
		{"nginx", false},
		{"my.release.v1", false},
		{"a", false},
		{"a-b-c", false},
		{"release-123", false},
		// Invalid cases
		{"", true},
		{"-starts-with-dash", true},
		{"ends-with-dash-", true},
		{"UPPERCASE", true},
		{"has spaces", true},
		{"has_underscores", true},
		{strings.Repeat("a", 254), true},
	}

	for _, tt := range tests {
		err := ValidateReleaseName(tt.name)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateReleaseName(%q) expected error, got nil", tt.name)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidateReleaseName(%q) unexpected error: %v", tt.name, err)
		}
	}
}

func TestValidateNamespace(t *testing.T) {
	tests := []struct {
		ns      string
		wantErr bool
	}{
		{"", false}, // empty is valid
		{"default", false},
		{"kube-system", false},
		{"my-namespace", false},
		// Invalid
		{"INVALID", true},
		{"has spaces", true},
		{strings.Repeat("a", 254), true},
	}

	for _, tt := range tests {
		err := ValidateNamespace(tt.ns)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateNamespace(%q) expected error, got nil", tt.ns)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidateNamespace(%q) unexpected error: %v", tt.ns, err)
		}
	}
}

func TestValidateKubeConfig(t *testing.T) {
	// Empty path is valid
	if err := ValidateKubeConfig(""); err != nil {
		t.Errorf("ValidateKubeConfig('') unexpected error: %v", err)
	}

	// Path traversal
	if err := ValidateKubeConfig("/tmp/../etc/shadow"); err == nil {
		t.Error("ValidateKubeConfig with traversal expected error")
	}

	// Non-existent file
	if err := ValidateKubeConfig("/nonexistent/path/kubeconfig"); err == nil {
		t.Error("ValidateKubeConfig with non-existent file expected error")
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
	}{
		{"https://charts.example.com", false},
		{"http://localhost:8080", false},
		{"oci://registry.example.com/charts", false},
		// Invalid
		{"", true},
		{"ftp://invalid.com", true},
		{"not-a-url", true},
		{"file:///etc/passwd", true},
	}

	for _, tt := range tests {
		err := ValidateURL(tt.url)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateURL(%q) expected error, got nil", tt.url)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidateURL(%q) unexpected error: %v", tt.url, err)
		}
	}
}

func TestScrubCredentials(t *testing.T) {
	input := map[string]string{
		"HELM_REPOSITORY_CONFIG": "/home/user/.config/helm/repositories.yaml",
		"HELM_PASSWORD":          "super-secret",
		"HELM_TOKEN":             "my-token-123",
		"HELM_DRIVER":            "secret",
		"HELM_CACHE":             "normal-value",
	}

	scrubbed := ScrubCredentials(input)

	if scrubbed["HELM_REPOSITORY_CONFIG"] != input["HELM_REPOSITORY_CONFIG"] {
		t.Error("expected HELM_REPOSITORY_CONFIG to not be scrubbed")
	}
	if scrubbed["HELM_PASSWORD"] != "***REDACTED***" {
		t.Errorf("expected HELM_PASSWORD to be redacted, got %q", scrubbed["HELM_PASSWORD"])
	}
	if scrubbed["HELM_TOKEN"] != "***REDACTED***" {
		t.Errorf("expected HELM_TOKEN to be redacted, got %q", scrubbed["HELM_TOKEN"])
	}
	if scrubbed["HELM_CACHE"] != "normal-value" {
		t.Error("expected HELM_CACHE to not be scrubbed")
	}
}

func TestScrubError(t *testing.T) {
	// Nil error
	if err := ScrubError(nil); err != nil {
		t.Errorf("ScrubError(nil) = %v, want nil", err)
	}

	// Error with bearer token
	err := errors.New("failed: bearer eyJhbGciOiJSUzI1NiIs...")
	scrubbed := ScrubError(err)
	if strings.Contains(scrubbed.Error(), "eyJhbGciOiJSUzI1NiIs") {
		t.Error("expected bearer token to be scrubbed")
	}
	if !strings.Contains(scrubbed.Error(), "***REDACTED***") {
		t.Error("expected ***REDACTED*** in scrubbed error")
	}

	// Error with URL password
	err = errors.New("failed to connect to https://user:password123@registry.example.com")
	scrubbed = ScrubError(err)
	if strings.Contains(scrubbed.Error(), "password123") {
		t.Error("expected URL password to be scrubbed")
	}

	// Clean error passes through
	err = errors.New("release not found")
	scrubbed = ScrubError(err)
	if scrubbed.Error() != "release not found" {
		t.Errorf("clean error should pass through unchanged, got %q", scrubbed.Error())
	}
}

func TestScrubCredentials_EmptyMap(t *testing.T) {
	scrubbed := ScrubCredentials(map[string]string{})
	if len(scrubbed) != 0 {
		t.Errorf("expected empty map, got %d items", len(scrubbed))
	}
}

// --- EKS / GKE / AKS Kubeconfig Validation Tests ---

func TestValidateKubeConfig_PathTraversalVariants(t *testing.T) {
	traversals := []string{
		"/tmp/../etc/shadow",
		"/home/user/../../etc/passwd",
		"../../../etc/shadow",
		"/tmp/foo/../../etc/shadow",
	}
	for _, path := range traversals {
		if err := ValidateKubeConfig(path); err == nil {
			t.Errorf("ValidateKubeConfig(%q) should reject path traversal", path)
		}
	}
}

func TestValidateKubeConfig_DirectoryRejection(t *testing.T) {
	// /tmp exists and is a directory
	if err := ValidateKubeConfig("/tmp"); err == nil {
		t.Error("ValidateKubeConfig should reject directories")
	}
}

func TestValidateReleaseName_KubernetesNameRules(t *testing.T) {
	// Kubernetes DNS-1123 subdomain names: max 253 chars, lowercase alphanumeric, '-', '.'
	tests := []struct {
		name    string
		wantErr bool
		desc    string
	}{
		{"a", false, "single char valid"},
		{"a.b.c", false, "dots are valid"},
		{strings.Repeat("a", 253), false, "max length valid"},
		{strings.Repeat("a", 254), true, "exceeds max length"},
		{"My-Release", true, "uppercase rejected"},
		{"my_release", true, "underscore rejected"},
		{"my release", true, "space rejected"},
		{"-leading-dash", true, "leading dash"},
		{"trailing-dash-", true, "trailing dash"},
		{".leading-dot", true, "leading dot rejected"},
		{"trailing-dot.", true, "trailing dot rejected"},
		{"release@v1", true, "special chars rejected"},
		{"release/v1", true, "slash rejected"},
		{"release:latest", true, "colon rejected"},
	}
	for _, tt := range tests {
		err := ValidateReleaseName(tt.name)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateReleaseName(%q) [%s] expected error, got nil", tt.name, tt.desc)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidateReleaseName(%q) [%s] unexpected error: %v", tt.name, tt.desc, err)
		}
	}
}

func TestValidateNamespace_KubernetesRules(t *testing.T) {
	tests := []struct {
		ns      string
		wantErr bool
		desc    string
	}{
		{"", false, "empty valid"},
		{"default", false, "default ns"},
		{"kube-system", false, "kube-system"},
		{"kube-public", false, "kube-public"},
		{"production", false, "custom ns"},
		{"team-a", false, "with dash"},
		{"istio-system", false, "istio ns"},
		// Invalid
		{"Default", true, "uppercase"},
		{"my_namespace", true, "underscore"},
		{"my namespace", true, "space"},
		{"-leading", true, "leading dash"},
		{"trailing-", true, "trailing dash"},
	}
	for _, tt := range tests {
		err := ValidateNamespace(tt.ns)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateNamespace(%q) [%s] expected error", tt.ns, tt.desc)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidateNamespace(%q) [%s] unexpected error: %v", tt.ns, tt.desc, err)
		}
	}
}

func TestValidateURL_CloudProviders(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
		desc    string
	}{
		// EKS API server endpoints
		{"https://ABCDEF1234567890.gr7.us-east-1.eks.amazonaws.com", false, "EKS endpoint"},
		{"https://ABCDEF.yl4.us-west-2.eks.amazonaws.com", false, "EKS us-west-2"},
		// GKE API server endpoints
		{"https://35.202.100.1", false, "GKE IP endpoint"},
		{"https://container.googleapis.com/v1/projects/my-project/locations/us-central1/clusters/my-cluster", false, "GKE API"},
		// AKS API server endpoints
		{"https://my-aks-cluster-dns-abc123.hcp.eastus.azmk8s.io:443", false, "AKS endpoint"},
		// OCI registries
		{"oci://123456789.dkr.ecr.us-east-1.amazonaws.com", false, "ECR OCI"},
		{"oci://gcr.io/my-project/charts", false, "GCR OCI"},
		{"oci://myregistry.azurecr.io/charts", false, "ACR OCI"},
		// Invalid
		{"ftp://invalid.eks.amazonaws.com", true, "FTP not allowed"},
		{"file:///etc/passwd", true, "file scheme not allowed"},
		{"", true, "empty URL"},
		{"not-a-url", true, "no scheme"},
		{"ssh://git@github.com/repo", true, "SSH not allowed"},
	}
	for _, tt := range tests {
		err := ValidateURL(tt.url)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateURL(%q) [%s] expected error", tt.url, tt.desc)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidateURL(%q) [%s] unexpected error: %v", tt.url, tt.desc, err)
		}
	}
}

func TestScrubError_CloudTokens(t *testing.T) {
	// AWS STS token (typical EKS)
	err := errors.New("failed: bearer k8s-aws-v1.aHR0cHM6Ly9zdHMuYW1hem9uYXdzLmNvbS...")
	scrubbed := ScrubError(err)
	if strings.Contains(scrubbed.Error(), "k8s-aws-v1") {
		t.Error("expected EKS bearer token to be scrubbed")
	}

	// GKE access token
	err = errors.New("failed: bearer ya29.a0AfH6SMBxxxx...")
	scrubbed = ScrubError(err)
	if strings.Contains(scrubbed.Error(), "ya29") {
		t.Error("expected GKE token to be scrubbed")
	}

	// Azure token
	err = errors.New("failed: bearer eyJ0eXAiOiJKV1QiLCJhbGciOi...")
	scrubbed = ScrubError(err)
	if strings.Contains(scrubbed.Error(), "eyJ0eXAiOiJKV1Qi") {
		t.Error("expected Azure JWT token to be scrubbed")
	}

	// Basic auth in URL
	err = errors.New("failed: https://admin:MyP@ssw0rd@registry.example.com/v2/repo")
	scrubbed = ScrubError(err)
	if strings.Contains(scrubbed.Error(), "MyP@ssw0rd") {
		t.Error("expected URL password to be scrubbed")
	}

	// Token in query parameter style
	err = errors.New("request failed: token=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9")
	scrubbed = ScrubError(err)
	if strings.Contains(scrubbed.Error(), "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9") {
		t.Error("expected token= value to be scrubbed")
	}
}

func TestScrubCredentials_CloudProviderKeys(t *testing.T) {
	input := map[string]string{
		"AWS_ACCESS_KEY_ID":     "AKIAIOSFODNN7EXAMPLE",
		"AWS_SECRET_ACCESS_KEY": "test-secret-value-not-real", //nolint:gosec
		"AZURE_CLIENT_SECRET":   "azure-secret-value",
		"GOOGLE_APPLICATION_CREDENTIALS": "/path/to/creds.json",
		"HELM_REGISTRY_PASSWORD": "registry-pass",
		"KUBECONFIG":            "/home/user/.kube/config",
		"HELM_NAMESPACE":        "production",
	}

	scrubbed := ScrubCredentials(input)

	// Should be redacted (contain "key", "secret", "password", "credential")
	if scrubbed["AWS_ACCESS_KEY_ID"] != "***REDACTED***" {
		t.Error("AWS_ACCESS_KEY_ID should be redacted (contains 'key')")
	}
	if scrubbed["AWS_SECRET_ACCESS_KEY"] != "***REDACTED***" {
		t.Error("AWS_SECRET_ACCESS_KEY should be redacted (contains 'secret' and 'key')")
	}
	if scrubbed["AZURE_CLIENT_SECRET"] != "***REDACTED***" {
		t.Error("AZURE_CLIENT_SECRET should be redacted (contains 'secret')")
	}
	if scrubbed["GOOGLE_APPLICATION_CREDENTIALS"] != "***REDACTED***" {
		t.Error("GOOGLE_APPLICATION_CREDENTIALS should be redacted (contains 'credential')")
	}
	if scrubbed["HELM_REGISTRY_PASSWORD"] != "***REDACTED***" {
		t.Error("HELM_REGISTRY_PASSWORD should be redacted (contains 'password')")
	}

	// Should NOT be redacted
	if scrubbed["KUBECONFIG"] != "/home/user/.kube/config" {
		t.Error("KUBECONFIG should not be redacted")
	}
	if scrubbed["HELM_NAMESPACE"] != "production" {
		t.Error("HELM_NAMESPACE should not be redacted")
	}
}

func TestScrubError_BasicAuth(t *testing.T) {
	err := errors.New("failed: basic dXNlcjpwYXNzd29yZA==")
	scrubbed := ScrubError(err)
	if strings.Contains(scrubbed.Error(), "dXNlcjpwYXNzd29yZA==") {
		t.Error("expected basic auth to be scrubbed")
	}
	if !strings.Contains(scrubbed.Error(), "***REDACTED***") {
		t.Error("expected ***REDACTED*** in scrubbed error")
	}
}

// --- ValidatePath ---

func TestValidatePath(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{"", false},
		{"/tmp/mychart", false},
		{"./relative/path", false},
		{"../etc/shadow", true},
		{"/tmp/../etc/passwd", true},
		{"foo/../../bar", true},
	}
	for _, tt := range tests {
		err := ValidatePath(tt.path)
		if tt.wantErr && err == nil {
			t.Errorf("ValidatePath(%q) expected error", tt.path)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidatePath(%q) unexpected error: %v", tt.path, err)
		}
	}
}

// --- ValidateTimeout ---

func TestValidateTimeout(t *testing.T) {
	tests := []struct {
		timeout string
		wantErr bool
	}{
		{"", false},
		{"5m", false},
		{"5m0s", false},
		{"1h", false},
		{"30s", false},
		{"24h", false},
		// Invalid
		{"25h", true},
		{"invalid", true},
		{"-5m", true},
		{"999999h", true},
	}
	for _, tt := range tests {
		err := ValidateTimeout(tt.timeout)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateTimeout(%q) expected error", tt.timeout)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidateTimeout(%q) unexpected error: %v", tt.timeout, err)
		}
	}
}

// --- ValidateKubeConfig additional coverage ---

func TestValidateKubeConfig_ValidFile(t *testing.T) {
	// Create a temporary regular file that should pass validation
	dir := t.TempDir()
	validFile := filepath.Join(dir, "kubeconfig")
	if err := os.WriteFile(validFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if err := ValidateKubeConfig(validFile); err != nil {
		t.Errorf("ValidateKubeConfig(%q) unexpected error for valid file: %v", validFile, err)
	}
}

func TestValidateKubeConfig_SymlinkRejection(t *testing.T) {
	dir := t.TempDir()
	targetFile := filepath.Join(dir, "kubeconfig-target")
	if err := os.WriteFile(targetFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	symlinkPath := filepath.Join(dir, "kubeconfig-symlink")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	err := ValidateKubeConfig(symlinkPath)
	if err == nil {
		t.Error("ValidateKubeConfig should reject symlinks")
	}
	if err != nil && !strings.Contains(err.Error(), "symlink") {
		t.Errorf("expected error about symlink, got: %v", err)
	}
}
