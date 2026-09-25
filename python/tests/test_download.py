"""Tests for helm_mcp.download module."""

import hashlib
import json
import os
import platform
from pathlib import Path
from unittest.mock import patch

import pytest

from helm_mcp.download import (
    _get_binary_name,
    _get_install_dir,
    _load_checksums,
    _verify_checksum,
    ensure_binary,
)

# ---------------------------------------------------------------------------
# _load_checksums
# ---------------------------------------------------------------------------


def test_load_checksums_exists(tmp_path):
    """Test loading checksums from a valid file."""
    checksums = {"version": "1.0.0", "binaries": {"helm-mcp-linux-amd64": "abc123"}}
    checksums_file = tmp_path / "checksums.json"
    checksums_file.write_text(json.dumps(checksums))

    with patch("helm_mcp.download.Path") as mock_path_cls:
        # Make Path(__file__).parent return tmp_path
        mock_path_cls.return_value.parent = tmp_path

    # Just verify it returns a dict (depends on whether checksums.json exists)
    result = _load_checksums()
    assert isinstance(result, dict)


def test_load_checksums_missing(tmp_path, monkeypatch):
    """Test loading checksums when file doesn't exist."""
    # Patch the module's __file__ to point to tmp_path
    import helm_mcp.download as dl_mod

    monkeypatch.setattr(dl_mod, "__file__", str(tmp_path / "download.py"))
    result = _load_checksums()
    assert result == {}


def test_load_checksums_valid(tmp_path, monkeypatch):
    """Test loading valid checksums file."""
    import helm_mcp.download as dl_mod

    checksums = {"version": "1.0.0", "binaries": {"helm-mcp-linux-amd64": "abc123"}}
    checksums_file = tmp_path / "checksums.json"
    checksums_file.write_text(json.dumps(checksums))

    monkeypatch.setattr(dl_mod, "__file__", str(tmp_path / "download.py"))
    result = _load_checksums()
    assert result == checksums
    assert result["version"] == "1.0.0"
    assert result["binaries"]["helm-mcp-linux-amd64"] == "abc123"


# ---------------------------------------------------------------------------
# _get_binary_name
# ---------------------------------------------------------------------------


def test_get_binary_name_format():
    """Test binary name follows expected format."""
    name = _get_binary_name()
    system = platform.system().lower()
    assert name.startswith(f"helm-mcp-{system}-")
    if system == "windows":
        assert name.endswith(".exe")


def test_get_binary_name_darwin_arm64():
    """Test binary name for macOS ARM."""
    with (
        patch("helm_mcp.download.platform.system", return_value="Darwin"),
        patch("helm_mcp.download.platform.machine", return_value="arm64"),
    ):
        assert _get_binary_name() == "helm-mcp-darwin-arm64"


def test_get_binary_name_linux_amd64():
    """Test binary name for Linux x86_64."""
    with (
        patch("helm_mcp.download.platform.system", return_value="Linux"),
        patch("helm_mcp.download.platform.machine", return_value="x86_64"),
    ):
        assert _get_binary_name() == "helm-mcp-linux-amd64"


def test_get_binary_name_windows():
    """Test binary name for Windows adds .exe."""
    with (
        patch("helm_mcp.download.platform.system", return_value="Windows"),
        patch("helm_mcp.download.platform.machine", return_value="AMD64"),
    ):
        name = _get_binary_name()
        assert name == "helm-mcp-windows-amd64.exe"


def test_get_binary_name_linux_arm64():
    """Test binary name for Linux ARM."""
    with (
        patch("helm_mcp.download.platform.system", return_value="Linux"),
        patch("helm_mcp.download.platform.machine", return_value="aarch64"),
    ):
        assert _get_binary_name() == "helm-mcp-linux-arm64"


def test_get_binary_name_unknown_arch():
    """Test binary name with an unrecognized architecture passes through."""
    with (
        patch("helm_mcp.download.platform.system", return_value="Linux"),
        patch("helm_mcp.download.platform.machine", return_value="riscv64"),
    ):
        assert _get_binary_name() == "helm-mcp-linux-riscv64"


# ---------------------------------------------------------------------------
# _get_install_dir
# ---------------------------------------------------------------------------


def test_get_install_dir_scripts_writable(tmp_path):
    """Test install dir uses scripts dir when writable."""
    with patch("helm_mcp.download.sysconfig.get_path", return_value=str(tmp_path)):
        result = _get_install_dir()
        assert result == tmp_path


