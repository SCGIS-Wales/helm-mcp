"""Functional test: build a platform wheel, install it, and verify CLI entry points.

This test simulates the end-to-end user experience:
  1. Build a universal wheel from the source tree
  2. Build a platform-specific wheel (with a fake binary)
  3. Install the platform wheel into an isolated virtualenv
  4. Verify that ``helm-mcp`` and ``helm-mcp-python`` commands are
     both available and executable

Run with:
    pytest -v scripts/test_wheel_install.py
"""

from __future__ import annotations

import os
import platform as plat
import subprocess
import sys
from pathlib import Path
from zipfile import ZipFile

import pytest

# Root of the project
PROJECT_ROOT = Path(__file__).resolve().parent.parent
PYTHON_DIR = PROJECT_ROOT / "python"


def _make_fake_binary(binary_path: Path) -> None:
    """Create a fake Go binary that prints a version string."""
    binary_path.parent.mkdir(parents=True, exist_ok=True)
    script = "#!/bin/sh\necho 'helm-mcp 0.0.0-test'\n"
    binary_path.write_text(script)
    binary_path.chmod(0o755)


class TestWheelInstall:
    """End-to-end test: build wheel, install, verify commands work."""

    @pytest.fixture
    def venv(self, tmp_path):
        """Create an isolated virtualenv for installation testing."""
        venv_dir = tmp_path / "venv"
        subprocess.check_call([sys.executable, "-m", "venv", str(venv_dir)])
        return venv_dir

    @pytest.fixture
    def venv_python(self, venv):
        """Return the Python executable inside the virtualenv."""
        return str(venv / "bin" / "python")

    @pytest.fixture
    def venv_bin(self, venv):
        """Return the bin directory inside the virtualenv."""
        return venv / "bin"

    @pytest.fixture
    def platform_wheel(self, tmp_path):
        """Build a platform-specific wheel with a fake binary."""
        # 1. Build the universal wheel from the source tree
        dist_dir = tmp_path / "dist"
        dist_dir.mkdir()
        subprocess.check_call(
            [sys.executable, "-m", "pip", "install", "build"],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        subprocess.check_call(
            [
                sys.executable,
                "-m",
                "build",
                "--wheel",
                str(PYTHON_DIR),
                "--outdir",
                str(dist_dir),
            ],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )

        # Find the built universal wheel
        wheels = list(dist_dir.glob("helm_mcp-*-py3-none-any.whl"))
        assert len(wheels) == 1, f"Expected 1 universal wheel, got: {wheels}"
        universal_wheel = wheels[0]

        # 2. Create a fake binary
        system = plat.system().lower()
        machine = plat.machine().lower()
        binary_map = {
            ("darwin", "arm64"): "helm-mcp-darwin-arm64",
            ("darwin", "x86_64"): "helm-mcp-darwin-amd64",
            ("linux", "x86_64"): "helm-mcp-linux-amd64",
            ("linux", "aarch64"): "helm-mcp-linux-amd64",
        }
        binary_name = binary_map.get((system, machine))
        if binary_name is None:
            pytest.skip(f"Unsupported platform: {system}/{machine}")

        binary_dir = tmp_path / "binaries"
        fake_binary = binary_dir / binary_name
        _make_fake_binary(fake_binary)

        # 3. Build the platform-specific wheel
        sys.path.insert(0, str(PROJECT_ROOT / "scripts"))
        try:
            from build_wheels import PLATFORM_MAP, build_platform_wheel

            platform_tag = PLATFORM_MAP[binary_name]
            output_dir = tmp_path / "platform-dist"
            output_dir.mkdir()
            wheel_path = build_platform_wheel(
                universal_wheel, fake_binary, platform_tag, output_dir
            )
        finally:
            sys.path.pop(0)

        return wheel_path

    def test_wheel_installs_successfully(self, venv_python, platform_wheel):
        """Test that the platform wheel can be installed without errors."""
        result = subprocess.run(
            [venv_python, "-m", "pip", "install", str(platform_wheel)],
            capture_output=True,
            text=True,
        )
        assert result.returncode == 0, f"pip install failed:\n{result.stderr}"

    def test_helm_mcp_command_available(self, venv_python, venv_bin, platform_wheel):
        """Test that the 'helm-mcp' command is available after installation."""
        subprocess.check_call(
            [venv_python, "-m", "pip", "install", str(platform_wheel)],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        mcp_cmd = venv_bin / "helm-mcp"
        assert mcp_cmd.exists(), f"helm-mcp command not found at {mcp_cmd}"
        assert os.access(str(mcp_cmd), os.X_OK), "helm-mcp command is not executable"

    def test_helm_mcp_python_command_available(
        self, venv_python, venv_bin, platform_wheel
    ):
        """Test that the 'helm-mcp-python' command is available after installation."""
        subprocess.check_call(
            [venv_python, "-m", "pip", "install", str(platform_wheel)],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        py_cmd = venv_bin / "helm-mcp-python"
        assert py_cmd.exists(), f"helm-mcp-python command not found at {py_cmd}"
        assert os.access(str(py_cmd), os.X_OK), "helm-mcp-python command is not executable"

    def test_helm_mcp_command_executes_binary(
        self, venv_python, venv_bin, platform_wheel
    ):
        """Test that 'helm-mcp' command executes the bundled binary."""
        subprocess.check_call(
            [venv_python, "-m", "pip", "install", str(platform_wheel)],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        result = subprocess.run(
            [str(venv_bin / "helm-mcp")],
            capture_output=True,
            text=True,
            timeout=10,
        )
        # The fake binary prints "helm-mcp 0.0.0-test"
        assert "helm-mcp 0.0.0-test" in result.stdout

    def test_helm_mcp_python_help(self, venv_python, venv_bin, platform_wheel):
        """Test that 'helm-mcp-python --help' works."""
        subprocess.check_call(
            [venv_python, "-m", "pip", "install", str(platform_wheel)],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        result = subprocess.run(
            [str(venv_bin / "helm-mcp-python"), "--help"],
            capture_output=True,
            text=True,
            timeout=10,
        )
        assert result.returncode == 0
        assert "helm-mcp" in result.stdout.lower()

    def test_binary_bundled_in_package(self, venv_python, platform_wheel):
        """Test that the binary is bundled inside the helm_mcp/bin/ package directory."""
        subprocess.check_call(
            [venv_python, "-m", "pip", "install", str(platform_wheel)],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        result = subprocess.run(
            [
                venv_python,
                "-c",
                "from pathlib import Path; import helm_mcp; "
                "bin_dir = Path(helm_mcp.__file__).parent / 'bin'; "
                "print(list(sorted(p.name for p in bin_dir.iterdir())))",
            ],
            capture_output=True,
            text=True,
            timeout=10,
        )
        assert result.returncode == 0, f"Failed: {result.stderr}"
        assert "'helm-mcp'" in result.stdout
