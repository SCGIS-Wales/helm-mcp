package chart

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type TemplateInput struct {
	tools.GlobalInput
	ReleaseName      string                 `json:"release_name" jsonschema:"required" jsonschema_description:"Release name for template rendering"`
	Chart            string                 `json:"chart" jsonschema:"required" jsonschema_description:"Chart reference"`
	Version          string                 `json:"version,omitempty" jsonschema_description:"Chart version"`
	Values           map[string]interface{} `json:"values,omitempty" jsonschema_description:"Inline values"`
	ValuesFiles      []string               `json:"values_files,omitempty" jsonschema_description:"Values files"`
	ShowOnly         []string               `json:"show_only,omitempty" jsonschema_description:"Only show manifests from these templates"`
	Validate         bool                   `json:"validate,omitempty" jsonschema_description:"Validate against the cluster"`
	KubeVersion      string                 `json:"kube_version,omitempty" jsonschema_description:"Kubernetes version for capabilities"`
	APIVersions      []string               `json:"api_versions,omitempty" jsonschema_description:"API versions for capabilities"`
	IncludeCRDs      bool                   `json:"include_crds,omitempty" jsonschema_description:"Include CRDs"`
	SkipCRDs         bool                   `json:"skip_crds,omitempty" jsonschema_description:"Skip CRDs"`
	NoHooks          bool                   `json:"no_hooks,omitempty" jsonschema_description:"Skip hooks"`
	DependencyUpdate bool                   `json:"dependency_update,omitempty" jsonschema_description:"Update dependencies"`
}

var TemplateTool = &mcp.Tool{
	Name:        "helm_template",
	Description: "Render chart templates locally without installing. Useful for previewing manifests.",
}

func HandleTemplate(ctx context.Context, req *mcp.CallToolRequest, input TemplateInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.Template(ctx, cfg, &helmengine.TemplateOptions{
		ReleaseName:      input.ReleaseName,
		Chart:            input.Chart,
		Version:          input.Version,
		Values:           input.Values,
		ValuesFiles:      input.ValuesFiles,
		ShowOnly:         input.ShowOnly,
		Validate:         input.Validate,
		KubeVersion:      input.KubeVersion,
		APIVersions:      input.APIVersions,
		IncludeCRDs:      input.IncludeCRDs,
		SkipCRDs:         input.SkipCRDs,
		NoHooks:          input.NoHooks,
		DependencyUpdate: input.DependencyUpdate,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
