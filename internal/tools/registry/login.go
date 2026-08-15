package registry

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type LoginInput struct {
	tools.GlobalInput
	Hostname string `json:"hostname" jsonschema:"Registry hostname"`
	Username string `json:"username,omitempty" jsonschema:"Username"`
	Password string `json:"password,omitempty" jsonschema:"Password"`
	Insecure bool   `json:"insecure,omitempty" jsonschema:"Allow insecure connections"`
	CAFile   string `json:"ca_file,omitempty" jsonschema:"CA bundle file"`
}

var LoginTool = &mcp.Tool{
	Name:        "helm_registry_login",
	Description: "Login to an OCI registry for chart storage.",
	Annotations: tools.Mutating("Log in to an OCI registry", true),
}

func HandleLogin(ctx context.Context, _ *mcp.CallToolRequest, input LoginInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	defer input.ZeroBearerToken()

	opts := &helmengine.RegistryLoginOptions{
		Hostname: input.Hostname,
		Username: input.Username,
		Password: input.Password,
		Insecure: input.Insecure,
		CAFile:   input.CAFile,
	}
	defer opts.ZeroPassword()

	err := engine.RegistryLogin(ctx, opts)
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult("Login successful"), nil, nil
}
