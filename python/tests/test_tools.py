"""Tests for helm_mcp.tools — resilient async tool wrappers."""

import asyncio
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from helm_mcp.tools import (
    DEFAULT_MAX_RECONNECTS,
    DEFAULT_TIMEOUT,
    HelmClient,
    HelmConnectionError,
    HelmError,
    HelmTimeoutError,
    HelmToolError,
    _clean_args,
    _extract_text,
    _require,
    close_default_client,
    helm_list,
    helm_status,
)

# ---------------------------------------------------------------------------
# Exception hierarchy
# ---------------------------------------------------------------------------


class TestExceptionHierarchy:
    """Test the custom exception hierarchy."""

    def test_helm_error_is_base(self):
        assert issubclass(HelmTimeoutError, HelmError)
        assert issubclass(HelmConnectionError, HelmError)
        assert issubclass(HelmToolError, HelmError)

    def test_helm_error_is_exception(self):
        assert issubclass(HelmError, Exception)

    def test_helm_tool_error_attributes(self):
        err = HelmToolError("helm_list", {"message": "namespace not found"})
        assert err.tool_name == "helm_list"
        assert err.error_content == {"message": "namespace not found"}
        assert "helm_list" in str(err)

    def test_helm_timeout_error_message(self):
        err = HelmTimeoutError("timed out after 5s")
        assert "timed out" in str(err)

    def test_helm_connection_error_message(self):
        err = HelmConnectionError("subprocess crashed")
        assert "subprocess crashed" in str(err)


# ---------------------------------------------------------------------------
# Helper functions
# ---------------------------------------------------------------------------


class TestHelpers:
    """Test helper functions."""

    def test_require_valid_string(self):
        _require("name", "my-release")  # Should not raise

    def test_require_none_raises(self):
        with pytest.raises(ValueError, match="release_name"):
            _require("release_name", None)

    def test_require_empty_string_raises(self):
        with pytest.raises(ValueError, match="chart"):
            _require("chart", "")

    def test_require_whitespace_only_raises(self):
        with pytest.raises(ValueError, match="chart"):
            _require("chart", "   ")

    def test_require_non_string(self):
        _require("revision", 42)  # Should not raise for non-string

    def test_require_zero_is_valid(self):
        _require("revision", 0)  # Zero is valid for non-string

    def test_clean_args_removes_none(self):
        result = _clean_args({"a": 1, "b": None, "c": "hello", "d": None})
        assert result == {"a": 1, "c": "hello"}

    def test_clean_args_empty(self):
        assert _clean_args({}) == {}

    def test_clean_args_all_none(self):
        assert _clean_args({"a": None, "b": None}) == {}

    def test_clean_args_preserves_false(self):
        result = _clean_args({"a": False, "b": 0, "c": ""})
        assert result == {"a": False, "b": 0, "c": ""}

    def test_extract_text_none(self):
        assert _extract_text(None) is None

    def test_extract_text_string(self):
        assert _extract_text("hello") == "hello"

    def test_extract_text_with_text_attribute(self):
        obj = MagicMock()
        obj.text = "content"
        assert _extract_text(obj) == "content"

    def test_extract_text_list_of_text_objects(self):
        items = [MagicMock(text="line1"), MagicMock(text="line2")]
        result = _extract_text(items)
        assert result == "line1\nline2"

    def test_extract_text_list_of_dicts(self):
        items = [{"text": "a"}, {"text": "b"}]
        result = _extract_text(items)
        assert result == "a\nb"


# ---------------------------------------------------------------------------
# HelmClient lifecycle
# ---------------------------------------------------------------------------


class TestHelmClientLifecycle:
    """Test HelmClient construction and configuration."""

    def test_default_config(self):
        client = HelmClient()
        assert client.timeout == DEFAULT_TIMEOUT
        assert client.max_reconnects == DEFAULT_MAX_RECONNECTS
        assert client._connected is False
        assert client._client is None

    def test_custom_config(self):
        client = HelmClient(timeout=60.0, max_reconnects=5)
        assert client.timeout == 60.0
        assert client.max_reconnects == 5

    @pytest.mark.asyncio
    async def test_connect_success(self):
        mock_fastmcp_client = AsyncMock()
        mock_fastmcp_client.__aenter__ = AsyncMock(return_value=mock_fastmcp_client)

        with patch("helm_mcp.tools.create_client", return_value=mock_fastmcp_client):
            client = HelmClient()
            await client.connect()
            assert client._connected is True
            await client.close()

    @pytest.mark.asyncio
    async def test_connect_failure_raises_connection_error(self):
        with patch("helm_mcp.tools.create_client", side_effect=FileNotFoundError("not found")):
            client = HelmClient()
            with pytest.raises(HelmConnectionError, match="Failed to connect"):
                await client.connect()
            assert client._connected is False

    @pytest.mark.asyncio
    async def test_context_manager(self):
        mock_fastmcp_client = AsyncMock()
        mock_fastmcp_client.__aenter__ = AsyncMock(return_value=mock_fastmcp_client)
        mock_fastmcp_client.__aexit__ = AsyncMock(return_value=None)

        with patch("helm_mcp.tools.create_client", return_value=mock_fastmcp_client):
            async with HelmClient() as helm:
                assert helm._connected is True
            assert helm._connected is False

    @pytest.mark.asyncio
    async def test_close_idempotent(self):
        client = HelmClient()
        await client.close()  # Should not raise even without connect
        await client.close()  # Double close should be safe


