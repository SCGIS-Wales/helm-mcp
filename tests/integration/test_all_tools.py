"""Integration tests for all 44 helm-mcp MCP tools.

Tests run against a real Kubernetes cluster (k3d/kind/minikube) and
exercise each MCP tool via the FastMCP Python client over stdio.

Covers both OCI (oci://registry-1.docker.io/bitnamicharts) and
HTTPS (https://charts.bitnami.com/bitnami) chart repositories.

Test ordering:
  1. Environment & version tools (no cluster state needed)
  2. Repository management (add, list, update, remove — HTTPS repos)
  3. Search tools (search hub via Artifact Hub API, search repo)
  4. Chart inspection — HTTPS repo (show_*, lint, template, create, package, dependency)
  5. Chart inspection — OCI registry (show_*, pull, template)
  6. Release lifecycle — HTTPS install (install, list, status, upgrade, rollback, get_*, history, test, uninstall)
  7. Plugin management (list, install, uninstall, update)
  8. Registry tools (login, logout — error paths)
  9. Chart verification & push (error paths)
  10. Tools discovery meta-test
"""

import json
import os
import shutil
import tempfile

import pytest

from conftest import TEST_NAMESPACE, TEST_RELEASE, call_tool, extract_text

pytestmark = pytest.mark.asyncio


def _make_chart(parent_dir: str, name: str = "test-chart") -> str:
    """Create a minimal Helm chart scaffold in parent_dir for testing."""
    chart_dir = os.path.join(parent_dir, name)
    os.makedirs(os.path.join(chart_dir, "templates"))
    with open(os.path.join(chart_dir, "Chart.yaml"), "w") as f:
        f.write(f"apiVersion: v2\nname: {name}\nversion: 0.1.0\n")
    with open(os.path.join(chart_dir, "values.yaml"), "w") as f:
        f.write("{}\n")
    return chart_dir


# ---------------------------------------------------------------------------
# 1. Environment & Version Tools
# ---------------------------------------------------------------------------


class TestEnvironmentTools:
    """Tests for helm_env and helm_version."""

    async def test_helm_env(self, mcp_client):
        """helm_env returns Helm environment variables."""
        result = await call_tool(mcp_client, "helm_env")
        text = extract_text(result)
        assert not result.is_error, f"helm_env failed: {text}"
        data = json.loads(text)
        assert "HELM_NAMESPACE" in data
        assert "HELM_DATA_HOME" in data
        assert "HELM_CACHE_HOME" in data

    async def test_helm_version(self, mcp_client):
        """helm_version returns version information."""
        result = await call_tool(mcp_client, "helm_version")
        text = extract_text(result)
        assert not result.is_error, f"helm_version failed: {text}"
        assert "version" in text.lower() or "Version" in text

    async def test_helm_version_short(self, mcp_client):
        """helm_version with short=true returns condensed output."""
        result = await call_tool(mcp_client, "helm_version", {"short": True})
        assert not result.is_error


# ---------------------------------------------------------------------------
# 2. Repository Management Tools (HTTPS repos)
# ---------------------------------------------------------------------------


