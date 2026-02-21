package v3

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

// V3Engine implements helmengine.Engine using the Helm v3 SDK.
type V3Engine struct{}

// New creates a new V3Engine.
func New() *V3Engine {
	return &V3Engine{}
}

// newActionConfig creates a new Helm v3 action.Configuration initialized
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

	actionConfig := new(action.Configuration)

	var logFunc action.DebugLog
	if cfg.Debug {
		logFunc = log.Printf
	} else {
		logFunc = func(format string, v ...interface{}) { /* debug disabled */ }
	}

	if err := actionConfig.Init(
		settings.RESTClientGetter(),
		settings.Namespace(),
		os.Getenv("HELM_DRIVER"),
		logFunc,
	); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize helm v3 configuration: %w", err)
	}

	return actionConfig, settings, nil
}

// newActionConfigNoCluster creates a Configuration without cluster access
// (for chart-only operations like lint, create, package).
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

	actionConfig := new(action.Configuration)
	actionConfig.Log = func(format string, v ...interface{}) { /* no-cluster: logging disabled */ }

	return actionConfig, settings
}

func (e *V3Engine) Version(_ context.Context) (*helmengine.VersionInfo, error) {
	return &helmengine.VersionInfo{
		Version:   "v3 (SDK)",
		GoVersion: runtime.Version(),
	}, nil
}

func (e *V3Engine) Env(_ context.Context) (map[string]string, error) {
	settings := cli.New()
	envVars := settings.EnvVars()
	result := make(map[string]string, len(envVars))
	for k, v := range envVars {
		result[k] = v
	}
	return result, nil
}

// discardLogger returns an io.Writer that discards all output.
var _ io.Writer = io.Discard
