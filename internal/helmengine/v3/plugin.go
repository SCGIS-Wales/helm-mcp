package v3

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/plugin"
)

// pluginExecTimeout is the maximum duration for plugin CLI operations.
const pluginExecTimeout = 5 * time.Minute

func (e *V3Engine) PluginInstall(ctx context.Context, opts *helmengine.PluginInstallOptions) error {
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

func (e *V3Engine) PluginList(ctx context.Context) ([]*helmengine.PluginInfo, error) {
	settings := cli.New()
	plugins, err := plugin.FindPlugins(settings.PluginsDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to find plugins: %w", err)
	}

	result := make([]*helmengine.PluginInfo, 0, len(plugins))
	for _, p := range plugins {
		result = append(result, &helmengine.PluginInfo{
			Name:        p.Metadata.Name,
			Version:     p.Metadata.Version,
			Description: p.Metadata.Description,
		})
	}

	return result, nil
}

func (e *V3Engine) PluginUninstall(ctx context.Context, opts *helmengine.PluginUninstallOptions) error {
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

func (e *V3Engine) PluginUpdate(ctx context.Context, opts *helmengine.PluginUpdateOptions) error {
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