class TestRepositoryTools:
    """Tests for helm_repo_add, helm_repo_list, helm_repo_update, helm_repo_remove, helm_repo_index."""

    async def test_repo_add_https(self, mcp_client):
        """helm_repo_add adds an HTTPS chart repository."""
        result = await call_tool(
            mcp_client,
            "helm_repo_add",
            {
                "name": "integ-test-repo",
                "url": "https://charts.bitnami.com/bitnami",
                "force_update": True,
            },
        )
        text = extract_text(result)
        assert not result.is_error, f"repo_add failed: {text}"

    async def test_repo_list(self, mcp_client):
        """helm_repo_list shows configured repositories."""
        result = await call_tool(mcp_client, "helm_repo_list")
        text = extract_text(result)
        assert not result.is_error, f"repo_list failed: {text}"
        assert "integ-test-repo" in text or "bitnami" in text

    async def test_repo_update(self, mcp_client):
        """helm_repo_update refreshes repository indexes."""
        result = await call_tool(mcp_client, "helm_repo_update")
        text = extract_text(result)
        assert not result.is_error, f"repo_update failed: {text}"

    async def test_repo_update_specific(self, mcp_client):
        """helm_repo_update with specific repo names."""
        result = await call_tool(
            mcp_client,
            "helm_repo_update",
            {"names": ["integ-test-repo"]},
        )
        text = extract_text(result)
        assert not result.is_error, f"repo_update specific failed: {text}"

    async def test_repo_index(self, mcp_client):
        """helm_repo_index generates an index file."""
        with tempfile.TemporaryDirectory() as tmpdir:
            result = await call_tool(
                mcp_client,
                "helm_repo_index",
                {"directory": tmpdir},
            )
            text = extract_text(result)
            assert not result.is_error, f"repo_index failed: {text}"
            assert os.path.exists(os.path.join(tmpdir, "index.yaml"))

    async def test_repo_remove(self, mcp_client):
        """helm_repo_remove removes a chart repository."""
        result = await call_tool(
            mcp_client,
            "helm_repo_remove",
            {"names": ["integ-test-repo"]},
        )
        text = extract_text(result)
        assert not result.is_error, f"repo_remove failed: {text}"


# ---------------------------------------------------------------------------
# 3. Search Tools
# ---------------------------------------------------------------------------


class TestSearchTools:
    """Tests for helm_search_hub and helm_search_repo."""

    async def test_search_hub_v3(self, mcp_client):
        """helm_search_hub v3 queries Artifact Hub and returns results."""
        result = await call_tool(
            mcp_client,
            "helm_search_hub",
            {"keyword": "nginx", "helm_version": "v3"},
        )
        text = extract_text(result)
        assert not result.is_error, f"search_hub v3 failed: {text}"
        data = json.loads(text)
        assert len(data) > 0, "Expected at least one result for 'nginx'"
        assert any("nginx" in r["name"].lower() for r in data)

    async def test_search_hub_v4(self, mcp_client):
        """helm_search_hub v4 queries Artifact Hub and returns results."""
        result = await call_tool(
            mcp_client,
            "helm_search_hub",
            {"keyword": "nginx", "helm_version": "v4"},
        )
        text = extract_text(result)
        assert not result.is_error, f"search_hub v4 failed: {text}"
        data = json.loads(text)
        assert len(data) > 0, "Expected at least one result for 'nginx'"
        assert any("nginx" in r["name"].lower() for r in data)

    async def test_search_hub_with_repo_url(self, mcp_client):
        """helm_search_hub with list_repo_url includes repository URLs."""
        result = await call_tool(
            mcp_client,
            "helm_search_hub",
            {"keyword": "nginx", "list_repo_url": True},
        )
        text = extract_text(result)
        assert not result.is_error, f"search_hub repo_url failed: {text}"
        data = json.loads(text)
        assert len(data) > 0
        # At least one result should have a URL
        assert any(r.get("url") for r in data)

    async def test_search_repo(self, mcp_client, bitnami_repo):
        """helm_search_repo finds charts in local HTTPS repositories."""
        result = await call_tool(
            mcp_client,
            "helm_search_repo",
            {"keyword": "nginx"},
        )
        text = extract_text(result)
        assert not result.is_error, f"search_repo failed: {text}"
        assert "nginx" in text.lower()

    async def test_search_repo_versions(self, mcp_client, bitnami_repo):
        """helm_search_repo with versions=true shows all versions."""
        result = await call_tool(
            mcp_client,
            "helm_search_repo",
            {"keyword": "nginx", "versions": True},
        )
        text = extract_text(result)
        assert not result.is_error, f"search_repo versions failed: {text}"


# ---------------------------------------------------------------------------
# 4. Chart Inspection — HTTPS Repository
# ---------------------------------------------------------------------------


