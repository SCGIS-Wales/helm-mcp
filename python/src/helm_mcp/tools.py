"""Resilient async tool wrappers for all 44 helm-mcp MCP tools.

Provides typed, production-grade async functions wrapping every tool exposed
by the helm-mcp Go binary. Built on top of the FastMCP ``Client`` and designed
for enterprise workloads that demand reliability.

Resilience features
-------------------
- **Auto-reconnect**: If the Go subprocess crashes, the client transparently
  reconnects (up to ``max_reconnects`` times) before raising.
- **Timeouts**: Every tool call respects a configurable ``timeout`` (default
  300 s) via ``asyncio.wait_for``.
- **Structured errors**: Custom exception hierarchy —
  ``HelmError`` > ``HelmTimeoutError`` / ``HelmConnectionError`` / ``HelmToolError``.
- **Graceful shutdown**: ``HelmClient`` is an async context manager that
  cleanly terminates the subprocess on exit.
- **Input validation**: Required parameters are checked *before* the MCP
  round-trip, raising a clear ``ValueError``.
- **Logging**: All calls logged at DEBUG with timing; errors at WARNING.

Usage
-----
Context manager (recommended)::

    from helm_mcp.tools import HelmClient

    async with HelmClient() as helm:
        releases = await helm.list(namespace="default")
        status = await helm.status(release_name="my-app")

Module-level convenience functions::

    from helm_mcp.tools import helm_list, helm_status

    releases = await helm_list(namespace="default")
"""

from __future__ import annotations

import asyncio
import logging
import time
from typing import Any

from fastmcp import Client

from helm_mcp.client import create_client

logger = logging.getLogger("helm_mcp.tools")

# ---------------------------------------------------------------------------
# Exception hierarchy
# ---------------------------------------------------------------------------


class HelmError(Exception):
    """Base class for all helm-mcp tool errors."""


class HelmTimeoutError(HelmError):
    """Raised when a tool call exceeds its timeout."""


class HelmConnectionError(HelmError):
    """Raised when the Go subprocess is unreachable or crashed."""


class HelmToolError(HelmError):
    """Raised when the MCP tool returns an error result.

    Attributes:
        tool_name: Name of the tool that failed.
        error_content: Raw error content from the MCP response.
    """

    def __init__(self, tool_name: str, error_content: Any) -> None:
        self.tool_name = tool_name
        self.error_content = error_content
        super().__init__(f"Tool {tool_name!r} returned error: {error_content}")


# ---------------------------------------------------------------------------
# Default configuration
# ---------------------------------------------------------------------------

DEFAULT_TIMEOUT: float = 300.0  # seconds
DEFAULT_MAX_RECONNECTS: int = 3

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _require(name: str, value: Any) -> None:
    """Validate that a required parameter is provided."""
    if value is None or (isinstance(value, str) and not value.strip()):
        raise ValueError(f"Required parameter {name!r} must be provided (got {value!r})")


def _clean_args(args: dict[str, Any]) -> dict[str, Any]:
    """Remove ``None`` values so only explicitly-set params reach the Go binary."""
    return {k: v for k, v in args.items() if v is not None}


