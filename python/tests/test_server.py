"""Tests for helm_mcp server and client modules."""

import os
import platform
import shutil
from pathlib import Path
from unittest.mock import patch

import pytest

# ---------------------------------------------------------------------------
# Package metadata
# ---------------------------------------------------------------------------


def test_version():
    """Test package version is set."""
    from helm_mcp import __version__

    assert __version__  # version is set (value managed by auto-tag)


def test_exports():
    """Test that public API is properly exported."""
    import helm_mcp

    assert hasattr(helm_mcp, "create_server")
    assert hasattr(helm_mcp, "create_client")
    assert hasattr(helm_mcp, "__version__")
    assert callable(helm_mcp.create_server)
    assert callable(helm_mcp.create_client)


def test_all_exports():
    """Test __all__ contains core exports and tool wrappers."""
    import helm_mcp

    all_exports = set(helm_mcp.__all__)
    # Core exports
    assert "__version__" in all_exports
    assert "create_server" in all_exports
    assert "create_client" in all_exports
    # Client and exceptions
    assert "HelmClient" in all_exports
    assert "HelmError" in all_exports
    assert "HelmTimeoutError" in all_exports
    assert "HelmConnectionError" in all_exports
    assert "HelmToolError" in all_exports
    # Sample of tool wrappers (all 44)
    assert "helm_list" in all_exports
    assert "helm_install" in all_exports
    assert "helm_status" in all_exports
    assert "helm_version" in all_exports
    # Total: 3 core + 6 client/exceptions + 44 tools = 53
    assert len(all_exports) == 64


def test_py_typed_marker():
    """Test PEP 561 py.typed marker exists."""
    import helm_mcp

    pkg_dir = Path(helm_mcp.__file__).parent
    assert (pkg_dir / "py.typed").exists()


# ---------------------------------------------------------------------------
# Binary discovery
# ---------------------------------------------------------------------------


def test_find_binary_env_var(tmp_path):
    """Test binary discovery via HELM_MCP_BINARY env var."""
    from helm_mcp.server import _find_binary

    fake_binary = tmp_path / "helm-mcp"
    fake_binary.write_text("#!/bin/sh\necho hello")
    fake_binary.chmod(0o755)

    with patch.dict(os.environ, {"HELM_MCP_BINARY": str(fake_binary)}):
        result = _find_binary()
        assert result == str(fake_binary)


def test_find_binary_env_var_not_found():
    """Test error when HELM_MCP_BINARY points to nonexistent file."""
    from helm_mcp.server import _find_binary

    with (
        patch.dict(os.environ, {"HELM_MCP_BINARY": "/nonexistent/helm-mcp"}),
        pytest.raises(FileNotFoundError, match="HELM_MCP_BINARY"),
    ):
        _find_binary()


def test_find_binary_env_var_not_executable(tmp_path):
    """Test error when HELM_MCP_BINARY exists but is not executable."""
    from helm_mcp.server import _find_binary

    fake_binary = tmp_path / "helm-mcp"
    fake_binary.write_text("not executable")
    fake_binary.chmod(0o644)

    with (
        patch.dict(os.environ, {"HELM_MCP_BINARY": str(fake_binary)}),
        pytest.raises(FileNotFoundError, match="HELM_MCP_BINARY"),
    ):
        _find_binary()


def test_find_binary_path_lookup(tmp_path):
    """Test binary discovery via PATH."""
    from helm_mcp.server import _find_binary

    fake_binary = tmp_path / "helm-mcp"
    fake_binary.write_text("#!/bin/sh\necho hello")
    fake_binary.chmod(0o755)

    env = {k: v for k, v in os.environ.items() if k != "HELM_MCP_BINARY"}
    env["PATH"] = f"{tmp_path}:{env.get('PATH', '')}"

    with patch.dict(os.environ, env, clear=True):
        result = _find_binary()
        assert result == str(fake_binary)


