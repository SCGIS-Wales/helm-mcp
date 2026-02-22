package chart

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ShowInput struct {
	tools.GlobalInput
	Chart    string `json:"chart" jsonschema:"required" jsonschema_description:"Chart reference"`
	Version  string `json:"version,omitempty" jsonschema_description:"Chart version"`
	Repo     string `json:"repo,omitempty" jsonschema_description:"Repository URL"`
	Devel    bool   `json:"devel,omitempty" jsonschema_description:"Include development versions"`
	JSONPath string `json:"jsonpath,omitempty" jsonschema_description:"JSONPath template for values (v4 only)"`
}

var ShowAllTool = &mcp.Tool{
	Name:        "helm_show_all",
	Description: "Show all information for a chart (Chart.yaml, values, README, CRDs).",
}

func HandleShowAll(ctx context.Context, req *mcp.CallToolRequest, input ShowInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()
	result, err := engine.ShowAll(ctx, cfg, toShowOpts(&input))
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	return tools.TextResult(result), nil, nil
}

var ShowChartTool = &mcp.Tool{
	Name:        "helm_show_chart",
	Description: "Show the Chart.yaml of a chart.",
}

func HandleShowChart(ctx context.Context, req *mcp.CallToolRequest, input ShowInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()
	result, err := engine.ShowChart(ctx, cfg, toShowOpts(&input))
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	return tools.TextResult(result), nil, nil
}

var ShowCRDsTool = &mcp.Tool{
	Name:        "helm_show_crds",
	Description: "Show the CRDs of a chart.",
}

func HandleShowCRDs(ctx context.Context, req *mcp.CallToolRequest, input ShowInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()
	result, err := engine.ShowCRDs(ctx, cfg, toShowOpts(&input))
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	return tools.TextResult(result), nil, nil
}

var ShowReadmeTool = &mcp.Tool{
	Name:        "helm_show_readme",
	Description: "Show the README of a chart.",
}

func HandleShowReadme(ctx context.Context, req *mcp.CallToolRequest, input ShowInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()
	result, err := engine.ShowReadme(ctx, cfg, toShowOpts(&input))
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	return tools.TextResult(result), nil, nil
}

var ShowValuesTool = &mcp.Tool{
	Name:        "helm_show_values",
	Description: "Show the default values of a chart.",
}

func HandleShowValues(ctx context.Context, req *mcp.CallToolRequest, input ShowInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()
	result, err := engine.ShowValues(ctx, cfg, toShowOpts(&input))
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	return tools.TextResult(result), nil, nil
}

func toShowOpts(input *ShowInput) *helmengine.ShowOptions {
	return &helmengine.ShowOptions{
		Chart:    input.Chart,
		Version:  input.Version,
		Repo:     input.Repo,
		Devel:    input.Devel,
		JSONPath: input.JSONPath,
	}
}
