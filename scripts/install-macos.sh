#!/usr/bin/env bash
set -euo pipefail

SERVICE_LABEL="com.example.icloud-reminders-bridge"
SERVICE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_DIR="${HOME}/Library/Application Support/icloud-reminders-bridge"
BIN_DIR="${INSTALL_DIR}/bin"
BRIDGE_BIN="${BIN_DIR}/icloud-reminders-bridge"
EVENTKIT_BIN="${BIN_DIR}/icloud-reminders-eventkit"
CONFIG_DIR="${HOME}/.config/icloud-reminders-bridge"
CONFIG_PATH="${CONFIG_DIR}/config.json"
LOG_DIR="${HOME}/Library/Logs/icloud-reminders-bridge"
PLIST_PATH="${HOME}/Library/LaunchAgents/${SERVICE_LABEL}.plist"

mkdir -p "${SERVICE_DIR}/build" "${BIN_DIR}" "${CONFIG_DIR}" "${LOG_DIR}" "$(dirname "${PLIST_PATH}")"
chmod 0700 "${INSTALL_DIR}" "${BIN_DIR}"

(
  cd "${SERVICE_DIR}"
  go build -trimpath -o "build/icloud-reminders-bridge" ./cmd/icloud-reminders-bridge
  swiftc -O -parse-as-library eventkit-helper/main.swift \
    -Xlinker -sectcreate -Xlinker __TEXT -Xlinker __info_plist \
    -Xlinker "${SERVICE_DIR}/deployment/eventkit-helper-Info.plist" \
    -o "build/icloud-reminders-eventkit"
)
install -m 0755 "${SERVICE_DIR}/build/icloud-reminders-bridge" "${BRIDGE_BIN}"
install -m 0755 "${SERVICE_DIR}/build/icloud-reminders-eventkit" "${EVENTKIT_BIN}"
codesign --force --sign - --options runtime \
  --identifier "${SERVICE_LABEL}.eventkit" \
  --entitlements "${SERVICE_DIR}/deployment/reminders.entitlements" \
  "${EVENTKIT_BIN}"

if [[ ! -f "${CONFIG_PATH}" ]]; then
  install -m 0600 "${SERVICE_DIR}/config.example.json" "${CONFIG_PATH}"
  echo "Created ${CONFIG_PATH}; add exact list IDs from:" >&2
  echo "  '${BRIDGE_BIN}' discover" >&2
  if [[ "${1:-}" != "--install-only" ]]; then
    exit 2
  fi
fi
chmod 0600 "${CONFIG_PATH}"
if [[ "${1:-}" == "--install-only" ]]; then
  echo "Installed binaries; LaunchAgent activation was not requested."
  exit 0
fi
"${BRIDGE_BIN}" check-config --config "${CONFIG_PATH}"

sed \
  -e "s|@@UID@@|${UID}|g" \
  -e "s|@@HOME@@|${HOME}|g" \
  -e "s|@@USER@@|${USER}|g" \
  -e "s|@@TMPDIR@@|${TMPDIR}|g" \
  -e "s|@@BINARY@@|${BRIDGE_BIN}|g" \
  -e "s|@@CONFIG@@|${CONFIG_PATH}|g" \
  -e "s|@@LOG_DIR@@|${LOG_DIR}|g" \
  "${SERVICE_DIR}/deployment/${SERVICE_LABEL}.plist.tmpl" > "${PLIST_PATH}.new"
plutil -lint "${PLIST_PATH}.new" >/dev/null
chmod 0600 "${PLIST_PATH}.new"
mv "${PLIST_PATH}.new" "${PLIST_PATH}"

launchctl bootout "gui/${UID}/${SERVICE_LABEL}" 2>/dev/null || true
# launchd removes an unloaded service asynchronously. Wait for that removal so
# an immediate reinstall does not fail with a transient bootstrap error 5.
for _ in {1..20}; do
  if ! launchctl print "gui/${UID}/${SERVICE_LABEL}" >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done
launchctl bootstrap "gui/${UID}" "${PLIST_PATH}"
launchctl kickstart -k "gui/${UID}/${SERVICE_LABEL}"

echo "Installed and started ${SERVICE_LABEL}."
echo "Status: launchctl print gui/${UID}/${SERVICE_LABEL}"
echo "Logs:   tail -f '${LOG_DIR}/icloud-reminders-bridge.log'"
