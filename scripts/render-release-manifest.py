#!/usr/bin/env python3
"""Record the source revision and hashes for a local release candidate."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import platform
from pathlib import Path
import subprocess


def command_version(command: list[str]) -> str | None:
    try:
        result = subprocess.run(
            command,
            check=True,
            capture_output=True,
            text=True,
        )
    except (OSError, subprocess.CalledProcessError):
        return None
    line = (result.stdout or result.stderr).splitlines()
    return line[0].strip() if line else None


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--version")
    parser.add_argument("--output", type=Path)
    parser.add_argument(
        "--base-manifest",
        type=Path,
        help="preserve verified toolchain/package metadata from an earlier candidate",
    )
    args = parser.parse_args()
    root = args.root.resolve()
    version = args.version or (root / "release/VERSION").read_text().strip()
    output = args.output or root / "dist/release-manifest.json"
    compatibility = json.loads((root / "release/compatibility.json").read_text())
    source_revision = subprocess.check_output(
        ["git", "rev-parse", "HEAD"], cwd=root, text=True
    ).strip()

    base_metadata: dict[str, object] = {}
    if args.base_manifest is not None:
        base_path = args.base_manifest.expanduser()
        if not base_path.is_absolute():
            base_path = root / base_path
        if base_path.is_symlink() or not base_path.is_file():
            raise SystemExit(f"base manifest is not a regular file: {base_path}")
        try:
            base_path.resolve().relative_to(root / "dist")
        except ValueError as exc:
            raise SystemExit("base manifest must be inside dist/") from exc
        base_metadata = json.loads(base_path.read_text())
        if not isinstance(base_metadata, dict):
            raise SystemExit("base manifest must contain a JSON object")
        if base_metadata.get("source_revision") != source_revision:
            raise SystemExit("base manifest does not describe the current source revision")
        if base_metadata.get("source_dirty"):
            raise SystemExit("base manifest describes a dirty source tree")
        for field in ("toolchains", "package_tools"):
            if not isinstance(base_metadata.get(field), dict):
                raise SystemExit(f"base manifest has invalid {field} metadata")

    artifacts: list[dict[str, object]] = []
    for directory in (root / "dist/ha", root / "dist/macos", root / "dist/mcp"):
        if not directory.is_dir():
            continue
        for path in sorted(directory.iterdir()):
            if path.is_file():
                artifacts.append(
                    {
                        "path": str(path.relative_to(root)),
                        "sha256": hashlib.sha256(path.read_bytes()).hexdigest(),
                        "bytes": path.stat().st_size,
                    }
                )

    export = root / "dist/ha-fruit-forwarder"
    if export.is_dir():
        manifest = json.loads(
            (export / "custom_components/icloud_reminders_bridge/manifest.json").read_text()
        )
        artifacts.append(
            {
                "path": str(export.relative_to(root)),
                "kind": "directory",
                "manifest_version": manifest["version"],
            }
        )

    local_toolchains = {
        "platform": platform.platform(),
        "python": platform.python_version(),
        "go": command_version(["go", "version"]),
        "swift": command_version(["swift", "--version"]),
    }
    local_package_tools = {
        "mcpb_cli": os.environ.get("MCPB_VERSION", "2.1.2"),
        "macos_codesign_identity": os.environ.get("FRUIT_FORWARDER_SIGN_IDENTITY", "-"),
    }
    result = {
        "product": "Fruit Forwarder",
        "version": version,
        "compatibility": compatibility,
        "toolchains": base_metadata.get("toolchains", local_toolchains),
        "package_tools": base_metadata.get("package_tools", local_package_tools),
        "source_revision": source_revision,
        "source_dirty": bool(
            subprocess.check_output(["git", "status", "--porcelain"], cwd=root, text=True).strip()
        ),
        "artifacts": artifacts,
    }
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(result, indent=2) + "\n")
    print(output)


if __name__ == "__main__":
    main()
