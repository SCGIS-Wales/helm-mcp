package chart

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PackageInput struct {
	tools.GlobalInput
	Path             string `json:"path" jsonschema:"Path to the chart directory"`
	Destination      string `json:"destination,omitempty" jsonschema:"Output directory"`
	Version          string `json:"version,omitempty" jsonschema:"Override chart version"`
	AppVersion       string `json:"app_version,omitempty" jsonschema:"Override app version"`
	Sign             bool   `json:"sign,omitempty" jsonschema:"Sign the package"`
	Key              string `json:"key,omitempty" jsonschema:"Signing key name"`
	Keyring          string `json:"keyring,omitempty" jsonschema:"Keyring path"`
	PassphraseFile   string `json:"passphrase_file,omitempty" jsonschema:"File holding the signing key passphrase"`
	DependencyUpdate bool   `json:"dependency_update,omitempty" jsonschema:"Update dependencies before packaging"`
}

var PackageTool = &mcp.Tool{
	Name:        "helm_package",
	Description: "Package a chart directory into a versioned chart archive (.tgz).",
	Annotations: tools.Mutating("Package a chart", true),
}

func HandlePackage(ctx context.Context, _ *mcp.CallToolRequest, input PackageInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if r := tools.ValidateRequired("path", input.Path); r != nil {
		return r, nil, nil
	}
	if err := tools.ValidateFilePaths(input.Path, input.Destination, input.Keyring, input.PassphraseFile); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.Package(ctx, &helmengine.PackageOptions{
		Path:             input.Path,
		Destination:      input.Destination,
		Version:          input.Version,
		AppVersion:       input.AppVersion,
		Sign:             input.Sign,
		Key:              input.Key,
		Keyring:          input.Keyring,
		PassphraseFile:   input.PassphraseFile,
		DependencyUpdate: input.DependencyUpdate,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
