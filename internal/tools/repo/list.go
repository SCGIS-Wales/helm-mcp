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
	Annotations: tools.ReadOnly("List repositories", false),
}

func HandleList(ctx context.Context, _ *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, tools.RepoListOutput, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), tools.RepoListOutput{}, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.RepoList(ctx, &helmengine.RepoListOptions{})
	if err != nil {
		return tools.ErrorResult(err), tools.RepoListOutput{}, nil
	}

	return tools.TextResult(result), tools.RepoListOutput{Repositories: result, Count: len(result)}, nil
}
