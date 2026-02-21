# Integration Tests

End-to-end integration tests that exercise all 44 helm-mcp MCP tools against a real Kubernetes cluster.

## Prerequisites

| Requirement | Version | Notes |
|-------------|---------|-------|
| Go | 1.24+ | To build the `helm-mcp` binary |
| Python | 3.12+ | Test runner |
| Docker | 24+ | Required by k3d |
| k3d | 5+ | Lightweight K3s-in-Docker (or kind/minikube) |
| helm | 3 or 4 | System CLI used by plugin operations |

### Python packages

```
pip install fastmcp pytest pytest-asyncio pytest-timeout
```

## Quick start

```bash
# 1. Create a k3d cluster (skip if you already have a cluster)
k3d cluster create helm-mcp-test --wait

# 2. Add the bitnami HTTPS chart repo
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# 3. Build the helm-mcp binary
go build -o helm-mcp ./cmd/helm-mcp/

# 4. Run the tests
python3 -m pytest tests/integration/ -v --timeout=600
```

## What is tested

### Test structure (67 tests, 11 classes)

| Class | Tools covered | Description |
|-------|---------------|-------------|
| `TestEnvironmentTools` | `helm_env`, `helm_version` | Environment variables and version info |
| `TestRepositoryTools` | `helm_repo_add`, `helm_repo_list`, `helm_repo_update`, `helm_repo_remove`, `helm_repo_index` | HTTPS repository lifecycle |
| `TestSearchTools` | `helm_search_hub`, `helm_search_repo` | Chart search via Artifact Hub API (v3 and v4) |
| `TestChartInspectionHTTPS` | `helm_show_chart`, `helm_show_values`, `helm_show_readme`, `helm_show_crds`, `helm_show_all` | Chart metadata from HTTPS repos |
| `TestChartInspectionOCI` | Same show tools + `helm_pull`, `helm_template` | OCI registry (Docker Hub + GHCR) |
| `TestChartManagement` | `helm_create`, `helm_lint`, `helm_template`, `helm_package`, `helm_pull`, `helm_dependency_*` | Chart creation, linting, packaging |
| `TestReleaseLifecycle` | `helm_install`, `helm_list`, `helm_status`, `helm_get_*`, `helm_history`, `helm_upgrade`, `helm_rollback`, `helm_test`, `helm_uninstall` | Full release lifecycle with bitnami/nginx |
| `TestPluginTools` | `helm_plugin_list`, `helm_plugin_install`, `helm_plugin_update`, `helm_plugin_uninstall` | Plugin management |
| `TestRegistryTools` | `helm_registry_login`, `helm_registry_logout` | Registry auth (error paths) |
| `TestChartVerifyAndPush` | `helm_verify`, `helm_push` | Chart verification and push (error paths) |
| `TestToolsDiscovery` | All 44 tools | Meta-test verifying tool registration and schemas |

### Protocol coverage

- **HTTPS charts**: Full lifecycle with `bitnami/nginx` via `https://charts.bitnami.com/bitnami`
- **OCI charts**: Full `show_*`, `pull`, and `template` operations against Docker Hub (`oci://registry-1.docker.io/bitnamicharts/nginx`) and GHCR (`oci://ghcr.io/prometheus-community/charts/kube-prometheus-stack`)
- **Artifact Hub search**: Both v3 and v4 engines query the Artifact Hub REST API directly

### Release lifecycle flow

```
install -> list -> status -> get_values -> get_manifest -> get_hooks
  -> get_notes -> get_metadata -> get_all -> history -> upgrade
  -> history -> rollback -> test -> upgrade (dry_run) -> uninstall -> list
```

## CI integration

The integration tests run in GitHub Actions as the `integration-test` job:

1. Starts a k3d cluster inside the runner
2. Builds the `helm-mcp` binary
3. Adds the bitnami chart repository
4. Runs all integration tests via pytest

See `.github/workflows/ci.yml` for details.

## Files

```
tests/integration/
  conftest.py          # Shared fixtures (mcp_client, call_tool, namespace setup)
  test_all_tools.py    # 67 tests covering all 44 tools
  pytest.ini           # asyncio_mode=auto, timeout=600
```
