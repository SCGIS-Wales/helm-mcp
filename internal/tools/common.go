package tools

import (
	"encoding/json"
	"fmt"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	v3 "github.com/ssddgreg/helm-mcp/internal/helmengine/v3"
	v4 "github.com/ssddgreg/helm-mcp/internal/helmengine/v4"
	"github.com/ssddgreg/helm-mcp/internal/security"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GlobalInput is embedded in every tool input struct to provide shared fields.
type GlobalInput struct {
	HelmVersion       string  `json:"helm_version,omitempty" jsonschema_description:"Helm SDK version: v3 or v4 (default: v4)"`
	Namespace         string  `json:"namespace,omitempty" jsonschema_description:"Kubernetes namespace"`
	KubeContext       string  `json:"kube_context,omitempty" jsonschema_description:"Kubernetes context name from kubeconfig"`
	KubeConfig        string  `json:"kubeconfig,omitempty" jsonschema_description:"Path to kubeconfig file (defaults to $KUBECONFIG or ~/.kube/config)"`
	KubeAPIServer     string  `json:"kube_apiserver,omitempty" jsonschema_description:"Kubernetes API server URL (overrides kubeconfig)"`
	KubeBearerToken   string  `json:"kube_token,omitempty" jsonschema_description:"Bearer token for Kubernetes API authentication"`
	KubeTLSServerName string  `json:"kube_tls_server_name,omitempty" jsonschema_description:"Server name for TLS certificate validation"`
	KubeInsecureTLS   bool    `json:"kube_insecure_tls,omitempty" jsonschema_description:"Skip TLS certificate verification (insecure)"`
	Debug             bool    `json:"debug,omitempty" jsonschema_description:"Enable debug output"`
	BurstLimit        int     `json:"burst_limit,omitempty" jsonschema_description:"Client-side default throttling limit"`
	QPS               float32 `json:"qps,omitempty" jsonschema_description:"Client-side QPS rate limit"`
}

// ZeroBearerToken zeroes the bearer token field in the input after use.
// Call via defer to reduce the credential lifetime in memory.
func (g *GlobalInput) ZeroBearerToken() {
	if g == nil {
		return
	}
	g.KubeBearerToken = ""
}

// ToGlobalConfig converts GlobalInput to a helmengine.GlobalConfig.
func (g *GlobalInput) ToGlobalConfig() *helmengine.GlobalConfig {
	return &helmengine.GlobalConfig{
		Namespace:         g.Namespace,
		KubeContext:       g.KubeContext,
		KubeConfig:        g.KubeConfig,
		KubeAPIServer:     g.KubeAPIServer,
		KubeBearerToken:   g.KubeBearerToken,
		KubeTLSServerName: g.KubeTLSServerName,
		KubeInsecureTLS:   g.KubeInsecureTLS,
		Debug:             g.Debug,
		BurstLimit:        g.BurstLimit,
		QPS:               g.QPS,
	}
}

var (
	v3Engine helmengine.Engine = v3.New()
	v4Engine helmengine.Engine = v4.New()
)

// SetEnginesForTest replaces the engines used by all tool handlers.
// Returns a cleanup function to restore originals.
func SetEnginesForTest(v3e, v4e helmengine.Engine) func() {
	origV3, origV4 := v3Engine, v4Engine
	v3Engine = v3e
	v4Engine = v4e
	return func() {
		v3Engine = origV3
		v4Engine = origV4
	}
}

// SelectEngine returns the appropriate engine based on the helm_version field.
func SelectEngine(version string) helmengine.Engine {
	switch version {
	case "v3", "3":
		return v3Engine
	default:
		return v4Engine
	}
}

// ValidateRequired checks that a required string field is non-empty.
// Returns an error result if validation fails, nil otherwise.
func ValidateRequired(fieldName, value string) *mcp.CallToolResult {
	if value == "" {
		return ErrorResult(fmt.Errorf("%s is required", fieldName))
	}
	return nil
}

// ValidateGlobalInput validates the shared GlobalInput fields (namespace, kubeconfig path).
func ValidateGlobalInput(g *GlobalInput) error {
	if err := security.ValidateNamespace(g.Namespace); err != nil {
		return err
	}
	if err := security.ValidatePath(g.KubeConfig); err != nil {
		return err
	}
	return nil
}

// ValidateReleaseName delegates to security.ValidateReleaseName.
func ValidateReleaseName(name string) error {
	return security.ValidateReleaseName(name)
}

// ValidateTimeout delegates to security.ValidateTimeout.
func ValidateTimeout(timeout string) error {
	return security.ValidateTimeout(timeout)
}

// TextResult creates a CallToolResult with text content.
func TextResult(data interface{}) *mcp.CallToolResult {
	var text string
	switch v := data.(type) {
	case string:
		text = v
	default:
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			text = fmt.Sprintf("%v", data)
		} else {
			text = string(b)
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

// ErrorResult creates an error CallToolResult with credential scrubbing.
func ErrorResult(err error) *mcp.CallToolResult {
	scrubbed := security.ScrubError(err)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Error: %s", scrubbed.Error())},
		},
		IsError: true,
	}
}
