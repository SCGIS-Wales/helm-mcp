package plugin

import (
	"context"
	"strings"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/security"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type InstallInput struct {
	tools.GlobalInput
	URLOrPath string `json:"url_or_path" jsonschema:"required" jsonschema_description:"Plugin URL or local path"`
	Version   string `json:"version,omitempty" jsonschema_description:"Plugin version"`
}

var InstallTool = &mcp.Tool{
	Name:        "helm_plugin_install",
	Description: "Install a Helm plugin.",
}

func HandleInstall(ctx context.Context, req *mcp.CallToolRequest, input InstallInput) (*mcp.CallToolResult, any, error) {
	// Validate URL or path
	if strings.Contains(input.URLOrPath, "://") {
		if err := security.ValidateURL(input.URLOrPath); err != nil {
			return tools.ErrorResult(err), nil, nil
		}
	} else {
		if err := security.ValidatePath(input.URLOrPath); err != nil {
			return tools.ErrorResult(err), nil, nil
		}
	}

	engine := tools.SelectEngine(input.HelmVersion)

	err := engine.PluginInstall(ctx, &helmengine.PluginInstallOptions{
		URLOrPath: input.URLOrPath,
		Version:   input.Version,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult("Plugin installed successfully"), nil, nil
}
