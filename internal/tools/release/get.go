package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Get All ---

type GetAllInput struct {
	tools.GlobalInput
	ReleaseName string `json:"release_name" jsonschema:"required" jsonschema_description:"Name of the release"`
	Revision    int    `json:"revision,omitempty" jsonschema_description:"Release revision number"`
}

var GetAllTool = &mcp.Tool{
	Name:        "helm_get_all",
	Description: "Get all information (values, manifest, hooks, notes) for a release.",
}

func HandleGetAll(ctx context.Context, req *mcp.CallToolRequest, input GetAllInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateReleaseName(input.ReleaseName); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.GetAll(ctx, cfg, &helmengine.GetOptions{
		ReleaseName: input.ReleaseName,
		Revision:    input.Revision,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}

// --- Get Hooks ---

type GetHooksInput struct {
	tools.GlobalInput
	ReleaseName string `json:"release_name" jsonschema:"required" jsonschema_description:"Name of the release"`
	Revision    int    `json:"revision,omitempty" jsonschema_description:"Release revision number"`
}

var GetHooksTool = &mcp.Tool{
	Name:        "helm_get_hooks",
	Description: "Get all hooks for a release.",
}

func HandleGetHooks(ctx context.Context, req *mcp.CallToolRequest, input GetHooksInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateReleaseName(input.ReleaseName); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.GetHooks(ctx, cfg, &helmengine.GetOptions{
		ReleaseName: input.ReleaseName,
		Revision:    input.Revision,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}

// --- Get Manifest ---

type GetManifestInput struct {
	tools.GlobalInput
	ReleaseName string `json:"release_name" jsonschema:"required" jsonschema_description:"Name of the release"`
	Revision    int    `json:"revision,omitempty" jsonschema_description:"Release revision number"`
}

var GetManifestTool = &mcp.Tool{
	Name:        "helm_get_manifest",
	Description: "Get the Kubernetes manifest for a release.",
}

func HandleGetManifest(ctx context.Context, req *mcp.CallToolRequest, input GetManifestInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateReleaseName(input.ReleaseName); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.GetManifest(ctx, cfg, &helmengine.GetOptions{
		ReleaseName: input.ReleaseName,
		Revision:    input.Revision,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}

// --- Get Metadata ---

type GetMetadataInput struct {
	tools.GlobalInput
	ReleaseName string `json:"release_name" jsonschema:"required" jsonschema_description:"Name of the release"`
	Revision    int    `json:"revision,omitempty" jsonschema_description:"Release revision number"`
}

var GetMetadataTool = &mcp.Tool{
	Name:        "helm_get_metadata",
	Description: "Get metadata for a release.",
}

func HandleGetMetadata(ctx context.Context, req *mcp.CallToolRequest, input GetMetadataInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateReleaseName(input.ReleaseName); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.GetMetadata(ctx, cfg, &helmengine.GetOptions{
		ReleaseName: input.ReleaseName,
		Revision:    input.Revision,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}

// --- Get Notes ---

type GetNotesInput struct {
	tools.GlobalInput
	ReleaseName string `json:"release_name" jsonschema:"required" jsonschema_description:"Name of the release"`
	Revision    int    `json:"revision,omitempty" jsonschema_description:"Release revision number"`
}

var GetNotesTool = &mcp.Tool{
	Name:        "helm_get_notes",
	Description: "Get the notes for a release.",
}

func HandleGetNotes(ctx context.Context, req *mcp.CallToolRequest, input GetNotesInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateReleaseName(input.ReleaseName); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.GetNotes(ctx, cfg, &helmengine.GetOptions{
		ReleaseName: input.ReleaseName,
		Revision:    input.Revision,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}

// --- Get Values ---

type GetValuesInput struct {
	tools.GlobalInput
	ReleaseName string `json:"release_name" jsonschema:"required" jsonschema_description:"Name of the release"`
	Revision    int    `json:"revision,omitempty" jsonschema_description:"Release revision number"`
	All         bool   `json:"all,omitempty" jsonschema_description:"Include computed values"`
}

var GetValuesTool = &mcp.Tool{
	Name:        "helm_get_values",
	Description: "Get the values for a release. Use all=true to include computed values.",
}

func HandleGetValues(ctx context.Context, req *mcp.CallToolRequest, input GetValuesInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateReleaseName(input.ReleaseName); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.GetValues(ctx, cfg, &helmengine.GetValuesOptions{
		ReleaseName: input.ReleaseName,
		Revision:    input.Revision,
		All:         input.All,
	})
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult(result), nil, nil
}
