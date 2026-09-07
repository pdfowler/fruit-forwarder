#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/fruit-forwarder-hacs.XXXXXX")"
trap 'rm -rf "$tmp_dir"' EXIT

first="${tmp_dir}/first"
second="${tmp_dir}/second"
"${repo_dir}/scripts/export-hacs.sh" "$first" >/dev/null
"${repo_dir}/scripts/export-hacs.sh" "$second" >/dev/null
diff -ru "$first" "$second"
echo "validated deterministic HACS export"
