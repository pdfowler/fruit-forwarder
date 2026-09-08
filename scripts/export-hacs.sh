#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out_dir="${1:-${repo_dir}/dist/ha-fruit-forwarder}"
component="${repo_dir}/homeassistant/custom_components/icloud_reminders_bridge"

case "${out_dir}" in
  ""|"/"|"."|"..")
    echo "refusing an unsafe HACS export path: ${out_dir}" >&2
    exit 2
    ;;
esac
rm -rf "$out_dir"
mkdir -p "$out_dir/custom_components"
cp -R "$component" "$out_dir/custom_components/"
cp "$repo_dir/hacs.json" "$out_dir/hacs.json"
cp "$repo_dir/README-HACS.md" "$out_dir/README.md"
cp "$repo_dir/LICENSE" "$out_dir/LICENSE"
if [[ -d "$repo_dir/packaging/hacs/.github" ]]; then
  cp -R "$repo_dir/packaging/hacs/.github" "$out_dir/.github"
fi
if [[ -d "$repo_dir/brand" ]]; then
  cp -R "$repo_dir/brand" "$out_dir/brand"
fi
# Never ship interpreter caches created by local HA tests.
find "$out_dir" -type d -name '__pycache__' -prune -exec rm -rf {} +
find "$out_dir" -type f -name '*.pyc' -delete

source_revision="$(git -C "${repo_dir}" rev-parse HEAD)"
source_dirty=false
if [[ -n "$(git -C "${repo_dir}" status --porcelain)" ]]; then
  source_dirty=true
fi
python3 - "${out_dir}" "${source_revision}" "${source_dirty}" <<'PY'
import hashlib
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])


def tree_digest() -> str:
    digest = hashlib.sha256()
    for path in sorted(root.rglob("*")):
        if not path.is_file() or path.name == "fruit-forwarder-source.json":
            continue
        relative = path.relative_to(root).as_posix().encode()
        digest.update(relative)
        digest.update(b"\0")
        digest.update(path.read_bytes())
        digest.update(b"\0")
    return digest.hexdigest()


metadata = {
    "product": "Fruit Forwarder",
    "source_repository": "https://github.com/pdfowler/fruit-forwarder",
    "source_revision": sys.argv[2],
    "source_dirty": sys.argv[3] == "true",
}
metadata["export_tree_sha256"] = tree_digest()
(root / "fruit-forwarder-source.json").write_text(json.dumps(metadata, indent=2) + "\n")
PY

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
if not (root / "custom_components" / "icloud_reminders_bridge" / "brand" / "icon.png").is_file():
    raise SystemExit("component-local brand/icon.png is required for current HA")
source = json.loads((root / "fruit-forwarder-source.json").read_text())
if not source.get("source_revision") or len(source["source_revision"]) != 40:
    raise SystemExit("fruit-forwarder-source.json must contain a full source revision")
for workflow in ("validate.yml", "hassfest.yml"):
    if not (root / ".github" / "workflows" / workflow).is_file():
        raise SystemExit(f"missing HACS release workflow: {workflow}")
print(f"validated HACS export: {root}")
PY