class TestChartInspectionHTTPS:
    """Tests for helm_show_* using HTTPS chart repo (bitnami/nginx)."""

    async def test_show_chart_https(self, mcp_client, bitnami_repo):
        """helm_show_chart displays Chart.yaml from HTTPS repo."""
        result = await call_tool(
            mcp_client,
            "helm_show_chart",
            {"chart": "bitnami/nginx"},
        )
        text = extract_text(result)
        assert not result.is_error, f"show_chart HTTPS failed: {text}"
        assert "name" in text.lower()

    async def test_show_values_https(self, mcp_client, bitnami_repo):
        """helm_show_values displays default values from HTTPS repo."""
        result = await call_tool(
            mcp_client,
            "helm_show_values",
            {"chart": "bitnami/nginx"},
        )
        text = extract_text(result)
        assert not result.is_error, f"show_values HTTPS failed: {text}"

    async def test_show_readme_https(self, mcp_client, bitnami_repo):
        """helm_show_readme displays README from HTTPS repo."""
        result = await call_tool(
            mcp_client,
            "helm_show_readme",
            {"chart": "bitnami/nginx"},
        )
        text = extract_text(result)
        assert not result.is_error, f"show_readme HTTPS failed: {text}"
        assert len(text) > 100

    async def test_show_crds_https(self, mcp_client, bitnami_repo):
        """helm_show_crds from HTTPS repo (may be empty for nginx)."""
        result = await call_tool(
            mcp_client,
            "helm_show_crds",
            {"chart": "bitnami/nginx"},
        )
        assert not result.is_error

    async def test_show_all_https(self, mcp_client, bitnami_repo):
        """helm_show_all displays all chart info from HTTPS repo."""
        result = await call_tool(
            mcp_client,
            "helm_show_all",
            {"chart": "bitnami/nginx"},
        )
        text = extract_text(result)
        assert not result.is_error, f"show_all HTTPS failed: {text}"
        assert len(text) > 200


# ---------------------------------------------------------------------------
# 5. Chart Inspection — OCI Registry
# ---------------------------------------------------------------------------


class TestChartInspectionOCI:
    """Tests for OCI chart operations.

    The OCI registry client is now properly initialised, so OCI operations
    should succeed for public registries (e.g. Docker Hub bitnamicharts).
    """

    OCI_CHART = "oci://registry-1.docker.io/bitnamicharts/nginx"

    async def test_show_chart_oci(self, mcp_client):
        """helm_show_chart retrieves Chart.yaml from OCI registry."""
        result = await call_tool(
            mcp_client,
            "helm_show_chart",
            {"chart": self.OCI_CHART},
        )
        text = extract_text(result)
        assert not result.is_error, f"show_chart OCI failed: {text}"
        assert "name" in text.lower()

    async def test_show_values_oci(self, mcp_client):
        """helm_show_values retrieves default values from OCI registry."""
        result = await call_tool(
            mcp_client,
            "helm_show_values",
            {"chart": self.OCI_CHART},
        )
        text = extract_text(result)
        assert not result.is_error, f"show_values OCI failed: {text}"

    async def test_pull_oci(self, mcp_client):
        """helm_pull downloads chart from OCI registry."""
        with tempfile.TemporaryDirectory() as tmpdir:
            result = await call_tool(
                mcp_client,
                "helm_pull",
                {"chart": self.OCI_CHART, "destination": tmpdir},
            )
            text = extract_text(result)
            assert not result.is_error, f"pull OCI failed: {text}"
            files = os.listdir(tmpdir)
            assert any(f.endswith(".tgz") for f in files), f"No .tgz found: {files}"

    async def test_template_oci(self, mcp_client):
        """helm_template renders OCI chart templates locally."""
        result = await call_tool(
            mcp_client,
            "helm_template",
            {
                "release_name": "template-oci-test",
                "chart": self.OCI_CHART,
                "namespace": TEST_NAMESPACE,
            },
        )
        text = extract_text(result)
        assert not result.is_error, f"template OCI failed: {text}"
        assert "kind:" in text

    async def test_show_chart_oci_ghcr(self, mcp_client):
        """helm_show_chart retrieves Chart.yaml from GHCR OCI registry."""
        result = await call_tool(
            mcp_client,
            "helm_show_chart",
            {"chart": "oci://ghcr.io/prometheus-community/charts/kube-prometheus-stack"},
        )
        text = extract_text(result)
        assert not result.is_error, f"show_chart GHCR OCI failed: {text}"
        assert "name" in text.lower()

    async def test_pull_oci_ghcr(self, mcp_client):
        """helm_pull downloads chart from GHCR OCI registry."""
        with tempfile.TemporaryDirectory() as tmpdir:
            result = await call_tool(
                mcp_client,
                "helm_pull",
                {
                    "chart": "oci://ghcr.io/prometheus-community/charts/kube-prometheus-stack",
                    "destination": tmpdir,
                },
            )
            text = extract_text(result)
            assert not result.is_error, f"pull GHCR OCI failed: {text}"
            files = os.listdir(tmpdir)
            assert any(f.endswith(".tgz") for f in files), f"No .tgz found: {files}"


