package search

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type HubInput struct {
	tools.GlobalInput
	Keyword     string `json:"keyword" jsonschema:"Search keyword"`
	MaxColWidth int    `json:"max_col_width,omitempty" jsonschema:"Max column width"`
	ListRepoURL bool   `json:"list_repo_url,omitempty" jsonschema:"Show repository URL"`
}

var HubTool = &mcp.Tool{
	Name:        "helm_search_hub",
	Description: "Search Artifact Hub for Helm charts.",
}

func HandleHub(ctx context.Context, _ *mcp.CallToolRequest, input HubInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.SearchHub(ctx, &helmengine.SearchHubOptions{
		Keyword:     input.Keyword,
		MaxColWidth: input.MaxColWidth,
		ListRepoURL: input.ListRepoURL,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
