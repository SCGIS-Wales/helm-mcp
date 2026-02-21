"""Auto-download helm-mcp Go binary from GitHub Releases.

Downloads the platform-appropriate binary, verifies its SHA256 checksum
against values embedded in the package (checksums.json), and installs
it to a PATH-accessible directory. This ensures ``pip install helm-mcp``
is all users need — no separate Go binary setup required.

Supply chain security:
  - Checksums are baked into the wheel at build time, not fetched at runtime.
  - Downloads use HTTPS with certificate verification.
  - Binary is written to a temp file and only renamed after checksum passes.
  - No shell commands or install-time hooks are used.
"""

import hashlib
import json
import logging
import os
import platform
import stat
import sys
import sysconfig
import tempfile
import urllib.request
from pathlib import Path

logger = logging.getLogger(__name__)

GITHUB_RELEASE_URL = (
    "https://github.com/SCGIS-Wales/helm-mcp/releases/download/v{version}/{binary_name}"
)


def _load_checksums() -> dict:
    """Load embedded checksums from package data.

    Returns:
        Parsed checksums dict, or empty dict if file is missing.
    """
    checksums_path = Path(__file__).parent / "checksums.json"
    if not checksums_path.exists():
        return {}
    with open(checksums_path) as f:
        return json.load(f)


def _get_binary_name() -> str:
    """Get the platform-specific binary filename.

    Returns:
        Binary name like ``helm-mcp-darwin-arm64`` or
        ``helm-mcp-windows-amd64.exe``.
    """
    system = platform.system().lower()
    machine = platform.machine().lower()
    arch_map = {
        "x86_64": "amd64",
        "aarch64": "arm64",
        "arm64": "arm64",
        "amd64": "amd64",
    }
    arch = arch_map.get(machine, machine)
    name = f"helm-mcp-{system}-{arch}"
    if system == "windows":
        name += ".exe"
    return name


def _get_install_dir() -> Path:
    """Get the best directory for installing the binary.

    Prefers the Python scripts directory (where pip puts console_scripts,
    which is on PATH). Falls back to ``~/.local/bin``.

    Returns:
        Writable directory path.
    """
    scripts_dir = Path(sysconfig.get_path("scripts"))
    if os.access(str(scripts_dir), os.W_OK):
        return scripts_dir
    # Fallback: user-local bin
    local_bin = Path.home() / ".local" / "bin"
    local_bin.mkdir(parents=True, exist_ok=True)
    return local_bin


def _verify_checksum(file_path: Path, expected_sha256: str) -> bool:
    """Verify SHA256 checksum of a file.

    Args:
        file_path: Path to the file to verify.
        expected_sha256: Expected hex-encoded SHA256 digest.

    Returns:
        True if checksum matches, False otherwise.
    """
    sha256 = hashlib.sha256()
    with open(file_path, "rb") as f:
        for chunk in iter(lambda: f.read(8192), b""):
            sha256.update(chunk)
    return sha256.hexdigest() == expected_sha256


def ensure_binary(version: str) -> str | None:
    """Ensure the helm-mcp binary is available, downloading if needed.

    If the binary already exists in the install directory and is executable,
    returns its path immediately. Otherwise downloads from GitHub Releases,
    verifies the SHA256 checksum, and installs it.

    Args:
        version: Package version (e.g. ``"0.1.5"``). Used to construct
            the download URL and match against checksums.

    Returns:
        Absolute path to the binary, or ``None`` if download is not
        possible (e.g. no checksums available for this platform).

    Raises:
        RuntimeError: If the downloaded binary fails checksum verification.
        urllib.error.URLError: If the download fails.
    """
    binary_name = _get_binary_name()
    install_dir = _get_install_dir()

    target_name = "helm-mcp.exe" if platform.system().lower() == "windows" else "helm-mcp"
    target = install_dir / target_name

    # Already installed?
    if target.exists() and os.access(str(target), os.X_OK):
        return str(target)

    # Load embedded checksums
    checksums = _load_checksums()
    expected = checksums.get("binaries", {}).get(binary_name)
    if not expected:
        logger.debug("No checksum for %s — skipping auto-download", binary_name)
        return None

    url = GITHUB_RELEASE_URL.format(version=version, binary_name=binary_name)
    logger.info("Downloading helm-mcp binary from %s", url)
    print(
        f"Downloading helm-mcp binary for {platform.system()}/{platform.machine()}...",
        file=sys.stderr,
    )

    # Download to temp file, verify checksum, then atomic rename
    fd, tmp_path = tempfile.mkstemp(dir=str(install_dir), prefix=".helm-mcp-")
    try:
        os.close(fd)
        urllib.request.urlretrieve(url, tmp_path)  # noqa: S310 — URL is hardcoded HTTPS

        if not _verify_checksum(Path(tmp_path), expected):
            raise RuntimeError(
                f"Checksum mismatch for {binary_name}. "
                "The downloaded binary does not match the expected hash. "
                "This could indicate a tampered download."
            )

        # Make executable and atomically move into place
        os.chmod(tmp_path, os.stat(tmp_path).st_mode | stat.S_IEXEC | stat.S_IXGRP | stat.S_IXOTH)
        os.replace(tmp_path, str(target))

        print(f"Installed helm-mcp to {target}", file=sys.stderr)
        logger.info("Installed helm-mcp to %s", target)
        return str(target)
    except Exception:
        # Clean up temp file on any failure
        if os.path.exists(tmp_path):
            os.unlink(tmp_path)
        raise