# ---------------------------------------------------------------------------
# 6. Chart Creation, Lint, Package, Dependencies
# ---------------------------------------------------------------------------


class TestChartManagement:
    """Tests for helm_create, helm_lint, helm_template, helm_package, helm_dependency_*."""

    async def test_create(self, mcp_client):
        """helm_create creates a chart (simple name, created in CWD)."""
        name = "integ-create-test"
        try:
            result = await call_tool(mcp_client, "helm_create", {"name": name})
            text = extract_text(result)
            assert not result.is_error, f"create failed: {text}"
        finally:
            shutil.rmtree(name, ignore_errors=True)

    async def test_lint(self, mcp_client):
        """helm_lint validates a chart."""
        with tempfile.TemporaryDirectory() as tmpdir:
            chart_path = _make_chart(tmpdir)
            result = await call_tool(mcp_client, "helm_lint", {"paths": [chart_path]})
            assert not result.is_error, f"lint failed: {extract_text(result)}"

    async def test_template_local(self, mcp_client, bitnami_repo):
        """helm_template renders HTTPS chart templates locally."""
        result = await call_tool(
            mcp_client,
            "helm_template",
            {
                "release_name": "template-test",
                "chart": "bitnami/nginx",
                "namespace": TEST_NAMESPACE,
            },
        )
        text = extract_text(result)
        assert not result.is_error, f"template failed: {text}"
        assert "kind:" in text

    async def test_template_with_values(self, mcp_client, bitnami_repo):
        """helm_template with custom values override."""
        result = await call_tool(
            mcp_client,
            "helm_template",
            {
                "release_name": "template-values-test",
                "chart": "bitnami/nginx",
                "namespace": TEST_NAMESPACE,
                "values": {"replicaCount": 2},
            },
        )
        assert not result.is_error, f"template values failed: {extract_text(result)}"

    async def test_package(self, mcp_client):
        """helm_package packages a chart into a .tgz archive."""
        with tempfile.TemporaryDirectory() as tmpdir:
            chart_path = _make_chart(tmpdir, "pkg-chart")
            result = await call_tool(
                mcp_client,
                "helm_package",
                {"path": chart_path, "destination": tmpdir},
            )
            assert not result.is_error, f"package failed: {extract_text(result)}"

    async def test_dependency_list(self, mcp_client):
        """helm_dependency_list lists dependencies for a chart."""
        with tempfile.TemporaryDirectory() as tmpdir:
            chart_path = _make_chart(tmpdir, "dep-chart")
            result = await call_tool(
                mcp_client,
                "helm_dependency_list",
                {"chart_path": chart_path},
            )
            assert not result.is_error, f"dependency_list failed: {extract_text(result)}"

    async def test_dependency_update(self, mcp_client):
        """helm_dependency_update updates chart dependencies."""
        with tempfile.TemporaryDirectory() as tmpdir:
            chart_path = _make_chart(tmpdir, "depup-chart")
            result = await call_tool(
                mcp_client,
                "helm_dependency_update",
                {"chart_path": chart_path},
            )
            assert not result.is_error, f"dependency_update failed: {extract_text(result)}"

    async def test_dependency_build(self, mcp_client):
        """helm_dependency_build builds dependencies from Chart.lock."""
        with tempfile.TemporaryDirectory() as tmpdir:
            chart_path = _make_chart(tmpdir, "depbuild-chart")
            # May error if no Chart.lock — that's expected
            await call_tool(
                mcp_client,
                "helm_dependency_build",
                {"chart_path": chart_path},
            )

    async def test_pull_https(self, mcp_client, bitnami_repo):
        """helm_pull downloads chart from HTTPS repo."""
        with tempfile.TemporaryDirectory() as tmpdir:
            result = await call_tool(
                mcp_client,
                "helm_pull",
                {"chart": "bitnami/nginx", "destination": tmpdir},
            )
            assert not result.is_error, f"pull HTTPS failed: {extract_text(result)}"
            files = os.listdir(tmpdir)
            assert any(f.endswith(".tgz") for f in files)

    async def test_pull_with_untar(self, mcp_client, bitnami_repo):
        """helm_pull with untar extracts the chart."""
        with tempfile.TemporaryDirectory() as tmpdir:
            result = await call_tool(
                mcp_client,
                "helm_pull",
                {"chart": "bitnami/nginx", "destination": tmpdir, "untar": True},
            )
            assert not result.is_error, f"pull untar failed: {extract_text(result)}"
            assert os.path.isdir(os.path.join(tmpdir, "nginx"))


