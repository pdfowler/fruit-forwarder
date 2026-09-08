"""Release-manifest merge behavior stays provenance-preserving."""

import json
import os
from pathlib import Path
import subprocess
import sys


ROOT = Path(__file__).parents[1]
RENDERER = ROOT / "scripts/render-release-manifest.py"


def test_base_manifest_preserves_macos_provenance_when_merging_targets():
    dist = ROOT / "dist"
    dist.mkdir(exist_ok=True)
    suffix = f"{os.getpid()}"
    base = dist / f".test-base-manifest-{suffix}.json"
    output = dist / f".test-merged-manifest-{suffix}.json"
    source_revision = subprocess.check_output(
        ["git", "rev-parse", "HEAD"], cwd=ROOT, text=True
    ).strip()
    base.write_text(
        json.dumps(
            {
                "source_revision": source_revision,
                "source_dirty": False,
                "toolchains": {
                    "platform": "macOS-test-arm64",
                    "python": "3.14-test",
                    "go": "go version test",
                    "swift": "Apple Swift test",
                },
                "package_tools": {
                    "mcpb_cli": "2.1.2",
                    "macos_codesign_identity": "Developer ID Application: Test",
                },
            }
        )
    )
    try:
        subprocess.run(
            [
                sys.executable,
                str(RENDERER),
                "--base-manifest",
                str(base),
                "--output",
                str(output),
            ],
            cwd=ROOT,
            check=True,
        )
        result = json.loads(output.read_text())
        assert result["source_revision"] == source_revision
        assert result["toolchains"]["platform"] == "macOS-test-arm64"
        assert (
            result["package_tools"]["macos_codesign_identity"]
            == "Developer ID Application: Test"
        )
    finally:
        base.unlink(missing_ok=True)
        output.unlink(missing_ok=True)
