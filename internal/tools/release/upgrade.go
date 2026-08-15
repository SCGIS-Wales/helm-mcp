package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type UpgradeInput struct {
	tools.GlobalInput
	ReleaseName          string                 `json:"release_name" jsonschema:"Name of the release"`
	Chart                string                 `json:"chart" jsonschema:"Chart reference"`
	Version              string                 `json:"version,omitempty" jsonschema:"Chart version constraint"`
	Values               map[string]interface{} `json:"values,omitempty" jsonschema:"Inline values"`
	ValuesFiles          []string               `json:"values_files,omitempty" jsonschema:"Paths to values files"`
	Install              bool                   `json:"install,omitempty" jsonschema:"If release does not exist install it"`
	Force                bool                   `json:"force,omitempty" jsonschema:"Force resource updates"`
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
	ResetThenReuseValues bool                   `json:"reset_then_reuse_values,omitempty" jsonschema:"Reset then reuse values (v4 only)"`
	ServerSideApply      bool                   `json:"server_side_apply,omitempty" jsonschema:"Server-side apply (v4 only)"`
	TakeOwnership        bool                   `json:"take_ownership,omitempty" jsonschema:"Take ownership (v4 only)"`
	HideSecret           bool                   `json:"hide_secret,omitempty" jsonschema:"Hide secrets (v4 only)"`
	ForceConflicts       bool                   `json:"force_conflicts,omitempty" jsonschema:"Force conflicts (v4 only)"`
}

var UpgradeTool = &mcp.Tool{
	Name:        "helm_upgrade",
	Description: "Upgrade a Helm release to a new chart version or with new values.",
}

func HandleUpgrade(ctx context.Context, _ *mcp.CallToolRequest, input UpgradeInput) (*mcp.CallToolResult, any, error) {
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
