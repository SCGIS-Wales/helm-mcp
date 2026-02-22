package chart

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type VerifyInput struct {
	tools.GlobalInput
	ChartFile string `json:"chart_file" jsonschema:"required" jsonschema_description:"Path to chart archive to verify"`
	Keyring   string `json:"keyring,omitempty" jsonschema_description:"Keyring path"`
}

var VerifyTool = &mcp.Tool{
	Name:        "helm_verify",
	Description: "Verify that a chart has a valid provenance file.",
}

func HandleVerify(ctx context.Context, req *mcp.CallToolRequest, input VerifyInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.Verify(ctx, &helmengine.VerifyOptions{
		ChartFile: input.ChartFile,
		Keyring:   input.Keyring,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
