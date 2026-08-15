package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListInput struct {
	tools.GlobalInput
	AllNamespaces bool   `json:"all_namespaces,omitempty" jsonschema:"List releases across all namespaces"`
	Filter        string `json:"filter,omitempty" jsonschema:"Regular expression filter on release name"`
	Selector      string `json:"selector,omitempty" jsonschema:"Label selector filter (v4 only)"`
	SortBy        string `json:"sort_by,omitempty" jsonschema:"Sort by: name or date"`
	SortReverse   bool   `json:"sort_reverse,omitempty" jsonschema:"Reverse the sort order"`
	Limit         int    `json:"limit,omitempty" jsonschema:"Maximum number of releases to return"`
	Offset        int    `json:"offset,omitempty" jsonschema:"Number of releases to skip"`
	Deployed      bool   `json:"deployed,omitempty" jsonschema:"Show deployed releases"`
	Failed        bool   `json:"failed,omitempty" jsonschema:"Show failed releases"`
	Pending       bool   `json:"pending,omitempty" jsonschema:"Show pending releases"`
	Uninstalled   bool   `json:"uninstalled,omitempty" jsonschema:"Show uninstalled releases"`
	Superseded    bool   `json:"superseded,omitempty" jsonschema:"Show superseded releases"`
	Uninstalling  bool   `json:"uninstalling,omitempty" jsonschema:"Show releases that are currently being uninstalled"`
	All           bool   `json:"all,omitempty" jsonschema:"Show releases in every state, ignoring the individual status filters"`
	TimeFormat    string `json:"time_format,omitempty" jsonschema:"Go time layout used to format the last-updated timestamp (e.g. 2006-01-02)"`
}

var ListTool = &mcp.Tool{
	Name:        "helm_list",
	Description: "List Helm releases. Shows deployed releases by default. Use filter flags to show other statuses.",
	Annotations: tools.ReadOnly("List releases", false),
}

func HandleList(ctx context.Context, _ *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, tools.ListOutput, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), tools.ListOutput{}, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.List(ctx, cfg, &helmengine.ListOptions{
		AllNamespaces: input.AllNamespaces,
		Filter:        input.Filter,
		Selector:      input.Selector,
		SortBy:        input.SortBy,
		SortReverse:   input.SortReverse,
		Limit:         input.Limit,
		Offset:        input.Offset,
		Deployed:      input.Deployed,
		Failed:        input.Failed,
		Pending:       input.Pending,
		Uninstalled:   input.Uninstalled,
		Superseded:    input.Superseded,
		Uninstalling:  input.Uninstalling,
		All:           input.All,
		TimeFormat:    input.TimeFormat,
	})
	if err != nil {
		return tools.ErrorResult(err), tools.ListOutput{}, nil
	}

	return tools.TextResult(result), tools.ListOutput{Releases: result, Count: len(result)}, nil
}
