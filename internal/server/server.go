package server

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ssddgreg/helm-mcp/internal/tools/chart"
	"github.com/ssddgreg/helm-mcp/internal/tools/env"
	"github.com/ssddgreg/helm-mcp/internal/tools/plugin"
	"github.com/ssddgreg/helm-mcp/internal/tools/registry"
	"github.com/ssddgreg/helm-mcp/internal/tools/release"
	"github.com/ssddgreg/helm-mcp/internal/tools/repo"
	"github.com/ssddgreg/helm-mcp/internal/tools/search"
)

const ServerName = "helm-mcp"

// ServerVersion is the fallback version reported to clients when the binary
// was built without -ldflags "-X main.version=...". It matches the default
// in main rather than naming a release, which it would soon misreport.
//
// This is a var rather than a const so that a build can override it; -X only
// patches string variables.
var ServerVersion = "dev"

// instructions is sent to clients during discovery and is the server's only
// chance to explain itself outside of individual tool descriptions.
const instructions = `helm-mcp exposes the Helm CLI surface as MCP tools, backed by the Helm Go SDK.

Version selection: every tool accepts helm_version ("v4", the default, or "v3").
Stay on v4 unless the user explicitly needs v3 behaviour. Some arguments are
v4-only and are marked as such in their descriptions; passing them with
helm_version "v3" is an error rather than a silent no-op.

Cluster targeting: every tool also accepts namespace, kube_context, kubeconfig,
kube_apiserver and kube_token. When the user names a cluster, context or
namespace, pass it explicitly rather than relying on ambient defaults.

Safety: tools carry annotations. Anything with destructiveHint set changes or
removes live cluster state (helm_uninstall, helm_rollback, helm_upgrade,
helm_repo_remove, helm_plugin_uninstall) and should only be called on explicit
user intent. Prefer the read-only tools (helm_list, helm_status, helm_get_*) to
establish state first, and helm_template or a dry_run of "client"/"server" to
preview a change before applying it.`

// schemaCache is shared across every server instance.
//
// The HTTP transports construct a fresh mcp.Server per request, so without a
// shared cache each request would re-derive the JSON schema for all 46 tools
// by reflection.
var schemaCache = mcp.NewSchemaCache()

// NewServer creates a new MCP server with all Helm tools registered.
// If version is empty, the default ServerVersion is used.
func NewServer(version string) *mcp.Server {
	if version == "" {
		version = ServerVersion
	}
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    ServerName,
			Version: version,
		},
		&mcp.ServerOptions{
			Instructions: instructions,
			PageSize:     100,
			SchemaCache:  schemaCache,
			// Only tools are served. Without this, the SDK advertises the
			// logging capability for historical reasons, and logging is
			// deprecated as of protocol version 2026-07-28 (SEP-2577).
			Capabilities: &mcp.ServerCapabilities{},
		},
	)

	registerReleaseTools(server)
	registerChartTools(server)
	registerRepoTools(server)
	registerRegistryTools(server)
	registerSearchTools(server)
	registerPluginTools(server)
	registerEnvTools(server)

	return server
}

// addTool registers a tool with Tool.Title set from its annotation title, so
// clients that read the top-level title (the field the specification now
// prefers for display) show a readable name.
//
// The definition is copied: tools are shared package variables and the HTTP
// transport calls NewServer once per request, so mutating them in place
// would race. The schema cache is keyed by type, so copies cost nothing.
func addTool[In, Out any](s *mcp.Server, t *mcp.Tool, h mcp.ToolHandlerFor[In, Out]) {
	tool := *t
	if tool.Title == "" && tool.Annotations != nil {
		tool.Title = tool.Annotations.Title
	}
	mcp.AddTool(s, &tool, h)
}

func registerReleaseTools(s *mcp.Server) {
	addTool(s, release.ListTool, release.HandleList)
	addTool(s, release.InstallTool, release.HandleInstall)
	addTool(s, release.UpgradeTool, release.HandleUpgrade)
	addTool(s, release.UninstallTool, release.HandleUninstall)
	addTool(s, release.RollbackTool, release.HandleRollback)
	addTool(s, release.StatusTool, release.HandleStatus)
	addTool(s, release.HistoryTool, release.HandleHistory)
	addTool(s, release.TestTool, release.HandleTest)
	addTool(s, release.GetAllTool, release.HandleGetAll)
	addTool(s, release.GetHooksTool, release.HandleGetHooks)
	addTool(s, release.GetManifestTool, release.HandleGetManifest)
	addTool(s, release.GetMetadataTool, release.HandleGetMetadata)
	addTool(s, release.GetNotesTool, release.HandleGetNotes)
	addTool(s, release.GetValuesTool, release.HandleGetValues)
}

func registerChartTools(s *mcp.Server) {
	addTool(s, chart.CreateTool, chart.HandleCreate)
	addTool(s, chart.LintTool, chart.HandleLint)
	addTool(s, chart.TemplateTool, chart.HandleTemplate)
	addTool(s, chart.PackageTool, chart.HandlePackage)
	addTool(s, chart.PullTool, chart.HandlePull)
	addTool(s, chart.PushTool, chart.HandlePush)
	addTool(s, chart.VerifyTool, chart.HandleVerify)
	addTool(s, chart.ShowAllTool, chart.HandleShowAll)
	addTool(s, chart.ShowChartTool, chart.HandleShowChart)
	addTool(s, chart.ShowCRDsTool, chart.HandleShowCRDs)
	addTool(s, chart.ShowReadmeTool, chart.HandleShowReadme)
	addTool(s, chart.ShowValuesTool, chart.HandleShowValues)
	addTool(s, chart.DependencyBuildTool, chart.HandleDependencyBuild)
	addTool(s, chart.DependencyListTool, chart.HandleDependencyList)
	addTool(s, chart.DependencyUpdateTool, chart.HandleDependencyUpdate)
}

func registerRepoTools(s *mcp.Server) {
	addTool(s, repo.AddTool, repo.HandleAdd)
	addTool(s, repo.ListTool, repo.HandleList)
	addTool(s, repo.UpdateTool, repo.HandleUpdate)
	addTool(s, repo.RemoveTool, repo.HandleRemove)
	addTool(s, repo.IndexTool, repo.HandleIndex)
}

func registerRegistryTools(s *mcp.Server) {
	addTool(s, registry.LoginTool, registry.HandleLogin)
	addTool(s, registry.LogoutTool, registry.HandleLogout)
}

func registerSearchTools(s *mcp.Server) {
	addTool(s, search.HubTool, search.HandleHub)
	addTool(s, search.RepoTool, search.HandleRepo)
}

func registerPluginTools(s *mcp.Server) {
	addTool(s, plugin.InstallTool, plugin.HandleInstall)
	addTool(s, plugin.ListTool, plugin.HandleList)
	addTool(s, plugin.UninstallTool, plugin.HandleUninstall)
	addTool(s, plugin.UpdateTool, plugin.HandleUpdate)
	addTool(s, plugin.PackageTool, plugin.HandlePackage)
	addTool(s, plugin.VerifyTool, plugin.HandleVerify)
}

func registerEnvTools(s *mcp.Server) {
	addTool(s, env.EnvTool, env.HandleEnv)
	addTool(s, env.VersionTool, env.HandleVersion)
}
