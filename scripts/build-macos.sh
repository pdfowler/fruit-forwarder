#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="${1:-$(tr -d '[:space:]' < "${repo_dir}/release/VERSION")}"
output_dir="${2:-${repo_dir}/dist/build/macos-${version}}"
bridge_name="${3:-icloud-reminders-bridge}"
sign_identity="${FRUIT_FORWARDER_SIGN_IDENTITY:--}"

if [[ "${OSTYPE:-}" != darwin* ]]; then
  echo "macOS builds require Darwin" >&2
  exit 2
fi
if [[ ! "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$ ]]; then
  echo "invalid version: ${version}" >&2
  exit 2
fi
if [[ ! "${bridge_name}" =~ ^[A-Za-z0-9._-]+$ ]]; then
  echo "invalid bridge executable name: ${bridge_name}" >&2
  exit 2
fi

# This target writes only the two known build outputs. Refuse a symlinked output
# directory or bin directory so a caller cannot redirect a build into an
# unrelated path. System aliases such as macOS's /var -> /private/var remain
# valid because they are outside the caller-selected output components.
python3 - "${output_dir}" <<'PY'
from pathlib import Path
import sys

candidate = Path(sys.argv[1]).expanduser()
if not candidate.is_absolute():
    candidate = Path.cwd() / candidate
for component in (candidate, candidate / "bin"):
    if component.is_symlink():
        raise SystemExit(f"build output contains a symlinked component: {component}")
PY

mkdir -p "${output_dir}/bin"
go build -trimpath -ldflags "-X main.version=${version}" \
  -o "${output_dir}/bin/${bridge_name}" \
  "${repo_dir}/cmd/icloud-reminders-bridge"

codesign_options=(--force --sign "${sign_identity}" --options runtime)
if [[ "${sign_identity}" != "-" ]]; then
  codesign_options+=(--timestamp)
fi
codesign "${codesign_options[@]}" \
  --identifier com.pdfowler.fruitforwarder \
  "${output_dir}/bin/${bridge_name}"
codesign --verify --strict "${output_dir}/bin/${bridge_name}"

swiftc -O -parse-as-library "${repo_dir}/eventkit-helper/main.swift" \
  -Xlinker -sectcreate -Xlinker __TEXT -Xlinker __info_plist \
  -Xlinker "${repo_dir}/deployment/eventkit-helper-Info.plist" \
  -o "${output_dir}/bin/icloud-reminders-eventkit"
codesign "${codesign_options[@]}" \
  --identifier com.pdfowler.fruitforwarder.eventkit \
  --entitlements "${repo_dir}/deployment/reminders.entitlements" \
  "${output_dir}/bin/icloud-reminders-eventkit"
codesign --verify --strict "${output_dir}/bin/icloud-reminders-eventkit"

FRUIT_FORWARDER_EVENTKIT_BINARY="${output_dir}/bin/icloud-reminders-eventkit" \
  python3 "${repo_dir}/scripts/test-native-calendar.py"
"${output_dir}/bin/${bridge_name}" version | grep -Fx "${version}" >/dev/null
chmod 0755 "${output_dir}/bin/${bridge_name}" "${output_dir}/bin/icloud-reminders-eventkit"

echo "macOS build: ${output_dir}"
