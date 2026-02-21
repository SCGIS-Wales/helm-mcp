package chart

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PackageInput struct {
	tools.GlobalInput
	Path             string `json:"path" jsonschema:"required" jsonschema_description:"Path to the chart directory"`
	Destination      string `json:"destination,omitempty" jsonschema_description:"Output directory"`
	Version          string `json:"version,omitempty" jsonschema_description:"Override chart version"`
	AppVersion       string `json:"app_version,omitempty" jsonschema_description:"Override app version"`
	Sign             bool   `json:"sign,omitempty" jsonschema_description:"Sign the package"`
	Key              string `json:"key,omitempty" jsonschema_description:"Signing key name"`
	Keyring          string `json:"keyring,omitempty" jsonschema_description:"Keyring path"`
	DependencyUpdate bool   `json:"dependency_update,omitempty" jsonschema_description:"Update dependencies before packaging"`
}

var PackageTool = &mcp.Tool{
	Name:        "helm_package",
	Description: "Package a chart directory into a versioned chart archive (.tgz).",
}

func HandlePackage(ctx context.Context, req *mcp.CallToolRequest, input PackageInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)

	result, err := engine.Package(ctx, &helmengine.PackageOptions{
		Path:             input.Path,
		Destination:      input.Destination,
		Version:          input.Version,
		AppVersion:       input.AppVersion,
		Sign:             input.Sign,
		Key:              input.Key,
		Keyring:          input.Keyring,
		DependencyUpdate: input.DependencyUpdate,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
