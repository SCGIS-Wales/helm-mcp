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
"""

__version__ = "0.1.6"

from helm_mcp.client import create_client
from helm_mcp.server import create_server

__all__ = ["create_server", "create_client", "__version__"]
