#!/usr/bin/env python3
"""Verify that a release manifest describes the current clean checkout and bytes."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import subprocess


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--manifest", type=Path)
    args = parser.parse_args()

    root = args.root.resolve()
    manifest_candidate = args.manifest or root / "dist/release-manifest.json"
    if manifest_candidate.is_symlink():
        raise SystemExit(f"release manifest is a symlink: {manifest_candidate}")
    manifest_path = manifest_candidate.resolve()
    try:
        manifest_path.relative_to(root)
    except ValueError as exc:
        raise SystemExit("manifest must be inside the repository root") from exc
    if not manifest_path.is_file() or manifest_path.is_symlink():
        raise SystemExit(f"release manifest is not a regular file: {manifest_path}")

    manifest = json.loads(manifest_path.read_text())
    version = (root / "release/VERSION").read_text().strip()
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
    if manifest.get("version") != version:
        raise SystemExit("release manifest version does not match release/VERSION")
    if manifest.get("source_revision") != head or manifest.get("source_dirty"):
        raise SystemExit("release manifest does not describe the current clean source revision")

    artifacts = manifest.get("artifacts")
    if not isinstance(artifacts, list) or not artifacts:
        raise SystemExit("release manifest contains no artifacts")
    seen: set[str] = set()
    for artifact in artifacts:
        if not isinstance(artifact, dict) or not isinstance(artifact.get("path"), str):
            raise SystemExit("release manifest contains an invalid artifact record")
        relative = Path(artifact["path"])
        if relative.is_absolute() or ".." in relative.parts:
            raise SystemExit(f"artifact path is unsafe: {relative}")
        key = str(relative)
        if key in seen:
            raise SystemExit(f"artifact is listed more than once: {relative}")
        seen.add(key)
        candidate = root / relative
        if candidate.is_symlink():
            raise SystemExit(f"artifact is a symlink: {relative}")
        path = candidate.resolve()
        try:
            path.relative_to(root / "dist")
        except ValueError as exc:
            raise SystemExit(f"artifact is outside dist/: {relative}") from exc
        if artifact.get("kind") == "directory":
            if not path.is_dir() or path.is_symlink():
                raise SystemExit(f"artifact directory is missing or unsafe: {relative}")
            manifest_file = path / "custom_components/icloud_reminders_bridge/manifest.json"
            if artifact.get("manifest_version") is not None:
                if not manifest_file.is_file() or manifest_file.is_symlink():
                    raise SystemExit(f"artifact directory has no regular integration manifest: {relative}")
                component_version = json.loads(manifest_file.read_text()).get("version")
                if component_version != artifact["manifest_version"]:
                    raise SystemExit(f"HACS artifact version mismatch: {relative}")
            continue
        if not path.is_file() or path.is_symlink():
            raise SystemExit(f"artifact file is missing or unsafe: {relative}")
        actual_hash = hashlib.sha256(path.read_bytes()).hexdigest()
        if artifact.get("sha256") != actual_hash:
            raise SystemExit(f"artifact checksum mismatch: {relative}")
        if artifact.get("bytes") != path.stat().st_size:
            raise SystemExit(f"artifact size mismatch: {relative}")

    print(f"validated release manifest: {manifest_path}")


if __name__ == "__main__":
    main()
