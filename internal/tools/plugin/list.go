package plugin

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListInput struct {
	tools.GlobalInput
}

var ListTool = &mcp.Tool{
	Name:        "helm_plugin_list",
	Description: "List installed Helm plugins.",
	Annotations: tools.ReadOnly("List plugins", false),
}

func HandleList(ctx context.Context, _ *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, tools.PluginListOutput, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), tools.PluginListOutput{}, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.PluginList(ctx)
	if err != nil {
		return tools.ErrorResult(err), tools.PluginListOutput{}, nil
	}

	return tools.TextResult(result), tools.PluginListOutput{Plugins: result, Count: len(result)}, nil
}
