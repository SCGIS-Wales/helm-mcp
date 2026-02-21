"""FastMCP proxy server wrapping the helm-mcp Go binary.

The proxy pattern ensures forward-compatibility: when new tools are added
to the Go binary, they are automatically discovered and exposed by the
proxy without any Python code changes. The MCP protocol handles tool
discovery at runtime via the ``tools/list`` method.
"""

import logging
import os
import platform
import shutil
from pathlib import Path

from fastmcp.client.transports import StdioTransport
from fastmcp.server import create_proxy

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


def _find_binary() -> str:
    """Locate the helm-mcp binary.

    Search order:
      1. ``HELM_MCP_BINARY`` environment variable
      2. Bundled binary in the package ``bin/`` directory
      3. ``helm-mcp`` on ``PATH`` (platform-specific wheel installs here)
      4. Auto-download from GitHub Releases (fallback for universal wheel)

    Returns:
        Absolute path to the helm-mcp executable.

    Raises:
        FileNotFoundError: If the binary cannot be located.
    """
    # 1. Explicit env var
    env_path = os.environ.get("HELM_MCP_BINARY")
    if env_path:
        p = Path(env_path)
        if p.is_file() and os.access(str(p), os.X_OK):
            logger.info("using binary from HELM_MCP_BINARY: %s", p)
            return str(p)
        raise FileNotFoundError(f"HELM_MCP_BINARY={env_path} does not exist or is not executable")

    # 2. Bundled binary in package data
    pkg_dir = Path(__file__).parent
    system = platform.system().lower()
    machine = platform.machine().lower()
    arch_map = {"x86_64": "amd64", "aarch64": "arm64", "arm64": "arm64", "amd64": "amd64"}
    arch = arch_map.get(machine, machine)
    binary_name = f"helm-mcp-{system}-{arch}"
    if system == "windows":
        binary_name += ".exe"
    bundled = pkg_dir / "bin" / binary_name
    if bundled.is_file() and os.access(str(bundled), os.X_OK):
        logger.info("using bundled binary: %s", bundled)
        return str(bundled)

    # Also check for plain "helm-mcp" in bin/
    plain = pkg_dir / "bin" / "helm-mcp"
    if plain.is_file() and os.access(str(plain), os.X_OK):
        logger.info("using bundled binary: %s", plain)
        return str(plain)

    # 3. PATH lookup (preferred over download — works with platform-specific wheels)
    found = shutil.which("helm-mcp")
    if found:
        logger.info("using binary from PATH: %s", found)
        return found

    # 4. Auto-download from GitHub Releases (fallback for universal wheel)
    from helm_mcp import __version__
    from helm_mcp.download import ensure_binary

    try:
        downloaded = ensure_binary(__version__)
        if downloaded:
            logger.info("using auto-downloaded binary: %s", downloaded)
            return downloaded
    except Exception:
        logger.warning("auto-download failed", exc_info=True)

    raise FileNotFoundError(
        "helm-mcp binary not found. Either:\n"
        "  1. Set HELM_MCP_BINARY=/path/to/helm-mcp\n"
        "  2. Install helm-mcp and ensure it's on your PATH\n"
        "  3. Install the platform-specific wheel (pip install helm-mcp[binary])"
    )


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
):
    """Create a FastMCP proxy server wrapping the helm-mcp Go binary.

    The proxy transparently forwards all MCP requests to the Go binary,
    which means any new tools added to the binary are automatically
    available without changing this Python code.

    Args:
        binary_path: Explicit path to the helm-mcp binary. Auto-detected if ``None``.
        name: Server name advertised via MCP.
        env: Additional environment variables to pass to the subprocess.
            These are merged on top of the default passthrough list
            (``PASSTHROUGH_ENV_VARS``).

    Returns:
        A FastMCP server instance ready to run.

    Example::

        server = create_server()
        server.run()                                       # stdio
        server.run(transport="http", host="0.0.0.0", port=8080)  # HTTP
    """
    binary = binary_path or _find_binary()
    subprocess_env = _build_subprocess_env(extra_env=env)
    logger.info("creating proxy server with binary: %s", binary)
    transport = StdioTransport(
        command=binary,
        args=["--mode", "stdio"],
        env=subprocess_env or None,
    )
    return create_proxy(transport, name=name)
