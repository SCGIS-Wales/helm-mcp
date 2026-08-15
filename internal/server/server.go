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
// was built without -ldflags "-X main.version=...".
//
// This is a var rather than a const so that a build can override it; -X only
// patches string variables.
var ServerVersion = "0.2.0"

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

func registerReleaseTools(s *mcp.Server) {
	mcp.AddTool(s, release.ListTool, release.HandleList)
	mcp.AddTool(s, release.InstallTool, release.HandleInstall)
	mcp.AddTool(s, release.UpgradeTool, release.HandleUpgrade)
	mcp.AddTool(s, release.UninstallTool, release.HandleUninstall)
	mcp.AddTool(s, release.RollbackTool, release.HandleRollback)
	mcp.AddTool(s, release.StatusTool, release.HandleStatus)
	mcp.AddTool(s, release.HistoryTool, release.HandleHistory)
	mcp.AddTool(s, release.TestTool, release.HandleTest)
	mcp.AddTool(s, release.GetAllTool, release.HandleGetAll)
	mcp.AddTool(s, release.GetHooksTool, release.HandleGetHooks)
	mcp.AddTool(s, release.GetManifestTool, release.HandleGetManifest)
	mcp.AddTool(s, release.GetMetadataTool, release.HandleGetMetadata)
	mcp.AddTool(s, release.GetNotesTool, release.HandleGetNotes)
	mcp.AddTool(s, release.GetValuesTool, release.HandleGetValues)
}

func registerChartTools(s *mcp.Server) {
	mcp.AddTool(s, chart.CreateTool, chart.HandleCreate)
	mcp.AddTool(s, chart.LintTool, chart.HandleLint)
	mcp.AddTool(s, chart.TemplateTool, chart.HandleTemplate)
	mcp.AddTool(s, chart.PackageTool, chart.HandlePackage)
	mcp.AddTool(s, chart.PullTool, chart.HandlePull)
	mcp.AddTool(s, chart.PushTool, chart.HandlePush)
	mcp.AddTool(s, chart.VerifyTool, chart.HandleVerify)
	mcp.AddTool(s, chart.ShowAllTool, chart.HandleShowAll)
	mcp.AddTool(s, chart.ShowChartTool, chart.HandleShowChart)
	mcp.AddTool(s, chart.ShowCRDsTool, chart.HandleShowCRDs)
	mcp.AddTool(s, chart.ShowReadmeTool, chart.HandleShowReadme)
	mcp.AddTool(s, chart.ShowValuesTool, chart.HandleShowValues)
	mcp.AddTool(s, chart.DependencyBuildTool, chart.HandleDependencyBuild)
	mcp.AddTool(s, chart.DependencyListTool, chart.HandleDependencyList)
	mcp.AddTool(s, chart.DependencyUpdateTool, chart.HandleDependencyUpdate)
}

func registerRepoTools(s *mcp.Server) {
	mcp.AddTool(s, repo.AddTool, repo.HandleAdd)
	mcp.AddTool(s, repo.ListTool, repo.HandleList)
	mcp.AddTool(s, repo.UpdateTool, repo.HandleUpdate)
	mcp.AddTool(s, repo.RemoveTool, repo.HandleRemove)
	mcp.AddTool(s, repo.IndexTool, repo.HandleIndex)
}

func registerRegistryTools(s *mcp.Server) {
	mcp.AddTool(s, registry.LoginTool, registry.HandleLogin)
	mcp.AddTool(s, registry.LogoutTool, registry.HandleLogout)
}

func registerSearchTools(s *mcp.Server) {
	mcp.AddTool(s, search.HubTool, search.HandleHub)
	mcp.AddTool(s, search.RepoTool, search.HandleRepo)
}

func registerPluginTools(s *mcp.Server) {
	mcp.AddTool(s, plugin.InstallTool, plugin.HandleInstall)
	mcp.AddTool(s, plugin.ListTool, plugin.HandleList)
	mcp.AddTool(s, plugin.UninstallTool, plugin.HandleUninstall)
	mcp.AddTool(s, plugin.UpdateTool, plugin.HandleUpdate)
}

func registerEnvTools(s *mcp.Server) {
	mcp.AddTool(s, env.EnvTool, env.HandleEnv)
	mcp.AddTool(s, env.VersionTool, env.HandleVersion)
}
