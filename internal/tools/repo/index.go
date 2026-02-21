package repo

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type IndexInput struct {
	tools.GlobalInput
	Directory string `json:"directory" jsonschema:"required" jsonschema_description:"Directory containing packaged charts"`
	URL       string `json:"url,omitempty" jsonschema_description:"URL of the chart repository"`
	Merge     string `json:"merge,omitempty" jsonschema_description:"Path to existing index to merge into"`
}

var IndexTool = &mcp.Tool{
	Name:        "helm_repo_index",
	Description: "Generate an index file for a directory of chart archives.",
}

func HandleIndex(ctx context.Context, req *mcp.CallToolRequest, input IndexInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)

	err := engine.RepoIndex(ctx, &helmengine.RepoIndexOptions{
		Directory: input.Directory,
		URL:       input.URL,
		Merge:     input.Merge,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult("Index file generated successfully"), nil, nil
}
