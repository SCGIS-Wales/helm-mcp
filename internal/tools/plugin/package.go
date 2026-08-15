package plugin

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PackageInput struct {
	tools.GlobalInput
	PluginPath     string `json:"plugin_path" jsonschema:"Path to the plugin directory to package"`
	Destination    string `json:"destination,omitempty" jsonschema:"Directory to write the plugin tarball to (default: current directory)"`
	NoSign         bool   `json:"no_sign,omitempty" jsonschema:"Skip PGP signing. Signing is on by default and is recommended for any plugin you intend to distribute"`
	Key            string `json:"key,omitempty" jsonschema:"Name of the PGP key to sign with"`
	Keyring        string `json:"keyring,omitempty" jsonschema:"Path to the keyring containing the signing key"`
	PassphraseFile string `json:"passphrase_file,omitempty" jsonschema:"Path to a file holding the signing key passphrase"`
}

var PackageTool = &mcp.Tool{
	Name:        "helm_plugin_package",
	Description: "Package a Helm plugin directory into a distributable archive, signing it with a PGP key by default. Requires helm_version v4.",
	Annotations: tools.Mutating("Package a plugin", true),
}

func HandlePackage(ctx context.Context, _ *mcp.CallToolRequest, input PackageInput) (*mcp.CallToolResult, any, error) {
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

	out, err := engine.PluginPackage(ctx, &helmengine.PluginPackageOptions{
		PluginPath:     input.PluginPath,
		Destination:    input.Destination,
		Sign:           !input.NoSign,
		Key:            input.Key,
		Keyring:        input.Keyring,
		PassphraseFile: input.PassphraseFile,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(out), nil, nil
}
