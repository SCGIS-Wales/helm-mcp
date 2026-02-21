package repo

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type AddInput struct {
	tools.GlobalInput
	Name        string `json:"name" jsonschema:"required" jsonschema_description:"Repository name"`
	URL         string `json:"url" jsonschema:"required" jsonschema_description:"Repository URL"`
	Username    string `json:"username,omitempty" jsonschema_description:"Repository username"`
	Password    string `json:"password,omitempty" jsonschema_description:"Repository password"`
	ForceUpdate bool   `json:"force_update,omitempty" jsonschema_description:"Replace existing entry"`
	CAFile      string `json:"ca_file,omitempty" jsonschema_description:"CA bundle file"`
	InsecureSkipTLS bool `json:"insecure_skip_tls,omitempty" jsonschema_description:"Skip TLS verification"`
}

var AddTool = &mcp.Tool{
	Name:        "helm_repo_add",
	Description: "Add a chart repository.",
}

func HandleAdd(ctx context.Context, req *mcp.CallToolRequest, input AddInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)

	err := engine.RepoAdd(ctx, &helmengine.RepoAddOptions{
		Name:                  input.Name,
		URL:                   input.URL,
		Username:              input.Username,
		Password:              input.Password,
		ForceUpdate:           input.ForceUpdate,
		CAFile:                input.CAFile,
		InsecureSkipTLSVerify: input.InsecureSkipTLS,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult("Repository added successfully"), nil, nil
}
