#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="${1:-$(tr -d '[:space:]' < "${repo_dir}/release/VERSION")}"
output_dir="${2:-${repo_dir}/dist/mcp}"
artifact="${output_dir}/fruit-forwarder-mcp-${version}.mcpb"
sidecar="${artifact}.sha256"
metadata="${output_dir}/server.json"

[[ -f "${artifact}" && -f "${sidecar}" && -f "${metadata}" ]] || {
  echo "MCP candidate files are incomplete" >&2
  exit 2
}
actual="$(openssl dgst -sha256 "${artifact}" | awk '{print $NF}')"
expected="$(awk 'NF {print $1; exit}' "${sidecar}")"
[[ "${actual}" == "${expected}" ]] || {
  echo "MCPB checksum sidecar does not match the artifact" >&2
  exit 1
}

python3 - "${artifact}" "${metadata}" "${version}" "${actual}" <<'PY'
import json
import sys
import zipfile

artifact, metadata, version, checksum = sys.argv[1:]
with zipfile.ZipFile(artifact) as bundle:
    manifest = json.loads(bundle.read("manifest.json"))
    if manifest.get("version") != version:
        raise SystemExit("MCPB manifest version does not match the candidate")
    if manifest.get("server", {}).get("type") != "binary":
        raise SystemExit("MCPB manifest is not a binary server")
registry = json.load(open(metadata))
if registry.get("version") != version:
    raise SystemExit("Registry metadata version does not match the candidate")
if registry.get("name") != "io.github.pdfowler/fruit-forwarder":
    raise SystemExit("Registry metadata namespace is not the Fruit Forwarder namespace")
packages = registry.get("packages")
if not isinstance(packages, list) or len(packages) != 1:
    raise SystemExit("Registry metadata must contain exactly one package")
package = packages[0]
if package.get("fileSha256") != checksum:
    raise SystemExit("Registry metadata hash does not match the candidate")
if package.get("version") != version or package.get("registryType") != "mcpb":
    raise SystemExit("Registry metadata package is inconsistent")
identifier = package.get("identifier", "")
expected_identifier = (
    f"https://github.com/pdfowler/fruit-forwarder/releases/download/v{version}/"
    f"fruit-forwarder-mcp-{version}.mcpb"
)
if identifier != expected_identifier:
    raise SystemExit("Registry metadata MCPB URL does not point to the versioned GitHub release")
if "mcp" not in identifier.lower():
    raise SystemExit("Registry metadata MCPB URL must contain 'mcp'")
if package.get("transport", {}).get("type") != "stdio":
    raise SystemExit("Registry metadata MCPB transport must be stdio")
print("validated MCPB, checksum sidecar, and Registry metadata")
PY
