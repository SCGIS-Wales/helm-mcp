package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type RollbackInput struct {
	tools.GlobalInput
	ReleaseName     string `json:"release_name" jsonschema:"Name of the release"`
	Revision        int    `json:"revision" jsonschema:"Revision number to rollback to"`
	Wait            bool   `json:"wait,omitempty" jsonschema:"Wait for resources to be ready"`
	WaitForJobs     bool   `json:"wait_for_jobs,omitempty" jsonschema:"Wait for jobs to complete"`
	Timeout         string `json:"timeout,omitempty" jsonschema:"Timeout duration"`
	Force           bool   `json:"force,omitempty" jsonschema:"Force resource updates"`
	DryRun          bool   `json:"dry_run,omitempty" jsonschema:"Simulate a rollback"`
	DisableHooks    bool   `json:"disable_hooks,omitempty" jsonschema:"Disable hooks"`
	CleanupOnFail   bool   `json:"cleanup_on_fail,omitempty" jsonschema:"Cleanup on failure"`
	MaxHistory      int    `json:"max_history,omitempty" jsonschema:"Max history revisions"`
	ServerSideApply bool   `json:"server_side_apply,omitempty" jsonschema:"Server-side apply (v4 only)"`
	ForceConflicts  bool   `json:"force_conflicts,omitempty" jsonschema:"Force conflicts (v4 only)"`
	WaitStrategy    string `json:"wait_strategy,omitempty" jsonschema:"How to wait for resources: watcher (kstatus), legacy, or hookOnly. Overrides wait. v4 only"`
}

var RollbackTool = &mcp.Tool{
	Name:        "helm_rollback",
	Description: "Rollback a Helm release to a previous revision.",
}

func HandleRollback(ctx context.Context, _ *mcp.CallToolRequest, input RollbackInput) (*mcp.CallToolResult, any, error) {
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

	err := engine.Rollback(ctx, cfg, &helmengine.RollbackOptions{
		ReleaseName:     input.ReleaseName,
		Revision:        input.Revision,
		Wait:            input.Wait,
		WaitForJobs:     input.WaitForJobs,
		Timeout:         input.Timeout,
		Force:           input.Force,
		DryRun:          input.DryRun,
		DisableHooks:    input.DisableHooks,
		CleanupOnFail:   input.CleanupOnFail,
		MaxHistory:      input.MaxHistory,
		ServerSideApply: input.ServerSideApply,
		ForceConflicts:  input.ForceConflicts,
		WaitStrategy:    input.WaitStrategy,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult("Rollback successful"), nil, nil
}
