"""CLI entry point for helm-mcp-python."""

import argparse
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
    args = parser.parse_args()

    from helm_mcp.server import create_server

    try:
        server = create_server(binary_path=args.binary)
    except FileNotFoundError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

    if args.transport == "stdio":
        server.run()
    else:
        server.run(transport=args.transport, host=args.host, port=args.port)


if __name__ == "__main__":
    main()
