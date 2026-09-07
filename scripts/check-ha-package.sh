#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="${1:-$(tr -d '[:space:]' < "${repo_dir}/release/VERSION")}"
archive="${2:-${repo_dir}/dist/ha/fruit-forwarder-ha-${version}.zip}"
sidecar="${archive}.sha256"

[[ -f "${archive}" && -f "${sidecar}" ]] || {
  echo "HA archive or checksum sidecar is missing" >&2
  exit 2
}
expected="$(awk 'NF {print $1; exit}' "${sidecar}")"
actual="$(openssl dgst -sha256 "${archive}" | awk '{print $NF}')"
[[ -n "${expected}" && "${expected}" == "${actual}" ]] || {
  echo "HA archive checksum mismatch" >&2
  exit 1
}

python3 - "${archive}" "${version}" <<'PY'
import json
import sys
import zipfile

archive, version = sys.argv[1:]
root = f"fruit-forwarder-ha-{version}/"
required = {
    root + "custom_components/icloud_reminders_bridge/manifest.json",
    root + "custom_components/icloud_reminders_bridge/__init__.py",
    root + "hacs.json",
    root + "brand/icon.png",
    root + "fruit-forwarder-source.json",
    root + "README.md",
}
with zipfile.ZipFile(archive) as bundle:
    names = set(bundle.namelist())
    if not required <= names:
        raise SystemExit(f"HA archive missing files: {sorted(required - names)}")
    if any("__pycache__/" in name or name.endswith(".pyc") for name in names):
        raise SystemExit("HA archive contains interpreter cache files")
    if any(not name.startswith(root) for name in names):
        raise SystemExit("HA archive contains a path outside its versioned root")
    manifest = json.loads(bundle.read(root + "custom_components/icloud_reminders_bridge/manifest.json"))
    if manifest.get("version") != version:
        raise SystemExit("HA archive manifest version does not match archive version")
    source = json.loads(bundle.read(root + "fruit-forwarder-source.json"))
    if len(source.get("source_revision", "")) != 40 or source.get("source_dirty"):
        raise SystemExit("HA archive source metadata is missing or dirty")
print("validated HA archive, checksum, and manifest")
PY