def test_find_binary_path_before_download(tmp_path):
    """Test that PATH lookup is tried before auto-download.

    Platform-specific wheel installs place the binary on PATH, so PATH
    lookup must come before the (potentially failing) download attempt.
    """
    from helm_mcp.server import _find_binary

    fake_binary = tmp_path / "helm-mcp"
    fake_binary.write_text("#!/bin/sh\necho hello")
    fake_binary.chmod(0o755)

    env = {k: v for k, v in os.environ.items() if k != "HELM_MCP_BINARY"}
    env["PATH"] = f"{tmp_path}:{env.get('PATH', '')}"

    download_called = False

    def mock_ensure_binary(version):
        nonlocal download_called
        download_called = True
        return None

    with (
        patch.dict(os.environ, env, clear=True),
        patch("helm_mcp.download.ensure_binary", mock_ensure_binary),
    ):
        result = _find_binary()
        assert result == str(fake_binary)
        # Download should NOT have been called because PATH found it first
        assert not download_called


def test_find_binary_not_found():
    """Test error when binary cannot be found."""
    from helm_mcp.server import _find_binary

    env = {k: v for k, v in os.environ.items() if k != "HELM_MCP_BINARY"}
    env["PATH"] = "/nonexistent"

    with (
        patch.dict(os.environ, env, clear=True),
        patch("helm_mcp.download.ensure_binary", return_value=None),
        pytest.raises(FileNotFoundError, match="helm-mcp binary not found"),
    ):
        _find_binary()


def test_is_python_script_detects_python_shebang(tmp_path):
    """Test that _is_python_script returns True for pip console_script wrappers."""
    from helm_mcp.server import _is_python_script

    wrapper = tmp_path / "helm-mcp"
    wrapper.write_text("#!/usr/bin/env python3\n# pip console_script wrapper\n")
    assert _is_python_script(str(wrapper)) is True


def test_is_python_script_false_for_shell_script(tmp_path):
    """Test that _is_python_script returns False for shell scripts."""
    from helm_mcp.server import _is_python_script

    script = tmp_path / "helm-mcp"
    script.write_text('#!/bin/sh\nexec helm-mcp-go "$@"\n')
    assert _is_python_script(str(script)) is False


def test_is_python_script_false_for_binary(tmp_path):
    """Test that _is_python_script returns False for native binaries."""
    from helm_mcp.server import _is_python_script

    binary = tmp_path / "helm-mcp"
    binary.write_bytes(b"\x7fELF\x02\x01\x01\x00")  # ELF header
    assert _is_python_script(str(binary)) is False


def test_is_python_script_false_for_missing_file():
    """Test that _is_python_script returns False for nonexistent files."""
    from helm_mcp.server import _is_python_script

    assert _is_python_script("/nonexistent/path/helm-mcp") is False


def test_find_binary_skips_python_wrapper_on_path(tmp_path):
    """Test that _find_binary skips Python console_script wrappers on PATH.

    When pip installs the universal wheel, it places a Python wrapper at
    ``bin/helm-mcp``.  The PATH lookup must skip this to avoid an infinite
    exec loop.
    """
    from helm_mcp.server import _find_binary

    # Create a Python wrapper (mimics pip console_script)
    wrapper = tmp_path / "helm-mcp"
    wrapper.write_text("#!/usr/bin/env python3\nfrom helm_mcp.cli import helm_mcp_main\n")
    wrapper.chmod(0o755)

    env = {k: v for k, v in os.environ.items() if k != "HELM_MCP_BINARY"}
    env["PATH"] = f"{tmp_path}:{env.get('PATH', '')}"

    with (
        patch.dict(os.environ, env, clear=True),
        patch("helm_mcp.download.ensure_binary", return_value=None),
        pytest.raises(FileNotFoundError, match="helm-mcp binary not found"),
    ):
        _find_binary()


def test_find_binary_bundled(tmp_path):
    """Test binary discovery from bundled bin/ directory."""
    import helm_mcp.server as server_mod
    from helm_mcp.server import _find_binary

    pkg_dir = Path(server_mod.__file__).parent
    bin_dir = pkg_dir / "bin"
    bin_dir.mkdir(exist_ok=True)
    bundled = bin_dir / "helm-mcp"
    bundled.write_text("#!/bin/sh\necho hello")
    bundled.chmod(0o755)

    try:
        env = {k: v for k, v in os.environ.items() if k != "HELM_MCP_BINARY"}
        env["PATH"] = "/nonexistent"
        with patch.dict(os.environ, env, clear=True):
            result = _find_binary()
            assert result == str(bundled)
    finally:
        shutil.rmtree(bin_dir, ignore_errors=True)


