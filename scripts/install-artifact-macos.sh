#!/usr/bin/env bash
set -euo pipefail

if [[ "${OSTYPE:-}" != darwin* ]]; then
  echo "the Fruit Forwarder package requires macOS" >&2
  exit 2
fi

if [[ $# -lt 1 || $# -gt 3 ]]; then
  echo "usage: $0 /path/to/fruit-forwarder-macos-VERSION.tar.gz [--install-only] [--migrate-home-ctrl]" >&2
  exit 2
fi

archive="$1"
shift
for option in "$@"; do
  case "${option}" in
    --install-only|--migrate-home-ctrl) ;;
    *) echo "unknown option: ${option}" >&2; exit 2 ;;
  esac
done

[[ -f "${archive}" && ! -L "${archive}" ]] || {
  echo "package archive must be a regular, non-symlink file: ${archive}" >&2
  exit 2
}

source_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="$(basename "${archive}" .tar.gz)"
version="${version#fruit-forwarder-macos-}"
[[ "${version}" != "${archive}" && "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$ ]] || {
  echo "archive name must be fruit-forwarder-macos-VERSION.tar.gz" >&2
  exit 2
}

"${source_dir}/scripts/check-macos-package.sh" "${version}" "${archive}"

staging="$(mktemp -d "${TMPDIR:-/tmp}/fruit-forwarder-install.XXXXXX")"
trap 'rm -rf "${staging}"' EXIT

python3 - "${archive}" "${staging}" "${version}" <<'PY'
from __future__ import annotations

import os
import pathlib
import sys
import tarfile

archive, destination, version = sys.argv[1:]
root_name = f"fruit-forwarder-macos-{version}"
root = pathlib.Path(destination).resolve()
seen: set[str] = set()

with tarfile.open(archive, "r:gz") as bundle:
    members = bundle.getmembers()
    for member in members:
        name = pathlib.PurePosixPath(member.name)
        if member.name in seen:
            raise SystemExit(f"archive contains a duplicate member: {member.name}")
        seen.add(member.name)
        if name.is_absolute() or ".." in name.parts:
            raise SystemExit(f"archive contains an unsafe path: {member.name}")
        if not name.parts or name.parts[0] != root_name:
            raise SystemExit(f"archive member is outside its versioned root: {member.name}")
        if member.issym() or member.islnk() or not (member.isdir() or member.isfile()):
            raise SystemExit(f"archive contains an unsupported member: {member.name}")

    for member in members:
        target = (root / pathlib.Path(*pathlib.PurePosixPath(member.name).parts)).resolve()
        try:
            target.relative_to(root)
        except ValueError as exc:
            raise SystemExit(f"archive member escapes extraction root: {member.name}") from exc
        if member.isdir():
            target.mkdir(parents=True, exist_ok=True)
            continue
        target.parent.mkdir(parents=True, exist_ok=True)
        source = bundle.extractfile(member)
        if source is None:
            raise SystemExit(f"unable to read archive member: {member.name}")
        with target.open("xb") as output:
            while chunk := source.read(1024 * 1024):
                output.write(chunk)
        os.chmod(target, member.mode & 0o777)

package_dir = root / root_name
installer = package_dir / "scripts" / "install-package-macos.sh"
if not installer.is_file() or installer.is_symlink():
    raise SystemExit("package does not contain a regular install-package-macos.sh")
PY

exec "${staging}/fruit-forwarder-macos-${version}/scripts/install-package-macos.sh" "$@"
