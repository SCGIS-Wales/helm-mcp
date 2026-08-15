"""FastMCP proxy server wrapping the helm-mcp Go binary.

The proxy pattern ensures forward-compatibility: when new tools are added
to the Go binary, they are automatically discovered and exposed by the
proxy without any Python code changes. The MCP protocol handles tool
discovery at runtime via the ``tools/list`` method.
"""

from __future__ import annotations

import logging
import os
from typing import TYPE_CHECKING

from fastmcp.client.transports import StdioTransport
from fastmcp.server import create_proxy

from helm_mcp.discovery import find_binary, is_python_script
from helm_mcp.resilience import ResilienceConfig, build_middleware, setup_otel

if TYPE_CHECKING:
    from fastmcp import FastMCP

logger = logging.getLogger(__name__)


# Environment variables forwarded to the Go subprocess.
# Extend this list to pass additional variables — the proxy itself
# never needs to know about individual Helm tools.
PASSTHROUGH_ENV_VARS: list[str] = [
    # Core system
    "HOME",
    "USER",
    "PATH",
    # Kubernetes
    "KUBECONFIG",
    "KUBERNETES_SERVICE_HOST",
    "KUBERNETES_SERVICE_PORT",
    # Forward proxy
    "HTTP_PROXY",
    "HTTPS_PROXY",
    "NO_PROXY",
    "http_proxy",
    "https_proxy",
    "no_proxy",
    # Helm-specific
    "HELM_CACHE_HOME",
    "HELM_CONFIG_HOME",
    "HELM_DATA_HOME",
    "HELM_DRIVER",
    "HELM_REGISTRY_CONFIG",
    "HELM_REPOSITORY_CACHE",
    "HELM_REPOSITORY_CONFIG",
    "HELM_PLUGINS",
    "HELM_DEBUG",
    # AWS (EKS)
    "AWS_ACCESS_KEY_ID",
    "AWS_SECRET_ACCESS_KEY",
    "AWS_SESSION_TOKEN",
    "AWS_DEFAULT_REGION",
    "AWS_REGION",
    "AWS_PROFILE",
    "AWS_SHARED_CREDENTIALS_FILE",
    "AWS_CONFIG_FILE",
    # Google Cloud (GKE)
    "GOOGLE_APPLICATION_CREDENTIALS",
    "CLOUDSDK_COMPUTE_ZONE",
    "CLOUDSDK_COMPUTE_REGION",
    "CLOUDSDK_CORE_PROJECT",
    # Azure (AKS)
    "AZURE_TENANT_ID",
    "AZURE_CLIENT_ID",
    "AZURE_CLIENT_SECRET",
    "AZURE_SUBSCRIPTION_ID",
    "AZURE_AUTHORITY_HOST",
    # TLS / CA
    "SSL_CERT_FILE",
    "SSL_CERT_DIR",
    "REQUESTS_CA_BUNDLE",
]


# Binary discovery lives in helm_mcp.discovery so both entry points share it.
# These aliases keep the historical private names importable.
_is_python_script = is_python_script
_find_binary = find_binary


def _build_subprocess_env(
    extra_env: dict[str, str] | None = None,
    passthrough: list[str] | None = None,
) -> dict[str, str]:
    """Build the environment dict for the Go subprocess.

    Collects variables from ``PASSTHROUGH_ENV_VARS`` (or a custom list)
    and merges in any extra overrides.

    Args:
        extra_env: Additional variables that take precedence.
        passthrough: Override the default passthrough list.

    Returns:
        Environment dict for subprocess execution.
    """
    vars_to_pass = passthrough or PASSTHROUGH_ENV_VARS
    env: dict[str, str] = {}
    for var in vars_to_pass:
        val = os.environ.get(var)
        if val is not None:
            env[var] = val
    if extra_env:
        env.update(extra_env)
    return env


def create_server(
    binary_path: str | None = None,
    name: str = "helm-mcp",
    env: dict[str, str] | None = None,
    resilience: ResilienceConfig | None = None,
) -> FastMCP:
    """Create a FastMCP proxy server wrapping the helm-mcp Go binary.

    The proxy transparently forwards all MCP requests to the Go binary,
    which means any new tools added to the binary are automatically
    available without changing this Python code.

    Resilience middleware (retry, rate limiting, caching, error handling,
    timing) is applied based on the ``resilience`` config.  If ``None``,
    the config is read from ``HELM_MCP_*`` environment variables with
    sensible defaults.

    Args:
        binary_path: Explicit path to the helm-mcp binary. Auto-detected if ``None``.
        name: Server name advertised via MCP.
        env: Additional environment variables to pass to the subprocess.
            These are merged on top of the default passthrough list
            (``PASSTHROUGH_ENV_VARS``).
        resilience: Resilience configuration. If ``None``, reads from env vars.

    Returns:
        A FastMCP server instance ready to run.

    Example::

        server = create_server()
        server.run()                                       # stdio
        server.run(transport="http", host="0.0.0.0", port=8080)  # HTTP

    With custom resilience config::

        from helm_mcp.resilience import ResilienceConfig, RateLimitConfig
        config = ResilienceConfig(
            rate_limit=RateLimitConfig(enabled=True, max_requests_per_second=50),
        )
        server = create_server(resilience=config)
    """
    config = resilience or ResilienceConfig()

    # Set up OpenTelemetry SDK if enabled
    setup_otel(config.otel)

    binary = binary_path or _find_binary()
    subprocess_env = _build_subprocess_env(extra_env=env)
    logger.info("creating proxy server with binary: %s", binary)
    transport = StdioTransport(
        command=binary,
        args=["--mode", "stdio"],
        env=subprocess_env or None,
    )
    proxy = create_proxy(transport, name=name)

    # Apply resilience middleware
    middlewares = build_middleware(config)
    for mw in middlewares:
        proxy.add_middleware(mw)
    if middlewares:
        logger.info("applied %d resilience middleware(s) to proxy server", len(middlewares))

    return proxy
