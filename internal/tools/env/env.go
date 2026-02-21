package env

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type EnvInput struct {
	tools.GlobalInput
}

var EnvTool = &mcp.Tool{
	Name:        "helm_env",
	Description: "Print Helm environment information (paths, settings, etc.).",
}

func HandleEnv(ctx context.Context, req *mcp.CallToolRequest, input EnvInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.Env(ctx)
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
