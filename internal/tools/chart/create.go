package chart

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CreateInput struct {
	tools.GlobalInput
	Name    string `json:"name" jsonschema:"Name of the chart to create; may be a path, in which case the last element is the chart name"`
	Starter string `json:"starter,omitempty" jsonschema:"Starter chart: a name under $HELM_DATA_HOME/starters or an absolute path"`
}

var CreateTool = &mcp.Tool{
	Name:        "helm_create",
	Description: "Create a new Helm chart with the given name, optionally from a starter chart.",
	Annotations: tools.Mutating("Scaffold a new chart", false),
}

func HandleCreate(ctx context.Context, _ *mcp.CallToolRequest, input CreateInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if r := tools.ValidateRequired("name", input.Name); r != nil {
		return r, nil, nil
	}
	// Both are filesystem paths: name is where the chart is written and a
	// starter is read from disk.
	if err := tools.ValidateFilePaths(input.Name, input.Starter); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.Create(ctx, &helmengine.CreateOptions{
		Name:    input.Name,
		Starter: input.Starter,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