def test_find_binary_bundled_platform_specific(tmp_path):
    """Test platform-specific bundled binary discovery."""
    import helm_mcp.server as server_mod
    from helm_mcp.server import _find_binary

    pkg_dir = Path(server_mod.__file__).parent
    bin_dir = pkg_dir / "bin"
    bin_dir.mkdir(exist_ok=True)

    system = platform.system().lower()
    machine = platform.machine().lower()
    arch_map = {"x86_64": "amd64", "aarch64": "arm64", "arm64": "arm64", "amd64": "amd64"}
    arch = arch_map.get(machine, machine)
    binary_name = f"helm-mcp-{system}-{arch}"

    bundled = bin_dir / binary_name
    bundled.write_text("#!/bin/sh\necho hello")
    bundled.chmod(0o755)

    try:
        env = {k: v for k, v in os.environ.items() if k != "HELM_MCP_BINARY"}
        env["PATH"] = "/nonexistent"
        with patch.dict(os.environ, env, clear=True):
            result = _find_binary()
            assert result == str(bundled)
    finally:
        shutil.rmtree(bin_dir, ignore_errors=True)


# ---------------------------------------------------------------------------
# Environment building
# ---------------------------------------------------------------------------


def test_build_subprocess_env_passthrough():
    """Test that _build_subprocess_env forwards expected variables."""
    from helm_mcp.server import _build_subprocess_env

    test_vars = {
        "HTTP_PROXY": "http://proxy:8080",
        "HTTPS_PROXY": "http://proxy:8443",
        "NO_PROXY": "localhost,.internal",
        "KUBECONFIG": "/home/user/.kube/config",
        "HOME": "/home/user",
    }

    with patch.dict(os.environ, test_vars, clear=True):
        result = _build_subprocess_env()
        assert result["HTTP_PROXY"] == "http://proxy:8080"
        assert result["HTTPS_PROXY"] == "http://proxy:8443"
        assert result["NO_PROXY"] == "localhost,.internal"
        assert result["KUBECONFIG"] == "/home/user/.kube/config"
        assert result["HOME"] == "/home/user"


def test_build_subprocess_env_extra_overrides():
    """Test that extra_env overrides passthrough values."""
    from helm_mcp.server import _build_subprocess_env

    with patch.dict(os.environ, {"HOME": "/home/user"}, clear=True):
        result = _build_subprocess_env(extra_env={"HOME": "/override", "CUSTOM": "value"})
        assert result["HOME"] == "/override"
        assert result["CUSTOM"] == "value"


def test_build_subprocess_env_custom_passthrough():
    """Test _build_subprocess_env with a custom passthrough list."""
    from helm_mcp.server import _build_subprocess_env

    with patch.dict(os.environ, {"FOO": "bar", "HOME": "/home/user"}, clear=True):
        result = _build_subprocess_env(passthrough=["FOO"])
        assert result == {"FOO": "bar"}
        assert "HOME" not in result


def test_build_subprocess_env_skips_unset():
    """Test that unset variables are not included."""
    from helm_mcp.server import _build_subprocess_env

    with patch.dict(os.environ, {}, clear=True):
        result = _build_subprocess_env()
        assert result == {}


def test_build_subprocess_env_empty_extra():
    """Test passing empty extra_env dict."""
    from helm_mcp.server import _build_subprocess_env

    with patch.dict(os.environ, {"HOME": "/home/user"}, clear=True):
        result = _build_subprocess_env(extra_env={})
        assert result["HOME"] == "/home/user"


# ---------------------------------------------------------------------------
# PASSTHROUGH_ENV_VARS completeness
# ---------------------------------------------------------------------------


def test_passthrough_env_vars_includes_proxy():
    """Test that PASSTHROUGH_ENV_VARS includes all proxy variants."""
    from helm_mcp.server import PASSTHROUGH_ENV_VARS

    for var in ["HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy"]:
        assert var in PASSTHROUGH_ENV_VARS, f"{var} missing from PASSTHROUGH_ENV_VARS"


