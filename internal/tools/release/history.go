package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type HistoryInput struct {
	tools.GlobalInput
	ReleaseName string `json:"release_name" jsonschema:"required" jsonschema_description:"Name of the release"`
	Max         int    `json:"max,omitempty" jsonschema_description:"Maximum number of revisions to return"`
}

var HistoryTool = &mcp.Tool{
	Name:        "helm_history",
	Description: "Show the revision history of a Helm release.",
}

func HandleHistory(ctx context.Context, req *mcp.CallToolRequest, input HistoryInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateReleaseName(input.ReleaseName); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.History(ctx, cfg, &helmengine.HistoryOptions{
		ReleaseName: input.ReleaseName,
		Max:         input.Max,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