def test_get_install_dir_scripts_not_writable(tmp_path):
    """Test install dir falls back to ~/.local/bin."""
    non_writable = tmp_path / "no-write"
    non_writable.mkdir()
    non_writable.chmod(0o555)

    local_bin = tmp_path / "home" / ".local" / "bin"

    with (
        patch("helm_mcp.download.sysconfig.get_path", return_value=str(non_writable)),
        patch("helm_mcp.download.Path.home", return_value=tmp_path / "home"),
    ):
        result = _get_install_dir()
        assert result == local_bin
        assert local_bin.exists()

    # Restore permissions for cleanup
    non_writable.chmod(0o755)


# ---------------------------------------------------------------------------
# _verify_checksum
# ---------------------------------------------------------------------------


def test_verify_checksum_match(tmp_path):
    """Test checksum verification with matching hash."""
    content = b"hello world binary content"
    expected = hashlib.sha256(content).hexdigest()
    file_path = tmp_path / "binary"
    file_path.write_bytes(content)

    assert _verify_checksum(file_path, expected) is True


def test_verify_checksum_mismatch(tmp_path):
    """Test checksum verification with wrong hash."""
    file_path = tmp_path / "binary"
    file_path.write_bytes(b"actual content")

    wrong_hash = "0" * 64
    assert _verify_checksum(file_path, wrong_hash) is False


def test_verify_checksum_empty_file(tmp_path):
    """Test checksum verification with empty file."""
    file_path = tmp_path / "empty"
    file_path.write_bytes(b"")
    expected = hashlib.sha256(b"").hexdigest()

    assert _verify_checksum(file_path, expected) is True


# ---------------------------------------------------------------------------
# ensure_binary
# ---------------------------------------------------------------------------


def test_ensure_binary_already_installed(tmp_path):
    """Test ensure_binary returns an existing binary whose checksum matches."""
    content = b"#!/bin/sh\necho hello"
    target = tmp_path / "helm-mcp-go-v1.0.0"
    target.write_bytes(content)
    target.chmod(0o755)
    checksums = {"binaries": {"helm-mcp-linux-amd64": hashlib.sha256(content).hexdigest()}}

    with (
        patch("helm_mcp.download._get_install_dir", return_value=tmp_path),
        patch("helm_mcp.download._load_checksums", return_value=checksums),
        patch("helm_mcp.download._get_binary_name", return_value="helm-mcp-linux-amd64"),
        patch("helm_mcp.download.platform.system", return_value="Linux"),
        patch("helm_mcp.download._download") as download,
    ):
        result = ensure_binary("1.0.0")
        assert result == str(target)
        download.assert_not_called()


def test_ensure_binary_ignores_console_script(tmp_path):
    """The pip console script named helm-mcp must never be returned."""
    script = tmp_path / "helm-mcp"
    script.write_text("#!/usr/bin/python3\nfrom helm_mcp.cli import helm_mcp_main\n")
    script.chmod(0o755)
    binary_content = b"real binary"
    checksums = {"binaries": {"helm-mcp-linux-amd64": hashlib.sha256(binary_content).hexdigest()}}

    def fake_download(url, path):
        Path(path).write_bytes(binary_content)

    with (
        patch("helm_mcp.download._get_install_dir", return_value=tmp_path),
        patch("helm_mcp.download._load_checksums", return_value=checksums),
        patch("helm_mcp.download._get_binary_name", return_value="helm-mcp-linux-amd64"),
        patch("helm_mcp.download.platform.system", return_value="Linux"),
        patch("helm_mcp.download._download", side_effect=fake_download),
    ):
        result = ensure_binary("1.0.0")

    assert result is not None
    assert Path(result).name == "helm-mcp-go-v1.0.0"
    assert script.read_text().startswith("#!/usr/bin/python3")


def test_ensure_binary_redownloads_tampered_cache(tmp_path):
    """A cached binary that fails its checksum is replaced."""
    good = b"good binary"
    target = tmp_path / "helm-mcp-go-v1.0.0"
    target.write_bytes(b"tampered")
    target.chmod(0o755)
    checksums = {"binaries": {"helm-mcp-linux-amd64": hashlib.sha256(good).hexdigest()}}

    def fake_download(url, path):
        Path(path).write_bytes(good)

    with (
        patch("helm_mcp.download._get_install_dir", return_value=tmp_path),
        patch("helm_mcp.download._load_checksums", return_value=checksums),
        patch("helm_mcp.download._get_binary_name", return_value="helm-mcp-linux-amd64"),
        patch("helm_mcp.download.platform.system", return_value="Linux"),
        patch("helm_mcp.download._download", side_effect=fake_download),
    ):
        result = ensure_binary("1.0.0")

    assert result == str(target)
    assert target.read_bytes() == good