def test_passthrough_env_vars_includes_kubernetes():
    """Test that PASSTHROUGH_ENV_VARS includes Kubernetes variables."""
    from helm_mcp.server import PASSTHROUGH_ENV_VARS

    for var in ["KUBECONFIG", "KUBERNETES_SERVICE_HOST", "KUBERNETES_SERVICE_PORT"]:
        assert var in PASSTHROUGH_ENV_VARS, f"{var} missing from PASSTHROUGH_ENV_VARS"


def test_passthrough_env_vars_includes_helm():
    """Test that PASSTHROUGH_ENV_VARS includes Helm variables."""
    from helm_mcp.server import PASSTHROUGH_ENV_VARS

    for var in [
        "HELM_CACHE_HOME",
        "HELM_CONFIG_HOME",
        "HELM_DATA_HOME",
        "HELM_DRIVER",
        "HELM_PLUGINS",
        "HELM_DEBUG",
    ]:
        assert var in PASSTHROUGH_ENV_VARS, f"{var} missing from PASSTHROUGH_ENV_VARS"


def test_passthrough_env_vars_includes_aws():
    """Test that PASSTHROUGH_ENV_VARS includes AWS variables."""
    from helm_mcp.server import PASSTHROUGH_ENV_VARS

    for var in ["AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_REGION", "AWS_PROFILE"]:
        assert var in PASSTHROUGH_ENV_VARS, f"{var} missing from PASSTHROUGH_ENV_VARS"


def test_passthrough_env_vars_includes_gcp():
    """Test that PASSTHROUGH_ENV_VARS includes GCP variables."""
    from helm_mcp.server import PASSTHROUGH_ENV_VARS

    assert "GOOGLE_APPLICATION_CREDENTIALS" in PASSTHROUGH_ENV_VARS


def test_passthrough_env_vars_includes_azure():
    """Test that PASSTHROUGH_ENV_VARS includes Azure variables."""
    from helm_mcp.server import PASSTHROUGH_ENV_VARS

    for var in ["AZURE_TENANT_ID", "AZURE_CLIENT_ID", "AZURE_CLIENT_SECRET"]:
        assert var in PASSTHROUGH_ENV_VARS, f"{var} missing from PASSTHROUGH_ENV_VARS"


def test_passthrough_env_vars_includes_tls():
    """Test that PASSTHROUGH_ENV_VARS includes TLS CA variables."""
    from helm_mcp.server import PASSTHROUGH_ENV_VARS

    for var in ["SSL_CERT_FILE", "SSL_CERT_DIR"]:
        assert var in PASSTHROUGH_ENV_VARS, f"{var} missing from PASSTHROUGH_ENV_VARS"


# ---------------------------------------------------------------------------
# Server / Client creation
# ---------------------------------------------------------------------------


def test_create_server_with_binary(tmp_path):
    """Test create_server with explicit binary path."""
    from helm_mcp.server import create_server

    fake_binary = tmp_path / "helm-mcp"
    fake_binary.write_text("#!/bin/sh\necho hello")
    fake_binary.chmod(0o755)

    server = create_server(binary_path=str(fake_binary))
    assert server is not None


def test_create_server_custom_name(tmp_path):
    """Test create_server with a custom server name."""
    from helm_mcp.server import create_server

    fake_binary = tmp_path / "helm-mcp"
    fake_binary.write_text("#!/bin/sh\necho hello")
    fake_binary.chmod(0o755)

    server = create_server(binary_path=str(fake_binary), name="my-helm-mcp")
    assert server is not None


def test_create_server_with_extra_env(tmp_path):
    """Test create_server with extra environment variables."""
    from helm_mcp.server import create_server

    fake_binary = tmp_path / "helm-mcp"
    fake_binary.write_text("#!/bin/sh\necho hello")
    fake_binary.chmod(0o755)

    server = create_server(
        binary_path=str(fake_binary),
        env={"CUSTOM_VAR": "custom_value"},
    )
    assert server is not None


