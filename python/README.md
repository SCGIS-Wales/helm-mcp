# helm-mcp (Python)

[![CI/CD Pipeline](https://github.com/SCGIS-Wales/helm-mcp/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/SCGIS-Wales/helm-mcp/actions/workflows/ci.yml)
[![PyPI version](https://img.shields.io/pypi/v/helm-mcp)](https://pypi.org/project/helm-mcp/)
[![Python versions](https://img.shields.io/pypi/pyversions/helm-mcp)](https://pypi.org/project/helm-mcp/)
[![Downloads](https://img.shields.io/pypi/dm/helm-mcp)](https://pypistats.org/packages/helm-mcp)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Python MCP wrapper for the [helm-mcp](https://github.com/SCGIS-Wales/helm-mcp) Go server.

Uses [FastMCP](https://github.com/PrefectHQ/fastmcp) to create a transparent proxy around the helm-mcp Go binary, exposing all Helm tools via the Model Context Protocol. **New tools added to the Go binary are automatically available without any Python code changes.**

> **Removed in 0.2.0:** `--transport sse`. The MCP specification deprecates the
> HTTP+SSE transport and the Go server no longer serves it. Use
> `--transport http`.

## Requirements

- Python 3.12+
- The `helm-mcp` Go binary is **automatically downloaded** on first use (with SHA256 checksum verification)

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

# HTTP mode (Streamable HTTP)
helm-mcp-python --transport http --host 0.0.0.0 --port 8080

# Pre-download binary
helm-mcp-python --setup

# Explicit binary path
helm-mcp-python --binary /usr/local/bin/helm-mcp
```

## Binary Discovery

Both the `helm-mcp` and `helm-mcp-python` commands locate the Go binary the
same way, via `helm_mcp.discovery`:

1. `HELM_MCP_BINARY` environment variable
2. Bundled binary in the package `bin/` directory (platform wheels)
3. `helm-mcp` on `PATH` — pip console-script wrappers are skipped, so a
   `go install` or Homebrew build wins over a network fetch
4. Auto-download from GitHub Releases (with SHA256 checksum verification)

## Environment Variables

The proxy forwards these environment variables to the Go binary:

| Category | Variables |
|----------|-----------|
| Proxy | `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY` (and lowercase variants) |
| Kubernetes | `KUBECONFIG`, `KUBERNETES_SERVICE_HOST`, `KUBERNETES_SERVICE_PORT` |
| Helm | `HELM_CACHE_HOME`, `HELM_CONFIG_HOME`, `HELM_DATA_HOME`, `HELM_DRIVER`, `HELM_PLUGINS`, `HELM_DEBUG`, `HELM_REGISTRY_CONFIG`, `HELM_REPOSITORY_CACHE`, `HELM_REPOSITORY_CONFIG` |
| AWS | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, `AWS_REGION`, `AWS_DEFAULT_REGION`, `AWS_PROFILE`, `AWS_SHARED_CREDENTIALS_FILE`, `AWS_CONFIG_FILE` |
| GCP | `GOOGLE_APPLICATION_CREDENTIALS`, `CLOUDSDK_COMPUTE_ZONE`, `CLOUDSDK_COMPUTE_REGION`, `CLOUDSDK_CORE_PROJECT` |
| Azure | `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_SUBSCRIPTION_ID`, `AZURE_AUTHORITY_HOST` |
| TLS | `SSL_CERT_FILE`, `SSL_CERT_DIR`, `REQUESTS_CA_BUNDLE` |

The full list is `helm_mcp.server.PASSTHROUGH_ENV_VARS`.

## Resilience

`HelmClient` and the proxy server wrap every call in a configurable resilience
stack — timeouts, retries with exponential backoff and jitter, a circuit
breaker, a concurrency bulkhead, optional rate limiting and response caching,
and optional OpenTelemetry tracing. Everything is configurable in code via
`ResilienceConfig`, or from the environment:

| Variable | Default | Effect |
|---|---|---|
| `HELM_MCP_TIMEOUT` | `300` | Per-call timeout in seconds |
| `HELM_MCP_RETRY_ENABLED` | `true` | Retry transient connection failures |
| `HELM_MCP_RETRY_MAX_ATTEMPTS` | `3` | Attempts before giving up |
| `HELM_MCP_CIRCUIT_BREAKER_ENABLED` | `true` | Trip after repeated connection failures |
| `HELM_MCP_BULKHEAD_MAX_CONCURRENT` | `10` | Concurrent in-flight tool calls |
| `HELM_MCP_RATE_LIMIT_ENABLED` | `false` | Client-side rate limiting |
| `HELM_MCP_CACHE_ENABLED` | `false` | Cache responses |
| `HELM_MCP_OTEL_ENABLED` | `false` | Export traces (needs the `otel` extra) |

The corresponding CLI flags are `--no-retry`, `--no-circuit-breaker`,
`--rate-limit`, `--cache`, `--bulkhead-max` and `--otel`.

## Scalability

This package uses the MCP proxy pattern: the Python layer never needs to know about individual Helm tools. All tool discovery, input schemas, and invocations are forwarded to the Go binary via the MCP protocol at runtime. When new capabilities are added to the Go server, they are immediately available through the Python wrapper.

## License

MIT
