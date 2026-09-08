#!/usr/bin/env bash
set -euo pipefail

SERVICE_LABEL="com.pdfowler.fruitforwarder"
LEGACY_SERVICE_LABELS=(
  "net.pdfowler.icloud-reminders-bridge"
  "com.example.icloud-reminders-bridge"
)
SERVICE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_DIR="${HOME}/Library/Application Support/icloud-reminders-bridge"
BIN_DIR="${INSTALL_DIR}/bin"
ROLLBACK_DIR="${INSTALL_DIR}/rollback"
BRIDGE_BIN="${BIN_DIR}/icloud-reminders-bridge"
EVENTKIT_BIN="${BIN_DIR}/icloud-reminders-eventkit"
CONFIG_DIR="${HOME}/.config/icloud-reminders-bridge"
CONFIG_PATH="${CONFIG_DIR}/config.json"
LEGACY_CONFIG_PATH="${HOME}/.config/home-ctrl/icloud-reminders-bridge/config.json"
LOG_DIR="${HOME}/Library/Logs/icloud-reminders-bridge"
PLIST_PATH="${HOME}/Library/LaunchAgents/${SERVICE_LABEL}.plist"

INSTALL_ONLY=false
MIGRATE_HOME_CTRL=false
for argument in "$@"; do
  case "${argument}" in
    --install-only) INSTALL_ONLY=true ;;
    --migrate-home-ctrl) MIGRATE_HOME_CTRL=true ;;
    *) echo "unknown option: ${argument}" >&2; exit 2 ;;
  esac
done

mkdir -p "${SERVICE_DIR}/build" "${BIN_DIR}" "${ROLLBACK_DIR}" "${CONFIG_DIR}" "${LOG_DIR}" "$(dirname "${PLIST_PATH}")"
chmod 0700 "${INSTALL_DIR}" "${BIN_DIR}"
chmod 0700 "${ROLLBACK_DIR}"

VERSION="$(tr -d '[:space:]' < "${SERVICE_DIR}/release/VERSION")"
if [[ ! "${VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$ ]]; then
  echo "invalid release version: ${VERSION}" >&2
  exit 2
fi

BUILD_OUTPUT="${SERVICE_DIR}/build/macos-${VERSION}"
"${SERVICE_DIR}/scripts/build-macos.sh" "${VERSION}" "${BUILD_OUTPUT}"
STAGE_BRIDGE="${BIN_DIR}/.icloud-reminders-bridge.new"
STAGE_EVENTKIT="${BIN_DIR}/.icloud-reminders-eventkit.new"
install -m 0755 "${BUILD_OUTPUT}/bin/icloud-reminders-bridge" "${STAGE_BRIDGE}"
install -m 0755 "${BUILD_OUTPUT}/bin/icloud-reminders-eventkit" "${STAGE_EVENTKIT}"
codesign --verify --strict "${STAGE_BRIDGE}"
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

if [[ -L "${CONFIG_DIR}" || -L "${CONFIG_PATH}" ]]; then
  echo "refusing a symlinked configuration path" >&2
  exit 2
fi
if [[ ! -f "${CONFIG_PATH}" ]]; then
  if [[ "${MIGRATE_HOME_CTRL}" == true && -f "${LEGACY_CONFIG_PATH}" ]]; then
    [[ ! -L "${LEGACY_CONFIG_PATH}" ]] || { echo "refusing a symlinked legacy configuration" >&2; exit 2; }
    install -m 0600 "${LEGACY_CONFIG_PATH}" "${CONFIG_PATH}"
    echo "Migrated the existing home-ctrl configuration; its explicit Keychain and state paths were preserved." >&2
  else
    install -m 0600 "${SERVICE_DIR}/config.example.json" "${CONFIG_PATH}"
  fi
  echo "Created ${CONFIG_PATH}; add exact list IDs from:" >&2
  echo "  '${BRIDGE_BIN}' discover" >&2
  if [[ "${INSTALL_ONLY}" != true ]]; then
    exit 2
  fi
fi
chmod 0600 "${CONFIG_PATH}"
if [[ "${INSTALL_ONLY}" == true ]]; then
  echo "Installed binaries; LaunchAgent activation was not requested."
  echo "Previous binaries (if any): ${BACKUP_DIR}"
  exit 0
fi
"${BRIDGE_BIN}" check-config --config "${CONFIG_PATH}"

python3 "${SERVICE_DIR}/scripts/render-launchagent.py" \
  --uid "${UID}" --home "${HOME}" --user "${USER}" --tmpdir "${TMPDIR}" \
  --binary "${BRIDGE_BIN}" --config "${CONFIG_PATH}" --log-dir "${LOG_DIR}" \
  --output "${PLIST_PATH}.new"
plutil -lint "${PLIST_PATH}.new" >/dev/null
chmod 0600 "${PLIST_PATH}.new"
mv "${PLIST_PATH}.new" "${PLIST_PATH}"

launchctl bootout "gui/${UID}/${SERVICE_LABEL}" 2>/dev/null || true
for legacy_label in "${LEGACY_SERVICE_LABELS[@]}"; do
  launchctl bootout "gui/${UID}/${legacy_label}" 2>/dev/null || true
  rm -f "${HOME}/Library/LaunchAgents/${legacy_label}.plist"
done
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
