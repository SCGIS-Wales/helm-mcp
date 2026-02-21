package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type UninstallInput struct {
	tools.GlobalInput
	ReleaseName  string `json:"release_name" jsonschema:"required" jsonschema_description:"Name of the release to uninstall"`
	KeepHistory  bool   `json:"keep_history,omitempty" jsonschema_description:"Remove all associated resources but keep release history"`
	DryRun       bool   `json:"dry_run,omitempty" jsonschema_description:"Simulate an uninstall"`
	Wait         bool   `json:"wait,omitempty" jsonschema_description:"Wait for deletion of all resources"`
	Timeout      string `json:"timeout,omitempty" jsonschema_description:"Timeout duration"`
	DisableHooks bool   `json:"disable_hooks,omitempty" jsonschema_description:"Disable pre/post uninstall hooks"`
	Cascade      string `json:"cascade,omitempty" jsonschema_description:"Deletion propagation: background foreground or orphan"`
}

var UninstallTool = &mcp.Tool{
	Name:        "helm_uninstall",
	Description: "Uninstall a Helm release and remove all associated Kubernetes resources.",
}

func HandleUninstall(ctx context.Context, req *mcp.CallToolRequest, input UninstallInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()

	result, err := engine.Uninstall(ctx, cfg, &helmengine.UninstallOptions{
		ReleaseName:  input.ReleaseName,
		KeepHistory:  input.KeepHistory,
		DryRun:       input.DryRun,
		Wait:         input.Wait,
		Timeout:      input.Timeout,
		DisableHooks: input.DisableHooks,
		Cascade:      input.Cascade,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