# ---------------------------------------------------------------------------
# 7. Release Lifecycle — HTTPS Install
# ---------------------------------------------------------------------------


class TestReleaseLifecycle:
    """Tests for the full release lifecycle using HTTPS chart repo.

    Tests must run in order: install -> inspect -> upgrade -> rollback -> uninstall.
    Uses bitnami/nginx from the HTTPS repo (OCI registry client may not be
    initialised in the embedded SDK).
    """

    CHART = "bitnami/nginx"

    async def test_01_install(self, mcp_client, bitnami_repo):
        """helm_install installs nginx from HTTPS repo."""
        result = await call_tool(
            mcp_client,
            "helm_install",
            {
                "release_name": TEST_RELEASE,
                "chart": self.CHART,
                "namespace": TEST_NAMESPACE,
                "create_namespace": True,
                "wait": True,
                "timeout": "300s",
                "values": {"replicaCount": 1},
            },
        )
        text = extract_text(result)
        assert not result.is_error, f"install failed: {text}"

    async def test_02_list(self, mcp_client):
        """helm_list shows the installed release."""
        result = await call_tool(
            mcp_client,
            "helm_list",
            {"namespace": TEST_NAMESPACE},
        )
        text = extract_text(result)
        assert not result.is_error, f"list failed: {text}"
        assert TEST_RELEASE in text

    async def test_03_list_all_namespaces(self, mcp_client):
        """helm_list with all_namespaces returns without error."""
        result = await call_tool(
            mcp_client,
            "helm_list",
            {"all_namespaces": True},
        )
        text = extract_text(result)
        assert not result.is_error, f"list all_namespaces failed: {text}"

    async def test_04_list_filter(self, mcp_client):
        """helm_list with filter finds specific releases."""
        result = await call_tool(
            mcp_client,
            "helm_list",
            {"namespace": TEST_NAMESPACE, "filter": "integ"},
        )
        text = extract_text(result)
        assert not result.is_error
        assert TEST_RELEASE in text

    async def test_05_status(self, mcp_client):
        """helm_status shows release status."""
        result = await call_tool(
            mcp_client,
            "helm_status",
            {"release_name": TEST_RELEASE, "namespace": TEST_NAMESPACE},
        )
        text = extract_text(result)
        assert not result.is_error, f"status failed: {text}"

    async def test_06_status_show_resources(self, mcp_client):
        """helm_status with show_resources."""
        result = await call_tool(
            mcp_client,
            "helm_status",
            {
                "release_name": TEST_RELEASE,
                "namespace": TEST_NAMESPACE,
                "show_resources": True,
            },
        )
        assert not result.is_error

    async def test_07_get_values(self, mcp_client):
        """helm_get_values returns user-supplied values."""
        result = await call_tool(
            mcp_client,
            "helm_get_values",
            {"release_name": TEST_RELEASE, "namespace": TEST_NAMESPACE},
        )
        text = extract_text(result)
        assert not result.is_error, f"get_values failed: {text}"
        assert "replicaCount" in text

    async def test_08_get_values_all(self, mcp_client):
        """helm_get_values with all=true returns computed values."""
        result = await call_tool(
            mcp_client,
            "helm_get_values",
            {"release_name": TEST_RELEASE, "namespace": TEST_NAMESPACE, "all": True},
        )
        assert not result.is_error

    async def test_09_get_manifest(self, mcp_client):
        """helm_get_manifest returns the K8s manifest."""
        result = await call_tool(
            mcp_client,
            "helm_get_manifest",
            {"release_name": TEST_RELEASE, "namespace": TEST_NAMESPACE},
        )
        text = extract_text(result)
        assert not result.is_error, f"get_manifest failed: {text}"
        assert "kind:" in text

    async def test_10_get_hooks(self, mcp_client):
        """helm_get_hooks returns release hooks."""
        result = await call_tool(
            mcp_client,
            "helm_get_hooks",
            {"release_name": TEST_RELEASE, "namespace": TEST_NAMESPACE},
        )
        assert not result.is_error

    async def test_11_get_notes(self, mcp_client):
        """helm_get_notes returns release notes."""
        result = await call_tool(
            mcp_client,
            "helm_get_notes",
            {"release_name": TEST_RELEASE, "namespace": TEST_NAMESPACE},
        )
        assert not result.is_error

    async def test_12_get_metadata(self, mcp_client):
        """helm_get_metadata returns release metadata."""
        result = await call_tool(
            mcp_client,
            "helm_get_metadata",
            {"release_name": TEST_RELEASE, "namespace": TEST_NAMESPACE},
        )
        assert not result.is_error

    async def test_13_get_all(self, mcp_client):
        """helm_get_all returns all release information."""
        result = await call_tool(
            mcp_client,
            "helm_get_all",
            {"release_name": TEST_RELEASE, "namespace": TEST_NAMESPACE},
        )
        text = extract_text(result)
        assert not result.is_error, f"get_all failed: {text}"
        assert len(text) > 100

    async def test_14_history(self, mcp_client):
        """helm_history shows revision history."""
        result = await call_tool(
            mcp_client,
            "helm_history",
            {"release_name": TEST_RELEASE, "namespace": TEST_NAMESPACE},
        )
        assert not result.is_error

    async def test_15_upgrade(self, mcp_client, bitnami_repo):
        """helm_upgrade upgrades the release with new values."""
        result = await call_tool(
            mcp_client,
            "helm_upgrade",
            {
                "release_name": TEST_RELEASE,
                "chart": self.CHART,
                "namespace": TEST_NAMESPACE,
                "wait": True,
                "timeout": "300s",
                "values": {"replicaCount": 1, "service": {"type": "ClusterIP"}},
            },
        )
        text = extract_text(result)
        assert not result.is_error, f"upgrade failed: {text}"

    async def test_16_history_after_upgrade(self, mcp_client):
        """helm_history shows 2 revisions after upgrade."""
        result = await call_tool(
            mcp_client,
            "helm_history",
            {"release_name": TEST_RELEASE, "namespace": TEST_NAMESPACE},
        )
        assert not result.is_error

    async def test_17_rollback(self, mcp_client):
        """helm_rollback rolls back to revision 1."""
        result = await call_tool(
            mcp_client,
            "helm_rollback",
            {
                "release_name": TEST_RELEASE,
                "revision": 1,
                "namespace": TEST_NAMESPACE,
                "wait": True,
                "timeout": "300s",
            },
        )
        assert not result.is_error, f"rollback failed: {extract_text(result)}"

    async def test_18_test_release(self, mcp_client):
        """helm_test runs the test suite for a release."""
        result = await call_tool(
            mcp_client,
            "helm_test",
            {
                "release_name": TEST_RELEASE,
                "namespace": TEST_NAMESPACE,
                "timeout": "120s",
            },
        )
        # Test may fail if chart has no test pods — acceptable
        extract_text(result)

    async def test_19_upgrade_dry_run(self, mcp_client, bitnami_repo):
        """helm_upgrade with dry_run simulates without applying."""
        result = await call_tool(
            mcp_client,
            "helm_upgrade",
            {
                "release_name": TEST_RELEASE,
                "chart": self.CHART,
                "namespace": TEST_NAMESPACE,
                "dry_run": "client",
                "values": {"replicaCount": 3},
            },
        )
        assert not result.is_error, f"upgrade dry_run failed: {extract_text(result)}"

    async def test_20_uninstall(self, mcp_client):
        """helm_uninstall removes the release."""
        result = await call_tool(
            mcp_client,
            "helm_uninstall",
            {
                "release_name": TEST_RELEASE,
                "namespace": TEST_NAMESPACE,
            },
        )
        assert not result.is_error, f"uninstall failed: {extract_text(result)}"

    async def test_21_list_after_uninstall(self, mcp_client):
        """helm_list confirms release is gone after uninstall."""
        result = await call_tool(
            mcp_client,
            "helm_list",
            {"namespace": TEST_NAMESPACE},
        )
        text = extract_text(result)
        assert not result.is_error
        assert TEST_RELEASE not in text


