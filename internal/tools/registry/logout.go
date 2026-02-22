package registry

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type LogoutInput struct {
	tools.GlobalInput
	Hostname string `json:"hostname" jsonschema:"required" jsonschema_description:"Registry hostname"`
}

var LogoutTool = &mcp.Tool{
	Name:        "helm_registry_logout",
	Description: "Logout from an OCI registry.",
}

func HandleLogout(ctx context.Context, req *mcp.CallToolRequest, input LogoutInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)

	err := engine.RegistryLogout(ctx, &helmengine.RegistryLogoutOptions{
		Hostname: input.Hostname,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult("Logout successful"), nil, nil
}
