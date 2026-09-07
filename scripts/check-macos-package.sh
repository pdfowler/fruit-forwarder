#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="${1:-$(tr -d '[:space:]' < "${repo_dir}/release/VERSION")}"
archive="${2:-${repo_dir}/dist/macos/fruit-forwarder-macos-${version}.tar.gz}"
sidecar="${archive}.sha256"

[[ -f "${archive}" && -f "${sidecar}" ]] || {
  echo "macOS archive or checksum sidecar is missing" >&2
  exit 2
}
expected="$(awk 'NF {print $1; exit}' "${sidecar}")"
actual="$(openssl dgst -sha256 "${archive}" | awk '{print $NF}')"
[[ -n "${expected}" && "${expected}" == "${actual}" ]] || {
  echo "macOS archive checksum mismatch" >&2
  exit 1
}

python3 - "${archive}" "${version}" <<'PY'
import sys
import tarfile

archive, version = sys.argv[1:]
root = f"fruit-forwarder-macos-{version}/"
required = {
    root + "bin/icloud-reminders-bridge",
    root + "bin/icloud-reminders-eventkit",
    root + "config.example.json",
    root + "README.md",
    root + "CHANGELOG.md",
    root + "LICENSE",
    root + "deployment/com.example.icloud-reminders-bridge.plist.tmpl",
    root + "scripts/install-package-macos.sh",
    root + "scripts/rollback-macos.sh",
    root + "scripts/uninstall-macos.sh",
}
with tarfile.open(archive, "r:gz") as bundle:
    names = {member.name for member in bundle.getmembers()}
    if not required <= names:
        raise SystemExit(f"macOS archive missing files: {sorted(required - names)}")
    if any(name != root.rstrip("/") and not name.startswith(root) for name in names):
        raise SystemExit("macOS archive contains a path outside its versioned root")
    if any(member.issym() or member.islnk() for member in bundle.getmembers()):
        raise SystemExit("macOS archive contains an unexpected link")
print("validated macOS archive, checksum, and install payload")
PY