# ---------------------------------------------------------------------------
# Tool call dispatch — happy path
# ---------------------------------------------------------------------------


class TestToolCallHappyPath:
    """Test successful tool calls."""

    @pytest.mark.asyncio
    async def test_call_tool_returns_result(self):
        mock_client = AsyncMock()
        mock_content = MagicMock(text="release-data", isError=False)
        mock_client.call_tool = AsyncMock(return_value=[mock_content])

        helm = HelmClient()
        helm._client = mock_client
        helm._connected = True

        result = await helm.call_tool("helm_list", {})
        assert result == "release-data"
        mock_client.call_tool.assert_called_once_with("helm_list", {})

    @pytest.mark.asyncio
    async def test_list_passes_kwargs(self):
        mock_client = AsyncMock()
        mock_content = MagicMock(text="data", isError=False)
        mock_client.call_tool = AsyncMock(return_value=[mock_content])

        helm = HelmClient()
        helm._client = mock_client
        helm._connected = True

        await helm.list(namespace="default", all_namespaces=True)
        mock_client.call_tool.assert_called_once_with(
            "helm_list", {"namespace": "default", "all_namespaces": True}
        )

    @pytest.mark.asyncio
    async def test_install_requires_params(self):
        helm = HelmClient()
        helm._client = AsyncMock()
        helm._connected = True

        with pytest.raises(ValueError, match="release_name"):
            await helm.install("", "chart")

        with pytest.raises(ValueError, match="chart"):
            await helm.install("my-release", "")

    @pytest.mark.asyncio
    async def test_status_calls_correct_tool(self):
        mock_client = AsyncMock()
        mock_content = MagicMock(text="deployed", isError=False)
        mock_client.call_tool = AsyncMock(return_value=[mock_content])

        helm = HelmClient()
        helm._client = mock_client
        helm._connected = True

        result = await helm.status("my-release")
        mock_client.call_tool.assert_called_once_with("helm_status", {"release_name": "my-release"})
        assert result == "deployed"


# ---------------------------------------------------------------------------
# Timeout handling
# ---------------------------------------------------------------------------


class TestTimeout:
    """Test timeout handling."""

    @pytest.mark.asyncio
    async def test_timeout_raises_helm_timeout_error(self):
        mock_client = AsyncMock()

        async def slow_call(*args, **kwargs):
            await asyncio.sleep(10)

        mock_client.call_tool = slow_call

        helm = HelmClient(timeout=0.01)
        helm._client = mock_client
        helm._connected = True

        with pytest.raises(HelmTimeoutError, match="timed out"):
            await helm.call_tool("helm_list", {})

    @pytest.mark.asyncio
    async def test_per_call_timeout_override(self):
        mock_client = AsyncMock()

        async def slow_call(*args, **kwargs):
            await asyncio.sleep(10)

        mock_client.call_tool = slow_call

        helm = HelmClient(timeout=300)  # Long default
        helm._client = mock_client
        helm._connected = True

        with pytest.raises(HelmTimeoutError):
            await helm.call_tool("helm_list", {}, timeout=0.01)


# ---------------------------------------------------------------------------
# Connection error and auto-reconnect
# ---------------------------------------------------------------------------


