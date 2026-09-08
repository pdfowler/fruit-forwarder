#!/usr/bin/env bash
set -euo pipefail

require_clean=false
if [[ "${1:-}" == "--require-clean" ]]; then
  require_clean=true
  shift
fi
target_dir="${1:-}"
if [[ -z "${target_dir}" || ! -d "${target_dir}" ]]; then
  echo "usage: scripts/check-hacs-target.sh [--require-clean] <checked-out HACS repository>" >&2
  exit 2
fi

python3 - "${target_dir}" "${require_clean}" <<'PY'
import hashlib
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
require_clean = sys.argv[2] == "true"
if root.is_symlink():
    raise SystemExit("HACS target must not be a symlink")
metadata_path = root / "fruit-forwarder-source.json"
if not metadata_path.is_file() or metadata_path.is_symlink():
    raise SystemExit("HACS target has no trusted Fruit Forwarder export metadata")

metadata = json.loads(metadata_path.read_text())
if metadata.get("product") != "Fruit Forwarder":
    raise SystemExit("HACS target metadata belongs to another product")
if metadata.get("source_repository") != "https://github.com/pdfowler/fruit-forwarder":
    raise SystemExit("HACS target metadata points at an unexpected source repository")
if require_clean and metadata.get("source_dirty"):
    raise SystemExit("HACS target was generated from a dirty source tree")
component_brand = root / "custom_components" / "icloud_reminders_bridge" / "brand" / "icon.png"
if not component_brand.is_file() or component_brand.is_symlink():
    raise SystemExit("HACS target has no component-local brand icon")
expected = metadata.get("export_tree_sha256")
if not isinstance(expected, str) or len(expected) != 64:
    raise SystemExit("HACS target metadata has no export tree digest")

digest = hashlib.sha256()
for path in sorted(root.rglob("*")):
    if ".git" in path.relative_to(root).parts:
        continue
    if path.is_symlink():
        raise SystemExit(f"HACS target contains an unexpected symlink: {path.relative_to(root)}")
    if not path.is_file() or path.name == metadata_path.name:
        continue
    relative = path.relative_to(root).as_posix().encode()
    digest.update(relative)
    digest.update(b"\0")
    digest.update(path.read_bytes())
    digest.update(b"\0")

actual = digest.hexdigest()
if actual != expected:
    raise SystemExit(
        "HACS target differs from its recorded generated export; "
        "review or restore it before publishing"
    )
print(f"validated generated HACS target: {root}")
PY
