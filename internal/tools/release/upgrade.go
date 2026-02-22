package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type UpgradeInput struct {
	tools.GlobalInput
	ReleaseName          string                 `json:"release_name" jsonschema:"required" jsonschema_description:"Name of the release"`
	Chart                string                 `json:"chart" jsonschema:"required" jsonschema_description:"Chart reference"`
	Version              string                 `json:"version,omitempty" jsonschema_description:"Chart version constraint"`
	Values               map[string]interface{} `json:"values,omitempty" jsonschema_description:"Inline values"`
	ValuesFiles          []string               `json:"values_files,omitempty" jsonschema_description:"Paths to values files"`
	Install              bool                   `json:"install,omitempty" jsonschema_description:"If release does not exist install it"`
	Force                bool                   `json:"force,omitempty" jsonschema_description:"Force resource updates"`
	ResetValues          bool                   `json:"reset_values,omitempty" jsonschema_description:"Reset values to chart defaults"`
	ReuseValues          bool                   `json:"reuse_values,omitempty" jsonschema_description:"Reuse last release values"`
	Wait                 bool                   `json:"wait,omitempty" jsonschema_description:"Wait for resources to be ready"`
	WaitForJobs          bool                   `json:"wait_for_jobs,omitempty" jsonschema_description:"Wait for jobs to complete"`
	Timeout              string                 `json:"timeout,omitempty" jsonschema_description:"Timeout duration"`
	DryRun               string                 `json:"dry_run,omitempty" jsonschema_description:"Dry run: none client or server"`
	Description          string                 `json:"description,omitempty" jsonschema_description:"Custom description"`
	DisableHooks         bool                   `json:"disable_hooks,omitempty" jsonschema_description:"Disable hooks"`
	SkipCRDs             bool                   `json:"skip_crds,omitempty" jsonschema_description:"Skip CRDs"`
	CleanupOnFail        bool                   `json:"cleanup_on_fail,omitempty" jsonschema_description:"Cleanup on failure"`
	DependencyUpdate     bool                   `json:"dependency_update,omitempty" jsonschema_description:"Update dependencies"`
	Labels               map[string]string      `json:"labels,omitempty" jsonschema_description:"Labels"`
	MaxHistory           int                    `json:"max_history,omitempty" jsonschema_description:"Max history revisions"`
	ResetThenReuseValues bool                   `json:"reset_then_reuse_values,omitempty" jsonschema_description:"Reset then reuse values (v4 only)"`
	ServerSideApply      bool                   `json:"server_side_apply,omitempty" jsonschema_description:"Server-side apply (v4 only)"`
	TakeOwnership        bool                   `json:"take_ownership,omitempty" jsonschema_description:"Take ownership (v4 only)"`
	HideSecret           bool                   `json:"hide_secret,omitempty" jsonschema_description:"Hide secrets (v4 only)"`
	ForceConflicts       bool                   `json:"force_conflicts,omitempty" jsonschema_description:"Force conflicts (v4 only)"`
}

var UpgradeTool = &mcp.Tool{
	Name:        "helm_upgrade",
	Description: "Upgrade a Helm release to a new chart version or with new values.",
}

func HandleUpgrade(ctx context.Context, req *mcp.CallToolRequest, input UpgradeInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateReleaseName(input.ReleaseName); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateTimeout(input.Timeout); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.Upgrade(ctx, cfg, &helmengine.UpgradeOptions{
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
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
