"""Shared fixtures for helm-mcp integration tests.

These tests require:
  - A running Kubernetes cluster (k3d, kind, minikube, etc.)
  - The helm-mcp binary built at the repo root
  - Python packages: fastmcp, pytest, pytest-asyncio, pytest-timeout
"""

import os
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


def _run(cmd: str, **kwargs) -> subprocess.CompletedProcess:
    """Run a shell command, raising on failure."""
    return subprocess.run(cmd, shell=True, capture_output=True, text=True, check=True, **kwargs)


@pytest.fixture(scope="session", autouse=True)
def ensure_binary():
    """Ensure the helm-mcp binary exists."""
    if not BINARY.exists():
        _run(f"cd {REPO_ROOT} && go build -o helm-mcp ./cmd/helm-mcp/")
    assert BINARY.exists(), f"Binary not found at {BINARY}"


@pytest.fixture(scope="session", autouse=True)
def ensure_cluster():
    """Ensure a Kubernetes cluster is accessible."""
    result = subprocess.run(
        "kubectl cluster-info",
        shell=True,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        pytest.skip("No Kubernetes cluster available")


@pytest.fixture(scope="session", autouse=True)
def setup_namespace():
    """Create and tear down the test namespace."""
    subprocess.run(
        f"kubectl create namespace {TEST_NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -",
        shell=True,
        capture_output=True,
    )
    yield
    subprocess.run(
        f"kubectl delete namespace {TEST_NAMESPACE} --ignore-not-found --wait=false",
        shell=True,
        capture_output=True,
    )


@pytest.fixture(scope="session")
def bitnami_repo():
    """Ensure bitnami repo is added (HTTPS, non-OCI)."""
    subprocess.run(
        "helm repo add bitnami https://charts.bitnami.com/bitnami 2>/dev/null; helm repo update",
        shell=True,
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
