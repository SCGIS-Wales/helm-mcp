package chart

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type DependencyInput struct {
	tools.GlobalInput
	ChartPath   string `json:"chart_path" jsonschema:"required" jsonschema_description:"Path to the chart directory"`
	Verify      bool   `json:"verify,omitempty" jsonschema_description:"Verify dependencies"`
	Keyring     string `json:"keyring,omitempty" jsonschema_description:"Keyring path"`
	SkipRefresh bool   `json:"skip_refresh,omitempty" jsonschema_description:"Skip refreshing repository cache"`
}

var DependencyBuildTool = &mcp.Tool{
	Name:        "helm_dependency_build",
	Description: "Build out the charts/ directory from Chart.lock.",
}

func HandleDependencyBuild(ctx context.Context, req *mcp.CallToolRequest, input DependencyInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	err := engine.DependencyBuild(ctx, cfg, toDependencyOpts(&input))
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult("Dependency build successful"), nil, nil
}

var DependencyListTool = &mcp.Tool{
	Name:        "helm_dependency_list",
	Description: "List the dependencies for a chart.",
}

type DependencyListInput struct {
	tools.GlobalInput
	ChartPath string `json:"chart_path" jsonschema:"required" jsonschema_description:"Path to the chart directory"`
}

func HandleDependencyList(ctx context.Context, req *mcp.CallToolRequest, input DependencyListInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.DependencyList(ctx, cfg, &helmengine.DependencyOptions{
		ChartPath: input.ChartPath,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}

var DependencyUpdateTool = &mcp.Tool{
	Name:        "helm_dependency_update",
	Description: "Update charts/ based on Chart.yaml contents.",
}

func HandleDependencyUpdate(ctx context.Context, req *mcp.CallToolRequest, input DependencyInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	err := engine.DependencyUpdate(ctx, cfg, toDependencyOpts(&input))
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult("Dependency update successful"), nil, nil
}

func toDependencyOpts(input *DependencyInput) *helmengine.DependencyOptions {
	return &helmengine.DependencyOptions{
		ChartPath:   input.ChartPath,
		Verify:      input.Verify,
		Keyring:     input.Keyring,
		SkipRefresh: input.SkipRefresh,
	}
}
