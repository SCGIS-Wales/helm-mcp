"""Shared fixtures for helm-mcp integration tests.

These tests require:
  - A running Kubernetes cluster (k3d, kind, minikube, etc.)
  - The helm-mcp binary built at the repo root
  - Python packages: fastmcp, pytest, pytest-asyncio, pytest-timeout
"""

import subprocess
from pathlib import Path

import pytest
from fastmcp import Client
from fastmcp.client.transports import StdioTransport

REPO_ROOT = Path(__file__).resolve().parent.parent.parent
BINARY = REPO_ROOT / "helm-mcp"

# Namespace used for all integration tests
TEST_NAMESPACE = "helm-mcp-integ"

# Release name for lifecycle tests
TEST_RELEASE = "integ-nginx"


def _run(cmd: str | list[str], **kwargs) -> subprocess.CompletedProcess:
    """Run a command, raising on failure. Prefers list-form (shell=False) for safety."""
    if isinstance(cmd, str):
        cmd = cmd.split()
    return subprocess.run(cmd, capture_output=True, text=True, check=True, **kwargs)


@pytest.fixture(scope="session", autouse=True)
def ensure_binary():
    """Ensure the helm-mcp binary exists."""
    if not BINARY.exists():
        _run(["go", "build", "-o", "helm-mcp", "./cmd/helm-mcp/"], cwd=str(REPO_ROOT))
    assert BINARY.exists(), f"Binary not found at {BINARY}"


@pytest.fixture(scope="session", autouse=True)
def ensure_cluster():
    """Ensure a Kubernetes cluster is accessible."""
    result = subprocess.run(
        ["kubectl", "cluster-info"],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        pytest.skip("No Kubernetes cluster available")


@pytest.fixture(scope="session", autouse=True)
def setup_namespace():
    """Create and tear down the test namespace."""
    # Create namespace: use two-step process to avoid shell=True piping
    create = subprocess.run(
        [
            "kubectl",
            "create",
            "namespace",
            TEST_NAMESPACE,
            "--dry-run=client",
            "-o",
            "yaml",
        ],
        capture_output=True,
        text=True,
    )
    if create.returncode == 0:
        subprocess.run(
            ["kubectl", "apply", "-f", "-"],
            input=create.stdout,
            capture_output=True,
            text=True,
        )
    yield
    subprocess.run(
        [
            "kubectl",
            "delete",
            "namespace",
            TEST_NAMESPACE,
            "--ignore-not-found",
            "--wait=false",
        ],
        capture_output=True,
    )


@pytest.fixture(scope="session")
def bitnami_repo():
    """Ensure bitnami repo is added (HTTPS, non-OCI)."""
    subprocess.run(
        ["helm", "repo", "add", "bitnami", "https://charts.bitnami.com/bitnami"],
        capture_output=True,
    )
    subprocess.run(
        ["helm", "repo", "update"],
        capture_output=True,
    )


@pytest.fixture
async def mcp_client():
    """Create a fresh MCP client connected to helm-mcp via stdio."""
    transport = StdioTransport(
        command=str(BINARY),
        args=["--mode", "stdio"],
    )
    client = Client(transport=transport)
    async with client:
        yield client


async def call_tool(client, name: str, args: dict | None = None):
    """Call an MCP tool without raising on error (returns result with is_error flag)."""
    return await client.call_tool(name, args or {}, raise_on_error=False)


def extract_text(result) -> str:
    """Extract text content from a CallToolResult."""
    if result.content and len(result.content) > 0:
        return result.content[0].text
    return ""
