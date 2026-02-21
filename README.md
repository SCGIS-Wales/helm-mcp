# helm-mcp

[![CI/CD Pipeline](https://github.com/SCGIS-Wales/helm-mcp/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/SCGIS-Wales/helm-mcp/actions/workflows/ci.yml)
[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen.svg)](https://goreportcard.com/report/github.com/SCGIS-Wales/helm-mcp)
[![PyPI version](https://img.shields.io/pypi/v/helm-mcp)](https://pypi.org/project/helm-mcp/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

An open-source MCP (Model Context Protocol) server that gives AI assistants **full access to Helm** — the Kubernetes package manager. Built with the native Helm Go SDK, supporting both Helm 3.x and 4.x in a single binary.

> **Use natural language to manage your Kubernetes deployments.** Connect helm-mcp to Claude, Cursor, VS Code, or any MCP-compatible client to install charts, manage releases, search repositories, and more — all through conversation.

---

## Table of Contents

- [Why helm-mcp?](#why-helm-mcp)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [MCP Client Configuration](#mcp-client-configuration)
- [Available Tools](#available-tools-44)
- [Kubernetes Authentication](#kubernetes-authentication)
- [Helm Version Selection](#helm-version-selection)
- [Python Package](#python-package)
- [Security](#security)
- [Development](#development)
- [Architecture](#architecture)
- [Contributing](#contributing)
- [Community](#community)
- [License](#license)

## Why helm-mcp?

- **44 MCP tools** covering every Helm CLI command (minus shell completion and help)
- **Dual Helm SDK support** — Helm v3 and v4 via native Go SDK (not CLI wrappers)
- **Three transport modes** — stdio (default), HTTP (Streamable HTTP), SSE
- **Cloud provider ready** — EKS, GKE, AKS kubeconfig formats work out of the box
- **Security first** — credential scrubbing, input validation, path traversal prevention
- **Python wrapper** — [FastMCP](https://github.com/PrefectHQ/fastmcp)-based proxy that auto-discovers all tools
- **Forward proxy support** — respects `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`

## Installation

### Pre-built Binaries

Download the latest release for your platform from [GitHub Releases](https://github.com/SCGIS-Wales/helm-mcp/releases):

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/SCGIS-Wales/helm-mcp/releases/latest/download/helm-mcp-darwin-arm64
chmod +x helm-mcp-darwin-arm64
sudo mv helm-mcp-darwin-arm64 /usr/local/bin/helm-mcp

# macOS (Intel)
curl -LO https://github.com/SCGIS-Wales/helm-mcp/releases/latest/download/helm-mcp-darwin-amd64
chmod +x helm-mcp-darwin-amd64
sudo mv helm-mcp-darwin-amd64 /usr/local/bin/helm-mcp

# Linux (amd64)
curl -LO https://github.com/SCGIS-Wales/helm-mcp/releases/latest/download/helm-mcp-linux-amd64
chmod +x helm-mcp-linux-amd64
sudo mv helm-mcp-linux-amd64 /usr/local/bin/helm-mcp

# Linux (arm64)
curl -LO https://github.com/SCGIS-Wales/helm-mcp/releases/latest/download/helm-mcp-linux-arm64
chmod +x helm-mcp-linux-arm64
sudo mv helm-mcp-linux-arm64 /usr/local/bin/helm-mcp
```

### Build from Source

Requires Go 1.25+.

```bash
git clone https://github.com/SCGIS-Wales/helm-mcp.git
cd helm-mcp
make build
```

### Docker

```bash
docker build -t helm-mcp .
docker run -v ~/.kube:/home/helmuser/.kube:ro helm-mcp --mode stdio
```

### Python Package

```bash
pip install helm-mcp
```

See [Python Package](#python-package) below for full details.

## Quick Start

### stdio mode (for Claude Code, Cursor, etc.)

```bash
helm-mcp --mode stdio
```

### HTTP mode (Streamable HTTP)

```bash
helm-mcp --mode http --addr :8080
```

### SSE mode (Server-Sent Events)

```bash
helm-mcp --mode sse --addr :8080
```

## MCP Client Configuration

### Claude Desktop

Add to `~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "helm": {
      "command": "helm-mcp",
      "args": ["--mode", "stdio"]
    }
  }
}
```

### Claude Code

```bash
claude mcp add helm -- helm-mcp --mode stdio
```

### Cursor / Windsurf / VS Code

Add to your MCP server configuration:

```json
{
  "helm-mcp": {
    "command": "helm-mcp",
    "args": ["--mode", "stdio"]
  }
}
```

### Remote / HTTP Clients

Start the server in HTTP mode, then connect any MCP-compatible client to the endpoint:

```bash
helm-mcp --mode http --addr :8080
# MCP endpoint: http://localhost:8080/mcp
```

## Available Tools (44)

### Release Management (14)

| Tool | Description |
|------|-------------|
| `helm_install` | Install a Helm chart as a new release |
| `helm_upgrade` | Upgrade a release to a new chart version or values |
| `helm_uninstall` | Uninstall a release and remove associated resources |
| `helm_rollback` | Rollback a release to a previous revision |
| `helm_list` | List releases (supports filters, sorting, pagination) |
| `helm_status` | Display release status, revision, chart, and values |
| `helm_history` | Show revision history of a release |
| `helm_test` | Run the test suite for a release |
| `helm_get_all` | Get all info (values, manifest, hooks, notes) for a release |
| `helm_get_hooks` | Get hooks for a release |
| `helm_get_manifest` | Get the Kubernetes manifest for a release |
| `helm_get_metadata` | Get metadata for a release |
| `helm_get_notes` | Get notes for a release |
| `helm_get_values` | Get values for a release (user-supplied or computed) |

### Chart Management (14)

| Tool | Description |
|------|-------------|
| `helm_create` | Create a new chart with the given name |
| `helm_lint` | Lint a chart for issues and best practices |
| `helm_template` | Render templates locally without installing |
| `helm_package` | Package a chart directory into an archive (.tgz) |
| `helm_pull` | Download a chart from a repository or OCI registry |
| `helm_push` | Push a chart archive to an OCI registry |
| `helm_verify` | Verify a chart has a valid provenance file |
| `helm_show_all` | Show all chart info (Chart.yaml, values, README, CRDs) |
| `helm_show_chart` | Show Chart.yaml of a chart |
| `helm_show_crds` | Show CRDs of a chart |
| `helm_show_readme` | Show README of a chart |
| `helm_show_values` | Show default values of a chart |
| `helm_dependency_build` | Build charts/ directory from Chart.lock |
| `helm_dependency_list` | List dependencies for a chart |

### Repository Management (5)

| Tool | Description |
|------|-------------|
| `helm_repo_add` | Add a chart repository |
| `helm_repo_list` | List configured chart repositories |
| `helm_repo_update` | Update chart repository indexes |
| `helm_repo_remove` | Remove chart repositories |
| `helm_repo_index` | Generate an index file for chart archives |

### Registry / OCI (2)

| Tool | Description |
|------|-------------|
| `helm_registry_login` | Login to an OCI registry |
| `helm_registry_logout` | Logout from an OCI registry |

### Search (2)

| Tool | Description |
|------|-------------|
| `helm_search_hub` | Search Artifact Hub for charts |
| `helm_search_repo` | Search locally configured repositories |

### Plugin Management (4)

| Tool | Description |
|------|-------------|
| `helm_plugin_install` | Install a Helm plugin |
| `helm_plugin_list` | List installed plugins |
| `helm_plugin_uninstall` | Uninstall a plugin |
| `helm_plugin_update` | Update a plugin |

### Environment (2)

| Tool | Description |
|------|-------------|
| `helm_env` | Print Helm environment information |
| `helm_version` | Print Helm SDK version information |

### Dependency Update (1)

| Tool | Description |
|------|-------------|
| `helm_dependency_update` | Update charts/ based on Chart.yaml |

## Kubernetes Authentication

Every tool accepts these authentication fields via the `GlobalInput`:

| Field | JSON Key | Description |
|-------|----------|-------------|
| Kubeconfig | `kubeconfig` | Path to kubeconfig file (defaults to `$KUBECONFIG` or `~/.kube/config`) |
| Context | `kube_context` | Kubernetes context name to use |
| API Server | `kube_apiserver` | Override the API server URL from kubeconfig |
| Bearer Token | `kube_token` | Bearer token for API authentication |
| TLS Server Name | `kube_tls_server_name` | Server name for TLS certificate validation |
| Insecure TLS | `kube_insecure_tls` | Skip TLS certificate verification |
| Namespace | `namespace` | Target Kubernetes namespace |

### EKS (AWS)

EKS uses exec-based authentication in kubeconfig. The standard kubeconfig generated by `aws eks update-kubeconfig` works out of the box:

```json
{
  "kubeconfig": "/home/user/.kube/config",
  "kube_context": "arn:aws:eks:us-east-1:123456789:cluster/my-cluster"
}
```

Or with direct token authentication:

```json
{
  "kube_apiserver": "https://ABCDEF.gr7.us-east-1.eks.amazonaws.com",
  "kube_token": "<bearer-token-from-aws-eks-get-token>"
}
```

### GKE (Google Cloud)

GKE kubeconfig generated by `gcloud container clusters get-credentials` works out of the box:

```json
{
  "kubeconfig": "/home/user/.kube/config",
  "kube_context": "gke_my-project_us-central1_my-cluster"
}
```

### AKS (Azure)

AKS kubeconfig generated by `az aks get-credentials` works out of the box:

```json
{
  "kubeconfig": "/home/user/.kube/config",
  "kube_context": "my-aks-cluster"
}
```

## Helm Version Selection

Every tool supports a `helm_version` field to select between Helm v3 and v4:

```json
{
  "helm_version": "v4",
  "release_name": "my-release"
}
```

- `"v4"` (default) — Uses Helm v4 SDK with Server-Side Apply, WASM plugins, label selectors
- `"v3"` — Uses Helm v3 SDK for backward compatibility

### v4-Only Features

These fields are only available when using `helm_version: "v4"`:

- `server_side_apply` — Use Kubernetes server-side apply
- `take_ownership` — Skip Helm annotation checks
- `rollback_on_failure` — Auto-rollback on install failure
- `hide_secret` — Hide secrets in dry-run output
- `force_conflicts` — Force conflict resolution
- `selector` — Label selector for list operations
- `show_resources` — Show resources table in status
- `reset_then_reuse_values` — Reset then reuse values in upgrade

## Python Package

A Python wrapper is available that uses [FastMCP](https://github.com/PrefectHQ/fastmcp) to create a transparent proxy around the helm-mcp Go binary. New tools added to the Go binary are automatically available in Python without code changes.

### Installation

```bash
pip install helm-mcp
```

Requires Python 3.14+. The Go binary is **automatically downloaded** on first use (with SHA256 checksum verification). You can also pre-download it:

```bash
helm-mcp-python --setup
```

### Usage as a Server

```python
from helm_mcp import create_server

# stdio mode (default, for MCP clients)
server = create_server()
server.run()

# HTTP mode
server = create_server()
server.run(transport="http", host="0.0.0.0", port=8080)
```

### Usage as a Client

```python
import asyncio
from helm_mcp import create_client

async def main():
    async with create_client() as client:
        # List all available tools
        tools = await client.list_tools()
        print(f"Available tools: {len(tools)}")

        # List Helm releases
        result = await client.call_tool("helm_list", {"namespace": "default"})
        print(result)

        # Install a chart
        result = await client.call_tool("helm_install", {
            "release_name": "my-app",
            "chart": "bitnami/nginx",
            "namespace": "default",
        })
        print(result)

asyncio.run(main())
```

### CLI

```bash
# stdio mode (for MCP clients like Claude Code)
helm-mcp-python

# HTTP mode
helm-mcp-python --transport http --host 0.0.0.0 --port 8080

# Custom binary path
helm-mcp-python --binary /usr/local/bin/helm-mcp
```

### Integrating with FastMCP

The Python package is built on [FastMCP](https://github.com/PrefectHQ/fastmcp) and returns standard FastMCP server/client objects. You can compose it with other FastMCP servers:

```python
from fastmcp import FastMCP
from helm_mcp import create_server as create_helm_server

# Create a composite server
app = FastMCP("my-platform")

# Mount helm-mcp as a sub-server
helm = create_helm_server()
app.mount("helm", helm)

# Add your own tools alongside Helm
@app.tool()
def my_custom_tool(param: str) -> str:
    return f"Custom: {param}"

app.run()
```

### Binary Discovery

The Python package locates the `helm-mcp` Go binary in this order:

1. `HELM_MCP_BINARY` environment variable
2. Bundled binary in the package `bin/` directory
3. Auto-download from GitHub Releases (with SHA256 checksum verification)
4. `helm-mcp` on `PATH`

### Environment Variables

The proxy forwards these environment variables to the Go subprocess:

| Category | Variables |
|----------|-----------|
| Proxy | `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY` (and lowercase variants) |
| Kubernetes | `KUBECONFIG`, `KUBERNETES_SERVICE_HOST`, `KUBERNETES_SERVICE_PORT` |
| Helm | `HELM_CACHE_HOME`, `HELM_CONFIG_HOME`, `HELM_DATA_HOME`, `HELM_PLUGINS`, `HELM_DEBUG` |
| AWS | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, `AWS_REGION`, `AWS_PROFILE` |
| GCP | `GOOGLE_APPLICATION_CREDENTIALS`, `CLOUDSDK_COMPUTE_ZONE` |
| Azure | `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_SUBSCRIPTION_ID` |
| TLS | `SSL_CERT_FILE`, `SSL_CERT_DIR` |

## Security

### Credential Scrubbing

All error messages are automatically scrubbed to remove:
- Bearer tokens (including EKS, GKE, and Azure JWT tokens)
- Basic authentication credentials
- URL-embedded passwords (`https://user:password@host`)

### Input Validation

The security package provides validators for:
- Release names (DNS-1123 compliant)
- Namespace names
- Kubeconfig file paths (path traversal prevention, symlink detection)
- URLs (scheme validation)
- File paths (traversal prevention)
- Timeout durations (max 24h)

### File Permissions

- Repository configuration files are written with `0600` (owner read/write only)
- Config directories are created with `0700` (owner only)

### HTTP Server Hardening

When running in HTTP or SSE mode:
- `ReadTimeout: 30s` — prevents slow client attacks
- `WriteTimeout: 60s` — prevents connection exhaustion
- `IdleTimeout: 120s` — reclaims idle connections
- `MaxHeaderBytes: 1MB` — prevents header-based DoS
- Graceful shutdown with 5-second timeout

### Forward Proxy Support

helm-mcp respects standard proxy environment variables:

```bash
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080
export NO_PROXY=localhost,127.0.0.1,.internal.company.com
```

## Development

### Prerequisites

- Go 1.25+
- Python 3.14+ (for the Python package)
- golangci-lint v2 (optional, for linting)

### Build

```bash
make build        # Build binary
make install      # Install to $GOPATH/bin
make build-all    # Cross-compile for Linux/macOS (amd64/arm64)
```

### Test

```bash
# Go tests (141 tests)
make test         # Run all tests with race detection and coverage
make test-short   # Run tests without integration tests

# Python tests (33 tests)
cd python && pip install -e ".[dev]" && pytest -v tests/
```

### Lint

```bash
make lint         # Run golangci-lint + go vet
make vet          # Run go vet only
```

### Security Check

```bash
make security     # Run govulncheck
```

### Coverage

```bash
make coverage     # Generate coverage report (coverage.html)
```

## Architecture

```
cmd/helm-mcp/          Entry point, transport selection
internal/
  helmengine/           Engine interface and shared types
    v3/                 Helm v3 SDK implementation
    v4/                 Helm v4 SDK implementation
  tools/                MCP tool handlers
    release/            Install, upgrade, uninstall, rollback, list, status, etc.
    chart/              Create, lint, template, package, pull, push, show, etc.
    repo/               Add, list, update, remove, index
    registry/           Login, logout
    search/             Hub, repo
    plugin/             Install, list, uninstall, update
    env/                Env, version
  security/             Input validation, credential scrubbing
  server/               MCP server creation and tool registration
python/                 FastMCP-based Python wrapper
  src/helm_mcp/         Python package source
  tests/                Python tests
```

## Contributing

We welcome contributions from the community! Whether it's bug reports, feature requests, documentation improvements, or code contributions — all help is appreciated.

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines on:
- Setting up your development environment
- Running tests and linters
- Submitting pull requests
- Commit message conventions

## Community

- **Bug reports & feature requests**: [GitHub Issues](https://github.com/SCGIS-Wales/helm-mcp/issues)
- **Discussions & questions**: [GitHub Discussions](https://github.com/SCGIS-Wales/helm-mcp/discussions)
- **Releases**: [GitHub Releases](https://github.com/SCGIS-Wales/helm-mcp/releases) (auto-published on every merge to main)

## License

This project is licensed under the [MIT License](LICENSE) — free to use, modify, and distribute.
