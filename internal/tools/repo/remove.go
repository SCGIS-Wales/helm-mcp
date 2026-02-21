package repo

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type RemoveInput struct {
	tools.GlobalInput
	Names []string `json:"names" jsonschema:"required" jsonschema_description:"Repository names to remove"`
}

var RemoveTool = &mcp.Tool{
	Name:        "helm_repo_remove",
	Description: "Remove chart repositories.",
}

func HandleRemove(ctx context.Context, req *mcp.CallToolRequest, input RemoveInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)

	err := engine.RepoRemove(ctx, &helmengine.RepoRemoveOptions{
		Names: input.Names,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult("Repositories removed successfully"), nil, nil
}
