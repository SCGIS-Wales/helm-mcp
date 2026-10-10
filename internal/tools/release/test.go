package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type TestInput struct {
	tools.GlobalInput
	ReleaseName string   `json:"release_name" jsonschema:"Name of the release to test"`
	Timeout     string   `json:"timeout,omitempty" jsonschema:"Timeout for test execution"`
	Filters     []string `json:"filters,omitempty" jsonschema:"Filter tests by name"`
	Logs        bool     `json:"logs,omitempty" jsonschema:"Include the test pod logs in the result (helm test --logs)"`
}

var TestTool = &mcp.Tool{
	Name:        "helm_test",
	Description: "Run the test suite for a Helm release.",
	Annotations: tools.Mutating("Run release tests", false),
}

func HandleTest(ctx context.Context, req *mcp.CallToolRequest, input TestInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateReleaseName(input.ReleaseName); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateTimeout(input.Timeout); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	opts := &helmengine.TestOptions{
		ReleaseName: input.ReleaseName,
		Timeout:     input.Timeout,
		Filters:     input.Filters,
		Logs:        input.Logs,
	}
	result, err := tools.WithProgress(ctx, req, "helm test", func() (*helmengine.ReleaseInfo, error) {
		return engine.Test(ctx, cfg, opts)
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
