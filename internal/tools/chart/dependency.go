package chart

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type DependencyInput struct {
	tools.GlobalInput
	ChartPath   string `json:"chart_path" jsonschema:"Path to the chart directory"`
	Verify      bool   `json:"verify,omitempty" jsonschema:"Verify dependencies"`
	Keyring     string `json:"keyring,omitempty" jsonschema:"Keyring path"`
	SkipRefresh bool   `json:"skip_refresh,omitempty" jsonschema:"Skip refreshing repository cache"`
}

var DependencyBuildTool = &mcp.Tool{
	Name:        "helm_dependency_build",
	Description: "Build out the charts/ directory from Chart.lock.",
	Annotations: tools.Mutating("Build chart dependencies", true),
}

func HandleDependencyBuild(ctx context.Context, _ *mcp.CallToolRequest, input DependencyInput) (*mcp.CallToolResult, any, error) {
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
	Annotations: tools.ReadOnly("List chart dependencies", false),
}

type DependencyListInput struct {
	tools.GlobalInput
	ChartPath string `json:"chart_path" jsonschema:"Path to the chart directory"`
}

func HandleDependencyList(ctx context.Context, _ *mcp.CallToolRequest, input DependencyListInput) (*mcp.CallToolResult, any, error) {
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
	Annotations: tools.Mutating("Update chart dependencies", true),
}

func HandleDependencyUpdate(ctx context.Context, _ *mcp.CallToolRequest, input DependencyInput) (*mcp.CallToolResult, any, error) {
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
