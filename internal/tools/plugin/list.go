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
}

func HandleList(ctx context.Context, req *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.PluginList(ctx)
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
