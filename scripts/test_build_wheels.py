#!/usr/bin/env python3
"""Tests for scripts/build_wheels.py — platform-specific wheel builder."""

from __future__ import annotations

import base64
import hashlib
import os
import sys
import tempfile
from pathlib import Path
from zipfile import ZipFile, ZIP_DEFLATED

import pytest

# Add scripts/ to sys.path so we can import build_wheels
sys.path.insert(0, str(Path(__file__).parent))

from build_wheels import (
    PLATFORM_MAP,
    _parse_wheel_filename,
    _record_entry,
    build_all,
    build_platform_wheel,
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_universal_wheel(tmp_path: Path, name: str = "helm_mcp", version: str = "1.0.0") -> Path:
    """Create a minimal universal wheel for testing."""
    tmp_path.mkdir(parents=True, exist_ok=True)
    dist_info = f"{name}-{version}.dist-info"
    wheel_name = f"{name}-{version}-py3-none-any.whl"
    wheel_path = tmp_path / wheel_name

    with ZipFile(wheel_path, "w", ZIP_DEFLATED) as zf:
        # Minimal Python module
        zf.writestr(f"{name}/__init__.py", f'__version__ = "{version}"\n')

        # WHEEL metadata
        wheel_meta = (
            "Wheel-Version: 1.0\n"
            "Generator: test\n"
            "Root-Is-Purelib: true\n"
            "Tag: py3-none-any\n"
        )
        zf.writestr(f"{dist_info}/WHEEL", wheel_meta)

        # METADATA
        metadata = (
            f"Metadata-Version: 2.1\n"
            f"Name: {name.replace('_', '-')}\n"
            f"Version: {version}\n"
        )
        zf.writestr(f"{dist_info}/METADATA", metadata)

        # RECORD (simplified)
        zf.writestr(f"{dist_info}/RECORD", "")

    return wheel_path


def _make_fake_binary(tmp_path: Path, name: str) -> Path:
    """Create a fake Go binary file."""
    tmp_path.mkdir(parents=True, exist_ok=True)
    binary = tmp_path / name
    binary.write_bytes(b"\x7fELF" + os.urandom(1024))  # Fake ELF header + random data
    return binary


# ---------------------------------------------------------------------------
# _record_entry
# ---------------------------------------------------------------------------


class TestRecordEntry:
    """Test RECORD entry generation."""

    def test_record_format(self):
        data = b"hello world"
        entry = _record_entry("some/file.py", data)

        digest = hashlib.sha256(data).digest()
        expected_b64 = base64.urlsafe_b64encode(digest).rstrip(b"=").decode("ascii")

        assert entry == f"some/file.py,sha256={expected_b64},{len(data)}"

    def test_record_empty_data(self):
        entry = _record_entry("empty.txt", b"")
        assert entry.startswith("empty.txt,sha256=")
        assert entry.endswith(",0")

    def test_record_large_data(self):
        data = os.urandom(1024 * 1024)  # 1MB
        entry = _record_entry("large.bin", data)
        parts = entry.split(",")
        assert len(parts) == 3
        assert parts[0] == "large.bin"
        assert parts[1].startswith("sha256=")
        assert parts[2] == str(len(data))


# ---------------------------------------------------------------------------
# _parse_wheel_filename
# ---------------------------------------------------------------------------


class TestParseWheelFilename:
    """Test wheel filename parsing."""

    def test_standard_wheel(self):
        name, version = _parse_wheel_filename(
            Path("helm_mcp-1.0.0-py3-none-any.whl")
        )
        assert name == "helm_mcp"
        assert version == "1.0.0"

    def test_platform_wheel(self):
        name, version = _parse_wheel_filename(
            Path("helm_mcp-2.1.0-py3-none-manylinux_2_17_x86_64.whl")
        )
        assert name == "helm_mcp"
        assert version == "2.1.0"

    def test_invalid_filename(self):
        with pytest.raises(ValueError, match="Invalid wheel filename"):
            _parse_wheel_filename(Path("invalid.whl"))

    def test_two_part_filename(self):
        with pytest.raises(ValueError, match="Invalid wheel filename"):
            _parse_wheel_filename(Path("name-version.whl"))


# ---------------------------------------------------------------------------
# PLATFORM_MAP
# ---------------------------------------------------------------------------


class TestPlatformMap:
    """Test the platform mapping constants."""

    def test_all_platforms_present(self):
        assert "helm-mcp-linux-amd64" in PLATFORM_MAP
        assert "helm-mcp-linux-arm64" in PLATFORM_MAP
        assert "helm-mcp-darwin-amd64" in PLATFORM_MAP
        assert "helm-mcp-darwin-arm64" in PLATFORM_MAP
        assert "helm-mcp-windows-amd64.exe" in PLATFORM_MAP

    def test_platform_count(self):
        assert len(PLATFORM_MAP) == 5

    def test_linux_tags_include_manylinux(self):
        assert "manylinux_2_17" in PLATFORM_MAP["helm-mcp-linux-amd64"]
        assert "manylinux_2_17" in PLATFORM_MAP["helm-mcp-linux-arm64"]

    def test_macos_tags(self):
        assert "macosx" in PLATFORM_MAP["helm-mcp-darwin-amd64"]
        assert "macosx" in PLATFORM_MAP["helm-mcp-darwin-arm64"]

    def test_windows_tag(self):
        assert PLATFORM_MAP["helm-mcp-windows-amd64.exe"] == "win_amd64"


# ---------------------------------------------------------------------------
# build_platform_wheel
# ---------------------------------------------------------------------------


class TestBuildPlatformWheel:
    """Test building a single platform-specific wheel."""

    def test_creates_wheel_file(self, tmp_path):
        source = _make_universal_wheel(tmp_path / "src")
        binary = _make_fake_binary(tmp_path / "bin", "helm-mcp-darwin-arm64")
        output_dir = tmp_path / "output"
        output_dir.mkdir()

        result = build_platform_wheel(
            source, binary, "macosx_11_0_arm64", output_dir
        )

        assert result.exists()
        assert result.suffix == ".whl"
        assert "macosx_11_0_arm64" in result.name

    def test_wheel_contains_binary(self, tmp_path):
        source = _make_universal_wheel(tmp_path / "src")
        binary = _make_fake_binary(tmp_path / "bin", "helm-mcp-darwin-arm64")
        output_dir = tmp_path / "output"
        output_dir.mkdir()

        result = build_platform_wheel(
            source, binary, "macosx_11_0_arm64", output_dir
        )

        with ZipFile(result, "r") as zf:
            names = zf.namelist()
            binary_path = "helm_mcp-1.0.0.data/scripts/helm-mcp"
            assert binary_path in names

    def test_binary_has_executable_permissions(self, tmp_path):
        source = _make_universal_wheel(tmp_path / "src")
        binary = _make_fake_binary(tmp_path / "bin", "helm-mcp-linux-amd64")
        output_dir = tmp_path / "output"
        output_dir.mkdir()

        result = build_platform_wheel(
            source, binary, "manylinux_2_17_x86_64", output_dir
        )

        with ZipFile(result, "r") as zf:
            info = zf.getinfo("helm_mcp-1.0.0.data/scripts/helm-mcp")
            unix_perms = (info.external_attr >> 16) & 0o777
            assert unix_perms == 0o755

    def test_wheel_metadata_has_platform_tag(self, tmp_path):
        source = _make_universal_wheel(tmp_path / "src")
        binary = _make_fake_binary(tmp_path / "bin", "helm-mcp-darwin-arm64")
        output_dir = tmp_path / "output"
        output_dir.mkdir()

        result = build_platform_wheel(
            source, binary, "macosx_11_0_arm64", output_dir
        )

        with ZipFile(result, "r") as zf:
            wheel_data = zf.read("helm_mcp-1.0.0.dist-info/WHEEL").decode("utf-8")
            assert "Tag: py3-none-macosx_11_0_arm64" in wheel_data

    def test_record_has_valid_hashes(self, tmp_path):
        source = _make_universal_wheel(tmp_path / "src")
        binary = _make_fake_binary(tmp_path / "bin", "helm-mcp-darwin-arm64")
        output_dir = tmp_path / "output"
        output_dir.mkdir()

        result = build_platform_wheel(
            source, binary, "macosx_11_0_arm64", output_dir
        )

        with ZipFile(result, "r") as zf:
            record = zf.read("helm_mcp-1.0.0.dist-info/RECORD").decode("utf-8")
            lines = [l for l in record.strip().split("\n") if l]

            # RECORD itself has no hash
            record_line = [l for l in lines if "RECORD" in l]
            assert any(l.endswith(",,") for l in record_line)

            # Other entries have hashes
            hash_lines = [l for l in lines if not l.endswith(",,")]
            for line in hash_lines:
                parts = line.split(",")
                assert len(parts) == 3
                assert parts[1].startswith("sha256=")
                assert int(parts[2]) >= 0

    def test_original_files_preserved(self, tmp_path):
        source = _make_universal_wheel(tmp_path / "src")
        binary = _make_fake_binary(tmp_path / "bin", "helm-mcp-darwin-arm64")
        output_dir = tmp_path / "output"
        output_dir.mkdir()

        result = build_platform_wheel(
            source, binary, "macosx_11_0_arm64", output_dir
        )

        with ZipFile(result, "r") as zf:
            names = zf.namelist()
            assert "helm_mcp/__init__.py" in names
            assert "helm_mcp-1.0.0.dist-info/METADATA" in names

    def test_windows_binary_name(self, tmp_path):
        source = _make_universal_wheel(tmp_path / "src")
        binary = _make_fake_binary(tmp_path / "bin", "helm-mcp-windows-amd64.exe")
        output_dir = tmp_path / "output"
        output_dir.mkdir()

        result = build_platform_wheel(
            source, binary, "win_amd64", output_dir
        )

        with ZipFile(result, "r") as zf:
            names = zf.namelist()
            assert "helm_mcp-1.0.0.data/scripts/helm-mcp.exe" in names


# ---------------------------------------------------------------------------
# build_all (batch mode)
# ---------------------------------------------------------------------------


class TestBuildAll:
    """Test batch wheel building."""

    def test_builds_all_matching_binaries(self, tmp_path):
        source = _make_universal_wheel(tmp_path / "src")
        binaries_dir = tmp_path / "binaries"
        binaries_dir.mkdir()
        output_dir = tmp_path / "output"
        output_dir.mkdir()

        # Create all 5 platform binaries
        for name in PLATFORM_MAP:
            _make_fake_binary(binaries_dir, name)

        wheels = build_all(source, binaries_dir, output_dir)
        assert len(wheels) == 5

        # Verify each has the correct platform tag
        wheel_names = {w.name for w in wheels}
        for platform_tag in PLATFORM_MAP.values():
            assert any(platform_tag in name for name in wheel_names)

    def test_skips_unknown_binaries(self, tmp_path):
        source = _make_universal_wheel(tmp_path / "src")
        binaries_dir = tmp_path / "binaries"
        binaries_dir.mkdir()
        output_dir = tmp_path / "output"
        output_dir.mkdir()

        # Only create one known binary plus one unknown
        _make_fake_binary(binaries_dir, "helm-mcp-darwin-arm64")
        _make_fake_binary(binaries_dir, "helm-mcp-freebsd-amd64")  # Not in PLATFORM_MAP

        wheels = build_all(source, binaries_dir, output_dir)
        assert len(wheels) == 1

    def test_empty_binaries_dir(self, tmp_path):
        source = _make_universal_wheel(tmp_path / "src")
        binaries_dir = tmp_path / "binaries"
        binaries_dir.mkdir()
        output_dir = tmp_path / "output"
        output_dir.mkdir()

        wheels = build_all(source, binaries_dir, output_dir)
        assert len(wheels) == 0
