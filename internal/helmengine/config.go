package helmengine

import "time"

// DefaultTimeout is the default timeout for Helm operations that require
// waiting (install --wait, upgrade --wait, etc.).
const DefaultTimeout = 300 * time.Second

// ParseDuration parses a Go duration string, returning DefaultTimeout for empty input.
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return DefaultTimeout, nil
	}
	return time.ParseDuration(s)
}

// GlobalConfig holds configuration shared across all Helm operations
// that require cluster access.
//
// Kubernetes Authentication:
//
// The Helm MCP server authenticates to Kubernetes clusters using the same
// mechanisms as the standard Helm CLI and kubectl:
//
//  1. Explicit kubeconfig: Set KubeConfig to the path of a kubeconfig file.
//     This takes highest precedence.
//
//  2. KUBECONFIG environment variable: If KubeConfig is empty, the Helm SDK
//     reads $KUBECONFIG (can be a colon-separated list of paths).
//
//  3. Default kubeconfig: If neither is set, ~/.kube/config is used.
//
//  4. In-cluster config: When running inside a Kubernetes pod, the SDK
//     automatically uses the service account token mounted at
//     /var/run/secrets/kubernetes.io/serviceaccount/.
//
//  5. Context selection: KubeContext selects a specific context from the
//     kubeconfig. If empty, the current-context is used.
//
//  6. API server override: KubeAPIServer overrides the server URL from
//     the kubeconfig. Useful for port-forwarding or proxy scenarios.
//
//  7. Bearer token: KubeBearerToken provides a bearer token for direct
//     authentication, bypassing kubeconfig-based auth.
//
//  8. TLS: KubeTLSServerName sets the server name for TLS certificate
//     validation. KubeInsecureTLS disables TLS verification (not
//     recommended for production).
//
// Rate Limiting:
//
//   - QPS: Maximum queries per second to the API server (default varies by SDK).
//   - BurstLimit: Maximum burst above QPS (default varies by SDK).
type GlobalConfig struct {
	// Namespace is the Kubernetes namespace for the operation.
	Namespace string `json:"namespace,omitempty"`

	// KubeContext selects a context from the kubeconfig file.
	KubeContext string `json:"kube_context,omitempty"`

	// KubeConfig is the path to a kubeconfig file.
	// If empty, $KUBECONFIG or ~/.kube/config is used.
	KubeConfig string `json:"kubeconfig,omitempty"`

	// KubeAPIServer overrides the API server URL from kubeconfig.
	KubeAPIServer string `json:"kube_apiserver,omitempty"`

	// KubeBearerToken is a bearer token for API server authentication.
	// Takes precedence over kubeconfig credentials when set.
	KubeBearerToken string `json:"kube_token,omitempty"`

	// KubeTLSServerName overrides the server name used for TLS validation.
	KubeTLSServerName string `json:"kube_tls_server_name,omitempty"`

	// KubeInsecureTLS disables TLS certificate verification.
	// WARNING: This is insecure and should not be used in production.
	KubeInsecureTLS bool `json:"kube_insecure_tls,omitempty"`

	// Debug enables verbose logging for Helm operations.
	Debug bool `json:"debug,omitempty"`

	// BurstLimit is the client-side throttling burst limit.
	BurstLimit int `json:"burst_limit,omitempty"`

	// QPS is the client-side queries-per-second rate limit.
	QPS float32 `json:"qps,omitempty"`
}
