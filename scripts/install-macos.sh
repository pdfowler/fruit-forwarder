#!/usr/bin/env bash
set -euo pipefail

SERVICE_LABEL="com.example.icloud-reminders-bridge"
SERVICE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_DIR="${HOME}/Library/Application Support/icloud-reminders-bridge"
BIN_DIR="${INSTALL_DIR}/bin"
ROLLBACK_DIR="${INSTALL_DIR}/rollback"
BRIDGE_BIN="${BIN_DIR}/icloud-reminders-bridge"
EVENTKIT_BIN="${BIN_DIR}/icloud-reminders-eventkit"
CONFIG_DIR="${HOME}/.config/icloud-reminders-bridge"
CONFIG_PATH="${CONFIG_DIR}/config.json"
LOG_DIR="${HOME}/Library/Logs/icloud-reminders-bridge"
PLIST_PATH="${HOME}/Library/LaunchAgents/${SERVICE_LABEL}.plist"

mkdir -p "${SERVICE_DIR}/build" "${BIN_DIR}" "${ROLLBACK_DIR}" "${CONFIG_DIR}" "${LOG_DIR}" "$(dirname "${PLIST_PATH}")"
chmod 0700 "${INSTALL_DIR}" "${BIN_DIR}"
chmod 0700 "${ROLLBACK_DIR}"

VERSION="$(tr -d '[:space:]' < "${SERVICE_DIR}/release/VERSION")"
if [[ ! "${VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$ ]]; then
  echo "invalid release version: ${VERSION}" >&2
  exit 2
fi

(
  cd "${SERVICE_DIR}"
  go build -trimpath -ldflags "-X main.version=${VERSION}" -o "build/icloud-reminders-bridge" ./cmd/icloud-reminders-bridge
  swiftc -O -parse-as-library eventkit-helper/main.swift \
    -Xlinker -sectcreate -Xlinker __TEXT -Xlinker __info_plist \
    -Xlinker "${SERVICE_DIR}/deployment/eventkit-helper-Info.plist" \
    -o "build/icloud-reminders-eventkit"
)
STAGE_BRIDGE="${BIN_DIR}/.icloud-reminders-bridge.new"
STAGE_EVENTKIT="${BIN_DIR}/.icloud-reminders-eventkit.new"
install -m 0755 "${SERVICE_DIR}/build/icloud-reminders-bridge" "${STAGE_BRIDGE}"
install -m 0755 "${SERVICE_DIR}/build/icloud-reminders-eventkit" "${STAGE_EVENTKIT}"
codesign --force --sign - --options runtime \
  --identifier "${SERVICE_LABEL}.eventkit" \
  --entitlements "${SERVICE_DIR}/deployment/reminders.entitlements" \
  "${STAGE_EVENTKIT}"
codesign --verify --strict "${STAGE_EVENTKIT}"

# Keep the previous pair recoverable and switch each executable with an atomic
# same-directory rename only after both staged files have passed validation.
INSTALL_STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
BACKUP_DIR="${ROLLBACK_DIR}/${VERSION}-${INSTALL_STAMP}"
mkdir -p "${BACKUP_DIR}"
chmod 0700 "${BACKUP_DIR}"
if [[ -f "${BRIDGE_BIN}" ]]; then
  cp -p "${BRIDGE_BIN}" "${BACKUP_DIR}/icloud-reminders-bridge"
fi
if [[ -f "${EVENTKIT_BIN}" ]]; then
  cp -p "${EVENTKIT_BIN}" "${BACKUP_DIR}/icloud-reminders-eventkit"
fi
mv "${STAGE_BRIDGE}" "${BRIDGE_BIN}"
if ! mv "${STAGE_EVENTKIT}" "${EVENTKIT_BIN}"; then
  # Do not leave launchd with a bridge/helper pair from different releases.
  if [[ -f "${BACKUP_DIR}/icloud-reminders-bridge" ]]; then
    cp -p "${BACKUP_DIR}/icloud-reminders-bridge" "${BRIDGE_BIN}"
  else
    rm -f "${BRIDGE_BIN}"
  fi
  echo "failed to switch the EventKit helper; restored the previous executable pair" >&2
  exit 1
fi

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
  echo "Previous binaries (if any): ${BACKUP_DIR}"
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
echo "Previous binaries (if any): ${BACKUP_DIR}"
echo "Status: launchctl print gui/${UID}/${SERVICE_LABEL}"
echo "Logs:   tail -f '${LOG_DIR}/icloud-reminders-bridge.log'"
