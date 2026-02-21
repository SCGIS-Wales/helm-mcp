package env

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type VersionInput struct {
	tools.GlobalInput
	Short bool `json:"short,omitempty" jsonschema_description:"Print only the version number"`
}

var VersionTool = &mcp.Tool{
	Name:        "helm_version",
	Description: "Print the Helm SDK version information.",
}

func HandleVersion(ctx context.Context, req *mcp.CallToolRequest, input VersionInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.Version(ctx)
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	if input.Short {
		return tools.TextResult(result.Version), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
