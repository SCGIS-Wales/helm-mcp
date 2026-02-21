#!/usr/bin/env python3
"""Build platform-specific wheels by injecting a Go binary into .data/scripts/.

When pip installs a wheel, files in .data/scripts/ are copied to the user's
scripts directory (e.g. ~/.local/bin, .venv/bin/) — the same place where
console_scripts entry points go. This makes the Go binary available on PATH.

Usage:
    # Single platform wheel:
    python scripts/build_wheels.py \\
        --wheel python/dist/helm_mcp-1.0.0-py3-none-any.whl \\
        --binary release-assets/helm-mcp-darwin-arm64 \\
        --platform macosx_11_0_arm64 \\
        --output dist/

    # All platform wheels at once:
    python scripts/build_wheels.py \\
        --wheel python/dist/helm_mcp-1.0.0-py3-none-any.whl \\
        --binaries-dir release-assets/ \\
        --output dist/
"""

from __future__ import annotations

import argparse
import base64
import hashlib
import logging
import os
import re
import sys
from pathlib import Path
from zipfile import ZipFile, ZipInfo, ZIP_DEFLATED

logger = logging.getLogger(__name__)

# Mapping from Go binary names to wheel platform tags.
# Go binaries are statically linked (CGO_ENABLED=0), so manylinux_2_17 is
# conservative and correct — no glibc dependency.
PLATFORM_MAP: dict[str, str] = {
    "helm-mcp-linux-amd64": "manylinux_2_17_x86_64.manylinux2014_x86_64",
    "helm-mcp-linux-arm64": "manylinux_2_17_aarch64.manylinux2014_aarch64",
    "helm-mcp-darwin-amd64": "macosx_10_15_x86_64",
    "helm-mcp-darwin-arm64": "macosx_11_0_arm64",
    "helm-mcp-windows-amd64.exe": "win_amd64",
}


def _record_entry(filename: str, data: bytes) -> str:
    """Create a RECORD entry: filename,sha256=<urlsafe-b64>,<size>."""
    digest = hashlib.sha256(data).digest()
    b64 = base64.urlsafe_b64encode(digest).rstrip(b"=").decode("ascii")
    return f"{filename},sha256={b64},{len(data)}"


def _parse_wheel_filename(wheel_path: Path) -> tuple[str, str]:
    """Extract (name, version) from a wheel filename."""
    # Wheel filenames: {name}-{version}-{python}-{abi}-{platform}.whl
    stem = wheel_path.stem
    parts = stem.split("-")
    if len(parts) < 3:
        raise ValueError(f"Invalid wheel filename: {wheel_path.name}")
    return parts[0], parts[1]


def build_platform_wheel(
    source_wheel: Path,
    binary: Path,
    platform_tag: str,
    output_dir: Path,
) -> Path:
    """Build a platform-specific wheel from a universal wheel + Go binary.

    Args:
        source_wheel: Path to the universal (py3-none-any) wheel.
        binary: Path to the Go binary for this platform.
        platform_tag: Wheel platform tag (e.g. 'macosx_11_0_arm64').
        output_dir: Directory to write the new wheel to.

    Returns:
        Path to the newly created platform wheel.
    """
    name, version = _parse_wheel_filename(source_wheel)
    dist_info = f"{name}-{version}.dist-info"
    data_dir = f"{name}-{version}.data"

    # Determine script name inside the wheel
    is_windows = "win" in platform_tag
    script_name = "helm-mcp.exe" if is_windows else "helm-mcp"

    # Output wheel filename
    out_name = f"{name}-{version}-py3-none-{platform_tag}.whl"
    out_path = output_dir / out_name

    records: list[str] = []

    with ZipFile(source_wheel, "r") as src, ZipFile(out_path, "w", ZIP_DEFLATED) as dst:
        # Copy all existing files except WHEEL and RECORD
        for item in src.infolist():
            if item.filename in (f"{dist_info}/WHEEL", f"{dist_info}/RECORD"):
                continue
            data = src.read(item.filename)
            dst.writestr(item, data)
            records.append(_record_entry(item.filename, data))

        # Add the Go binary inside the Python package for reliable access.
        # Using helm_mcp/bin/ instead of .data/scripts/ because pip does not
        # always preserve executable permissions on binary files placed in
        # .data/scripts/ (especially when create_system defaults to 0/Windows
        # in ZipInfo).  Console-script entry points created by pip handle
        # exec'ing these binaries and chmod'ing them on first run.
        binary_path_in_wheel = f"helm_mcp/bin/{script_name}"
        binary_data = binary.read_bytes()
        info = ZipInfo(binary_path_in_wheel)
        info.compress_type = ZIP_DEFLATED
        info.create_system = 3  # Unix
        info.external_attr = 0o100755 << 16  # regular file + rwxr-xr-x
        dst.writestr(info, binary_data)
        records.append(_record_entry(binary_path_in_wheel, binary_data))

        # Write updated WHEEL metadata with platform tag
        wheel_metadata = (
            "Wheel-Version: 1.0\n"
            "Generator: build_wheels.py\n"
            "Root-Is-Purelib: true\n"
            f"Tag: py3-none-{platform_tag}\n"
        )
        wheel_data = wheel_metadata.encode("utf-8")
        dst.writestr(f"{dist_info}/WHEEL", wheel_data)
        records.append(_record_entry(f"{dist_info}/WHEEL", wheel_data))

        # Write RECORD (self-entry has no hash)
        record_path = f"{dist_info}/RECORD"
        records.append(f"{record_path},,")
        record_content = "\n".join(records) + "\n"
        dst.writestr(record_path, record_content)

    logger.info("built %s (%.1f MB)", out_name, out_path.stat().st_size / 1e6)
    return out_path


