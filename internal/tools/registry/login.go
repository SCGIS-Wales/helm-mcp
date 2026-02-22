package registry

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type LoginInput struct {
	tools.GlobalInput
	Hostname string `json:"hostname" jsonschema:"required" jsonschema_description:"Registry hostname"`
	Username string `json:"username,omitempty" jsonschema_description:"Username"`
	Password string `json:"password,omitempty" jsonschema_description:"Password"`
	Insecure bool   `json:"insecure,omitempty" jsonschema_description:"Allow insecure connections"`
	CAFile   string `json:"ca_file,omitempty" jsonschema_description:"CA bundle file"`
}

var LoginTool = &mcp.Tool{
	Name:        "helm_registry_login",
	Description: "Login to an OCI registry for chart storage.",
}

func HandleLogin(ctx context.Context, req *mcp.CallToolRequest, input LoginInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)
	defer input.ZeroSensitiveFields()

	opts := &helmengine.RegistryLoginOptions{
		Hostname: input.Hostname,
		Username: input.Username,
		Password: input.Password,
		Insecure: input.Insecure,
		CAFile:   input.CAFile,
	}

	err := engine.RegistryLogin(ctx, opts)
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult("Login successful"), nil, nil
}