def test_create_server_binary_not_found():
    """Test create_server raises when binary not found."""
    from helm_mcp.server import create_server

    env = {k: v for k, v in os.environ.items() if k != "HELM_MCP_BINARY"}
    env["PATH"] = "/nonexistent"

    with (
        patch.dict(os.environ, env, clear=True),
        patch("helm_mcp.download.ensure_binary", return_value=None),
        pytest.raises(FileNotFoundError),
    ):
        create_server()


def test_create_client_with_binary(tmp_path):
    """Test create_client with explicit binary path."""
    from helm_mcp.client import create_client

    fake_binary = tmp_path / "helm-mcp"
    fake_binary.write_text("#!/bin/sh\necho hello")
    fake_binary.chmod(0o755)

    client = create_client(binary_path=str(fake_binary))
    assert client is not None


def test_create_client_with_extra_env(tmp_path):
    """Test create_client with extra environment variables."""
    from helm_mcp.client import create_client

    fake_binary = tmp_path / "helm-mcp"
    fake_binary.write_text("#!/bin/sh\necho hello")
    fake_binary.chmod(0o755)

    client = create_client(
        binary_path=str(fake_binary),
        env={"EXTRA_VAR": "extra"},
    )
    assert client is not None


def test_create_client_binary_not_found():
    """Test create_client raises when binary not found."""
    from helm_mcp.client import create_client

    env = {k: v for k, v in os.environ.items() if k != "HELM_MCP_BINARY"}
    env["PATH"] = "/nonexistent"

    with (
        patch.dict(os.environ, env, clear=True),
        patch("helm_mcp.download.ensure_binary", return_value=None),
        pytest.raises(FileNotFoundError),
    ):
        create_client()


# ---------------------------------------------------------------------------
# CLI module
# ---------------------------------------------------------------------------


def test_find_binary_bundled_chmod():
    """Test bundled binary gets chmod'd if it exists but is not executable."""
    import helm_mcp.server as server_mod
    from helm_mcp.server import _find_binary

    pkg_dir = Path(server_mod.__file__).parent
    bin_dir = pkg_dir / "bin"
    bin_dir.mkdir(exist_ok=True)
    bundled = bin_dir / "helm-mcp"
    bundled.write_bytes(b"\x7fELF...")  # fake ELF header
    bundled.chmod(0o644)  # NOT executable

    try:
        env = {k: v for k, v in os.environ.items() if k != "HELM_MCP_BINARY"}
        env["PATH"] = "/nonexistent"
        with patch.dict(os.environ, env, clear=True):
            result = _find_binary()
            assert result == str(bundled)
            # Verify it was made executable
            assert os.access(str(bundled), os.X_OK)
    finally:
        shutil.rmtree(bin_dir, ignore_errors=True)


def test_cli_module_exists():
    """Test that CLI entry point module exists."""
    from helm_mcp import cli

    assert hasattr(cli, "main")
    assert callable(cli.main)
    assert hasattr(cli, "helm_mcp_main")
    assert callable(cli.helm_mcp_main)


def test_cli_help(capsys):
    """Test CLI --help exits cleanly."""
    from helm_mcp.cli import main

    with (
        pytest.raises(SystemExit) as exc_info,
        patch("sys.argv", ["helm-mcp-python", "--help"]),
    ):
        main()
    assert exc_info.value.code == 0


def test_cli_invalid_transport(capsys):
    """Test CLI rejects invalid transport."""
    from helm_mcp.cli import main

    with (
        pytest.raises(SystemExit) as exc_info,
        patch("sys.argv", ["helm-mcp-python", "--transport", "invalid"]),
    ):
        main()
    assert exc_info.value.code != 0


def test_cli_binary_not_found(capsys):
    """Test CLI exits with error when binary not found."""
    from helm_mcp.cli import main

    env = {k: v for k, v in os.environ.items() if k != "HELM_MCP_BINARY"}
    env["PATH"] = "/nonexistent"

    with (
        patch.dict(os.environ, env, clear=True),
        patch("helm_mcp.download.ensure_binary", return_value=None),
        patch("sys.argv", ["helm-mcp-python"]),
        pytest.raises(SystemExit) as exc_info,
    ):
        main()
    assert exc_info.value.code == 1