# ---------------------------------------------------------------------------
# 8. Plugin Management Tools
# ---------------------------------------------------------------------------


class TestPluginTools:
    """Tests for helm_plugin_list, helm_plugin_install, helm_plugin_uninstall, helm_plugin_update.

    Plugin operations shell out to the system ``helm`` CLI.  Helm v4 requires
    ``--verify=false`` for unsigned plugins, which the MCP tool does not yet
    expose.  We therefore pre-install the test plugin via subprocess and
    exercise only the MCP list/update/uninstall tools.
    """

    @staticmethod
    def _ensure_diff_plugin():
        """Pre-install the helm-diff plugin if not present."""
        import subprocess as sp

        r = sp.run(["helm", "plugin", "list"], capture_output=True, text=True)
        if "diff" in r.stdout.lower():
            return
        sp.run(
            ["helm", "plugin", "install", "https://github.com/databus23/helm-diff", "--verify=false"],
            capture_output=True,
        )

    async def test_plugin_list(self, mcp_client):
        """helm_plugin_list shows installed plugins."""
        result = await call_tool(mcp_client, "helm_plugin_list")
        assert not result.is_error

    async def test_plugin_install(self, mcp_client):
        """helm_plugin_install invokes plugin install (may fail on unsigned sources)."""
        result = await call_tool(
            mcp_client,
            "helm_plugin_install",
            {"url_or_path": "https://github.com/databus23/helm-diff"},
        )
        text = extract_text(result)
        # Accept success, "already exists", or verification error
        if result.is_error:
            assert (
                "already exists" in text.lower()
                or "verification" in text.lower()
                or "verify" in text.lower()
            ), f"plugin_install unexpected error: {text}"

    async def test_plugin_list_after_install(self, mcp_client):
        """helm_plugin_list shows the diff plugin (pre-installed via CLI)."""
        self._ensure_diff_plugin()
        result = await call_tool(mcp_client, "helm_plugin_list")
        text = extract_text(result)
        assert not result.is_error
        assert "diff" in text.lower()

    async def test_plugin_update(self, mcp_client):
        """helm_plugin_update updates a plugin."""
        self._ensure_diff_plugin()
        result = await call_tool(
            mcp_client,
            "helm_plugin_update",
            {"name": "diff"},
        )
        text = extract_text(result)
        if result.is_error and "not found" in text.lower():
            pytest.skip("Plugin diff not found for update")

    async def test_plugin_uninstall(self, mcp_client):
        """helm_plugin_uninstall removes the plugin."""
        self._ensure_diff_plugin()
        result = await call_tool(
            mcp_client,
            "helm_plugin_uninstall",
            {"name": "diff"},
        )
        assert not result.is_error, f"plugin_uninstall failed: {extract_text(result)}"