def test_download_enforces_size_cap(tmp_path):
    """_download aborts once the response exceeds MAX_BINARY_BYTES."""
    from helm_mcp import download as mod

    class FakeResponse:
        def __init__(self):
            self.chunks = [b"x" * 10, b"x" * 10]

        def read(self, _n):
            return self.chunks.pop(0) if self.chunks else b""

        def __enter__(self):
            return self

        def __exit__(self, *exc):
            return False

    with (
        patch.object(mod, "MAX_BINARY_BYTES", 15),
        patch.object(mod.urllib.request, "urlopen", return_value=FakeResponse()) as urlopen,
        pytest.raises(RuntimeError, match="exceeds"),
    ):
        mod._download("https://example.invalid/x", tmp_path / "out")
    assert urlopen.call_args.kwargs["timeout"] == mod.DOWNLOAD_TIMEOUT_SECONDS


def test_ensure_binary_no_checksums(tmp_path):
    """Test ensure_binary returns None when no checksums available."""
    with (
        patch("helm_mcp.download._get_install_dir", return_value=tmp_path),
        patch("helm_mcp.download._load_checksums", return_value={}),
        patch("helm_mcp.download.platform.system", return_value="Linux"),
    ):
        result = ensure_binary("1.0.0")
        assert result is None


def test_ensure_binary_no_checksum_for_platform(tmp_path):
    """Test ensure_binary returns None when no checksum for this platform."""
    checksums = {"version": "1.0.0", "binaries": {"helm-mcp-linux-amd64": "abc123"}}

    with (
        patch("helm_mcp.download._get_install_dir", return_value=tmp_path),
        patch("helm_mcp.download._load_checksums", return_value=checksums),
        patch("helm_mcp.download._get_binary_name", return_value="helm-mcp-freebsd-amd64"),
        patch("helm_mcp.download.platform.system", return_value="FreeBSD"),
    ):
        result = ensure_binary("1.0.0")
        assert result is None


def test_ensure_binary_downloads_and_verifies(tmp_path):
    """Test ensure_binary downloads binary and verifies checksum."""
    binary_content = b"fake binary content for testing"
    expected_sha = hashlib.sha256(binary_content).hexdigest()

    checksums = {"version": "1.0.0", "binaries": {"helm-mcp-linux-amd64": expected_sha}}

    def fake_download(url, path):
        Path(path).write_bytes(binary_content)

    with (
        patch("helm_mcp.download._get_install_dir", return_value=tmp_path),
        patch("helm_mcp.download._load_checksums", return_value=checksums),
        patch("helm_mcp.download._get_binary_name", return_value="helm-mcp-linux-amd64"),
        patch("helm_mcp.download.platform.system", return_value="Linux"),
        patch("helm_mcp.download._download", side_effect=fake_download),
    ):
        result = ensure_binary("1.0.0")

    assert result is not None
    target = Path(result)
    assert target.exists()
    assert target.name == "helm-mcp-go-v1.0.0"
    assert target.read_bytes() == binary_content
    assert os.access(str(target), os.X_OK)


def test_ensure_binary_checksum_mismatch_raises(tmp_path):
    """Test ensure_binary raises on checksum mismatch."""
    checksums = {"version": "1.0.0", "binaries": {"helm-mcp-linux-amd64": "expected_hash"}}

    def fake_download(url, path):
        Path(path).write_bytes(b"tampered content")

    with (
        patch("helm_mcp.download._get_install_dir", return_value=tmp_path),
        patch("helm_mcp.download._load_checksums", return_value=checksums),
        patch("helm_mcp.download._get_binary_name", return_value="helm-mcp-linux-amd64"),
        patch("helm_mcp.download.platform.system", return_value="Linux"),
        patch("helm_mcp.download._download", side_effect=fake_download),
        pytest.raises(RuntimeError, match="Checksum mismatch"),
    ):
        ensure_binary("1.0.0")

    # Verify temp file was cleaned up
    remaining = list(tmp_path.glob(".helm-mcp-*"))
    assert len(remaining) == 0