def test_cli_stdio_transport(tmp_path):
    """Test CLI with stdio transport calls server.run()."""
    from unittest.mock import MagicMock

    from helm_mcp.cli import main

    mock_server = MagicMock()

    fake_binary = tmp_path / "helm-mcp"
    fake_binary.write_text("#!/bin/sh\necho hello")
    fake_binary.chmod(0o755)

    with (
        patch("sys.argv", ["helm-mcp-python", "--binary", str(fake_binary)]),
        patch("helm_mcp.server.create_server", return_value=mock_server) as mock_create,
    ):
        main()

    mock_create.assert_called_once()
    call_kwargs = mock_create.call_args[1]
    assert call_kwargs["binary_path"] == str(fake_binary)
    assert "resilience" in call_kwargs
    mock_server.run.assert_called_once_with()


def test_cli_http_transport(tmp_path):
    """Test CLI with http transport passes host and port."""
    from unittest.mock import MagicMock

    from helm_mcp.cli import main

    mock_server = MagicMock()

    fake_binary = tmp_path / "helm-mcp"
    fake_binary.write_text("#!/bin/sh\necho hello")
    fake_binary.chmod(0o755)

    with (
        patch(
            "sys.argv",
            [
                "helm-mcp-python",
                "--binary",
                str(fake_binary),
                "--transport",
                "http",
                "--host",
                "127.0.0.1",
                "--port",
                "9090",
            ],
        ),
        patch("helm_mcp.server.create_server", return_value=mock_server),
    ):
        main()

    mock_server.run.assert_called_once_with(transport="http", host="127.0.0.1", port=9090)


# ---------------------------------------------------------------------------
# CLI entry point: helm_mcp_main
# ---------------------------------------------------------------------------


def test_helm_mcp_main_execs_binary(tmp_path):
    """Test helm_mcp_main finds bundled binary and calls os.execvp."""
    from helm_mcp.cli import helm_mcp_main

    fake_binary = tmp_path / "helm-mcp"
    fake_binary.write_text("#!/bin/sh\necho helm-mcp")
    fake_binary.chmod(0o755)

    with (
        patch("helm_mcp.cli._find_binary", return_value=str(fake_binary)),
        patch("os.execvp") as mock_exec,
        patch("sys.argv", ["helm-mcp", "--mode", "stdio"]),
    ):
        helm_mcp_main()
        mock_exec.assert_called_once_with(str(fake_binary), [str(fake_binary), "--mode", "stdio"])


def test_helm_mcp_main_not_found(capsys):
    """Test helm_mcp_main exits 1 when binary not found."""
    from helm_mcp.cli import helm_mcp_main

    with (
        patch("helm_mcp.cli._find_binary", side_effect=FileNotFoundError("not found")),
        pytest.raises(SystemExit) as exc_info,
    ):
        helm_mcp_main()
    assert exc_info.value.code == 1


def test_cli_is_python_script(tmp_path):
    """Test that cli._is_python_script detects Python wrappers."""
    from helm_mcp.cli import _is_python_script

    wrapper = tmp_path / "helm-mcp"
    wrapper.write_text("#!/usr/bin/python3\nimport sys\n")
    assert _is_python_script(str(wrapper)) is True

    shell = tmp_path / "helm-mcp-go"
    shell.write_text("#!/bin/sh\nexec helm-mcp-go\n")
    assert _is_python_script(str(shell)) is False


def test_cli_find_binary_skips_python_wrapper(tmp_path):
    """Test that cli._find_binary skips Python wrappers and tries auto-download."""
    from helm_mcp.cli import _find_binary

    wrapper = tmp_path / "helm-mcp"
    wrapper.write_text("#!/usr/bin/env python3\nfrom helm_mcp.cli import main\n")
    wrapper.chmod(0o755)

    downloaded = tmp_path / "helm-mcp-downloaded"
    downloaded.write_bytes(b"\x7fELF")
    downloaded.chmod(0o755)

    env = {k: v for k, v in os.environ.items() if k != "HELM_MCP_BINARY"}
    env["PATH"] = f"{tmp_path}:{env.get('PATH', '')}"

    with (
        patch.dict(os.environ, env, clear=True),
        patch("helm_mcp.download.ensure_binary", return_value=str(downloaded)),
    ):
        result = _find_binary("helm-mcp")
        assert result == str(downloaded)


