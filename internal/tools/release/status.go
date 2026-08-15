package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type StatusInput struct {
	tools.GlobalInput
	ReleaseName   string `json:"release_name" jsonschema:"Name of the release"`
	Revision      int    `json:"revision,omitempty" jsonschema:"Show status for a specific revision"`
	ShowResources bool   `json:"show_resources,omitempty" jsonschema:"Show resources table (v4 only)"`
}

var StatusTool = &mcp.Tool{
	Name:        "helm_status",
	Description: "Display the status of a Helm release including its revision, chart, and values.",
	Annotations: tools.ReadOnly("Release status", false),
}

func HandleStatus(ctx context.Context, _ *mcp.CallToolRequest, input StatusInput) (*mcp.CallToolResult, tools.StatusOutput, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), tools.StatusOutput{}, nil
	}
	if err := tools.ValidateReleaseName(input.ReleaseName); err != nil {
		return tools.ErrorResult(err), tools.StatusOutput{}, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.Status(ctx, cfg, &helmengine.StatusOptions{
		ReleaseName:   input.ReleaseName,
		Revision:      input.Revision,
		ShowResources: input.ShowResources,
	})
	if err != nil {
		return tools.ErrorResult(err), tools.StatusOutput{}, nil
	}

	return tools.TextResult(result), tools.StatusOutput{Release: result}, nil
}
