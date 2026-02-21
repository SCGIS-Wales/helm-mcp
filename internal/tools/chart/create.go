package chart

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CreateInput struct {
	tools.GlobalInput
	Name    string `json:"name" jsonschema:"required" jsonschema_description:"Name of the chart to create"`
	Starter string `json:"starter,omitempty" jsonschema_description:"Starter chart name"`
}

var CreateTool = &mcp.Tool{
	Name:        "helm_create",
	Description: "Create a new Helm chart with the given name in the current directory.",
}

func HandleCreate(ctx context.Context, req *mcp.CallToolRequest, input CreateInput) (*mcp.CallToolResult, any, error) {
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
