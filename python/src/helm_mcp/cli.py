"""CLI entry points for helm-mcp.

Provides two commands:
  - ``helm-mcp``:        Thin wrapper that execs the bundled Go ``helm-mcp`` binary.
  - ``helm-mcp-python``: Python MCP proxy server wrapping the Go binary via FastMCP.
"""

import argparse
import logging
import os
import shutil
import stat
import sys
from pathlib import Path


def _find_bundled_binary(name: str) -> str | None:
    """Locate a binary bundled inside the package ``bin/`` directory.

    If the binary exists but is not executable, it is chmod'd on first use.

    Returns:
        Absolute path to the binary, or ``None`` if not found.
    """
    pkg_dir = Path(__file__).parent
    bundled = pkg_dir / "bin" / name
    if not bundled.is_file():
        return None
    # Ensure the binary is executable — pip may not preserve permissions
    # for package-data files extracted from wheels.
    if not os.access(str(bundled), os.X_OK):
        try:
            bundled.chmod(bundled.stat().st_mode | stat.S_IEXEC | stat.S_IXGRP | stat.S_IXOTH)
        except OSError:
            return None
    return str(bundled)


def _is_python_script(path: str) -> bool:
    """Check if *path* is a Python script (e.g. a pip console_script wrapper).

    This prevents ``shutil.which("helm-mcp")`` from returning the
    Python wrapper installed by pip, which would cause an infinite
    ``exec`` loop when ``helm_mcp_main`` tries to replace itself with
    the "real" binary.
    """
    try:
        with open(path, "rb") as fh:
            head = fh.read(128)
        first_line = head.split(b"\n", 1)[0].lower()
        return head[:2] == b"#!" and b"python" in first_line
    except OSError:
        return False


def _find_binary(name: str) -> str:
    """Find the Go binary by name: bundled → PATH → auto-download.

    Raises:
        FileNotFoundError: If the binary cannot be located.
    """
    # 1. Bundled binary inside the Python package
    bundled = _find_bundled_binary(name)
    if bundled:
        return bundled

    # 2. Binary on PATH (e.g. installed via go install or Homebrew).
    #    Skip Python console_script wrappers to avoid an infinite exec loop.
    found = shutil.which(name)
    if found and not _is_python_script(found):
        return found

    # 3. Auto-download from GitHub Releases (fallback for universal wheel)
    try:
        from helm_mcp import __version__
        from helm_mcp.download import ensure_binary

        downloaded = ensure_binary(__version__)
        if downloaded:
            return downloaded
    except Exception:
        pass

    raise FileNotFoundError(
        f"{name} binary not found. Install helm-mcp via:\n"
        "  pip install helm-mcp  (platform wheel bundles the binary)\n"
        "  or: go install github.com/SCGIS-Wales/helm-mcp/cmd/helm-mcp@latest\n"
        "  or: set HELM_MCP_BINARY=/path/to/helm-mcp"
    )


def helm_mcp_main() -> None:
    """Entry point for the ``helm-mcp`` command.

    Locates the bundled Go ``helm-mcp`` binary and replaces the current
    process with it, forwarding all command-line arguments.
    """
    try:
        binary = _find_binary("helm-mcp")
    except FileNotFoundError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
    os.execvp(binary, [binary] + sys.argv[1:])


def main() -> None:
    """Run the helm-mcp proxy server (``helm-mcp-python`` command)."""
    parser = argparse.ArgumentParser(
        description="helm-mcp: MCP server for Helm operations",
    )
    parser.add_argument(
        "--transport",
        choices=["stdio", "http", "sse"],
        default="stdio",
        help="Transport mode (default: stdio)",
    )
    parser.add_argument(
        "--host",
        default="0.0.0.0",
        help="Host for HTTP/SSE mode (default: 0.0.0.0)",
    )
    parser.add_argument(
        "--port",
        type=int,
        default=8080,
        help="Port for HTTP/SSE mode (default: 8080)",
    )
    parser.add_argument(
        "--binary",
        default=None,
        help="Path to helm-mcp Go binary (auto-detected if not set)",
    )
    parser.add_argument(
        "--setup",
        action="store_true",
        help="Download the helm-mcp Go binary and exit",
    )
    parser.add_argument(
        "--verbose",
        "-v",
        action="store_true",
        help="Enable verbose logging",
    )

    # Resilience arguments
    res_group = parser.add_argument_group(
        "resilience", "Resilience pattern configuration (overrides HELM_MCP_* env vars)"
    )
    res_group.add_argument(
        "--no-retry",
        action="store_true",
        help="Disable proxy-level retry middleware",
    )
    res_group.add_argument(
        "--rate-limit",
        type=float,
        default=None,
        metavar="RPS",
        help="Enable rate limiting at RPS requests/second",
    )
    res_group.add_argument(
        "--cache",
        action="store_true",
        help="Enable response caching",
    )
    res_group.add_argument(
        "--no-circuit-breaker",
        action="store_true",
        help="Disable circuit breaker on tool calls",
    )
    res_group.add_argument(
        "--bulkhead-max",
        type=int,
        default=None,
        metavar="N",
        help="Max concurrent tool calls (bulkhead limit)",
    )
    res_group.add_argument(
        "--otel",
        action="store_true",
        help="Enable OpenTelemetry tracing (requires pip install helm-mcp[otel])",
    )

    args = parser.parse_args()

    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
        stream=sys.stderr,
    )
    logger = logging.getLogger(__name__)

    if args.setup:
        from helm_mcp import __version__
        from helm_mcp.download import ensure_binary

        try:
            path = ensure_binary(__version__)
            if path:
                logger.info("helm-mcp binary ready at: %s", path)
                print(f"helm-mcp binary ready at: {path}")
            else:
                logger.error("no checksums available for this platform")
                print(
                    "No checksums available for this platform. Install the binary manually.",
                    file=sys.stderr,
                )
                sys.exit(1)
        except Exception as e:
            logger.error("failed to download binary: %s", e)
            print(f"Error downloading binary: {e}", file=sys.stderr)
            sys.exit(1)
        return

    # CLI overrides for resilience config
    if args.no_retry:
        os.environ["HELM_MCP_RETRY_ENABLED"] = "false"
    if args.rate_limit is not None:
        os.environ["HELM_MCP_RATE_LIMIT_ENABLED"] = "true"
        os.environ["HELM_MCP_RATE_LIMIT_MAX_RPS"] = str(args.rate_limit)
    if args.cache:
        os.environ["HELM_MCP_CACHE_ENABLED"] = "true"
    if args.no_circuit_breaker:
        os.environ["HELM_MCP_CIRCUIT_BREAKER_ENABLED"] = "false"
    if args.bulkhead_max is not None:
        os.environ["HELM_MCP_BULKHEAD_MAX_CONCURRENT"] = str(args.bulkhead_max)
    if args.otel:
        os.environ["HELM_MCP_OTEL_ENABLED"] = "true"

    from helm_mcp.resilience import ResilienceConfig
    from helm_mcp.server import create_server

    resilience_config = ResilienceConfig()

    try:
        server = create_server(binary_path=args.binary, resilience=resilience_config)
    except FileNotFoundError as e:
        logger.error("binary not found: %s", e)
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

    logger.info("starting server with transport=%s", args.transport)
    if args.transport == "stdio":
        server.run()
    else:
        logger.info("listening on %s:%d", args.host, args.port)
        server.run(transport=args.transport, host=args.host, port=args.port)


if __name__ == "__main__":
    main()
