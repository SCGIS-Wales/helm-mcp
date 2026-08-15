package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type HistoryInput struct {
	tools.GlobalInput
	ReleaseName string `json:"release_name" jsonschema:"Name of the release"`
	Max         int    `json:"max,omitempty" jsonschema:"Maximum number of revisions to return"`
}

var HistoryTool = &mcp.Tool{
	Name:        "helm_history",
	Description: "Show the revision history of a Helm release.",
	Annotations: tools.ReadOnly("Release history", false),
}

func HandleHistory(ctx context.Context, _ *mcp.CallToolRequest, input HistoryInput) (*mcp.CallToolResult, tools.HistoryOutput, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), tools.HistoryOutput{}, nil
	}
	if err := tools.ValidateReleaseName(input.ReleaseName); err != nil {
		return tools.ErrorResult(err), tools.HistoryOutput{}, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.History(ctx, cfg, &helmengine.HistoryOptions{
		ReleaseName: input.ReleaseName,
		Max:         input.Max,
	})
	if err != nil {
		return tools.ErrorResult(err), tools.HistoryOutput{}, nil
	}

	return tools.TextResult(result), tools.HistoryOutput{Revisions: result, Count: len(result)}, nil
}
