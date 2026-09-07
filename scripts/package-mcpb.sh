#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ $# -ge 1 ]]; then
  version="$1"
else
  version="$(tr -d '[:space:]' < "${repo_dir}/release/VERSION")"
fi
output_dir="${2:-${repo_dir}/dist/mcp}"
output="${output_dir}/fruit-forwarder-mcp-${version}.mcpb"
mcpb_version="${MCPB_VERSION:-2.1.2}"

if [[ "${OSTYPE:-}" != darwin* ]]; then
  echo "MCPB packaging currently requires macOS for EventKit" >&2
  exit 2
fi
if [[ ! "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$ ]]; then
  echo "invalid version: ${version}" >&2
  exit 2
fi

if [[ -n "${MCPB_BIN:-}" ]]; then
  mcpb=("${MCPB_BIN}")
elif command -v mcpb >/dev/null 2>&1; then
  mcpb=(mcpb)
else
  npx_bin="$(command -v npx || true)"
  if [[ -z "${npx_bin}" ]]; then
    for candidate in /opt/homebrew/opt/node/bin/npx /usr/local/opt/node/bin/npx \
      /opt/homebrew/Cellar/node/*/bin/npx /usr/local/Cellar/node/*/bin/npx; do
      if [[ -x "${candidate}" ]]; then
        npx_bin="${candidate}"
        PATH="$(dirname "${candidate}"):${PATH}"
        break
      fi
    done
  fi
  if [[ -z "${npx_bin}" ]]; then
    echo "install @anthropic-ai/mcpb or set MCPB_BIN" >&2
    exit 2
  fi
  mcpb=("${npx_bin}" --yes "@anthropic-ai/mcpb@${mcpb_version}")
fi

staging="$(mktemp -d "${TMPDIR:-/tmp}/fruit-forwarder-mcpb.XXXXXX")"
trap 'rm -rf "${staging}"' EXIT
mkdir -p "${staging}/server" "${staging}/brand" "${output_dir}"

go build -trimpath -ldflags "-X main.version=${version}" \
  -o "${staging}/server/fruit-forwarder-mcp" \
  "${repo_dir}/cmd/icloud-reminders-bridge"
swiftc -O -parse-as-library "${repo_dir}/eventkit-helper/main.swift" \
  -Xlinker -sectcreate -Xlinker __TEXT -Xlinker __info_plist \
  -Xlinker "${repo_dir}/deployment/eventkit-helper-Info.plist" \
  -o "${staging}/server/icloud-reminders-eventkit"
codesign --force --sign - --options runtime \
  --identifier com.example.icloud-reminders-bridge.eventkit \
  --entitlements "${repo_dir}/deployment/reminders.entitlements" \
  "${staging}/server/icloud-reminders-eventkit"
FRUIT_FORWARDER_EVENTKIT_BINARY="${staging}/server/icloud-reminders-eventkit" \
  python3 "${repo_dir}/scripts/test-native-calendar.py"

sed -e "s|@@VERSION@@|${version}|g" \
  "${repo_dir}/packaging/mcp/manifest.json.in" > "${staging}/manifest.json"
cp "${repo_dir}/brand/icon.png" "${staging}/brand/icon.png"
cp "${repo_dir}/LICENSE" "${staging}/LICENSE"
cp "${repo_dir}/packaging/mcp/config.example.json" "${staging}/config.example.json"
chmod 0755 "${staging}/server/fruit-forwarder-mcp" "${staging}/server/icloud-reminders-eventkit"

"${mcpb[@]}" validate "${staging}/manifest.json"
"${mcpb[@]}" pack "${staging}" "${output}"
python3 "${repo_dir}/scripts/normalize-zip.py" "${output}"
if [[ -n "${MCPB_CERT:-}" || -n "${MCPB_KEY:-}" ]]; then
  if [[ -z "${MCPB_CERT:-}" || -z "${MCPB_KEY:-}" ]]; then
    echo "MCPB_CERT and MCPB_KEY must be provided together" >&2
    exit 2
  fi
  "${mcpb[@]}" sign "${output}" --cert "${MCPB_CERT}" --key "${MCPB_KEY}"
  "${mcpb[@]}" verify "${output}"
fi
checksum="$(openssl dgst -sha256 "${output}" | awk '{print $NF}')"
printf '%s  %s\n' "${checksum}" "$(basename "${output}")" > "${output}.sha256"
"${mcpb[@]}" info "${output}"
echo "MCPB: ${output}"
echo "SHA-256: ${checksum}"
