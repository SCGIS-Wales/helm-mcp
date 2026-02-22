// Package security provides input validation, path sanitization, and
// credential scrubbing for the Helm MCP server.
package security

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// validNamePattern matches Helm release names and chart names.
// Helm names must be lowercase alphanumeric, dashes, or dots.
var validNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-\.]*[a-z0-9])?$`)

// validPluginNamePattern matches Helm plugin names.
// Plugin names must be alphanumeric (case-insensitive), dashes, or underscores,
// and must not start with a dash (to prevent argument injection).
var validPluginNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_\-]*$`)

// Regex patterns for credential scrubbing (compiled once at package init).
var (
	scrubTokenPattern       = regexp.MustCompile(`(?i)(bearer\s+|token[=:]\s*)[^\s"']+`)
	scrubBasicAuthPattern   = regexp.MustCompile(`(?i)(basic\s+)[^\s"']+`)
	scrubURLPasswordPattern = regexp.MustCompile(`://[^:]+:[^@]+@`)
)

// maxNameLength is the maximum length for release names.
const maxNameLength = 253

// ValidateReleaseName validates a Helm release name.
func ValidateReleaseName(name string) error {
	if name == "" {
		return fmt.Errorf("release name is required")
	}
	if len(name) > maxNameLength {
		return fmt.Errorf("release name %q exceeds maximum length of %d", name, maxNameLength)
	}
	if !validNamePattern.MatchString(name) {
		return fmt.Errorf("release name %q is invalid: must consist of lowercase alphanumeric characters, dashes, or dots, and must start and end with an alphanumeric character", name)
	}
	return nil
}

// ValidateNamespace validates a Kubernetes namespace name.
func ValidateNamespace(ns string) error {
	if ns == "" {
		return nil // empty is valid (uses default)
	}
	if len(ns) > maxNameLength {
		return fmt.Errorf("namespace %q exceeds maximum length of %d", ns, maxNameLength)
	}
	if !validNamePattern.MatchString(ns) {
		return fmt.Errorf("namespace %q is invalid: must consist of lowercase alphanumeric characters or dashes", ns)
	}
	return nil
}

// ValidateKubeConfig validates a kubeconfig file path.
// It checks that the path exists and is a regular file (not a directory,
// symlink to /etc/shadow, etc.).
func ValidateKubeConfig(path string) error {
	if path == "" {
		return nil // empty means use default
	}

	// Resolve to absolute path to prevent traversal
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid kubeconfig path %q: %w", path, err)
	}

	// Check for obvious path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("kubeconfig path %q must not contain '..'", path)
	}

	// Use Lstat to detect symlinks without following them.
	// absPath is derived from filepath.Abs (which resolves and cleans the path)
	// and we've already rejected paths containing "..".
	cleanPath := filepath.Clean(absPath)
	info, err := os.Lstat(cleanPath) //#nosec G703 -- path is sanitized above
	if err != nil {
		return fmt.Errorf("kubeconfig %q not accessible: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("kubeconfig %q is a symlink, which is not allowed for security", path)
	}
	if info.IsDir() {
		return fmt.Errorf("kubeconfig %q is a directory, not a file", path)
	}

	return nil
}

// ValidateURL performs basic URL validation.
func ValidateURL(u string) error {
	if u == "" {
		return fmt.Errorf("URL is required")
	}
	// Must start with a valid scheme
	if !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") &&
		!strings.HasPrefix(u, "oci://") {
		return fmt.Errorf("URL %q must start with https://, http://, or oci://", u)
	}
	return nil
}

// ValidatePath checks that a file path does not contain path traversal sequences.
func ValidatePath(path string) error {
	if path == "" {
		return nil
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("path %q must not contain '..'", path)
	}
	return nil
}

// ValidateTimeout checks that a timeout duration string is parseable and
// within reasonable bounds (max 24 hours).
func ValidateTimeout(timeout string) error {
	if timeout == "" {
		return nil
	}
	d, err := time.ParseDuration(timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout %q: %w", timeout, err)
	}
	if d > 24*time.Hour {
		return fmt.Errorf("timeout %q exceeds maximum of 24h", timeout)
	}
	if d < 0 {
		return fmt.Errorf("timeout must not be negative")
	}
	return nil
}

// ScrubCredentials removes sensitive values from a string map.
// It replaces values for keys matching common credential patterns.
func ScrubCredentials(m map[string]string) map[string]string {
	scrubbed := make(map[string]string, len(m))
	for k, v := range m {
		lower := strings.ToLower(k)
		if strings.Contains(lower, "password") ||
			strings.Contains(lower, "token") ||
			strings.Contains(lower, "secret") ||
			strings.Contains(lower, "key") ||
			strings.Contains(lower, "credential") {
			scrubbed[k] = "***REDACTED***"
		} else {
			scrubbed[k] = v
		}
	}
	return scrubbed
}

// ValidatePluginName validates a Helm plugin name.
// Plugin names must be alphanumeric, dashes, or underscores, and must not
// start with a dash (to prevent argument injection when passed to helm CLI).
func ValidatePluginName(name string) error {
	if name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if len(name) > maxNameLength {
		return fmt.Errorf("plugin name %q exceeds maximum length of %d", name, maxNameLength)
	}
	if !validPluginNamePattern.MatchString(name) {
		return fmt.Errorf("plugin name %q is invalid: must consist of alphanumeric characters, dashes, or underscores, and must start with an alphanumeric character", name)
	}
	return nil
}

// ScrubError removes potentially sensitive information from error messages
// (tokens, passwords) before returning them to the user.
func ScrubError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	msg = scrubTokenPattern.ReplaceAllString(msg, "${1}***REDACTED***")
	msg = scrubBasicAuthPattern.ReplaceAllString(msg, "${1}***REDACTED***")
	msg = scrubURLPasswordPattern.ReplaceAllString(msg, "://***:***@")

	if msg != err.Error() {
		return fmt.Errorf("%s", msg)
	}
	return err
}
