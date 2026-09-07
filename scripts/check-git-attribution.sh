#!/usr/bin/env bash
set -euo pipefail

expected_email="${FRUIT_FORWARDER_COMMIT_EMAIL:-pfowler@icloud.com}"
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ ! "${expected_email}" =~ ^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$ ]]; then
  echo "invalid expected commit email: ${expected_email}" >&2
  exit 2
fi

violations="$(git -C "${repo_dir}" log --all --format='%H%x09%ae%x09%ce' | awk -F '\t' -v expected="${expected_email}" '$2 != expected || $3 != expected { print }')"
if [[ -n "${violations}" ]]; then
  echo "commits with unexpected author or committer email (expected ${expected_email}):" >&2
  while IFS=$'\t' read -r revision author committer; do
    printf '  %s author=%s committer=%s\n' "${revision}" "${author}" "${committer}" >&2
  done <<< "${violations}"
  exit 1
fi

echo "validated Git author and committer email across all refs: ${expected_email}"
