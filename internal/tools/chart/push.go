package chart

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PushInput struct {
	tools.GlobalInput
	ChartRef  string `json:"chart_ref" jsonschema:"required" jsonschema_description:"Chart archive to push"`
	Remote    string `json:"remote" jsonschema:"required" jsonschema_description:"OCI registry URL"`
	PlainHTTP bool   `json:"plain_http,omitempty" jsonschema_description:"Use plain HTTP"`
	CAFile    string `json:"ca_file,omitempty" jsonschema_description:"CA bundle file"`
	CertFile  string `json:"cert_file,omitempty" jsonschema_description:"TLS client certificate"`
	KeyFile   string `json:"key_file,omitempty" jsonschema_description:"TLS client key"`
}

var PushTool = &mcp.Tool{
	Name:        "helm_push",
	Description: "Push a chart archive to an OCI registry.",
}

func HandlePush(ctx context.Context, req *mcp.CallToolRequest, input PushInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.Push(ctx, cfg, &helmengine.PushOptions{
		ChartRef:  input.ChartRef,
		Remote:    input.Remote,
		PlainHTTP: input.PlainHTTP,
		CAFile:    input.CAFile,
		CertFile:  input.CertFile,
		KeyFile:   input.KeyFile,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	if result == "" {
		result = "Chart pushed successfully"
	}
	return tools.TextResult(result), nil, nil
}