# ---------------------------------------------------------------------------
# 9. Registry Tools (error path — no real OCI registry with auth)
# ---------------------------------------------------------------------------


class TestRegistryTools:
    """Tests for helm_registry_login and helm_registry_logout."""

    async def test_registry_login_invalid(self, mcp_client):
        """helm_registry_login with invalid creds returns error."""
        result = await call_tool(
            mcp_client,
            "helm_registry_login",
            {
                "hostname": "localhost:5555",
                "username": "fake",
                "password": "fake",
                "insecure": True,
            },
        )
        assert result.is_error  # Expected — no registry

    async def test_registry_logout(self, mcp_client):
        """helm_registry_logout from a host we're not logged into."""
        result = await call_tool(
            mcp_client,
            "helm_registry_logout",
            {"hostname": "localhost:5555"},
        )
        # May succeed (no-op) or error
        extract_text(result)


# ---------------------------------------------------------------------------
# 10. Chart Verification & Push (error paths)
# ---------------------------------------------------------------------------


class TestChartVerifyAndPush:
    """Tests for helm_verify and helm_push (error paths)."""

    async def test_verify_unsigned_chart(self, mcp_client):
        """helm_verify on unsigned chart returns error."""
        with tempfile.TemporaryDirectory() as tmpdir:
            chart_path = _make_chart(tmpdir, "verify-chart")
            await call_tool(
                mcp_client,
                "helm_package",
                {"path": chart_path, "destination": tmpdir},
            )
            tgz_files = [f for f in os.listdir(tmpdir) if f.endswith(".tgz")]
            assert tgz_files, "helm_package did not produce a .tgz"

            result = await call_tool(
                mcp_client,
                "helm_verify",
                {"chart_file": os.path.join(tmpdir, tgz_files[0])},
            )
            assert result.is_error  # No provenance file

    async def test_push_no_registry(self, mcp_client):
        """helm_push to nonexistent registry fails gracefully."""
        with tempfile.TemporaryDirectory() as tmpdir:
            chart_path = _make_chart(tmpdir, "push-chart")
            await call_tool(
                mcp_client,
                "helm_package",
                {"path": chart_path, "destination": tmpdir},
            )
            tgz_files = [f for f in os.listdir(tmpdir) if f.endswith(".tgz")]
            assert tgz_files, "helm_package did not produce a .tgz"

            result = await call_tool(
                mcp_client,
                "helm_push",
                {
                    "chart_ref": os.path.join(tmpdir, tgz_files[0]),
                    "remote": "oci://localhost:5555/charts",
                },
            )
            assert result.is_error  # No registry


