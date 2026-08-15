package release

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/resilience"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Get All ---

type GetAllInput struct {
	tools.GlobalInput
	ReleaseName string `json:"release_name" jsonschema:"Name of the release"`
	Revision    int    `json:"revision,omitempty" jsonschema:"Release revision number"`
}

var GetAllTool = &mcp.Tool{
	Name:        "helm_get_all",
	Description: "Get all information (values, manifest, hooks, notes) for a release.",
	Annotations: tools.ReadOnly("Get all release data", false),
}

func HandleGetAll(ctx context.Context, _ *mcp.CallToolRequest, input GetAllInput) (*mcp.CallToolResult, any, error) {
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

	// Strip noisy Kubernetes fields from manifest and hooks to reduce
	// payload size for LLM consumption.
	if result != nil {
		result.Manifest = resilience.SanitizeManifest(result.Manifest)
		result.Hooks = resilience.SanitizeManifest(result.Hooks)
	}

	return tools.TextResult(result), nil, nil
}

// --- Get Hooks ---

type GetHooksInput struct {
	tools.GlobalInput
	ReleaseName string `json:"release_name" jsonschema:"Name of the release"`
	Revision    int    `json:"revision,omitempty" jsonschema:"Release revision number"`
}

var GetHooksTool = &mcp.Tool{
	Name:        "helm_get_hooks",
	Description: "Get all hooks for a release.",
	Annotations: tools.ReadOnly("Get release hooks", false),
}

func HandleGetHooks(ctx context.Context, _ *mcp.CallToolRequest, input GetHooksInput) (*mcp.CallToolResult, any, error) {
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

	return tools.TextResult(resilience.SanitizeManifest(result)), nil, nil
}

// --- Get Manifest ---

type GetManifestInput struct {
	tools.GlobalInput
	ReleaseName string `json:"release_name" jsonschema:"Name of the release"`
	Revision    int    `json:"revision,omitempty" jsonschema:"Release revision number"`
}

var GetManifestTool = &mcp.Tool{
	Name:        "helm_get_manifest",
	Description: "Get the Kubernetes manifest for a release.",
	Annotations: tools.ReadOnly("Get release manifest", false),
}

func HandleGetManifest(ctx context.Context, _ *mcp.CallToolRequest, input GetManifestInput) (*mcp.CallToolResult, any, error) {
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

	// Strip noisy Kubernetes fields (managedFields, last-applied-configuration)
	// to reduce payload size for LLM consumption.
	return tools.TextResult(resilience.SanitizeManifest(result)), nil, nil
}

// --- Get Metadata ---

type GetMetadataInput struct {
	tools.GlobalInput
	ReleaseName string `json:"release_name" jsonschema:"Name of the release"`
	Revision    int    `json:"revision,omitempty" jsonschema:"Release revision number"`
}

var GetMetadataTool = &mcp.Tool{
	Name:        "helm_get_metadata",
	Description: "Get metadata for a release.",
	Annotations: tools.ReadOnly("Get release metadata", false),
}

func HandleGetMetadata(ctx context.Context, _ *mcp.CallToolRequest, input GetMetadataInput) (*mcp.CallToolResult, tools.MetadataOutput, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), tools.MetadataOutput{}, nil
	}
	if err := tools.ValidateReleaseName(input.ReleaseName); err != nil {
		return tools.ErrorResult(err), tools.MetadataOutput{}, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	cfg := input.ToGlobalConfig()
	defer cfg.ZeroCredentials()

	result, err := engine.GetMetadata(ctx, cfg, &helmengine.GetOptions{
		ReleaseName: input.ReleaseName,
		Revision:    input.Revision,
	})
	if err != nil {
		return tools.ErrorResult(err), tools.MetadataOutput{}, nil
	}

	return tools.TextResult(result), tools.MetadataOutput{Metadata: result}, nil
}

// --- Get Notes ---

type GetNotesInput struct {
	tools.GlobalInput
	ReleaseName string `json:"release_name" jsonschema:"Name of the release"`
	Revision    int    `json:"revision,omitempty" jsonschema:"Release revision number"`
}

var GetNotesTool = &mcp.Tool{
	Name:        "helm_get_notes",
	Description: "Get the notes for a release.",
	Annotations: tools.ReadOnly("Get release notes", false),
}

func HandleGetNotes(ctx context.Context, _ *mcp.CallToolRequest, input GetNotesInput) (*mcp.CallToolResult, any, error) {
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
	ReleaseName string `json:"release_name" jsonschema:"Name of the release"`
	Revision    int    `json:"revision,omitempty" jsonschema:"Release revision number"`
	All         bool   `json:"all,omitempty" jsonschema:"Include computed values"`
}

var GetValuesTool = &mcp.Tool{
	Name:        "helm_get_values",
	Description: "Get the values for a release. Use all=true to include computed values.",
	Annotations: tools.ReadOnly("Get release values", false),
}

func HandleGetValues(ctx context.Context, _ *mcp.CallToolRequest, input GetValuesInput) (*mcp.CallToolResult, tools.ValuesOutput, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), tools.ValuesOutput{}, nil
	}
	if err := tools.ValidateReleaseName(input.ReleaseName); err != nil {
		return tools.ErrorResult(err), tools.ValuesOutput{}, nil
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
		return tools.ErrorResult(err), tools.ValuesOutput{}, nil
	}

	return tools.TextResult(result), tools.ValuesOutput{Values: result}, nil
}
