package chart

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type LintInput struct {
	tools.GlobalInput
	Paths         []string               `json:"paths,omitempty" jsonschema_description:"Chart paths to lint (default: current directory)"`
	Values        map[string]interface{} `json:"values,omitempty" jsonschema_description:"Inline values"`
	ValuesFiles   []string               `json:"values_files,omitempty" jsonschema_description:"Values files"`
	Strict        bool                   `json:"strict,omitempty" jsonschema_description:"Treat warnings as errors"`
	WithSubcharts bool                   `json:"with_subcharts,omitempty" jsonschema_description:"Lint subcharts"`
	Quiet         bool                   `json:"quiet,omitempty" jsonschema_description:"Only show warnings and errors"`
}

var LintTool = &mcp.Tool{
	Name:        "helm_lint",
	Description: "Lint a Helm chart for possible issues and best practices.",
}

func HandleLint(ctx context.Context, req *mcp.CallToolRequest, input LintInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.Lint(ctx, &helmengine.LintOptions{
		Paths:         input.Paths,
		Values:        input.Values,
		ValuesFiles:   input.ValuesFiles,
		Strict:        input.Strict,
		WithSubcharts: input.WithSubcharts,
		Quiet:         input.Quiet,
		Namespace:     input.Namespace,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