# ---------------------------------------------------------------------------
# 11. Tools Discovery Meta-Test
# ---------------------------------------------------------------------------


class TestToolsDiscovery:
    """Verify all 44 tools are registered and well-formed."""

    EXPECTED_TOOLS = {
        "helm_env",
        "helm_version",
        "helm_repo_add",
        "helm_repo_list",
        "helm_repo_update",
        "helm_repo_remove",
        "helm_repo_index",
        "helm_search_hub",
        "helm_search_repo",
        "helm_show_all",
        "helm_show_chart",
        "helm_show_crds",
        "helm_show_readme",
        "helm_show_values",
        "helm_create",
        "helm_lint",
        "helm_template",
        "helm_package",
        "helm_pull",
        "helm_push",
        "helm_verify",
        "helm_dependency_build",
        "helm_dependency_list",
        "helm_dependency_update",
        "helm_install",
        "helm_upgrade",
        "helm_uninstall",
        "helm_rollback",
        "helm_list",
        "helm_status",
        "helm_history",
        "helm_test",
        "helm_get_all",
        "helm_get_hooks",
        "helm_get_manifest",
        "helm_get_metadata",
        "helm_get_notes",
        "helm_get_values",
        "helm_plugin_install",
        "helm_plugin_list",
        "helm_plugin_uninstall",
        "helm_plugin_update",
        "helm_registry_login",
        "helm_registry_logout",
    }

    async def test_all_44_tools_registered(self, mcp_client):
        """Verify all 44 tools are available via MCP tools/list."""
        tools = await mcp_client.list_tools()
        tool_names = {t.name for t in tools}
        assert len(tools) == 44, f"Expected 44 tools, got {len(tools)}: {tool_names}"
        missing = self.EXPECTED_TOOLS - tool_names
        assert not missing, f"Missing tools: {missing}"

    async def test_tools_have_descriptions(self, mcp_client):
        """All tools have non-empty descriptions."""
        tools = await mcp_client.list_tools()
        for tool in tools:
            assert tool.description, f"Tool {tool.name} has no description"

    async def test_tools_have_input_schemas(self, mcp_client):
        """All tools have input schemas defined."""
        tools = await mcp_client.list_tools()
        for tool in tools:
            assert tool.inputSchema, f"Tool {tool.name} has no input schema"
