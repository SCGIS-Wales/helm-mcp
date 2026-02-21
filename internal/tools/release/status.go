package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type StatusInput struct {
	tools.GlobalInput
	ReleaseName   string `json:"release_name" jsonschema:"required" jsonschema_description:"Name of the release"`
	Revision      int    `json:"revision,omitempty" jsonschema_description:"Show status for a specific revision"`
	ShowResources bool   `json:"show_resources,omitempty" jsonschema_description:"Show resources table (v4 only)"`
}

var StatusTool = &mcp.Tool{
	Name:        "helm_status",
	Description: "Display the status of a Helm release including its revision, chart, and values.",
}

func HandleStatus(ctx context.Context, req *mcp.CallToolRequest, input StatusInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()

	result, err := engine.Status(ctx, cfg, &helmengine.StatusOptions{
		ReleaseName:   input.ReleaseName,
		Revision:      input.Revision,
		ShowResources: input.ShowResources,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
