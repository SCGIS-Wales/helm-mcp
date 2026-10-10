package chart

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type LintInput struct {
	tools.GlobalInput
	Paths         []string               `json:"paths,omitempty" jsonschema:"Chart paths to lint (default: current directory)"`
	Values        map[string]interface{} `json:"values,omitempty" jsonschema:"Inline values, deep-merged over values_files"`
	ValuesFiles   []string               `json:"values_files,omitempty" jsonschema:"Values files"`
	Strict        bool                   `json:"strict,omitempty" jsonschema:"Treat warnings as errors"`
	WithSubcharts bool                   `json:"with_subcharts,omitempty" jsonschema:"Also lint the subcharts under each chart's charts/ directory"`
	Quiet         bool                   `json:"quiet,omitempty" jsonschema:"Only show warnings and errors"`
	KubeVersion   string                 `json:"kube_version,omitempty" jsonschema:"Kubernetes version to lint against (e.g. 1.34.0)"`
	// SkipSchemaValidation mirrors helm lint --skip-schema-validation.
	SkipSchemaValidation bool `json:"skip_schema_validation,omitempty" jsonschema:"Skip validation of values against the chart's values.schema.json"`
}

var LintTool = &mcp.Tool{
	Name:        "helm_lint",
	Description: "Lint a Helm chart for possible issues and best practices.",
	Annotations: tools.ReadOnly("Lint a chart", false),
}

func HandleLint(ctx context.Context, _ *mcp.CallToolRequest, input LintInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateValuesFiles(input.ValuesFiles); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateFilePaths(input.Paths...); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.Lint(ctx, &helmengine.LintOptions{
		Paths:         input.Paths,
		Values:        input.Values,
		ValuesFiles:   input.ValuesFiles,
		Strict:        input.Strict,
		WithSubcharts: input.WithSubcharts,
		Quiet:         input.Quiet,
		Namespace:     input.Namespace,
		KubeVersion:   input.KubeVersion,

		SkipSchemaValidation: input.SkipSchemaValidation,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
