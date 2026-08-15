package plugin

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type VerifyInput struct {
	tools.GlobalInput
	PluginPath string `json:"plugin_path" jsonschema:"Path to the packaged plugin archive to verify"`
	Keyring    string `json:"keyring,omitempty" jsonschema:"Path to the keyring containing the trusted public keys"`
}

var VerifyTool = &mcp.Tool{
	Name:        "helm_plugin_verify",
	Description: "Verify that a packaged Helm plugin has been signed and that its provenance is valid. Requires helm_version v4.",
	Annotations: tools.ReadOnly("Verify plugin signature", false),
}

func HandleVerify(ctx context.Context, _ *mcp.CallToolRequest, input VerifyInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if result := tools.ValidateRequired("plugin_path", input.PluginPath); result != nil {
		return result, nil, nil
	}
	if err := tools.ValidatePluginPath(input.PluginPath); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)

	out, err := engine.PluginVerify(ctx, &helmengine.PluginVerifyOptions{
		PluginPath: input.PluginPath,
		Keyring:    input.Keyring,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(out), nil, nil
}
