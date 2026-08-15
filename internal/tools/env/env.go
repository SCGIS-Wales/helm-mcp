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
	Annotations: tools.ReadOnly("Show Helm environment", false),
}

func HandleEnv(ctx context.Context, _ *mcp.CallToolRequest, input EnvInput) (*mcp.CallToolResult, tools.EnvOutput, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), tools.EnvOutput{}, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.Env(ctx)
	if err != nil {
		return tools.ErrorResult(err), tools.EnvOutput{}, nil
	}

	return tools.TextResult(result), tools.EnvOutput{Environment: result}, nil
}