def build_all(
    source_wheel: Path,
    binaries_dir: Path,
    output_dir: Path,
) -> list[Path]:
    """Build platform wheels for all Go binaries found in binaries_dir.

    Returns:
        List of paths to the created platform wheels.
    """
    wheels: list[Path] = []

    for filename in sorted(os.listdir(binaries_dir)):
        if filename not in PLATFORM_MAP:
            continue
        platform_tag = PLATFORM_MAP[filename]
        binary = binaries_dir / filename
        if not binary.is_file():
            continue
        wheel = build_platform_wheel(source_wheel, binary, platform_tag, output_dir)
        wheels.append(wheel)

    return wheels


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Build platform-specific Python wheels with bundled Go binary",
    )
    parser.add_argument(
        "--wheel",
        required=True,
        help="Path to the universal (py3-none-any) wheel",
    )
    parser.add_argument(
        "--binary",
        help="Path to a single Go binary (use with --platform)",
    )
    parser.add_argument(
        "--platform",
        help="Wheel platform tag (e.g. macosx_11_0_arm64)",
    )
    parser.add_argument(
        "--binaries-dir",
        help="Directory containing Go binaries (batch mode)",
    )
    parser.add_argument(
        "--output",
        required=True,
        help="Output directory for platform wheels",
    )
    args = parser.parse_args()

    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(message)s",
        stream=sys.stderr,
    )

    source_wheel = Path(args.wheel)
    output_dir = Path(args.output)
    output_dir.mkdir(parents=True, exist_ok=True)

    if not source_wheel.is_file():
        # Support glob patterns (e.g. dist/helm_mcp-*.whl)
        import glob

        matches = glob.glob(str(source_wheel))
        if len(matches) == 1:
            source_wheel = Path(matches[0])
        elif len(matches) == 0:
            logger.error("wheel not found: %s", args.wheel)
            sys.exit(1)
        else:
            logger.error("multiple wheels match: %s", matches)
            sys.exit(1)

    if args.binaries_dir:
        binaries_dir = Path(args.binaries_dir)
        if not binaries_dir.is_dir():
            logger.error("binaries directory not found: %s", binaries_dir)
            sys.exit(1)
        wheels = build_all(source_wheel, binaries_dir, output_dir)
        logger.info("built %d platform wheels", len(wheels))
    elif args.binary and args.platform:
        binary = Path(args.binary)
        if not binary.is_file():
            logger.error("binary not found: %s", binary)
            sys.exit(1)
        build_platform_wheel(source_wheel, binary, args.platform, output_dir)
    else:
        logger.error("specify either --binaries-dir or both --binary and --platform")
        sys.exit(1)


if __name__ == "__main__":
    main()
