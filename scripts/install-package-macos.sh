#!/usr/bin/env bash
set -euo pipefail

if [[ "${OSTYPE:-}" != darwin* ]]; then
  echo "the Fruit Forwarder package requires macOS" >&2
  exit 2
fi

PACKAGE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVICE_LABEL="com.pdfowler.fruitforwarder"
LEGACY_SERVICE_LABELS=(
  "net.pdfowler.icloud-reminders-bridge"
  "com.example.icloud-reminders-bridge"
)
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
PACKAGE_BRIDGE="${PACKAGE_DIR}/bin/icloud-reminders-bridge"
PACKAGE_EVENTKIT="${PACKAGE_DIR}/bin/icloud-reminders-eventkit"

INSTALL_ONLY=false
MIGRATE_HOME_CTRL=false
for argument in "$@"; do
  case "${argument}" in
    --install-only) INSTALL_ONLY=true ;;
    --migrate-home-ctrl) MIGRATE_HOME_CTRL=true ;;
    *) echo "unknown option: ${argument}" >&2; exit 2 ;;
  esac
done

[[ -x "${PACKAGE_BRIDGE}" && -x "${PACKAGE_EVENTKIT}" ]] || {
  echo "package is missing executable bridge files" >&2
  exit 2
}
codesign --verify --strict "${PACKAGE_EVENTKIT}"
VERSION="$("${PACKAGE_BRIDGE}" version)"
[[ "${VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$ ]] || {
  echo "package bridge reported an invalid version: ${VERSION}" >&2
  exit 2
}

if [[ -L "${CONFIG_DIR}" || -L "${CONFIG_PATH}" ]]; then
  echo "refusing a symlinked configuration path" >&2
  exit 2
fi
for path in "${INSTALL_DIR}" "${BIN_DIR}" "${ROLLBACK_DIR}" "${LOG_DIR}"; do
  if [[ -L "${path}" ]]; then
    echo "refusing a symlinked install path: ${path}" >&2
    exit 2
  fi
done
mkdir -p "${BIN_DIR}" "${ROLLBACK_DIR}" "${CONFIG_DIR}" "${LOG_DIR}" "$(dirname "${PLIST_PATH}")"
chmod 0700 "${INSTALL_DIR}" "${BIN_DIR}" "${ROLLBACK_DIR}"
for path in "${INSTALL_DIR}" "${BIN_DIR}" "${ROLLBACK_DIR}" "${LOG_DIR}"; do
  if [[ -L "${path}" ]]; then
    echo "refusing a symlinked install path: ${path}" >&2
    exit 2
  fi
done

if [[ ! -f "${CONFIG_PATH}" ]]; then
  if [[ "${MIGRATE_HOME_CTRL}" == true && -f "${LEGACY_CONFIG_PATH}" ]]; then
    [[ ! -L "${LEGACY_CONFIG_PATH}" ]] || { echo "refusing a symlinked legacy configuration" >&2; exit 2; }
    install -m 0600 "${LEGACY_CONFIG_PATH}" "${CONFIG_PATH}"
    echo "Migrated the existing home-ctrl configuration; its explicit Keychain and state paths were preserved." >&2
  else
    install -m 0600 "${PACKAGE_DIR}/config.example.json" "${CONFIG_PATH}"
    echo "Created ${CONFIG_PATH}; add exact IDs, then rerun this installer." >&2
    if [[ "${INSTALL_ONLY}" != true ]]; then
      exit 2
    fi
  fi
fi
chmod 0600 "${CONFIG_PATH}"

STAGE_BRIDGE="${BIN_DIR}/.icloud-reminders-bridge.new"
STAGE_EVENTKIT="${BIN_DIR}/.icloud-reminders-eventkit.new"
install -m 0755 "${PACKAGE_BRIDGE}" "${STAGE_BRIDGE}"
install -m 0755 "${PACKAGE_EVENTKIT}" "${STAGE_EVENTKIT}"
codesign --verify --strict "${STAGE_EVENTKIT}"
if [[ "${INSTALL_ONLY}" != true ]]; then
  "${STAGE_BRIDGE}" check-config --config "${CONFIG_PATH}"
fi

INSTALL_STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
BACKUP_DIR="${ROLLBACK_DIR}/${VERSION}-${INSTALL_STAMP}"
mkdir -p "${BACKUP_DIR}"
chmod 0700 "${BACKUP_DIR}"
if [[ -f "${BRIDGE_BIN}" ]]; then cp -p "${BRIDGE_BIN}" "${BACKUP_DIR}/icloud-reminders-bridge"; fi
if [[ -f "${EVENTKIT_BIN}" ]]; then cp -p "${EVENTKIT_BIN}" "${BACKUP_DIR}/icloud-reminders-eventkit"; fi
mv "${STAGE_BRIDGE}" "${BRIDGE_BIN}"
if ! mv "${STAGE_EVENTKIT}" "${EVENTKIT_BIN}"; then
  if [[ -f "${BACKUP_DIR}/icloud-reminders-bridge" ]]; then
    cp -p "${BACKUP_DIR}/icloud-reminders-bridge" "${BRIDGE_BIN}"
  else
    rm -f "${BRIDGE_BIN}"
  fi
  echo "failed to switch the EventKit helper; restored the previous executable pair" >&2
  exit 1
fi

if [[ "${INSTALL_ONLY}" == true ]]; then
  echo "Installed Fruit Forwarder ${VERSION}; LaunchAgent activation was not requested."
  echo "Previous binaries (if any): ${BACKUP_DIR}"
  exit 0
fi
python3 "${PACKAGE_DIR}/scripts/render-launchagent.py" \
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
for _ in {1..20}; do
  if ! launchctl print "gui/${UID}/${SERVICE_LABEL}" >/dev/null 2>&1; then break; fi
  sleep 0.25
done
launchctl bootstrap "gui/${UID}" "${PLIST_PATH}"
launchctl kickstart -k "gui/${UID}/${SERVICE_LABEL}"
echo "Installed and started Fruit Forwarder ${VERSION}."
echo "Previous binaries (if any): ${BACKUP_DIR}"
