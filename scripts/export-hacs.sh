#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out_dir="${1:-${repo_dir}/dist/ha-fruit-forwarder}"
component="${repo_dir}/homeassistant/custom_components/icloud_reminders_bridge"

rm -rf "$out_dir"
mkdir -p "$out_dir/custom_components"
cp -R "$component" "$out_dir/custom_components/"
cp "$repo_dir/hacs.json" "$out_dir/hacs.json"
cp "$repo_dir/README-HACS.md" "$out_dir/README.md"
cp "$repo_dir/LICENSE" "$out_dir/LICENSE"
if [[ -d "$repo_dir/brand" ]]; then
  cp -R "$repo_dir/brand" "$out_dir/brand"
fi

python3 - "$out_dir" <<'PY'
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
manifest_path = root / "custom_components" / "icloud_reminders_bridge" / "manifest.json"
manifest = json.loads(manifest_path.read_text())
required = {"domain", "documentation", "issue_tracker", "codeowners", "name", "version"}
missing = required - manifest.keys()
if missing:
    raise SystemExit(f"manifest missing required HACS keys: {sorted(missing)}")
if manifest["domain"] != "icloud_reminders_bridge":
    raise SystemExit("unexpected integration domain")
if not (root / "brand" / "icon.png").is_file():
    raise SystemExit("brand/icon.png is required for HACS export")
print(f"validated HACS export: {root}")
PY
