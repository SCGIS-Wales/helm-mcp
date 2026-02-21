package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type InstallInput struct {
	tools.GlobalInput
	ReleaseName      string                 `json:"release_name" jsonschema:"required" jsonschema_description:"Name of the release"`
	Chart            string                 `json:"chart" jsonschema:"required" jsonschema_description:"Chart reference (name or path or URL)"`
	Version          string                 `json:"version,omitempty" jsonschema_description:"Chart version constraint"`
	Values           map[string]interface{} `json:"values,omitempty" jsonschema_description:"Inline values (equivalent to --set)"`
	ValuesFiles      []string               `json:"values_files,omitempty" jsonschema_description:"Paths to values files"`
	CreateNamespace  bool                   `json:"create_namespace,omitempty" jsonschema_description:"Create namespace if not present"`
	Wait             bool                   `json:"wait,omitempty" jsonschema_description:"Wait for resources to be ready"`
	WaitForJobs      bool                   `json:"wait_for_jobs,omitempty" jsonschema_description:"Wait for jobs to complete"`
	Timeout          string                 `json:"timeout,omitempty" jsonschema_description:"Timeout duration (e.g. 5m0s)"`
	DryRun           string                 `json:"dry_run,omitempty" jsonschema_description:"Dry run strategy: none client or server"`
	Description      string                 `json:"description,omitempty" jsonschema_description:"Custom release description"`
	DisableHooks     bool                   `json:"disable_hooks,omitempty" jsonschema_description:"Disable pre/post hooks"`
	Replace          bool                   `json:"replace,omitempty" jsonschema_description:"Re-use a release name"`
	SkipCRDs         bool                   `json:"skip_crds,omitempty" jsonschema_description:"Skip CRD installation"`
	IncludeCRDs      bool                   `json:"include_crds,omitempty" jsonschema_description:"Include CRDs in rendering"`
	DependencyUpdate bool                   `json:"dependency_update,omitempty" jsonschema_description:"Update dependencies before install"`
	GenerateName     bool                   `json:"generate_name,omitempty" jsonschema_description:"Auto-generate release name"`
	NameTemplate     string                 `json:"name_template,omitempty" jsonschema_description:"Go template for name generation"`
	Labels           map[string]string      `json:"labels,omitempty" jsonschema_description:"Labels to add to release metadata"`
	// v4-specific
	ServerSideApply   bool `json:"server_side_apply,omitempty" jsonschema_description:"Use Kubernetes server-side apply (v4 only)"`
	TakeOwnership     bool `json:"take_ownership,omitempty" jsonschema_description:"Skip helm annotation checks (v4 only)"`
	RollbackOnFailure bool `json:"rollback_on_failure,omitempty" jsonschema_description:"Rollback on install failure (v4 only)"`
	HideSecret        bool `json:"hide_secret,omitempty" jsonschema_description:"Hide secrets in dry-run output (v4 only)"`
	ForceConflicts    bool `json:"force_conflicts,omitempty" jsonschema_description:"Force conflict resolution (v4 only)"`
}

var InstallTool = &mcp.Tool{
	Name:        "helm_install",
	Description: "Install a Helm chart as a new release. Supports both local charts and repository charts.",
}

func HandleInstall(ctx context.Context, req *mcp.CallToolRequest, input InstallInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()

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
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
