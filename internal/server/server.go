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

const (
	ServerName    = "helm-mcp"
	ServerVersion = "0.1.0"
)

// NewServer creates a new MCP server with all Helm tools registered.
// If version is empty, the default ServerVersion constant is used.
func NewServer(version string) *mcp.Server {
	if version == "" {
		version = ServerVersion
	}
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    ServerName,
			Version: version,
		},
		nil,
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
