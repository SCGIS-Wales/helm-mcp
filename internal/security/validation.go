// Package security provides input validation, path sanitization, and
// credential scrubbing for the Helm MCP server.
package security

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
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

// privateIPNets contains CIDR ranges considered private/internal.
// Used by ValidateURL to block SSRF attempts.
var privateIPNets []*net.IPNet

func init() {
	cidrs := []string{
		"127.0.0.0/8",    // loopback
		"10.0.0.0/8",     // RFC 1918
		"172.16.0.0/12",  // RFC 1918
		"192.168.0.0/16", // RFC 1918
		"169.254.0.0/16", // link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique-local
	}
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Fatalf("invalid CIDR in privateIPNets: %s", cidr)
		}
		privateIPNets = append(privateIPNets, ipNet)
	}
}

// isPrivateIP checks whether an IP address falls within a private/internal range.
func isPrivateIP(ip net.IP) bool {
	for _, ipNet := range privateIPNets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

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
//
// NOTE: There is an inherent TOCTOU (time-of-check-time-of-use) race between
// the Lstat check here and the actual file open by the Helm SDK. An attacker
// with local filesystem access could replace the file with a symlink between
// the two operations. Mitigating this fully would require opening the file
// descriptor here with O_NOFOLLOW and passing it through. The risk is limited
// to local privilege escalation scenarios.
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

// ValidateURL validates a URL and blocks SSRF attempts targeting private/internal networks.
// The ctx parameter allows the caller to cancel DNS resolution if the parent
// operation times out (previously this used context.Background which could hang).
//
// NOTE: The DNS check is inherently TOCTOU — the hostname may resolve to a
// different IP at actual connection time. This is a best-effort pre-flight
// check; full SSRF protection requires network-level controls.
func ValidateURL(u string, ctx ...context.Context) error {
	if u == "" {
		return fmt.Errorf("URL is required")
	}
	// Must start with a valid scheme
	if !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") &&
		!strings.HasPrefix(u, "oci://") {
		return fmt.Errorf("URL %q must start with https://, http://, or oci://", u)
	}

	// Swap oci:// to https:// for net/url parsing (oci isn't a standard scheme).
	parseURL := u
	if strings.HasPrefix(u, "oci://") {
		parseURL = "https://" + strings.TrimPrefix(u, "oci://")
	}

	parsed, err := url.Parse(parseURL)
	if err != nil {
		return fmt.Errorf("URL %q is malformed: %w", u, err)
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL %q has no hostname", u)
	}

	// Block literal "localhost"
	if strings.EqualFold(hostname, "localhost") {
		return fmt.Errorf("URL %q targets localhost, which is not allowed", u)
	}

	// If hostname is a literal IP address, check immediately.
	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("URL %q resolves to a private IP address", u)
		}
		return nil
	}

	// Use the provided context (if any) so DNS resolution can be cancelled
	// when the parent request times out. Falls back to a 10-second timeout
	// to avoid hanging indefinitely on slow DNS.
	dnsCtx := context.Background()
	if len(ctx) > 0 && ctx[0] != nil {
		dnsCtx = ctx[0]
	} else {
		var cancel context.CancelFunc
		dnsCtx, cancel = context.WithTimeout(dnsCtx, 10*time.Second)
		defer cancel()
	}

	// Resolve hostname and check all returned IPs.
	// Permanent DNS failures ("no such host") are safe — the domain doesn't
	// exist so it can't be used for SSRF. Temporary/timeout failures are
	// blocked because they could mask a private-IP resolution (DNS rebinding).
	addrs, err := net.DefaultResolver.LookupHost(dnsCtx, hostname)
	if err != nil {
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			return nil // permanent failure — domain doesn't exist, not an SSRF risk
		}
		return fmt.Errorf("DNS resolution failed for %q: %w", hostname, err)
	}
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip != nil && isPrivateIP(ip) {
			return fmt.Errorf("URL %q resolves to a private IP address", u)
		}
	}

	return nil
}

// ValidatePath checks that a file path does not contain path traversal
// sequences and is not a symlink (which could point to sensitive files).
//
// NOTE: Same TOCTOU caveat as ValidateKubeConfig — see its documentation.
func ValidatePath(path string) error {
	if path == "" {
		return nil
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("path %q must not contain '..'", path)
	}

	// Resolve to absolute and check for symlinks — without this a symlink
	// to /etc/shadow (for example) could be read via values files.
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", path, err)
	}
	cleanPath := filepath.Clean(absPath)
	info, err := os.Lstat(cleanPath)
	if err != nil {
		// File doesn't exist yet — that's OK; the caller will get an error
		// when it actually tries to open it.
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("path %q not accessible: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("path %q is a symlink, which is not allowed for security", path)
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
