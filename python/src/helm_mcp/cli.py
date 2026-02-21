"""CLI entry point for helm-mcp-python."""

import argparse
import logging
import sys


def main() -> None:
    """Run the helm-mcp proxy server."""
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
