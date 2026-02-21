package repo

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListInput struct {
	tools.GlobalInput
}

var ListTool = &mcp.Tool{
	Name:        "helm_repo_list",
	Description: "List configured chart repositories.",
}

func HandleList(ctx context.Context, req *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.RepoList(ctx, &helmengine.RepoListOptions{})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
