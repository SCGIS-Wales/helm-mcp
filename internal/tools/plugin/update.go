package plugin

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/security"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type UpdateInput struct {
	tools.GlobalInput
	Name string `json:"name" jsonschema:"required" jsonschema_description:"Plugin name"`
}

var UpdateTool = &mcp.Tool{
	Name:        "helm_plugin_update",
	Description: "Update a Helm plugin.",
}

func HandleUpdate(ctx context.Context, req *mcp.CallToolRequest, input UpdateInput) (*mcp.CallToolResult, any, error) {
	if err := security.ValidatePluginName(input.Name); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)

	err := engine.PluginUpdate(ctx, &helmengine.PluginUpdateOptions{
		Name: input.Name,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult("Plugin updated successfully"), nil, nil
}