def _extract_text(result: Any) -> Any:
    """Extract text content from an MCP result, handling various FastMCP response shapes.

    When the result is a list with exactly one text block, the raw text is
    returned directly.  If the single text block looks like JSON it is parsed
    and returned as a dict/list so callers get structured data instead of a
    string.  Multiple text blocks are still joined with newlines — but a
    warning is logged because this may indicate truncation or unexpected
    multi-part responses from the Go binary.
    """
    if result is None:
        return result

    # If the result is a CallToolResult (or similar object with .content),
    # unwrap to the inner content list before further processing.
    if hasattr(result, "content") and isinstance(result.content, list):
        return _extract_text(result.content)

    # If the result is a list of content blocks, extract text
    if isinstance(result, list):
        texts = []
        for item in result:
            if hasattr(item, "text"):
                texts.append(item.text)
            elif isinstance(item, dict) and "text" in item:
                texts.append(item["text"])
            else:
                texts.append(str(item))

        if not texts:
            return str(result)

        # Single block — try to return structured JSON if applicable
        if len(texts) == 1:
            raw = texts[0]
            stripped = raw.strip()
            if stripped and stripped[0] in ("{", "["):
                import json

                try:
                    return json.loads(stripped)
                except (json.JSONDecodeError, ValueError):
                    pass
            return raw

        # Multiple blocks — warn and join
        logger.warning(
            "MCP result contained %d content blocks; joining with newline "
            "(may indicate unexpected multi-part response)",
            len(texts),
        )
        return "\n".join(texts)

    # If it has a text attribute directly
    if hasattr(result, "text"):
        return result.text

    return result


# ---------------------------------------------------------------------------
# HelmClient — managed connection with resilience
# ---------------------------------------------------------------------------


