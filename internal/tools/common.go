package tools

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	v3 "github.com/ssddgreg/helm-mcp/internal/helmengine/v3"
	v4 "github.com/ssddgreg/helm-mcp/internal/helmengine/v4"
	"github.com/ssddgreg/helm-mcp/internal/resilience"
	"github.com/ssddgreg/helm-mcp/internal/security"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MaxResponseBytes controls the maximum response size before truncation.
// Defaults to resilience.DefaultMaxResponseBytes (256 KB).
// Set to 0 to disable truncation.
var MaxResponseBytes = resilience.DefaultMaxResponseBytes

// sensitivePrefixes lists path prefixes that should never be accepted as kubeconfig.
var sensitivePrefixes = []string{
	"/etc/shadow",
	"/etc/passwd",
	"/etc/master.passwd",
	"/proc/",
	"/dev/",
	"/sys/",
}

// GlobalInput is embedded in every tool input struct to provide shared fields.
type GlobalInput struct {
	HelmVersion       string  `json:"helm_version,omitempty" jsonschema:"Helm SDK version: v3 or v4 (default: v4)"`
	Namespace         string  `json:"namespace,omitempty" jsonschema:"Kubernetes namespace"`
	KubeContext       string  `json:"kube_context,omitempty" jsonschema:"Kubernetes context name from kubeconfig"`
	KubeConfig        string  `json:"kubeconfig,omitempty" jsonschema:"Path to kubeconfig file (defaults to $KUBECONFIG or ~/.kube/config)"`
	KubeAPIServer     string  `json:"kube_apiserver,omitempty" jsonschema:"Kubernetes API server URL (overrides kubeconfig)"`
	KubeBearerToken   string  `json:"kube_token,omitempty" jsonschema:"Bearer token for Kubernetes API authentication"`
	KubeTLSServerName string  `json:"kube_tls_server_name,omitempty" jsonschema:"Server name for TLS certificate validation"`
	KubeInsecureTLS   bool    `json:"kube_insecure_tls,omitempty" jsonschema:"Skip TLS certificate verification (insecure)"`
	Debug             bool    `json:"debug,omitempty" jsonschema:"Enable debug output"`
	BurstLimit        int     `json:"burst_limit,omitempty" jsonschema:"Client-side default throttling limit"`
	QPS               float32 `json:"qps,omitempty" jsonschema:"Client-side QPS rate limit"`
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

// ValidateGlobalInput validates the shared GlobalInput fields (namespace, kubeconfig path, API server URL).
func ValidateGlobalInput(g *GlobalInput) error {
	if err := security.ValidateNamespace(g.Namespace); err != nil {
		return err
	}
	if err := security.ValidatePath(g.KubeConfig); err != nil {
		return err
	}
	if err := validateNotSensitivePath(g.KubeConfig); err != nil {
		return err
	}
	if g.KubeAPIServer != "" {
		if err := security.ValidateURL(g.KubeAPIServer); err != nil {
			return fmt.Errorf("invalid kube_apiserver: %w", err)
		}
	}
	return nil
}

// validateNotSensitivePath rejects paths targeting sensitive system files.
func validateNotSensitivePath(path string) error {
	if path == "" {
		return nil
	}
	cleaned := filepath.Clean(path)
	for _, prefix := range sensitivePrefixes {
		if strings.HasPrefix(cleaned, prefix) {
			return fmt.Errorf("path %q targets a sensitive system location", path)
		}
	}
	return nil
}

// Tool annotation helpers.
//
// The MCP spec defaults destructiveHint and openWorldHint to true, so both are
// pointers in the SDK and must be set explicitly to claim otherwise. These
// constructors make the three meaningful shapes explicit at each call site.
//
// Each returns freshly allocated pointers rather than the address of a shared
// package-level bool, so one tool's annotations can never alias another's.
//
// Annotations are hints, not a security boundary — a client may ignore them.
// They exist so an agent can tell helm_list apart from helm_uninstall.

// boolPtr returns a pointer to a copy of v.
func boolPtr(v bool) *bool { return &v }

// ReadOnly describes a tool that only reads state and can be repeated safely.
// openWorld reports whether it reaches beyond the configured cluster and
// repositories — Artifact Hub search and chart downloads do.
func ReadOnly(title string, openWorld bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    true,
		IdempotentHint:  true,
		DestructiveHint: boolPtr(false),
		OpenWorldHint:   boolPtr(openWorld),
	}
}

// Mutating describes a tool that changes state additively — creating or
// updating something without removing or replacing existing resources.
func Mutating(title string, idempotent bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    false,
		IdempotentHint:  idempotent,
		DestructiveHint: boolPtr(false),
		OpenWorldHint:   boolPtr(true),
	}
}

// Destructive describes a tool that can remove or replace live state and so
// needs explicit user intent before an agent calls it.
func Destructive(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    false,
		IdempotentHint:  false,
		DestructiveHint: boolPtr(true),
		OpenWorldHint:   boolPtr(true),
	}
}

// ValidatePluginPath validates a filesystem path supplied as a tool argument,
// applying the same traversal and sensitive-location checks that
// ValidateGlobalInput applies to kubeconfig.
func ValidatePluginPath(path string) error {
	if err := security.ValidatePath(path); err != nil {
		return err
	}
	return validateNotSensitivePath(path)
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
// Responses exceeding MaxResponseBytes are automatically truncated
// with metadata indicating the original size.
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

	text = resilience.TruncateResponse(text, MaxResponseBytes)

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
