package v4

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"
)

// Plugin operations in Helm v4 use internal packages that are not publicly
// importable. We use the helm CLI for all plugin operations.

// pluginExecTimeout is the maximum duration for plugin CLI operations.
const pluginExecTimeout = 5 * time.Minute

func (e *V4Engine) PluginInstall(ctx context.Context, opts *helmengine.PluginInstallOptions) error {
	args := []string{"plugin", "install"}
	if opts.Version != "" {
		args = append(args, "--version", opts.Version)
	}
	args = append(args, "--", opts.URLOrPath)

	execCtx, cancel := context.WithTimeout(ctx, pluginExecTimeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "helm", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm plugin install failed: %s: %w", stderr.String(), err)
	}

	return nil
}

func (e *V4Engine) PluginList(ctx context.Context) ([]*helmengine.PluginInfo, error) {
	execCtx, cancel := context.WithTimeout(ctx, pluginExecTimeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "helm", "plugin", "list")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("helm plugin list failed: %s: %w", stderr.String(), err)
	}

	// Parse the output (tab-separated: NAME VERSION DESCRIPTION)
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	var result []*helmengine.PluginInfo
	for i, line := range lines {
		if i == 0 {
			continue // skip header
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		info := &helmengine.PluginInfo{
			Name:    fields[0],
			Version: fields[1],
		}
		if len(fields) > 2 {
			info.Description = strings.Join(fields[2:], " ")
		}
		result = append(result, info)
	}

	return result, nil
}

func (e *V4Engine) PluginUninstall(ctx context.Context, opts *helmengine.PluginUninstallOptions) error {
	execCtx, cancel := context.WithTimeout(ctx, pluginExecTimeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "helm", "plugin", "uninstall", "--", opts.Name)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm plugin uninstall failed: %s: %w", stderr.String(), err)
	}

	return nil
}

func (e *V4Engine) PluginUpdate(ctx context.Context, opts *helmengine.PluginUpdateOptions) error {
	execCtx, cancel := context.WithTimeout(ctx, pluginExecTimeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "helm", "plugin", "update", "--", opts.Name)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm plugin update failed: %s: %w", stderr.String(), err)
	}

	return nil
}

// pluginPackageArgs builds the `helm plugin package` argument list.
//
// Split out from PluginPackage so the flag mapping is testable without
// depending on whatever helm binary happens to be on PATH.
func pluginPackageArgs(opts *helmengine.PluginPackageOptions) []string {
	args := []string{"plugin", "package"}
	// Signing is on by default in the CLI; only pass the flag when the caller
	// explicitly opts out, so the CLI default stays authoritative.
	if !opts.Sign {
		args = append(args, "--sign=false")
	}
	if opts.Key != "" {
		args = append(args, "--key", opts.Key)
	}
	if opts.Keyring != "" {
		args = append(args, "--keyring", opts.Keyring)
	}
	if opts.PassphraseFile != "" {
		args = append(args, "--passphrase-file", opts.PassphraseFile)
	}
	if opts.Destination != "" {
		args = append(args, "--destination", opts.Destination)
	}
	// "--" keeps a path that starts with a dash from being read as a flag.
	return append(args, "--", opts.PluginPath)
}

// pluginVerifyArgs builds the `helm plugin verify` argument list.
func pluginVerifyArgs(opts *helmengine.PluginVerifyOptions) []string {
	args := []string{"plugin", "verify"}
	if opts.Keyring != "" {
		args = append(args, "--keyring", opts.Keyring)
	}
	return append(args, "--", opts.PluginPath)
}

// PluginPackage packages a plugin directory into a signed archive.
// New in Helm v4.
func (e *V4Engine) PluginPackage(ctx context.Context, opts *helmengine.PluginPackageOptions) (string, error) {
	args := pluginPackageArgs(opts)

	execCtx, cancel := context.WithTimeout(ctx, pluginExecTimeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "helm", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("helm plugin package failed: %s: %w", stderr.String(), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// PluginVerify verifies that a packaged plugin has been signed and is valid.
// New in Helm v4.
func (e *V4Engine) PluginVerify(ctx context.Context, opts *helmengine.PluginVerifyOptions) (string, error) {
	args := pluginVerifyArgs(opts)

	execCtx, cancel := context.WithTimeout(ctx, pluginExecTimeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "helm", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("helm plugin verify failed: %s: %w", stderr.String(), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}
