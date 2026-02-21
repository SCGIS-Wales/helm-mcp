package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type RollbackInput struct {
	tools.GlobalInput
	ReleaseName     string `json:"release_name" jsonschema:"required" jsonschema_description:"Name of the release"`
	Revision        int    `json:"revision" jsonschema:"required" jsonschema_description:"Revision number to rollback to"`
	Wait            bool   `json:"wait,omitempty" jsonschema_description:"Wait for resources to be ready"`
	WaitForJobs     bool   `json:"wait_for_jobs,omitempty" jsonschema_description:"Wait for jobs to complete"`
	Timeout         string `json:"timeout,omitempty" jsonschema_description:"Timeout duration"`
	Force           bool   `json:"force,omitempty" jsonschema_description:"Force resource updates"`
	DryRun          bool   `json:"dry_run,omitempty" jsonschema_description:"Simulate a rollback"`
	DisableHooks    bool   `json:"disable_hooks,omitempty" jsonschema_description:"Disable hooks"`
	CleanupOnFail   bool   `json:"cleanup_on_fail,omitempty" jsonschema_description:"Cleanup on failure"`
	MaxHistory      int    `json:"max_history,omitempty" jsonschema_description:"Max history revisions"`
	ServerSideApply bool   `json:"server_side_apply,omitempty" jsonschema_description:"Server-side apply (v4 only)"`
	ForceConflicts  bool   `json:"force_conflicts,omitempty" jsonschema_description:"Force conflicts (v4 only)"`
}

var RollbackTool = &mcp.Tool{
	Name:        "helm_rollback",
	Description: "Rollback a Helm release to a previous revision.",
}

func HandleRollback(ctx context.Context, req *mcp.CallToolRequest, input RollbackInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()

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
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult("Rollback successful"), nil, nil
}