class TestReconnect:
    """Test auto-reconnect on connection failures."""

    @pytest.mark.asyncio
    async def test_reconnect_on_connection_error(self):
        call_count = 0

        mock_client = AsyncMock()

        async def failing_then_success(tool_name, args):
            nonlocal call_count
            call_count += 1
            if call_count == 1:
                raise ConnectionError("subprocess died")
            return [MagicMock(text="recovered", isError=False)]

        mock_client.call_tool = failing_then_success

        mock_new_client = AsyncMock()
        mock_new_client.__aenter__ = AsyncMock(return_value=mock_new_client)
        mock_new_client.__aexit__ = AsyncMock(return_value=None)
        mock_content = MagicMock(text="recovered", isError=False)
        mock_new_client.call_tool = AsyncMock(return_value=[mock_content])

        helm = HelmClient(max_reconnects=3)
        helm._client = mock_client
        helm._connected = True

        with patch("helm_mcp.tools.create_client", return_value=mock_new_client):
            result = await helm.call_tool("helm_list", {})
            assert result == "recovered"

    @pytest.mark.asyncio
    async def test_max_reconnects_exceeded(self):
        mock_client = AsyncMock()
        mock_client.call_tool = AsyncMock(side_effect=ConnectionError("dead"))

        mock_new_client = AsyncMock()
        mock_new_client.__aenter__ = AsyncMock(return_value=mock_new_client)
        mock_new_client.__aexit__ = AsyncMock(return_value=None)
        mock_new_client.call_tool = AsyncMock(side_effect=ConnectionError("still dead"))

        helm = HelmClient(max_reconnects=2)
        helm._client = mock_client
        helm._connected = True

        with (
            patch("helm_mcp.tools.create_client", return_value=mock_new_client),
            pytest.raises(HelmConnectionError, match="after 2 reconnect"),
        ):
            await helm.call_tool("helm_list", {})


# ---------------------------------------------------------------------------
# Tool error handling
# ---------------------------------------------------------------------------


class TestToolError:
    """Test MCP tool error response handling."""

    @pytest.mark.asyncio
    async def test_tool_error_raises_helm_tool_error(self):
        mock_client = AsyncMock()
        error_content = MagicMock(text="namespace not found", isError=True)
        mock_client.call_tool = AsyncMock(return_value=[error_content])

        helm = HelmClient()
        helm._client = mock_client
        helm._connected = True

        with pytest.raises(HelmToolError, match="helm_status") as exc_info:
            await helm.call_tool("helm_status", {"release_name": "missing"})
        assert exc_info.value.tool_name == "helm_status"


# ---------------------------------------------------------------------------
# Input validation on tool methods
# ---------------------------------------------------------------------------


class TestInputValidation:
    """Test that tool methods validate required inputs."""

    @pytest.mark.asyncio
    async def test_install_requires_release_name(self):
        helm = HelmClient()
        helm._connected = True
        with pytest.raises(ValueError, match="release_name"):
            await helm.install(None, "chart")

    @pytest.mark.asyncio
    async def test_upgrade_requires_chart(self):
        helm = HelmClient()
        helm._connected = True
        with pytest.raises(ValueError, match="chart"):
            await helm.upgrade("release", None)

    @pytest.mark.asyncio
    async def test_uninstall_requires_release_name(self):
        helm = HelmClient()
        helm._connected = True
        with pytest.raises(ValueError, match="release_name"):
            await helm.uninstall("")

    @pytest.mark.asyncio
    async def test_rollback_requires_revision(self):
        helm = HelmClient()
        helm._connected = True
        with pytest.raises(ValueError, match="revision"):
            await helm.rollback("release", None)

    @pytest.mark.asyncio
    async def test_status_requires_release_name(self):
        helm = HelmClient()
        helm._connected = True
        with pytest.raises(ValueError, match="release_name"):
            await helm.status("")

    @pytest.mark.asyncio
    async def test_repo_add_requires_name_and_url(self):
        helm = HelmClient()
        helm._connected = True
        with pytest.raises(ValueError, match="name"):
            await helm.repo_add("", "https://charts.example.com")
        with pytest.raises(ValueError, match="url"):
            await helm.repo_add("bitnami", "")

    @pytest.mark.asyncio
    async def test_search_hub_requires_keyword(self):
        helm = HelmClient()
        helm._connected = True
        with pytest.raises(ValueError, match="keyword"):
            await helm.search_hub("")

    @pytest.mark.asyncio
    async def test_push_requires_chart_ref_and_remote(self):
        helm = HelmClient()
        helm._connected = True
        with pytest.raises(ValueError, match="chart_ref"):
            await helm.push("", "oci://registry.example.com")
        with pytest.raises(ValueError, match="remote"):
            await helm.push("mychart-0.1.0.tgz", "")

    @pytest.mark.asyncio
    async def test_create_requires_name(self):
        helm = HelmClient()
        helm._connected = True
        with pytest.raises(ValueError, match="name"):
            await helm.create("")

    @pytest.mark.asyncio
    async def test_template_requires_release_name_and_chart(self):
        helm = HelmClient()
        helm._connected = True
        with pytest.raises(ValueError, match="release_name"):
            await helm.template("", "chart")
        with pytest.raises(ValueError, match="chart"):
            await helm.template("release", "")


