package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListInput struct {
	tools.GlobalInput
	AllNamespaces bool   `json:"all_namespaces,omitempty" jsonschema_description:"List releases across all namespaces"`
	Filter        string `json:"filter,omitempty" jsonschema_description:"Regular expression filter on release name"`
	Selector      string `json:"selector,omitempty" jsonschema_description:"Label selector filter (v4 only)"`
	SortBy        string `json:"sort_by,omitempty" jsonschema_description:"Sort by: name or date"`
	SortReverse   bool   `json:"sort_reverse,omitempty" jsonschema_description:"Reverse the sort order"`
	Limit         int    `json:"limit,omitempty" jsonschema_description:"Maximum number of releases to return"`
	Offset        int    `json:"offset,omitempty" jsonschema_description:"Number of releases to skip"`
	Deployed      bool   `json:"deployed,omitempty" jsonschema_description:"Show deployed releases"`
	Failed        bool   `json:"failed,omitempty" jsonschema_description:"Show failed releases"`
	Pending       bool   `json:"pending,omitempty" jsonschema_description:"Show pending releases"`
	Uninstalled   bool   `json:"uninstalled,omitempty" jsonschema_description:"Show uninstalled releases"`
	Superseded    bool   `json:"superseded,omitempty" jsonschema_description:"Show superseded releases"`
}

var ListTool = &mcp.Tool{
	Name:        "helm_list",
	Description: "List Helm releases. Shows deployed releases by default. Use filter flags to show other statuses.",
}

func HandleList(ctx context.Context, req *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, any, error) {
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
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
