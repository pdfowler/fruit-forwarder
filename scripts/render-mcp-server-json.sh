#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 VERSION MCPB_URL MCPB_SHA256" >&2
  exit 2
fi

version="$1"
url="$2"
sha256="$3"
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
template="${repo_dir}/packaging/mcp/server.json.in"
output="${repo_dir}/dist/mcp/server.json"

[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$ ]] || { echo "invalid version" >&2; exit 2; }
[[ "$url" == https://* ]] || { echo "MCPB URL must use HTTPS" >&2; exit 2; }
[[ "$url" == *mcp* ]] || { echo "MCPB URL must identify an MCP artifact" >&2; exit 2; }
[[ "$sha256" =~ ^[a-f0-9]{64}$ ]] || { echo "invalid lowercase SHA-256" >&2; exit 2; }

mkdir -p "$(dirname "$output")"
sed \
  -e "s|@@VERSION@@|$version|g" \
  -e "s|@@MCPB_URL@@|$url|g" \
  -e "s|@@MCPB_SHA256@@|$sha256|g" \
  "$template" > "$output"
python3 -m json.tool "$output" >/dev/null
echo "$output"
