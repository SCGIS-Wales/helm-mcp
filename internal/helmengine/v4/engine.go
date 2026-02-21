package v4

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/cli"
)

// V4Engine implements helmengine.Engine using the Helm v4 SDK.
type V4Engine struct{}

// New creates a new V4Engine.
func New() *V4Engine {
	return &V4Engine{}
}

// newActionConfig creates a new Helm v4 action.Configuration initialized
// with the given global config. It configures Kubernetes authentication
// using kubeconfig, context, API server, and token settings.
func newActionConfig(cfg *helmengine.GlobalConfig) (*action.Configuration, *cli.EnvSettings, error) {
	settings := cli.New()

	if cfg.Namespace != "" {
		settings.SetNamespace(cfg.Namespace)
	}
	if cfg.KubeContext != "" {
		settings.KubeContext = cfg.KubeContext
	}
	if cfg.KubeConfig != "" {
		settings.KubeConfig = cfg.KubeConfig
	}
	if cfg.KubeAPIServer != "" {
		settings.KubeAPIServer = cfg.KubeAPIServer
	}
	if cfg.KubeBearerToken != "" {
		settings.KubeToken = cfg.KubeBearerToken
	}
	if cfg.KubeTLSServerName != "" {
		settings.KubeTLSServerName = cfg.KubeTLSServerName
	}
	if cfg.KubeInsecureTLS {
		settings.KubeInsecureSkipTLSVerify = true
	}
	if cfg.BurstLimit > 0 {
		settings.BurstLimit = cfg.BurstLimit
	}
	if cfg.QPS > 0 {
		settings.QPS = cfg.QPS
	}
	settings.Debug = cfg.Debug

	var logHandler slog.Handler
	if cfg.Debug {
		logHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		logHandler = slog.NewTextHandler(io.Discard, nil)
	}

	actionConfig := action.NewConfiguration(
		action.ConfigurationSetLogger(logHandler),
	)

	if err := actionConfig.Init(
		settings.RESTClientGetter(),
		settings.Namespace(),
		os.Getenv("HELM_DRIVER"),
	); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize helm v4 configuration: %w", err)
	}

	return actionConfig, settings, nil
}

// newActionConfigNoCluster creates a Configuration without cluster access.
func newActionConfigNoCluster(cfg *helmengine.GlobalConfig) (*action.Configuration, *cli.EnvSettings) {
	settings := cli.New()

	if cfg != nil {
		if cfg.Namespace != "" {
			settings.SetNamespace(cfg.Namespace)
		}
		if cfg.KubeContext != "" {
			settings.KubeContext = cfg.KubeContext
		}
		if cfg.KubeConfig != "" {
			settings.KubeConfig = cfg.KubeConfig
		}
		settings.Debug = cfg.Debug
	}

	logHandler := slog.NewTextHandler(io.Discard, nil)
	actionConfig := action.NewConfiguration(
		action.ConfigurationSetLogger(logHandler),
	)

	return actionConfig, settings
}

func (e *V4Engine) Version(_ context.Context) (*helmengine.VersionInfo, error) {
	return &helmengine.VersionInfo{
		Version:   "v4 (SDK)",
		GoVersion: runtime.Version(),
	}, nil
}

func (e *V4Engine) Env(_ context.Context) (map[string]string, error) {
	settings := cli.New()
	envVars := settings.EnvVars()
	result := make(map[string]string, len(envVars))
	for k, v := range envVars {
		result[k] = v
	}
	return result, nil
}