# ---------------------------------------------------------------------------
# All 44 tool methods exist
# ---------------------------------------------------------------------------


class TestAllToolsExist:
    """Verify all 44 tool methods are defined on HelmClient."""

    EXPECTED_METHODS = [
        # Release management (14)
        "list",
        "install",
        "upgrade",
        "uninstall",
        "rollback",
        "status",
        "history",
        "test",
        "get_all",
        "get_hooks",
        "get_manifest",
        "get_metadata",
        "get_notes",
        "get_values",
        # Chart management (15)
        "create",
        "lint",
        "template",
        "package",
        "pull",
        "push",
        "verify",
        "show_all",
        "show_chart",
        "show_crds",
        "show_readme",
        "show_values",
        "dependency_build",
        "dependency_list",
        "dependency_update",
        # Repository management (5)
        "repo_add",
        "repo_list",
        "repo_update",
        "repo_remove",
        "repo_index",
        # Registry (2)
        "registry_login",
        "registry_logout",
        # Search (2)
        "search_hub",
        "search_repo",
        # Plugin (4)
        "plugin_install",
        "plugin_list",
        "plugin_uninstall",
        "plugin_update",
        # Environment (2)
        "env",
        "version",
    ]

    def test_all_methods_exist(self):
        for method_name in self.EXPECTED_METHODS:
            assert hasattr(HelmClient, method_name), f"HelmClient missing method: {method_name}"
            assert callable(getattr(HelmClient, method_name)), (
                f"HelmClient.{method_name} is not callable"
            )

    def test_method_count(self):
        assert len(self.EXPECTED_METHODS) == 44


# ---------------------------------------------------------------------------
# Module-level convenience functions
# ---------------------------------------------------------------------------


class TestModuleLevelFunctions:
    """Test module-level convenience functions."""

    @pytest.mark.asyncio
    async def test_helm_list_uses_default_client(self):
        mock_client = AsyncMock()
        mock_content = MagicMock(text="releases", isError=False)
        mock_client.call_tool = AsyncMock(return_value=[mock_content])

        mock_helm_client = HelmClient()
        mock_helm_client._client = mock_client
        mock_helm_client._connected = True

        with patch("helm_mcp.tools._get_default_client", return_value=mock_helm_client):
            result = await helm_list(namespace="default")
            assert result == "releases"

    @pytest.mark.asyncio
    async def test_helm_status_uses_default_client(self):
        mock_client = AsyncMock()
        mock_content = MagicMock(text="deployed", isError=False)
        mock_client.call_tool = AsyncMock(return_value=[mock_content])

        mock_helm_client = HelmClient()
        mock_helm_client._client = mock_client
        mock_helm_client._connected = True

        with patch("helm_mcp.tools._get_default_client", return_value=mock_helm_client):
            result = await helm_status("my-release")
            assert result == "deployed"

    @pytest.mark.asyncio
    async def test_close_default_client_safe_when_none(self):
        """Closing default client when none exists should not raise."""
        import helm_mcp.tools as tools_mod

        with patch.object(tools_mod, "_default_client", None):
            await close_default_client()


# ---------------------------------------------------------------------------
# Concurrent calls
# ---------------------------------------------------------------------------


class TestConcurrentCalls:
    """Test concurrent tool calls on the same client."""

    @pytest.mark.asyncio
    async def test_concurrent_calls(self):
        mock_client = AsyncMock()

        call_count = 0

        async def tracked_call(tool_name, args):
            nonlocal call_count
            call_count += 1
            await asyncio.sleep(0.01)  # Simulate some work
            return [MagicMock(text=f"result-{tool_name}", isError=False)]

        mock_client.call_tool = tracked_call

        helm = HelmClient()
        helm._client = mock_client
        helm._connected = True

        results = await asyncio.gather(
            helm.list(namespace="ns1"),
            helm.list(namespace="ns2"),
            helm.status("release1"),
        )

        assert len(results) == 3
        assert call_count == 3


# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------


class TestConstants:
    """Test default configuration constants."""

    def test_default_timeout(self):
        assert DEFAULT_TIMEOUT == 300.0

    def test_default_max_reconnects(self):
        assert DEFAULT_MAX_RECONNECTS == 3