def test_find_bundled_binary_found():
    """Test _find_bundled_binary finds and chmod's bundled binary."""
    import helm_mcp.cli as cli_mod
    from helm_mcp.cli import _find_bundled_binary

    pkg_dir = Path(cli_mod.__file__).parent
    bin_dir = pkg_dir / "bin"
    bin_dir.mkdir(exist_ok=True)
    bundled = bin_dir / "helm-mcp"
    bundled.write_bytes(b"\x7fELF...")
    bundled.chmod(0o644)  # NOT executable

    try:
        result = _find_bundled_binary("helm-mcp")
        assert result == str(bundled)
        assert os.access(str(bundled), os.X_OK)
    finally:
        shutil.rmtree(bin_dir, ignore_errors=True)


def test_find_bundled_binary_not_found():
    """Test _find_bundled_binary returns None when binary doesn't exist."""
    from helm_mcp.cli import _find_bundled_binary

    result = _find_bundled_binary("nonexistent-binary-xyz")
    assert result is None


# ---------------------------------------------------------------------------
# Middleware integration
# ---------------------------------------------------------------------------


class TestMiddlewareIntegration:
    """Test that create_server applies middleware from ResilienceConfig."""

    def test_server_with_default_config(self, tmp_path):
        """Default config applies timing, error handling, and retry middleware."""
        from helm_mcp.server import create_server

        fake_binary = tmp_path / "helm-mcp"
        fake_binary.write_text("#!/bin/sh\necho hello")
        fake_binary.chmod(0o755)

        with patch.dict(os.environ, {}, clear=True):
            server = create_server(binary_path=str(fake_binary))
            assert server is not None

    def test_server_with_all_middleware_disabled(self, tmp_path):
        """No middleware when everything is disabled."""
        from helm_mcp.resilience import (
            CacheConfig,
            ErrorHandlingConfig,
            RateLimitConfig,
            ResilienceConfig,
            RetryConfig,
            TimingConfig,
        )
        from helm_mcp.server import create_server

        fake_binary = tmp_path / "helm-mcp"
        fake_binary.write_text("#!/bin/sh\necho hello")
        fake_binary.chmod(0o755)

        config = ResilienceConfig(
            retry=RetryConfig(enabled=False),
            rate_limit=RateLimitConfig(enabled=False),
            cache=CacheConfig(enabled=False),
            error_handling=ErrorHandlingConfig(enabled=False),
            timing=TimingConfig(enabled=False),
        )
        server = create_server(binary_path=str(fake_binary), resilience=config)
        assert server is not None

    def test_server_with_custom_resilience(self, tmp_path):
        """Custom resilience config is passed through to server."""
        from helm_mcp.resilience import RateLimitConfig, ResilienceConfig
        from helm_mcp.server import create_server

        fake_binary = tmp_path / "helm-mcp"
        fake_binary.write_text("#!/bin/sh\necho hello")
        fake_binary.chmod(0o755)

        config = ResilienceConfig(
            rate_limit=RateLimitConfig(enabled=True, max_requests_per_second=50.0),
        )
        server = create_server(binary_path=str(fake_binary), resilience=config)
        assert server is not None


# ---------------------------------------------------------------------------
# CLI resilience flags
# ---------------------------------------------------------------------------


