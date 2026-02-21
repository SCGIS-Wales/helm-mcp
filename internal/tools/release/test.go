package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type TestInput struct {
	tools.GlobalInput
	ReleaseName string   `json:"release_name" jsonschema:"required" jsonschema_description:"Name of the release to test"`
	Timeout     string   `json:"timeout,omitempty" jsonschema_description:"Timeout for test execution"`
	Filters     []string `json:"filters,omitempty" jsonschema_description:"Filter tests by name"`
}

var TestTool = &mcp.Tool{
	Name:        "helm_test",
	Description: "Run the test suite for a Helm release.",
}

func HandleTest(ctx context.Context, req *mcp.CallToolRequest, input TestInput) (*mcp.CallToolResult, any, error) {
	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()

	result, err := engine.Test(ctx, cfg, &helmengine.TestOptions{
		ReleaseName: input.ReleaseName,
		Timeout:     input.Timeout,
		Filters:     input.Filters,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
