package plugin

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/security"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type UninstallInput struct {
	tools.GlobalInput
	Name string `json:"name" jsonschema:"required" jsonschema_description:"Plugin name"`
}

var UninstallTool = &mcp.Tool{
	Name:        "helm_plugin_uninstall",
	Description: "Uninstall a Helm plugin.",
}

func HandleUninstall(ctx context.Context, req *mcp.CallToolRequest, input UninstallInput) (*mcp.CallToolResult, any, error) {
	if err := security.ValidatePluginName(input.Name); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)

	err := engine.PluginUninstall(ctx, &helmengine.PluginUninstallOptions{
		Name: input.Name,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult("Plugin uninstalled successfully"), nil, nil
}
