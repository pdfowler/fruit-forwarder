#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="${1:-$(tr -d '[:space:]' < "${repo_dir}/release/VERSION")}"
output_dir="${2:-${repo_dir}/dist/macos}"
archive="${output_dir}/fruit-forwarder-macos-${version}.tar.gz"

if [[ "${OSTYPE:-}" != darwin* ]]; then
  echo "macOS packaging requires Darwin" >&2
  exit 2
fi
if [[ ! "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$ ]]; then
  echo "invalid version: ${version}" >&2
  exit 2
fi

staging="$(mktemp -d "${TMPDIR:-/tmp}/fruit-forwarder-macos.XXXXXX")"
trap 'rm -rf "${staging}"' EXIT
mkdir -p "${staging}/fruit-forwarder-macos-${version}/bin" "${staging}/fruit-forwarder-macos-${version}/deployment" "${staging}/fruit-forwarder-macos-${version}/scripts"

"${repo_dir}/scripts/build-macos.sh" \
  "${version}" "${staging}/fruit-forwarder-macos-${version}"

cp "${repo_dir}/deployment/com.pdfowler.fruitforwarder.plist.tmpl" \
  "${staging}/fruit-forwarder-macos-${version}/deployment/"
cp "${repo_dir}/config.example.json" "${staging}/fruit-forwarder-macos-${version}/"
cp "${repo_dir}/LICENSE" "${staging}/fruit-forwarder-macos-${version}/"
cp "${repo_dir}/README.md" "${staging}/fruit-forwarder-macos-${version}/"
cp "${repo_dir}/CHANGELOG.md" "${staging}/fruit-forwarder-macos-${version}/"
cp "${repo_dir}/SUPPORT.md" "${staging}/fruit-forwarder-macos-${version}/"
cp "${repo_dir}/docs/setup.md" "${staging}/fruit-forwarder-macos-${version}/"
cp "${repo_dir}/scripts/install-package-macos.sh" "${staging}/fruit-forwarder-macos-${version}/scripts/"
cp "${repo_dir}/scripts/rollback-macos.sh" "${staging}/fruit-forwarder-macos-${version}/scripts/"
cp "${repo_dir}/scripts/uninstall-macos.sh" "${staging}/fruit-forwarder-macos-${version}/scripts/"
cp "${repo_dir}/scripts/render-launchagent.py" "${staging}/fruit-forwarder-macos-${version}/scripts/"
chmod 0755 "${staging}/fruit-forwarder-macos-${version}/scripts/"*.sh

"${staging}/fruit-forwarder-macos-${version}/bin/icloud-reminders-bridge" version | grep -Fx "${version}" >/dev/null
mkdir -p "${output_dir}"
rm -f "${archive}" "${archive}.sha256"
python3 "${repo_dir}/scripts/package-tar.py" \
  "${staging}/fruit-forwarder-macos-${version}" "${archive}"
checksum="$(openssl dgst -sha256 "${archive}" | awk '{print $NF}')"
printf '%s  %s\n' "${checksum}" "$(basename "${archive}")" > "${archive}.sha256"
"${repo_dir}/scripts/check-macos-package.sh" "${version}" "${archive}"
echo "macOS package: ${archive}"
echo "SHA-256: ${checksum}"
