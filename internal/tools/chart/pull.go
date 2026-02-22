package chart

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PullInput struct {
	tools.GlobalInput
	Chart       string `json:"chart" jsonschema:"required" jsonschema_description:"Chart reference to pull"`
	Version     string `json:"version,omitempty" jsonschema_description:"Chart version"`
	Repo        string `json:"repo,omitempty" jsonschema_description:"Repository URL"`
	Destination string `json:"destination,omitempty" jsonschema_description:"Output directory"`
	Untar       bool   `json:"untar,omitempty" jsonschema_description:"Untar the chart after download"`
	UntarDir    string `json:"untar_dir,omitempty" jsonschema_description:"Directory to untar into"`
	Verify      bool   `json:"verify,omitempty" jsonschema_description:"Verify the chart"`
	Keyring     string `json:"keyring,omitempty" jsonschema_description:"Keyring path"`
	Username    string `json:"username,omitempty" jsonschema_description:"Repository username"`
	Password    string `json:"password,omitempty" jsonschema_description:"Repository password"`
}

var PullTool = &mcp.Tool{
	Name:        "helm_pull",
	Description: "Download a chart from a repository or OCI registry.",
}

func HandlePull(ctx context.Context, req *mcp.CallToolRequest, input PullInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()
	defer input.ZeroSensitiveFields()

	result, err := engine.Pull(ctx, cfg, &helmengine.PullOptions{
		Chart:       input.Chart,
		Version:     input.Version,
		Repo:        input.Repo,
		Destination: input.Destination,
		Untar:       input.Untar,
		UntarDir:    input.UntarDir,
		Verify:      input.Verify,
		Keyring:     input.Keyring,
		Username:    input.Username,
		Password:    input.Password,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	if result == "" {
		result = "Chart pulled successfully"
	}
	return tools.TextResult(result), nil, nil
}
