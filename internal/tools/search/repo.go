package search

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type RepoInput struct {
	tools.GlobalInput
	Keyword           string `json:"keyword" jsonschema:"required" jsonschema_description:"Search keyword"`
	Regexp            bool   `json:"regexp,omitempty" jsonschema_description:"Use regular expressions"`
	Versions          bool   `json:"versions,omitempty" jsonschema_description:"Show all versions"`
	Devel             bool   `json:"devel,omitempty" jsonschema_description:"Include development versions"`
	VersionConstraint string `json:"version_constraint,omitempty" jsonschema_description:"Semver version constraint"`
}

var RepoTool = &mcp.Tool{
	Name:        "helm_search_repo",
	Description: "Search locally configured repositories for charts.",
}

func HandleRepo(ctx context.Context, req *mcp.CallToolRequest, input RepoInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.SearchRepo(ctx, &helmengine.SearchRepoOptions{
		Keyword:           input.Keyword,
		Regexp:            input.Regexp,
		Versions:          input.Versions,
		Devel:             input.Devel,
		VersionConstraint: input.VersionConstraint,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
