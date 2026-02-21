# helm-mcp (Python)

A Python MCP wrapper for the [helm-mcp](https://github.com/ssddgreg/helm-mcp) Go server.

Uses [FastMCP](https://github.com/PrefectHQ/fastmcp) to create a transparent proxy around the helm-mcp Go binary, exposing all Helm tools via the Model Context Protocol. **New tools added to the Go binary are automatically available without any Python code changes.**

## Requirements

- Python 3.14+
- The `helm-mcp` Go binary on your `PATH` or pointed to via `HELM_MCP_BINARY`

## Installation

```bash
pip install helm-mcp
```

## Quick Start

### As a server

```python
from helm_mcp import create_server

server = create_server()
server.run()  # stdio mode (default)
```

### As a client

```python
import asyncio
from helm_mcp import create_client

async def main():
    async with create_client() as client:
        tools = await client.list_tools()
        print(f"Available tools: {len(tools)}")

        result = await client.call_tool("helm_list", {"namespace": "default"})
        print(result)

asyncio.run(main())
```

### CLI

```bash
# stdio mode (default, for MCP clients like Claude Code)
helm-mcp-python

# HTTP mode
helm-mcp-python --transport http --host 0.0.0.0 --port 8080

# Explicit binary path
helm-mcp-python --binary /usr/local/bin/helm-mcp
```

## Binary Discovery

The package locates the `helm-mcp` Go binary in this order:

1. `HELM_MCP_BINARY` environment variable
2. Bundled binary in the package `bin/` directory
3. `helm-mcp` on `PATH`

## Environment Variables

The proxy forwards these environment variables to the Go binary:

| Category | Variables |
|----------|-----------|
| Proxy | `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY` (and lowercase variants) |
| Kubernetes | `KUBECONFIG`, `KUBERNETES_SERVICE_HOST`, `KUBERNETES_SERVICE_PORT` |
| Helm | `HELM_CACHE_HOME`, `HELM_CONFIG_HOME`, `HELM_DATA_HOME`, `HELM_PLUGINS`, `HELM_DEBUG` |
| AWS | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, `AWS_REGION`, `AWS_PROFILE` |
| GCP | `GOOGLE_APPLICATION_CREDENTIALS`, `CLOUDSDK_COMPUTE_ZONE` |
| Azure | `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_SUBSCRIPTION_ID` |
| TLS | `SSL_CERT_FILE`, `SSL_CERT_DIR` |

## Scalability

This package uses the MCP proxy pattern: the Python layer never needs to know about individual Helm tools. All tool discovery, input schemas, and invocations are forwarded to the Go binary via the MCP protocol at runtime. When new capabilities are added to the Go server, they are immediately available through the Python wrapper.

## License

MIT
