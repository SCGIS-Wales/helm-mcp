# helm-mcp

A comprehensive MCP (Model Context Protocol) server that exposes **all Helm CLI capabilities** via the native Helm Go SDK. Supports both Helm 3.x and 4.x with a single binary.

## Features

- **44 MCP tools** covering every Helm CLI command (minus shell completion and help)
- **Dual Helm SDK support** — Helm v3 (helm.sh/helm/v3) and Helm v4 (helm.sh/helm/v4)
- **Three transport modes** — stdio (default), HTTP (Streamable HTTP), SSE
- **Full Kubernetes authentication** — kubeconfig, context selection, bearer tokens, TLS config
- **Forward proxy support** — respects `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY` environment variables
- **Credential scrubbing** — tokens, passwords, and secrets are redacted from error output
- **Cloud provider compatible** — EKS, GKE, AKS kubeconfig formats supported

## Quick Start

### Build

```bash
make build
```

### Run (stdio mode for Claude Code, Cursor, etc.)

```bash
./helm-mcp --mode stdio
```

### Run (HTTP mode)

```bash
./helm-mcp --mode http --addr :8080
```

### Run (SSE mode)

```bash
./helm-mcp --mode sse --addr :8080
```

## MCP Client Configuration

### Claude Code

Add to your Claude Code MCP configuration (`~/.claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "helm": {
      "command": "/path/to/helm-mcp",
      "args": ["--mode", "stdio"]
    }
  }
}
```

### Cursor / Windsurf / VS Code

Add to your MCP server configuration:

```json
{
  "helm-mcp": {
    "command": "/path/to/helm-mcp",
    "args": ["--mode", "stdio"]
  }
}
```

### Docker

```bash
docker build -t helm-mcp .
docker run -v ~/.kube:/home/helmuser/.kube:ro helm-mcp --mode stdio
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

## Forward Proxy Support

helm-mcp respects standard proxy environment variables:

```bash
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080
export NO_PROXY=localhost,127.0.0.1,.internal.company.com
```

These are automatically used by Go's `net/http` package (via `http.ProxyFromEnvironment`) for all HTTP operations including:

- Chart downloads (`helm_pull`)
- Repository index updates (`helm_repo_update`)
- Artifact Hub searches (`helm_search_hub`)
- OCI registry operations (`helm_registry_login`, `helm_push`, `helm_pull`)
- Kubernetes API server communication

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

## Development

### Prerequisites

- Go 1.26+
- golangci-lint (optional, for linting)

### Build

```bash
make build        # Build binary
make install      # Install to $GOPATH/bin
make build-all    # Cross-compile for Linux/macOS (amd64/arm64)
```

### Test

```bash
make test         # Run all tests with race detection and coverage
make test-short   # Run tests without integration tests
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
```

## Python Package

A Python wrapper is available that uses [FastMCP](https://github.com/PrefectHQ/fastmcp) to proxy the Go binary. This means new tools added to the Go binary are automatically available in Python without code changes.

```bash
pip install helm-mcp
```

Requires Python 3.14+ and the `helm-mcp` binary on PATH.

```python
from helm_mcp import create_server

server = create_server()
server.run()
```

See [python/README.md](python/README.md) for full documentation.

## License

MIT
