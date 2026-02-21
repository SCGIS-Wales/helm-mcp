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
	args := []string{"plugin", "install", opts.URLOrPath}
	if opts.Version != "" {
		args = append(args, "--version", opts.Version)
	}

	execCtx, cancel := context.WithTimeout(ctx, pluginExecTimeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "helm", args...)
	var stderr bytes.Buffer
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

	cmd := exec.CommandContext(execCtx, "helm", "plugin", "uninstall", opts.Name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm plugin uninstall failed: %s: %w", stderr.String(), err)
	}

	return nil
}

func (e *V4Engine) PluginUpdate(ctx context.Context, opts *helmengine.PluginUpdateOptions) error {
	execCtx, cancel := context.WithTimeout(ctx, pluginExecTimeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "helm", "plugin", "update", opts.Name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm plugin update failed: %s: %w", stderr.String(), err)
	}

	return nil
}
