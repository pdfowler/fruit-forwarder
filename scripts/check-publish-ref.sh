#!/usr/bin/env bash
set -euo pipefail

tag="${FRUIT_FORWARDER_RELEASE_TAG:-}"
[[ "${tag}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$ ]] || {
  echo "publication requires a semantic version tag such as v0.1.0" >&2
  exit 2
}
[[ -z "$(git status --porcelain)" ]] || {
  echo "publication requires a clean checkout" >&2
  exit 2
}
git rev-parse --verify --quiet "refs/tags/${tag}" >/dev/null || {
  echo "tag does not exist locally: ${tag}" >&2
  exit 2
}
[[ "$(git rev-parse HEAD)" == "$(git rev-list -n 1 "refs/tags/${tag}")" ]] || {
  echo "HEAD must be the exact commit referenced by ${tag}" >&2
  exit 2
}
scripts/check-git-attribution.sh