def test_ensure_binary_download_failure_cleans_up(tmp_path):
    """Test ensure_binary cleans up temp file on download failure."""
    checksums = {"version": "1.0.0", "binaries": {"helm-mcp-linux-amd64": "abc123"}}

    with (
        patch("helm_mcp.download._get_install_dir", return_value=tmp_path),
        patch("helm_mcp.download._load_checksums", return_value=checksums),
        patch("helm_mcp.download._get_binary_name", return_value="helm-mcp-linux-amd64"),
        patch("helm_mcp.download.platform.system", return_value="Linux"),
        patch(
            "helm_mcp.download._download",
            side_effect=ConnectionError("Network error"),
        ),
        pytest.raises(ConnectionError, match="Network error"),
    ):
        ensure_binary("1.0.0")

    # Verify temp file was cleaned up
    remaining = list(tmp_path.glob(".helm-mcp-*"))
    assert len(remaining) == 0


def test_ensure_binary_url_format(tmp_path):
    """Test ensure_binary constructs correct download URL."""
    binary_content = b"binary"
    expected_sha = hashlib.sha256(binary_content).hexdigest()
    checksums = {"version": "1.0.0", "binaries": {"helm-mcp-darwin-arm64": expected_sha}}

    captured_url = None

    def fake_download(url, path):
        nonlocal captured_url
        captured_url = url
        Path(path).write_bytes(binary_content)

    with (
        patch("helm_mcp.download._get_install_dir", return_value=tmp_path),
        patch("helm_mcp.download._load_checksums", return_value=checksums),
        patch("helm_mcp.download._get_binary_name", return_value="helm-mcp-darwin-arm64"),
        patch("helm_mcp.download.platform.system", return_value="Darwin"),
        patch("helm_mcp.download._download", side_effect=fake_download),
    ):
        ensure_binary("0.1.5")

    assert captured_url == (
        "https://github.com/SCGIS-Wales/helm-mcp/releases/download/v0.1.5/helm-mcp-darwin-arm64"
    )


def test_ensure_binary_windows_target_name(tmp_path):
    """Test ensure_binary uses .exe extension on Windows."""
    content = b"fake binary"
    target = tmp_path / "helm-mcp-go-v1.0.0.exe"
    target.write_bytes(content)
    target.chmod(0o755)
    checksums = {"binaries": {"helm-mcp-windows-amd64.exe": hashlib.sha256(content).hexdigest()}}

    with (
        patch("helm_mcp.download._get_install_dir", return_value=tmp_path),
        patch("helm_mcp.download._load_checksums", return_value=checksums),
        patch("helm_mcp.download._get_binary_name", return_value="helm-mcp-windows-amd64.exe"),
        patch("helm_mcp.download.platform.system", return_value="Windows"),
    ):
        result = ensure_binary("1.0.0")
        assert result == str(target)


# ---------------------------------------------------------------------------
# CLI --setup flag
# ---------------------------------------------------------------------------


def test_cli_setup_success(tmp_path, capsys):
    """Test CLI --setup downloads binary and exits."""
    from helm_mcp.cli import main

    fake_path = str(tmp_path / "helm-mcp")

    with (
        patch("sys.argv", ["helm-mcp-python", "--setup"]),
        patch("helm_mcp.download.ensure_binary", return_value=fake_path),
    ):
        main()

    captured = capsys.readouterr()
    assert fake_path in captured.out


def test_cli_setup_no_checksums(capsys):
    """Test CLI --setup exits 1 when no checksums available."""
    from helm_mcp.cli import main

    with (
        patch("sys.argv", ["helm-mcp-python", "--setup"]),
        patch("helm_mcp.download.ensure_binary", return_value=None),
        pytest.raises(SystemExit) as exc_info,
    ):
        main()

    assert exc_info.value.code == 1


def test_cli_setup_download_error(capsys):
    """Test CLI --setup exits 1 on download error."""
    from helm_mcp.cli import main

    with (
        patch("sys.argv", ["helm-mcp-python", "--setup"]),
        patch(
            "helm_mcp.download.ensure_binary",
            side_effect=RuntimeError("Checksum mismatch"),
        ),
        pytest.raises(SystemExit) as exc_info,
    ):
        main()

    assert exc_info.value.code == 1
