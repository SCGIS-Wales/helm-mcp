package chart

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/resilience"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type TemplateInput struct {
	tools.GlobalInput
	tools.ChartSourceInput
	tools.ValueSetInput
	tools.PostRenderInput
	ReleaseName      string                 `json:"release_name" jsonschema:"Release name for template rendering"`
	Chart            string                 `json:"chart" jsonschema:"Chart reference"`
	Version          string                 `json:"version,omitempty" jsonschema:"Chart version"`
	Values           map[string]interface{} `json:"values,omitempty" jsonschema:"Inline values, deep-merged over values_files (equivalent to --set)"`
	ValuesFiles      []string               `json:"values_files,omitempty" jsonschema:"Values files"`
	ShowOnly         []string               `json:"show_only,omitempty" jsonschema:"Only show manifests rendered from these templates; paths relative to the chart, globs allowed (e.g. templates/service.yaml)"`
	Validate         bool                   `json:"validate,omitempty" jsonschema:"Validate against the cluster"`
	KubeVersion      string                 `json:"kube_version,omitempty" jsonschema:"Kubernetes version for capabilities"`
	APIVersions      []string               `json:"api_versions,omitempty" jsonschema:"API versions for capabilities"`
	IncludeCRDs      bool                   `json:"include_crds,omitempty" jsonschema:"Include CRDs"`
	SkipCRDs         bool                   `json:"skip_crds,omitempty" jsonschema:"Skip CRDs"`
	NoHooks          bool                   `json:"no_hooks,omitempty" jsonschema:"Leave hooks out of the output"`
	SkipTests        bool                   `json:"skip_tests,omitempty" jsonschema:"Leave test hooks out of the output"`
	IsUpgrade        bool                   `json:"is_upgrade,omitempty" jsonschema:"Render with .Release.IsUpgrade set instead of .Release.IsInstall"`
	DependencyUpdate bool                   `json:"dependency_update,omitempty" jsonschema:"Fetch missing chart dependencies before rendering"`
}

var TemplateTool = &mcp.Tool{
	Name:        "helm_template",
	Description: "Render chart templates locally without installing. Useful for previewing manifests.",
	Annotations: tools.ReadOnly("Render chart templates", true),
}

func HandleTemplate(ctx context.Context, _ *mcp.CallToolRequest, input TemplateInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateChartRef(input.Chart); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateValuesFiles(input.ValuesFiles); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateReleaseName(input.ReleaseName); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateReleaseInputs(&input.ChartSourceInput, &input.ValueSetInput, &input.PostRenderInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	opts := &helmengine.TemplateOptions{
		ChartSourceOptions: input.ChartSourceInput.ToOptions(),
		ValueSetOptions:    input.ValueSetInput.ToOptions(),
		PostRenderOptions:  input.PostRenderInput.ToOptions(),
		ReleaseName:        input.ReleaseName,
		Chart:              input.Chart,
		Version:            input.Version,
		Values:             input.Values,
		ValuesFiles:        input.ValuesFiles,
		ShowOnly:           input.ShowOnly,
		Validate:           input.Validate,
		KubeVersion:        input.KubeVersion,
		APIVersions:        input.APIVersions,
		IncludeCRDs:        input.IncludeCRDs,
		SkipCRDs:           input.SkipCRDs,
		NoHooks:            input.NoHooks,
		SkipTests:          input.SkipTests,
		IsUpgrade:          input.IsUpgrade,
		DependencyUpdate:   input.DependencyUpdate,
	}
	defer opts.ZeroPassword()

	result, err := engine.Template(ctx, cfg, opts)
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	// Strip noisy Kubernetes fields from rendered templates to reduce
	// payload size for LLM consumption.
	return tools.TextResult(resilience.SanitizeManifest(result)), nil, nil
}
