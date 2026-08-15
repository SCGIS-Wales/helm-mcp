package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type InstallInput struct {
	tools.GlobalInput
	ReleaseName      string                 `json:"release_name" jsonschema:"Name of the release"`
	Chart            string                 `json:"chart" jsonschema:"Chart reference (name or path or URL)"`
	Version          string                 `json:"version,omitempty" jsonschema:"Chart version constraint"`
	Values           map[string]interface{} `json:"values,omitempty" jsonschema:"Inline values (equivalent to --set)"`
	ValuesFiles      []string               `json:"values_files,omitempty" jsonschema:"Paths to values files"`
	CreateNamespace  bool                   `json:"create_namespace,omitempty" jsonschema:"Create namespace if not present"`
	Wait             bool                   `json:"wait,omitempty" jsonschema:"Wait for resources to be ready"`
	WaitForJobs      bool                   `json:"wait_for_jobs,omitempty" jsonschema:"Wait for jobs to complete"`
	Timeout          string                 `json:"timeout,omitempty" jsonschema:"Timeout duration (e.g. 5m0s)"`
	DryRun           string                 `json:"dry_run,omitempty" jsonschema:"Dry run strategy: none client or server"`
	Description      string                 `json:"description,omitempty" jsonschema:"Custom release description"`
	DisableHooks     bool                   `json:"disable_hooks,omitempty" jsonschema:"Disable pre/post hooks"`
	Replace          bool                   `json:"replace,omitempty" jsonschema:"Re-use a release name"`
	SkipCRDs         bool                   `json:"skip_crds,omitempty" jsonschema:"Skip CRD installation"`
	IncludeCRDs      bool                   `json:"include_crds,omitempty" jsonschema:"Include CRDs in rendering"`
	DependencyUpdate bool                   `json:"dependency_update,omitempty" jsonschema:"Update dependencies before install"`
	GenerateName     bool                   `json:"generate_name,omitempty" jsonschema:"Auto-generate release name"`
	NameTemplate     string                 `json:"name_template,omitempty" jsonschema:"Go template for name generation"`
	Labels           map[string]string      `json:"labels,omitempty" jsonschema:"Labels to add to release metadata"`
	// Supported by both v3 (as --atomic) and v4 (as --rollback-on-failure).
	RollbackOnFailure        bool   `json:"rollback_on_failure,omitempty" jsonschema:"Roll the release back automatically if the install fails (helm --atomic)"`
	Devel                    bool   `json:"devel,omitempty" jsonschema:"Allow development chart versions (alpha/beta/rc) to satisfy the version constraint"`
	SubNotes                 bool   `json:"sub_notes,omitempty" jsonschema:"Also render the NOTES.txt of subcharts"`
	HideNotes                bool   `json:"hide_notes,omitempty" jsonschema:"Omit NOTES.txt from the output"`
	SkipSchemaValidation     bool   `json:"skip_schema_validation,omitempty" jsonschema:"Skip validation of values against the chart's values.schema.json"`
	DisableOpenAPIValidation bool   `json:"disable_openapi_validation,omitempty" jsonschema:"Skip validating rendered manifests against the Kubernetes OpenAPI schema"`
	EnableDNS                bool   `json:"enable_dns,omitempty" jsonschema:"Allow DNS lookups from templates via the getHostByName function"`
	OutputDir                string `json:"output_dir,omitempty" jsonschema:"Write rendered manifests to this directory instead of only applying them"`
	UseReleaseName           bool   `json:"use_release_name,omitempty" jsonschema:"Prefix files written to output_dir with the release name"`
	// v4-specific
	ServerSideApply bool   `json:"server_side_apply,omitempty" jsonschema:"Use Kubernetes server-side apply (v4 only)"`
	TakeOwnership   bool   `json:"take_ownership,omitempty" jsonschema:"Skip helm annotation checks (v4 only)"`
	HideSecret      bool   `json:"hide_secret,omitempty" jsonschema:"Hide secrets in dry-run output (v4 only)"`
	ForceConflicts  bool   `json:"force_conflicts,omitempty" jsonschema:"Force conflict resolution (v4 only)"`
	WaitStrategy    string `json:"wait_strategy,omitempty" jsonschema:"How to wait for resources: watcher (kstatus), legacy, or hookOnly. Overrides wait. v4 only"`
}

var InstallTool = &mcp.Tool{
	Name:        "helm_install",
	Description: "Install a Helm chart as a new release. Supports both local charts and repository charts.",
}

func HandleInstall(ctx context.Context, _ *mcp.CallToolRequest, input InstallInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if !input.GenerateName {
		if err := tools.ValidateReleaseName(input.ReleaseName); err != nil {
			return tools.ErrorResult(err), nil, nil
		}
	}
	if err := tools.ValidateTimeout(input.Timeout); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.Install(ctx, cfg, &helmengine.InstallOptions{
		ReleaseName:       input.ReleaseName,
		Chart:             input.Chart,
		Version:           input.Version,
		Values:            input.Values,
		ValuesFiles:       input.ValuesFiles,
		CreateNamespace:   input.CreateNamespace,
		Wait:              input.Wait,
		WaitForJobs:       input.WaitForJobs,
		Timeout:           input.Timeout,
		DryRun:            input.DryRun,
		Description:       input.Description,
		DisableHooks:      input.DisableHooks,
		Replace:           input.Replace,
		SkipCRDs:          input.SkipCRDs,
		IncludeCRDs:       input.IncludeCRDs,
		DependencyUpdate:  input.DependencyUpdate,
		GenerateName:      input.GenerateName,
		NameTemplate:      input.NameTemplate,
		Labels:            input.Labels,
		ServerSideApply:   input.ServerSideApply,
		TakeOwnership:     input.TakeOwnership,
		RollbackOnFailure: input.RollbackOnFailure,
		HideSecret:        input.HideSecret,
		ForceConflicts:    input.ForceConflicts,

		Devel:                    input.Devel,
		SubNotes:                 input.SubNotes,
		HideNotes:                input.HideNotes,
		SkipSchemaValidation:     input.SkipSchemaValidation,
		DisableOpenAPIValidation: input.DisableOpenAPIValidation,
		EnableDNS:                input.EnableDNS,
		OutputDir:                input.OutputDir,
		UseReleaseName:           input.UseReleaseName,
		WaitStrategy:             input.WaitStrategy,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
