"""helm-mcp: A Python MCP wrapper for the Helm MCP server.

This package provides a FastMCP proxy that wraps the helm-mcp Go binary,
automatically exposing all Helm tools via the Model Context Protocol.

The proxy pattern means this package requires zero code changes when new
tools are added to the Go binary — they are discovered and forwarded
automatically at runtime via the MCP protocol.

Usage as a server:
    from helm_mcp import create_server
    server = create_server()
    server.run()

Usage as a client:
    from helm_mcp import create_client
    async with create_client() as client:
        tools = await client.list_tools()
        result = await client.call_tool("helm_list", {"namespace": "default"})

Usage with custom resilience config:
    from helm_mcp import create_server, ResilienceConfig, RateLimitConfig
    config = ResilienceConfig(
        rate_limit=RateLimitConfig(enabled=True, max_requests_per_second=50),
    )
    server = create_server(resilience=config)
"""

__version__ = "0.1.24"

from helm_mcp.client import create_client
from helm_mcp.resilience import (
    BulkheadConfig,
    CacheConfig,
    CircuitBreakerConfig,
    ErrorHandlingConfig,
    OTelConfig,
    RateLimitConfig,
    ResilienceConfig,
    RetryConfig,
    TenacityConfig,
    TimingConfig,
)
from helm_mcp.server import create_server
from helm_mcp.tools import (
    HelmCircuitOpenError,
    HelmClient,
    HelmConnectionError,
    HelmError,
    HelmTimeoutError,
    HelmToolError,
    close_default_client,
    helm_create,
    helm_dependency_build,
    helm_dependency_list,
    helm_dependency_update,
    helm_env,
    helm_get_all,
    helm_get_hooks,
    helm_get_manifest,
    helm_get_metadata,
    helm_get_notes,
    helm_get_values,
    helm_history,
    helm_install,
    helm_lint,
    helm_list,
    helm_package,
    helm_plugin_install,
    helm_plugin_list,
    helm_plugin_uninstall,
    helm_plugin_update,
    helm_pull,
    helm_push,
    helm_registry_login,
    helm_registry_logout,
    helm_repo_add,
    helm_repo_index,
    helm_repo_list,
    helm_repo_remove,
    helm_repo_update,
    helm_rollback,
    helm_search_hub,
    helm_search_repo,
    helm_show_all,
    helm_show_chart,
    helm_show_crds,
    helm_show_readme,
    helm_show_values,
    helm_status,
    helm_template,
    helm_test,
    helm_uninstall,
    helm_upgrade,
    helm_verify,
    helm_version,
)

__all__ = [
    "__version__",
    "create_server",
    "create_client",
    # Client class and exceptions
    "HelmClient",
    "HelmError",
    "HelmTimeoutError",
    "HelmConnectionError",
    "HelmCircuitOpenError",
    "HelmToolError",
    "close_default_client",
    # Resilience configuration
    "ResilienceConfig",
    "RetryConfig",
    "RateLimitConfig",
    "CacheConfig",
    "ErrorHandlingConfig",
    "TimingConfig",
    "CircuitBreakerConfig",
    "TenacityConfig",
    "BulkheadConfig",
    "OTelConfig",
    # Async tool wrappers (all 44 tools)
    "helm_list",
    "helm_install",
    "helm_upgrade",
    "helm_uninstall",
    "helm_rollback",
    "helm_status",
    "helm_history",
    "helm_test",
    "helm_get_all",
    "helm_get_hooks",
    "helm_get_manifest",
    "helm_get_metadata",
    "helm_get_notes",
    "helm_get_values",
    "helm_create",
    "helm_lint",
    "helm_template",
    "helm_package",
    "helm_pull",
    "helm_push",
    "helm_verify",
    "helm_show_all",
    "helm_show_chart",
    "helm_show_crds",
    "helm_show_readme",
    "helm_show_values",
    "helm_dependency_build",
    "helm_dependency_list",
    "helm_dependency_update",
    "helm_repo_add",
    "helm_repo_list",
    "helm_repo_update",
    "helm_repo_remove",
    "helm_repo_index",
    "helm_registry_login",
    "helm_registry_logout",
    "helm_search_hub",
    "helm_search_repo",
    "helm_plugin_install",
    "helm_plugin_list",
    "helm_plugin_uninstall",
    "helm_plugin_update",
    "helm_env",
    "helm_version",
]
