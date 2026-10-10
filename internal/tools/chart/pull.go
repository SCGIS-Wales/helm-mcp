package chart

import (
	"context"
	"strings"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/security"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PullInput struct {
	tools.GlobalInput
	tools.ChartSourceInput
	Chart       string `json:"chart" jsonschema:"Chart reference to pull"`
	Version     string `json:"version,omitempty" jsonschema:"Chart version"`
	Destination string `json:"destination,omitempty" jsonschema:"Output directory"`
	Untar       bool   `json:"untar,omitempty" jsonschema:"Untar the chart after download"`
	UntarDir    string `json:"untar_dir,omitempty" jsonschema:"Directory to untar into"`
	Prov        bool   `json:"prov,omitempty" jsonschema:"Also download the provenance (.prov) file without verifying it (helm pull --prov)"`
}

var PullTool = &mcp.Tool{
	Name:        "helm_pull",
	Description: "Download a chart from a repository or OCI registry.",
	Annotations: tools.Mutating("Download a chart", true),
}

func HandlePull(ctx context.Context, _ *mcp.CallToolRequest, input PullInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateFilePaths(input.Destination, input.UntarDir); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := input.Validate(); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	// Validate URL-like chart references for SSRF protection.
	if strings.Contains(input.Chart, "://") {
		if err := security.ValidateURL(input.Chart); err != nil {
			return tools.ErrorResult(err), nil, nil
		}
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()
	defer input.ZeroBearerToken()

	opts := &helmengine.PullOptions{
		ChartSourceOptions: input.ToOptions(),
		Chart:              input.Chart,
		Version:            input.Version,
		Destination:        input.Destination,
		Untar:              input.Untar,
		UntarDir:           input.UntarDir,
		Prov:               input.Prov,
	}
	defer opts.ZeroPassword()

	result, err := engine.Pull(ctx, cfg, opts)
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	if result == "" {
		result = "Chart pulled successfully"
	}
	return tools.TextResult(result), nil, nil
}
