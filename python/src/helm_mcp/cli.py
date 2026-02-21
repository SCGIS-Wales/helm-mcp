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


def _find_binary(name: str) -> str:
    """Find a binary by name: bundled in package, then PATH.

    Raises:
        FileNotFoundError: If the binary cannot be located.
    """
    # 1. Bundled binary inside the Python package
    bundled = _find_bundled_binary(name)
    if bundled:
        return bundled

    # 2. Binary on PATH (e.g. installed via go install or Homebrew)
    found = shutil.which(name)
    if found:
        return found

    raise FileNotFoundError(
        f"{name} binary not found. Install helm-mcp via:\n"
        "  go install github.com/SCGIS-Wales/helm-mcp/cmd/helm-mcp@latest\n"
        "  or: pip install helm-mcp  (platform wheel bundles the binary)"
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

    from helm_mcp.server import create_server

    try:
        server = create_server(binary_path=args.binary)
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