class HelmClient:
    """Resilient async client for helm-mcp tools.

    Manages the lifecycle of the Go subprocess and transparently retries
    on transient failures.

    Args:
        binary_path: Explicit path to the helm-mcp Go binary.
        env: Extra environment variables passed to the subprocess.
        timeout: Default timeout (seconds) for every tool call.
        max_reconnects: Maximum reconnection attempts before giving up.
    """

    def __init__(
        self,
        binary_path: str | None = None,
        env: dict[str, str] | None = None,
        timeout: float = DEFAULT_TIMEOUT,
        max_reconnects: int = DEFAULT_MAX_RECONNECTS,
    ) -> None:
        self._binary_path = binary_path
        self._env = env
        self.timeout = timeout
        self.max_reconnects = max_reconnects
        self._client: Client | None = None
        self._connected = False

    # -- lifecycle -----------------------------------------------------------

    async def __aenter__(self) -> HelmClient:
        await self.connect()
        return self

    async def __aexit__(self, *exc: object) -> None:
        await self.close()

    async def connect(self) -> None:
        """Establish the connection to the Go subprocess."""
        try:
            self._client = create_client(
                binary_path=self._binary_path,
                env=self._env,
            )
            await self._client.__aenter__()
            self._connected = True
            logger.debug("connected to helm-mcp subprocess")
        except Exception as exc:
            self._connected = False
            raise HelmConnectionError(f"Failed to connect to helm-mcp: {exc}") from exc

    async def close(self) -> None:
        """Cleanly shut down the subprocess."""
        if self._client is not None:
            try:
                await self._client.__aexit__(None, None, None)
                logger.debug("disconnected from helm-mcp subprocess")
            except Exception:
                logger.warning("error during helm-mcp disconnect", exc_info=True)
            finally:
                self._client = None
                self._connected = False

    async def _reconnect(self, attempt: int = 0) -> None:
        """Reconnect after a subprocess crash with exponential backoff.

        Args:
            attempt: Current retry attempt number (0-indexed), used to
                calculate backoff delay.  Delay = min(2^attempt, 30) seconds.
        """
        delay = min(2**attempt, 30)
        logger.info(
            "attempting to reconnect to helm-mcp subprocess (attempt %d, backoff %.1fs)",
            attempt + 1,
            delay,
        )
        if delay > 0:
            await asyncio.sleep(delay)
        await self.close()
        await self.connect()

    # -- call dispatch -------------------------------------------------------

    async def call_tool(
        self,
        tool_name: str,
        arguments: dict[str, Any],
        timeout: float | None = None,
    ) -> Any:
        """Call an MCP tool with resilience (timeout + auto-reconnect).

        Args:
            tool_name: MCP tool name (e.g. ``"helm_list"``).
            arguments: Tool arguments dict.
            timeout: Per-call timeout override (seconds).

        Returns:
            Parsed tool result.

        Raises:
            HelmConnectionError: Subprocess unreachable after retries.
            HelmTimeoutError: Call exceeded timeout.
            HelmToolError: Tool returned an error result.
        """
        effective_timeout = timeout if timeout is not None else self.timeout
        attempts = 0
        last_exc: Exception | None = None

        while attempts <= self.max_reconnects:
            if not self._connected or self._client is None:
                try:
                    await self._reconnect(attempt=attempts)
                except HelmConnectionError:
                    attempts += 1
                    last_exc = HelmConnectionError(
                        f"Reconnect attempt {attempts}/{self.max_reconnects} failed"
                    )
                    continue

            t0 = time.monotonic()
            try:
                result = await asyncio.wait_for(
                    self._client.call_tool(tool_name, arguments),
                    timeout=effective_timeout,
                )
                elapsed = time.monotonic() - t0
                logger.debug("%s completed in %.2fs", tool_name, elapsed)

                # Check for error content in MCP result.
                # CallToolResult objects expose isError at the top level;
                # older list-of-content-blocks may carry it per item.
                if getattr(result, "isError", False):
                    raise HelmToolError(tool_name, _extract_text(result))

                if isinstance(result, list):
                    for item in result:
                        is_error = getattr(item, "isError", False) or (
                            isinstance(item, dict) and item.get("isError")
                        )
                        if is_error:
                            raise HelmToolError(tool_name, _extract_text(result))

                return _extract_text(result)

            except TimeoutError as exc:
                elapsed = time.monotonic() - t0
                logger.warning("%s timed out after %.2fs", tool_name, elapsed)
                raise HelmTimeoutError(
                    f"Tool {tool_name!r} timed out after {effective_timeout}s"
                ) from exc

            except HelmToolError:
                raise

            except (OSError, ConnectionError, BrokenPipeError) as exc:
                self._connected = False
                attempts += 1
                last_exc = exc
                logger.warning(
                    "%s failed (attempt %d/%d): %s",
                    tool_name,
                    attempts,
                    self.max_reconnects,
                    exc,
                )
                continue

        raise HelmConnectionError(
            f"Failed to call {tool_name!r} after {self.max_reconnects} reconnect attempts"
        ) from last_exc

    # -----------------------------------------------------------------------
    # Release management tools
    # -----------------------------------------------------------------------

    async def list(self, *, timeout: float | None = None, **kwargs: Any) -> Any:
        """List Helm releases."""
        return await self.call_tool("helm_list", _clean_args(kwargs), timeout=timeout)

    async def install(
        self,
        release_name: str,
        chart: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Install a Helm chart as a new release."""
        _require("release_name", release_name)
        _require("chart", chart)
        args = {"release_name": release_name, "chart": chart, **kwargs}
        return await self.call_tool("helm_install", _clean_args(args), timeout=timeout)

    async def upgrade(
        self,
        release_name: str,
        chart: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Upgrade a Helm release."""
        _require("release_name", release_name)
        _require("chart", chart)
        args = {"release_name": release_name, "chart": chart, **kwargs}
        return await self.call_tool("helm_upgrade", _clean_args(args), timeout=timeout)

    async def uninstall(
        self,
        release_name: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Uninstall a Helm release."""
        _require("release_name", release_name)
        args = {"release_name": release_name, **kwargs}
        return await self.call_tool("helm_uninstall", _clean_args(args), timeout=timeout)

    async def rollback(
        self,
        release_name: str,
        revision: int,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Rollback a Helm release to a previous revision."""
        _require("release_name", release_name)
        _require("revision", revision)
        args = {"release_name": release_name, "revision": revision, **kwargs}
        return await self.call_tool("helm_rollback", _clean_args(args), timeout=timeout)

    async def status(
        self,
        release_name: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Get the status of a Helm release."""
        _require("release_name", release_name)
        args = {"release_name": release_name, **kwargs}
        return await self.call_tool("helm_status", _clean_args(args), timeout=timeout)

    async def history(
        self,
        release_name: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Show revision history of a Helm release."""
        _require("release_name", release_name)
        args = {"release_name": release_name, **kwargs}
        return await self.call_tool("helm_history", _clean_args(args), timeout=timeout)

    async def test(
        self,
        release_name: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Run the test suite for a Helm release."""
        _require("release_name", release_name)
        args = {"release_name": release_name, **kwargs}
        return await self.call_tool("helm_test", _clean_args(args), timeout=timeout)

    async def get_all(
        self,
        release_name: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Get all information for a release."""
        _require("release_name", release_name)
        args = {"release_name": release_name, **kwargs}
        return await self.call_tool("helm_get_all", _clean_args(args), timeout=timeout)

    async def get_hooks(
        self,
        release_name: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Get all hooks for a release."""
        _require("release_name", release_name)
        args = {"release_name": release_name, **kwargs}
        return await self.call_tool("helm_get_hooks", _clean_args(args), timeout=timeout)

    async def get_manifest(
        self,
        release_name: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Get the Kubernetes manifest for a release."""
        _require("release_name", release_name)
        args = {"release_name": release_name, **kwargs}
        return await self.call_tool("helm_get_manifest", _clean_args(args), timeout=timeout)

    async def get_metadata(
        self,
        release_name: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Get metadata for a release."""
        _require("release_name", release_name)
        args = {"release_name": release_name, **kwargs}
        return await self.call_tool("helm_get_metadata", _clean_args(args), timeout=timeout)

    async def get_notes(
        self,
        release_name: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Get the notes for a release."""
        _require("release_name", release_name)
        args = {"release_name": release_name, **kwargs}
        return await self.call_tool("helm_get_notes", _clean_args(args), timeout=timeout)

    async def get_values(
        self,
        release_name: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Get the values for a release."""
        _require("release_name", release_name)
        args = {"release_name": release_name, **kwargs}
        return await self.call_tool("helm_get_values", _clean_args(args), timeout=timeout)

    # -----------------------------------------------------------------------
    # Chart management tools
    # -----------------------------------------------------------------------

    async def create(
        self,
        name: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Create a new Helm chart."""
        _require("name", name)
        args = {"name": name, **kwargs}
        return await self.call_tool("helm_create", _clean_args(args), timeout=timeout)

    async def lint(self, *, timeout: float | None = None, **kwargs: Any) -> Any:
        """Lint a Helm chart for possible issues."""
        return await self.call_tool("helm_lint", _clean_args(kwargs), timeout=timeout)

    async def template(
        self,
        release_name: str,
        chart: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Render chart templates locally."""
        _require("release_name", release_name)
        _require("chart", chart)
        args = {"release_name": release_name, "chart": chart, **kwargs}
        return await self.call_tool("helm_template", _clean_args(args), timeout=timeout)

    async def package(
        self,
        path: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Package a chart directory into a .tgz archive."""
        _require("path", path)
        args = {"path": path, **kwargs}
        return await self.call_tool("helm_package", _clean_args(args), timeout=timeout)

    async def pull(
        self,
        chart: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Download a chart from a repository or OCI registry."""
        _require("chart", chart)
        args = {"chart": chart, **kwargs}
        return await self.call_tool("helm_pull", _clean_args(args), timeout=timeout)

    async def push(
        self,
        chart_ref: str,
        remote: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Push a chart archive to an OCI registry."""
        _require("chart_ref", chart_ref)
        _require("remote", remote)
        args = {"chart_ref": chart_ref, "remote": remote, **kwargs}
        return await self.call_tool("helm_push", _clean_args(args), timeout=timeout)

    async def verify(
        self,
        chart_file: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Verify a chart has a valid provenance file."""
        _require("chart_file", chart_file)
        args = {"chart_file": chart_file, **kwargs}
        return await self.call_tool("helm_verify", _clean_args(args), timeout=timeout)

    async def show_all(
        self,
        chart: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Show all information for a chart."""
        _require("chart", chart)
        args = {"chart": chart, **kwargs}
        return await self.call_tool("helm_show_all", _clean_args(args), timeout=timeout)

    async def show_chart(
        self,
        chart: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Show the Chart.yaml of a chart."""
        _require("chart", chart)
        args = {"chart": chart, **kwargs}
        return await self.call_tool("helm_show_chart", _clean_args(args), timeout=timeout)

    async def show_crds(
        self,
        chart: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Show the CRDs of a chart."""
        _require("chart", chart)
        args = {"chart": chart, **kwargs}
        return await self.call_tool("helm_show_crds", _clean_args(args), timeout=timeout)

    async def show_readme(
        self,
        chart: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Show the README of a chart."""
        _require("chart", chart)
        args = {"chart": chart, **kwargs}
        return await self.call_tool("helm_show_readme", _clean_args(args), timeout=timeout)

    async def show_values(
        self,
        chart: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Show the default values of a chart."""
        _require("chart", chart)
        args = {"chart": chart, **kwargs}
        return await self.call_tool("helm_show_values", _clean_args(args), timeout=timeout)

    async def dependency_build(
        self,
        chart_path: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Build out the charts/ directory from Chart.lock."""
        _require("chart_path", chart_path)
        args = {"chart_path": chart_path, **kwargs}
        return await self.call_tool("helm_dependency_build", _clean_args(args), timeout=timeout)

    async def dependency_list(
        self,
        chart_path: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """List the dependencies for a chart."""
        _require("chart_path", chart_path)
        args = {"chart_path": chart_path, **kwargs}
        return await self.call_tool("helm_dependency_list", _clean_args(args), timeout=timeout)

    async def dependency_update(
        self,
        chart_path: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Update charts/ based on Chart.yaml contents."""
        _require("chart_path", chart_path)
        args = {"chart_path": chart_path, **kwargs}
        return await self.call_tool("helm_dependency_update", _clean_args(args), timeout=timeout)

    # -----------------------------------------------------------------------
    # Repository management tools
    # -----------------------------------------------------------------------

    async def repo_add(
        self,
        name: str,
        url: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Add a chart repository."""
        _require("name", name)
        _require("url", url)
        args = {"name": name, "url": url, **kwargs}
        return await self.call_tool("helm_repo_add", _clean_args(args), timeout=timeout)

    async def repo_list(self, *, timeout: float | None = None, **kwargs: Any) -> Any:
        """List configured chart repositories."""
        return await self.call_tool("helm_repo_list", _clean_args(kwargs), timeout=timeout)

    async def repo_update(self, *, timeout: float | None = None, **kwargs: Any) -> Any:
        """Update chart repository indexes."""
        return await self.call_tool("helm_repo_update", _clean_args(kwargs), timeout=timeout)

    async def repo_remove(
        self,
        names: list[str],
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Remove chart repositories."""
        _require("names", names)
        args = {"names": names, **kwargs}
        return await self.call_tool("helm_repo_remove", _clean_args(args), timeout=timeout)

    async def repo_index(
        self,
        directory: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Generate an index file for a directory of chart archives."""
        _require("directory", directory)
        args = {"directory": directory, **kwargs}
        return await self.call_tool("helm_repo_index", _clean_args(args), timeout=timeout)

    # -----------------------------------------------------------------------
    # Registry (OCI) tools
    # -----------------------------------------------------------------------

    async def registry_login(
        self,
        hostname: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Login to an OCI registry."""
        _require("hostname", hostname)
        args = {"hostname": hostname, **kwargs}
        return await self.call_tool("helm_registry_login", _clean_args(args), timeout=timeout)

    async def registry_logout(
        self,
        hostname: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Logout from an OCI registry."""
        _require("hostname", hostname)
        args = {"hostname": hostname, **kwargs}
        return await self.call_tool("helm_registry_logout", _clean_args(args), timeout=timeout)

    # -----------------------------------------------------------------------
    # Search tools
    # -----------------------------------------------------------------------

    async def search_hub(
        self,
        keyword: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Search Artifact Hub for Helm charts."""
        _require("keyword", keyword)
        args = {"keyword": keyword, **kwargs}
        return await self.call_tool("helm_search_hub", _clean_args(args), timeout=timeout)

    async def search_repo(
        self,
        keyword: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Search locally configured repositories for charts."""
        _require("keyword", keyword)
        args = {"keyword": keyword, **kwargs}
        return await self.call_tool("helm_search_repo", _clean_args(args), timeout=timeout)

    # -----------------------------------------------------------------------
    # Plugin management tools
    # -----------------------------------------------------------------------

    async def plugin_install(
        self,
        url_or_path: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Install a Helm plugin."""
        _require("url_or_path", url_or_path)
        args = {"url_or_path": url_or_path, **kwargs}
        return await self.call_tool("helm_plugin_install", _clean_args(args), timeout=timeout)

    async def plugin_list(self, *, timeout: float | None = None, **kwargs: Any) -> Any:
        """List installed Helm plugins."""
        return await self.call_tool("helm_plugin_list", _clean_args(kwargs), timeout=timeout)

    async def plugin_uninstall(
        self,
        name: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Uninstall a Helm plugin."""
        _require("name", name)
        args = {"name": name, **kwargs}
        return await self.call_tool("helm_plugin_uninstall", _clean_args(args), timeout=timeout)

    async def plugin_update(
        self,
        name: str,
        *,
        timeout: float | None = None,
        **kwargs: Any,
    ) -> Any:
        """Update a Helm plugin."""
        _require("name", name)
        args = {"name": name, **kwargs}
        return await self.call_tool("helm_plugin_update", _clean_args(args), timeout=timeout)

    # -----------------------------------------------------------------------
    # Environment tools
    # -----------------------------------------------------------------------

    async def env(self, *, timeout: float | None = None, **kwargs: Any) -> Any:
        """Print Helm environment information."""
        return await self.call_tool("helm_env", _clean_args(kwargs), timeout=timeout)

    async def version(self, *, timeout: float | None = None, **kwargs: Any) -> Any:
        """Print the Helm SDK version information."""
        return await self.call_tool("helm_version", _clean_args(kwargs), timeout=timeout)


# ---------------------------------------------------------------------------
# Module-level singleton client for convenience functions
# ---------------------------------------------------------------------------

_default_client: HelmClient | None = None
_default_lock = asyncio.Lock()


async def _get_default_client() -> HelmClient:
    """Get or create the module-level default client."""
    global _default_client  # noqa: PLW0603
    async with _default_lock:
        if _default_client is None or not _default_client._connected:
            if _default_client is not None:
                await _default_client.close()
            _default_client = HelmClient()
            await _default_client.connect()
        return _default_client


async def close_default_client() -> None:
    """Close the module-level default client (if open).

    Call this during application shutdown for clean cleanup.
    """
    global _default_client  # noqa: PLW0603
    async with _default_lock:
        if _default_client is not None:
            await _default_client.close()
            _default_client = None


# ---------------------------------------------------------------------------
# Module-level convenience functions
# ---------------------------------------------------------------------------


async def helm_list(**kwargs: Any) -> Any:
    """List Helm releases."""
    client = await _get_default_client()
    return await client.list(**kwargs)


async def helm_install(release_name: str, chart: str, **kwargs: Any) -> Any:
    """Install a Helm chart as a new release."""
    client = await _get_default_client()
    return await client.install(release_name, chart, **kwargs)


async def helm_upgrade(release_name: str, chart: str, **kwargs: Any) -> Any:
    """Upgrade a Helm release."""
    client = await _get_default_client()
    return await client.upgrade(release_name, chart, **kwargs)


async def helm_uninstall(release_name: str, **kwargs: Any) -> Any:
    """Uninstall a Helm release."""
    client = await _get_default_client()
    return await client.uninstall(release_name, **kwargs)


async def helm_rollback(release_name: str, revision: int, **kwargs: Any) -> Any:
    """Rollback a Helm release to a previous revision."""
    client = await _get_default_client()
    return await client.rollback(release_name, revision, **kwargs)


async def helm_status(release_name: str, **kwargs: Any) -> Any:
    """Get the status of a Helm release."""
    client = await _get_default_client()
    return await client.status(release_name, **kwargs)


async def helm_history(release_name: str, **kwargs: Any) -> Any:
    """Show revision history of a Helm release."""
    client = await _get_default_client()
    return await client.history(release_name, **kwargs)


async def helm_test(release_name: str, **kwargs: Any) -> Any:
    """Run the test suite for a Helm release."""
    client = await _get_default_client()
    return await client.test(release_name, **kwargs)


async def helm_get_all(release_name: str, **kwargs: Any) -> Any:
    """Get all information for a release."""
    client = await _get_default_client()
    return await client.get_all(release_name, **kwargs)


async def helm_get_hooks(release_name: str, **kwargs: Any) -> Any:
    """Get all hooks for a release."""
    client = await _get_default_client()
    return await client.get_hooks(release_name, **kwargs)


async def helm_get_manifest(release_name: str, **kwargs: Any) -> Any:
    """Get the Kubernetes manifest for a release."""
    client = await _get_default_client()
    return await client.get_manifest(release_name, **kwargs)


async def helm_get_metadata(release_name: str, **kwargs: Any) -> Any:
    """Get metadata for a release."""
    client = await _get_default_client()
    return await client.get_metadata(release_name, **kwargs)


async def helm_get_notes(release_name: str, **kwargs: Any) -> Any:
    """Get the notes for a release."""
    client = await _get_default_client()
    return await client.get_notes(release_name, **kwargs)


async def helm_get_values(release_name: str, **kwargs: Any) -> Any:
    """Get the values for a release."""
    client = await _get_default_client()
    return await client.get_values(release_name, **kwargs)


async def helm_create(name: str, **kwargs: Any) -> Any:
    """Create a new Helm chart."""
    client = await _get_default_client()
    return await client.create(name, **kwargs)


async def helm_lint(**kwargs: Any) -> Any:
    """Lint a Helm chart for possible issues."""
    client = await _get_default_client()
    return await client.lint(**kwargs)


async def helm_template(release_name: str, chart: str, **kwargs: Any) -> Any:
    """Render chart templates locally."""
    client = await _get_default_client()
    return await client.template(release_name, chart, **kwargs)


async def helm_package(path: str, **kwargs: Any) -> Any:
    """Package a chart directory into a .tgz archive."""
    client = await _get_default_client()
    return await client.package(path, **kwargs)


async def helm_pull(chart: str, **kwargs: Any) -> Any:
    """Download a chart from a repository or OCI registry."""
    client = await _get_default_client()
    return await client.pull(chart, **kwargs)


async def helm_push(chart_ref: str, remote: str, **kwargs: Any) -> Any:
    """Push a chart archive to an OCI registry."""
    client = await _get_default_client()
    return await client.push(chart_ref, remote, **kwargs)


async def helm_verify(chart_file: str, **kwargs: Any) -> Any:
    """Verify a chart has a valid provenance file."""
    client = await _get_default_client()
    return await client.verify(chart_file, **kwargs)


async def helm_show_all(chart: str, **kwargs: Any) -> Any:
    """Show all information for a chart."""
    client = await _get_default_client()
    return await client.show_all(chart, **kwargs)


async def helm_show_chart(chart: str, **kwargs: Any) -> Any:
    """Show the Chart.yaml of a chart."""
    client = await _get_default_client()
    return await client.show_chart(chart, **kwargs)


async def helm_show_crds(chart: str, **kwargs: Any) -> Any:
    """Show the CRDs of a chart."""
    client = await _get_default_client()
    return await client.show_crds(chart, **kwargs)


async def helm_show_readme(chart: str, **kwargs: Any) -> Any:
    """Show the README of a chart."""
    client = await _get_default_client()
    return await client.show_readme(chart, **kwargs)


async def helm_show_values(chart: str, **kwargs: Any) -> Any:
    """Show the default values of a chart."""
    client = await _get_default_client()
    return await client.show_values(chart, **kwargs)


async def helm_dependency_build(chart_path: str, **kwargs: Any) -> Any:
    """Build out the charts/ directory from Chart.lock."""
    client = await _get_default_client()
    return await client.dependency_build(chart_path, **kwargs)


async def helm_dependency_list(chart_path: str, **kwargs: Any) -> Any:
    """List the dependencies for a chart."""
    client = await _get_default_client()
    return await client.dependency_list(chart_path, **kwargs)


async def helm_dependency_update(chart_path: str, **kwargs: Any) -> Any:
    """Update charts/ based on Chart.yaml contents."""
    client = await _get_default_client()
    return await client.dependency_update(chart_path, **kwargs)


async def helm_repo_add(name: str, url: str, **kwargs: Any) -> Any:
    """Add a chart repository."""
    client = await _get_default_client()
    return await client.repo_add(name, url, **kwargs)


async def helm_repo_list(**kwargs: Any) -> Any:
    """List configured chart repositories."""
    client = await _get_default_client()
    return await client.repo_list(**kwargs)


async def helm_repo_update(**kwargs: Any) -> Any:
    """Update chart repository indexes."""
    client = await _get_default_client()
    return await client.repo_update(**kwargs)


async def helm_repo_remove(names: list[str], **kwargs: Any) -> Any:
    """Remove chart repositories."""
    client = await _get_default_client()
    return await client.repo_remove(names, **kwargs)


async def helm_repo_index(directory: str, **kwargs: Any) -> Any:
    """Generate an index file for a directory of chart archives."""
    client = await _get_default_client()
    return await client.repo_index(directory, **kwargs)


async def helm_registry_login(hostname: str, **kwargs: Any) -> Any:
    """Login to an OCI registry."""
    client = await _get_default_client()
    return await client.registry_login(hostname, **kwargs)


async def helm_registry_logout(hostname: str, **kwargs: Any) -> Any:
    """Logout from an OCI registry."""
    client = await _get_default_client()
    return await client.registry_logout(hostname, **kwargs)


async def helm_search_hub(keyword: str, **kwargs: Any) -> Any:
    """Search Artifact Hub for Helm charts."""
    client = await _get_default_client()
    return await client.search_hub(keyword, **kwargs)


async def helm_search_repo(keyword: str, **kwargs: Any) -> Any:
    """Search locally configured repositories for charts."""
    client = await _get_default_client()
    return await client.search_repo(keyword, **kwargs)


async def helm_plugin_install(url_or_path: str, **kwargs: Any) -> Any:
    """Install a Helm plugin."""
    client = await _get_default_client()
    return await client.plugin_install(url_or_path, **kwargs)


async def helm_plugin_list(**kwargs: Any) -> Any:
    """List installed Helm plugins."""
    client = await _get_default_client()
    return await client.plugin_list(**kwargs)


async def helm_plugin_uninstall(name: str, **kwargs: Any) -> Any:
    """Uninstall a Helm plugin."""
    client = await _get_default_client()
    return await client.plugin_uninstall(name, **kwargs)


async def helm_plugin_update(name: str, **kwargs: Any) -> Any:
    """Update a Helm plugin."""
    client = await _get_default_client()
    return await client.plugin_update(name, **kwargs)


async def helm_env(**kwargs: Any) -> Any:
    """Print Helm environment information."""
    client = await _get_default_client()
    return await client.env(**kwargs)


async def helm_version(**kwargs: Any) -> Any:
    """Print the Helm SDK version information."""
    client = await _get_default_client()
    return await client.version(**kwargs)
