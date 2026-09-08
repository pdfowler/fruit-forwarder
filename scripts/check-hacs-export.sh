#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/fruit-forwarder-hacs.XXXXXX")"
trap 'rm -rf "$tmp_dir"' EXIT

first="${tmp_dir}/fruit-forwarder-hacs-first"
second="${tmp_dir}/fruit-forwarder-hacs-second"
"${repo_dir}/scripts/export-hacs.sh" "$first" >/dev/null
"${repo_dir}/scripts/export-hacs.sh" "$second" >/dev/null
diff -ru "$first" "$second"
"${repo_dir}/scripts/check-hacs-target.sh" "$first" >/dev/null
echo "validated deterministic HACS export"
