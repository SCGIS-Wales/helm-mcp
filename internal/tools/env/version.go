package env

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type VersionInput struct {
	tools.GlobalInput
	Short bool `json:"short,omitempty" jsonschema:"Print only the version number"`
}

var VersionTool = &mcp.Tool{
	Name:        "helm_version",
	Description: "Print the Helm SDK version information.",
	Annotations: tools.ReadOnly("Show Helm version", false),
}

func HandleVersion(ctx context.Context, _ *mcp.CallToolRequest, input VersionInput) (*mcp.CallToolResult, tools.VersionOutput, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), tools.VersionOutput{}, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.Version(ctx)
	if err != nil {
		return tools.ErrorResult(err), tools.VersionOutput{}, nil
	}

	// `short` only changes the text rendering; the structured content stays
	// the full version object either way.
	if input.Short {
		return tools.TextResult(result.Version), tools.VersionOutput{Version: result}, nil
	}

	return tools.TextResult(result), tools.VersionOutput{Version: result}, nil
}
