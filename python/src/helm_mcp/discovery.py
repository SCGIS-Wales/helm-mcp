"""Locating the helm-mcp Go binary.

Both entry points need this: ``helm-mcp`` execs the binary directly and
``helm-mcp-python`` spawns it behind a FastMCP proxy. They used to carry
separate copies of the logic that had drifted apart — only one of them
honoured ``HELM_MCP_BINARY`` — so the search lives here now.

Search order:

1. ``HELM_MCP_BINARY`` environment variable
2. Bundled binary in the package ``bin/`` directory (platform wheels)
3. ``helm-mcp`` on ``PATH``
4. Auto-download from GitHub Releases (fallback for the universal wheel)
"""

from __future__ import annotations

import logging
import os
import platform
import shutil
import stat
from pathlib import Path

logger = logging.getLogger(__name__)

DEFAULT_BINARY_NAME = "helm-mcp"

_ARCH_MAP = {
    "x86_64": "amd64",
    "aarch64": "arm64",
    "arm64": "arm64",
    "amd64": "amd64",
}


def is_python_script(path: str) -> bool:
    """Report whether *path* is a Python script rather than a native binary.

    ``shutil.which("helm-mcp")`` will happily return the console-script
    wrapper pip installs for this very package. Exec'ing that would put the
    process into an infinite loop replacing itself, so the PATH lookup has to
    be able to recognise and skip it.
    """
    try:
        with Path(path).open("rb") as fh:
            head = fh.read(128)
        first_line = head.split(b"\n", 1)[0].lower()
        return head[:2] == b"#!" and b"python" in first_line
    except OSError:
        return False


def platform_binary_name() -> str:
    """Return the bundled binary name for the current platform."""
    system = platform.system().lower()
    arch = _ARCH_MAP.get(platform.machine().lower(), platform.machine().lower())
    name = f"{DEFAULT_BINARY_NAME}-{system}-{arch}"
    return f"{name}.exe" if system == "windows" else name


def _make_executable(path: Path) -> bool:
    """Ensure *path* is executable, returning whether it now is.

    pip does not reliably preserve the executable bit for package-data files
    extracted from a wheel, so the bundled binary often arrives without it.
    """
    if os.access(str(path), os.X_OK):
        return True
    try:
        path.chmod(path.stat().st_mode | stat.S_IEXEC | stat.S_IXGRP | stat.S_IXOTH)
    except OSError:
        return False
    return True


def find_bundled_binary(name: str | None = None) -> str | None:
    """Return the path to the binary bundled inside the package, if present."""
    bin_dir = Path(__file__).parent / "bin"
    candidates = [name] if name else [platform_binary_name(), DEFAULT_BINARY_NAME]
    for candidate in candidates:
        path = bin_dir / candidate
        if path.is_file() and _make_executable(path):
            return str(path)
    return None


def find_binary(name: str = DEFAULT_BINARY_NAME) -> str:
    """Locate the helm-mcp binary, downloading it if necessary.

    Raises:
        FileNotFoundError: If the binary cannot be located.
    """
    env_path = os.environ.get("HELM_MCP_BINARY")
    if env_path:
        path = Path(env_path)
        if path.is_file() and os.access(str(path), os.X_OK):
            logger.info("using binary from HELM_MCP_BINARY: %s", path)
            return str(path)
        raise FileNotFoundError(f"HELM_MCP_BINARY={env_path} does not exist or is not executable")

    bundled = find_bundled_binary()
    if bundled:
        logger.info("using bundled binary: %s", bundled)
        return bundled

    # PATH is tried before downloading so a `go install` or Homebrew build wins
    # over a network fetch. Console-script wrappers are skipped.
    found = shutil.which(name)
    if found and not is_python_script(found):
        logger.info("using binary from PATH: %s", found)
        return found

    from helm_mcp import __version__
    from helm_mcp.download import ensure_binary

    try:
        downloaded = ensure_binary(__version__)
        if downloaded:
            logger.info("using auto-downloaded binary: %s", downloaded)
            return downloaded
    except Exception:
        logger.warning("auto-download failed", exc_info=True)

    raise FileNotFoundError(
        f"{name} binary not found. Either:\n"
        "  1. Set HELM_MCP_BINARY=/path/to/helm-mcp\n"
        "  2. Install helm-mcp and ensure it's on your PATH\n"
        "  3. Run 'helm-mcp-python --setup' to download it"
    )
