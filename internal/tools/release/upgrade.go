package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type UpgradeInput struct {
	tools.GlobalInput
	tools.ChartSourceInput
	tools.ValueSetInput
	tools.PostRenderInput
	ReleaseName          string                 `json:"release_name" jsonschema:"Name of the release"`
	Chart                string                 `json:"chart" jsonschema:"Chart reference"`
	Version              string                 `json:"version,omitempty" jsonschema:"Chart version constraint"`
	Values               map[string]interface{} `json:"values,omitempty" jsonschema:"Inline values, deep-merged over values_files (equivalent to --set)"`
	ValuesFiles          []string               `json:"values_files,omitempty" jsonschema:"Paths to values files"`
	Install              bool                   `json:"install,omitempty" jsonschema:"If release does not exist install it"`
	Force                bool                   `json:"force,omitempty" jsonschema:"Replace resources by delete and recreate when an update is needed (helm --force-replace; --force on v3)"`
	ResetValues          bool                   `json:"reset_values,omitempty" jsonschema:"Reset values to chart defaults"`
	ReuseValues          bool                   `json:"reuse_values,omitempty" jsonschema:"Reuse last release values"`
	Wait                 bool                   `json:"wait,omitempty" jsonschema:"Wait for resources to be ready"`
	WaitForJobs          bool                   `json:"wait_for_jobs,omitempty" jsonschema:"Wait for jobs to complete"`
	Timeout              string                 `json:"timeout,omitempty" jsonschema:"Timeout duration"`
	DryRun               string                 `json:"dry_run,omitempty" jsonschema:"Dry run: none client or server"`
	Description          string                 `json:"description,omitempty" jsonschema:"Custom description"`
	DisableHooks         bool                   `json:"disable_hooks,omitempty" jsonschema:"Disable hooks"`
	SkipCRDs             bool                   `json:"skip_crds,omitempty" jsonschema:"Skip CRDs"`
	CleanupOnFail        bool                   `json:"cleanup_on_fail,omitempty" jsonschema:"Cleanup on failure"`
	DependencyUpdate     bool                   `json:"dependency_update,omitempty" jsonschema:"Update dependencies"`
	Labels               map[string]string      `json:"labels,omitempty" jsonschema:"Labels"`
	MaxHistory           int                    `json:"max_history,omitempty" jsonschema:"Max history revisions"`
	ResetThenReuseValues bool                   `json:"reset_then_reuse_values,omitempty" jsonschema:"Reset to the new chart's defaults, then reapply the last release's user values"`
	TakeOwnership        bool                   `json:"take_ownership,omitempty" jsonschema:"Adopt existing resources that lack Helm ownership annotations"`
	HideSecret           bool                   `json:"hide_secret,omitempty" jsonschema:"Hide Secret contents in dry-run output; requires dry_run"`
	ServerSideApply      *bool                  `json:"server_side_apply,omitempty" jsonschema:"Apply with Kubernetes server-side apply. Unset reuses the previous release's apply method (Helm's auto). v4 only"`
	ForceConflicts       bool                   `json:"force_conflicts,omitempty" jsonschema:"Take ownership of fields owned by other managers during server-side apply (v4 only)"`
	// Supported by both v3 (as --atomic) and v4 (as --rollback-on-failure).
	RollbackOnFailure        bool   `json:"rollback_on_failure,omitempty" jsonschema:"Roll the release back automatically if the upgrade fails (helm --atomic)"`
	Devel                    bool   `json:"devel,omitempty" jsonschema:"Allow development chart versions (alpha/beta/rc) to satisfy the version constraint"`
	SubNotes                 bool   `json:"sub_notes,omitempty" jsonschema:"Also render the NOTES.txt of subcharts"`
	HideNotes                bool   `json:"hide_notes,omitempty" jsonschema:"Omit NOTES.txt from the output"`
	SkipSchemaValidation     bool   `json:"skip_schema_validation,omitempty" jsonschema:"Skip validation of values against the chart's values.schema.json"`
	DisableOpenAPIValidation bool   `json:"disable_openapi_validation,omitempty" jsonschema:"Skip validating rendered manifests against the Kubernetes OpenAPI schema"`
	EnableDNS                bool   `json:"enable_dns,omitempty" jsonschema:"Allow DNS lookups from templates via the getHostByName function"`
	WaitStrategy             string `json:"wait_strategy,omitempty" jsonschema:"How to wait for resources: watcher (kstatus), legacy, or hookOnly. Overrides wait. v4 only"`
}

var UpgradeTool = &mcp.Tool{
	Name:        "helm_upgrade",
	Description: "Upgrade a Helm release to a new chart version or with new values.",
	Annotations: tools.Destructive("Upgrade a release"),
}

func HandleUpgrade(ctx context.Context, req *mcp.CallToolRequest, input UpgradeInput) (*mcp.CallToolResult, any, error) {
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
	if err := tools.ValidateTimeout(input.Timeout); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateReleaseInputs(&input.ChartSourceInput, &input.ValueSetInput, &input.PostRenderInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	opts := &helmengine.UpgradeOptions{
		ChartSourceOptions:   input.ChartSourceInput.ToOptions(),
		ValueSetOptions:      input.ValueSetInput.ToOptions(),
		PostRenderOptions:    input.PostRenderInput.ToOptions(),
		ReleaseName:          input.ReleaseName,
		Chart:                input.Chart,
		Version:              input.Version,
		Values:               input.Values,
		ValuesFiles:          input.ValuesFiles,
		Install:              input.Install,
		Force:                input.Force,
		ResetValues:          input.ResetValues,
		ReuseValues:          input.ReuseValues,
		Wait:                 input.Wait,
		WaitForJobs:          input.WaitForJobs,
		Timeout:              input.Timeout,
		DryRun:               input.DryRun,
		Description:          input.Description,
		DisableHooks:         input.DisableHooks,
		SkipCRDs:             input.SkipCRDs,
		CleanupOnFail:        input.CleanupOnFail,
		DependencyUpdate:     input.DependencyUpdate,
		Labels:               input.Labels,
		MaxHistory:           input.MaxHistory,
		ResetThenReuseValues: input.ResetThenReuseValues,
		ServerSideApply:      input.ServerSideApply,
		TakeOwnership:        input.TakeOwnership,
		HideSecret:           input.HideSecret,
		ForceConflicts:       input.ForceConflicts,

		RollbackOnFailure:        input.RollbackOnFailure,
		Devel:                    input.Devel,
		SubNotes:                 input.SubNotes,
		HideNotes:                input.HideNotes,
		SkipSchemaValidation:     input.SkipSchemaValidation,
		DisableOpenAPIValidation: input.DisableOpenAPIValidation,
		EnableDNS:                input.EnableDNS,
		WaitStrategy:             input.WaitStrategy,
	}
	defer opts.ZeroPassword()

	result, err := tools.WithProgress(ctx, req, "helm upgrade", func() (*helmengine.ReleaseInfo, error) {
		return engine.Upgrade(ctx, cfg, opts)
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
