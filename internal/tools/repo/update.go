package repo

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type UpdateInput struct {
	tools.GlobalInput
	Names []string `json:"names,omitempty" jsonschema_description:"Repository names to update (all if empty)"`
}

var UpdateTool = &mcp.Tool{
	Name:        "helm_repo_update",
	Description: "Update chart repository indexes.",
}

func HandleUpdate(ctx context.Context, req *mcp.CallToolRequest, input UpdateInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.RepoUpdate(ctx, &helmengine.RepoUpdateOptions{
		Names: input.Names,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
