#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="${1:-$(tr -d '[:space:]' < "${repo_dir}/release/VERSION")}"
export_dir="${repo_dir}/dist/ha-fruit-forwarder"
output_dir="${2:-${repo_dir}/dist/ha}"
archive="${output_dir}/fruit-forwarder-ha-${version}.zip"

"${repo_dir}/scripts/export-hacs.sh" "${export_dir}"
mkdir -p "${output_dir}"
python3 "${repo_dir}/scripts/package-hacs.py" "${export_dir}" "${archive}" "${version}"
checksum="$(openssl dgst -sha256 "${archive}" | awk '{print $NF}')"
printf '%s  %s\n' "${checksum}" "$(basename "${archive}")" > "${archive}.sha256"
"${repo_dir}/scripts/check-ha-package.sh" "${version}" "${archive}"
echo "HA package: ${archive}"
echo "SHA-256: ${checksum}"