class TestCLIResilienceFlags:
    """Test CLI resilience argument handling."""

    def test_cli_no_retry_flag(self, tmp_path):
        """--no-retry sets HELM_MCP_RETRY_ENABLED=false."""
        from unittest.mock import MagicMock

        from helm_mcp.cli import main

        mock_server = MagicMock()
        fake_binary = tmp_path / "helm-mcp"
        fake_binary.write_text("#!/bin/sh\necho hello")
        fake_binary.chmod(0o755)

        with (
            patch.dict(os.environ, os.environ.copy()),
            patch("sys.argv", ["helm-mcp-python", "--binary", str(fake_binary), "--no-retry"]),
            patch("helm_mcp.server.create_server", return_value=mock_server) as mock_create,
        ):
            main()

        call_kwargs = mock_create.call_args[1]
        config = call_kwargs["resilience"]
        assert config.retry.enabled is False

    def test_cli_rate_limit_flag(self, tmp_path):
        """--rate-limit RPS enables rate limiting at the specified rate."""
        from unittest.mock import MagicMock

        from helm_mcp.cli import main

        mock_server = MagicMock()
        fake_binary = tmp_path / "helm-mcp"
        fake_binary.write_text("#!/bin/sh\necho hello")
        fake_binary.chmod(0o755)

        with (
            patch.dict(os.environ, os.environ.copy()),
            patch(
                "sys.argv",
                ["helm-mcp-python", "--binary", str(fake_binary), "--rate-limit", "50"],
            ),
            patch("helm_mcp.server.create_server", return_value=mock_server) as mock_create,
        ):
            main()

        call_kwargs = mock_create.call_args[1]
        config = call_kwargs["resilience"]
        assert config.rate_limit.enabled is True
        assert config.rate_limit.max_requests_per_second == 50.0

    def test_cli_cache_flag(self, tmp_path):
        """--cache enables response caching."""
        from unittest.mock import MagicMock

        from helm_mcp.cli import main

        mock_server = MagicMock()
        fake_binary = tmp_path / "helm-mcp"
        fake_binary.write_text("#!/bin/sh\necho hello")
        fake_binary.chmod(0o755)

        with (
            patch.dict(os.environ, os.environ.copy()),
            patch("sys.argv", ["helm-mcp-python", "--binary", str(fake_binary), "--cache"]),
            patch("helm_mcp.server.create_server", return_value=mock_server) as mock_create,
        ):
            main()

        call_kwargs = mock_create.call_args[1]
        config = call_kwargs["resilience"]
        assert config.cache.enabled is True

    def test_cli_no_circuit_breaker_flag(self, tmp_path):
        """--no-circuit-breaker disables circuit breaker."""
        from unittest.mock import MagicMock

        from helm_mcp.cli import main

        mock_server = MagicMock()
        fake_binary = tmp_path / "helm-mcp"
        fake_binary.write_text("#!/bin/sh\necho hello")
        fake_binary.chmod(0o755)

        with (
            patch.dict(os.environ, os.environ.copy()),
            patch(
                "sys.argv",
                ["helm-mcp-python", "--binary", str(fake_binary), "--no-circuit-breaker"],
            ),
            patch("helm_mcp.server.create_server", return_value=mock_server) as mock_create,
        ):
            main()

        call_kwargs = mock_create.call_args[1]
        config = call_kwargs["resilience"]
        assert config.circuit_breaker.enabled is False

    def test_cli_bulkhead_max_flag(self, tmp_path):
        """--bulkhead-max N sets bulkhead max concurrent."""
        from unittest.mock import MagicMock

        from helm_mcp.cli import main

        mock_server = MagicMock()
        fake_binary = tmp_path / "helm-mcp"
        fake_binary.write_text("#!/bin/sh\necho hello")
        fake_binary.chmod(0o755)

        with (
            patch.dict(os.environ, os.environ.copy()),
            patch(
                "sys.argv",
                ["helm-mcp-python", "--binary", str(fake_binary), "--bulkhead-max", "5"],
            ),
            patch("helm_mcp.server.create_server", return_value=mock_server) as mock_create,
        ):
            main()

        call_kwargs = mock_create.call_args[1]
        config = call_kwargs["resilience"]
        assert config.bulkhead.max_concurrent == 5

    def test_cli_otel_flag(self, tmp_path):
        """--otel enables OpenTelemetry."""
        from unittest.mock import MagicMock

        from helm_mcp.cli import main

        mock_server = MagicMock()
        fake_binary = tmp_path / "helm-mcp"
        fake_binary.write_text("#!/bin/sh\necho hello")
        fake_binary.chmod(0o755)

        with (
            patch.dict(os.environ, os.environ.copy()),
            patch("sys.argv", ["helm-mcp-python", "--binary", str(fake_binary), "--otel"]),
            patch("helm_mcp.server.create_server", return_value=mock_server) as mock_create,
        ):
            main()

        call_kwargs = mock_create.call_args[1]
        config = call_kwargs["resilience"]
        assert config.otel.enabled is True
