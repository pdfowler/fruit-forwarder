#!/usr/bin/env python3
"""Record the source revision and hashes for a local release candidate."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import subprocess


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--version")
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    root = args.root.resolve()
    version = args.version or (root / "release/VERSION").read_text().strip()
    output = args.output or root / "dist/release-manifest.json"

    artifacts: list[dict[str, object]] = []
    for directory in (root / "dist/macos", root / "dist/mcp"):
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

    result = {
        "product": "Fruit Forwarder",
        "version": version,
        "source_revision": subprocess.check_output(
            ["git", "rev-parse", "HEAD"], cwd=root, text=True
        ).strip(),
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
