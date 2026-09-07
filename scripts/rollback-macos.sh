#!/usr/bin/env bash
set -euo pipefail

SERVICE_LABEL="com.pdfowler.fruitforwarder"
INSTALL_DIR="${HOME}/Library/Application Support/icloud-reminders-bridge"
BIN_DIR="${INSTALL_DIR}/bin"
ROLLBACK_DIR="${INSTALL_DIR}/rollback"
BRIDGE_BIN="${BIN_DIR}/icloud-reminders-bridge"
EVENTKIT_BIN="${BIN_DIR}/icloud-reminders-eventkit"
PLIST_PATH="${HOME}/Library/LaunchAgents/${SERVICE_LABEL}.plist"

reject_symlink_components() {
  local candidate="$1"
  local trusted_root="${HOME%/}"
  [[ -n "${trusted_root}" && "${trusted_root}" != "/" ]] || {
    echo "refusing an empty or root HOME path" >&2
    exit 2
  }
  [[ ! -L "${trusted_root}" ]] || {
    echo "refusing a symlinked HOME path" >&2
    exit 2
  }
  case "${candidate}" in
    "${trusted_root}"|"${trusted_root}"/*) ;;
    *) echo "refusing a path outside HOME: ${candidate}" >&2; exit 2 ;;
  esac
  local remaining="${candidate#"${trusted_root}"}"
  local prefix="${trusted_root}"
  while [[ -n "${remaining}" ]]; do
    remaining="${remaining#/}"
    [[ -n "${remaining}" ]] || break
    local component="${remaining%%/*}"
    if [[ "${remaining}" == "${component}" ]]; then
      remaining=""
    else
      remaining="${remaining#*/}"
    fi
    [[ "${component}" == "." ]] && continue
    [[ "${component}" != ".." ]] || {
      echo "refusing a path containing ..: ${candidate}" >&2
      exit 2
    }
    prefix="${prefix%/}/${component}"
    if [[ -L "${prefix}" ]]; then
      echo "refusing a path with a symlinked ancestor: ${candidate}" >&2
      exit 2
    fi
  done
}

if [[ $# -ne 1 ]]; then
  echo "usage: $0 /path/to/rollback/<version-timestamp>" >&2
  echo "available rollback directories:" >&2
  find "${ROLLBACK_DIR}" -mindepth 1 -maxdepth 1 -type d -print 2>/dev/null | sort >&2 || true
  exit 2
fi

backup="$1"
case "${backup}" in
  "${ROLLBACK_DIR}"/*) ;;
  *) echo "rollback path must be inside ${ROLLBACK_DIR}" >&2; exit 2 ;;
esac
for path in "${INSTALL_DIR}" "${BIN_DIR}" "${ROLLBACK_DIR}" "${backup}" "${PLIST_PATH}"; do
  reject_symlink_components "${path}"
done
[[ -d "${backup}" ]] || { echo "rollback directory not found: ${backup}" >&2; exit 2; }
[[ ! -L "${backup}" && ! -L "${BIN_DIR}" ]] || {
  echo "rollback target or install bin directory must not be a symlink" >&2
  exit 2
}
[[ -f "${backup}/icloud-reminders-bridge" && -f "${backup}/icloud-reminders-eventkit" ]] || {
  echo "rollback directory does not contain both bridge executables" >&2
  exit 2
}
[[ ! -L "${backup}/icloud-reminders-bridge" && ! -L "${backup}/icloud-reminders-eventkit" ]] || {
  echo "rollback executables must not be symlinks" >&2
  exit 2
}

STAGE_BRIDGE="${BIN_DIR}/.rollback-icloud-reminders-bridge.new"
STAGE_EVENTKIT="${BIN_DIR}/.rollback-icloud-reminders-eventkit.new"
install -m 0755 "${backup}/icloud-reminders-bridge" "${STAGE_BRIDGE}"
install -m 0755 "${backup}/icloud-reminders-eventkit" "${STAGE_EVENTKIT}"
codesign --verify --strict "${STAGE_EVENTKIT}"
mv "${STAGE_BRIDGE}" "${BRIDGE_BIN}"
if ! mv "${STAGE_EVENTKIT}" "${EVENTKIT_BIN}"; then
  # The staged bridge is the only file changed so far. Restore the current
  # pair from the same rollback target before returning an error.
  cp -p "${backup}/icloud-reminders-bridge" "${BRIDGE_BIN}"
  echo "failed to switch the EventKit helper; restored the previous executable pair" >&2
  exit 1
fi

if [[ -f "${PLIST_PATH}" ]]; then
  launchctl bootout "gui/${UID}/${SERVICE_LABEL}" 2>/dev/null || true
  for _ in {1..20}; do
    if ! launchctl print "gui/${UID}/${SERVICE_LABEL}" >/dev/null 2>&1; then
      break
    fi
    sleep 0.25
  done
  launchctl bootstrap "gui/${UID}" "${PLIST_PATH}"
  launchctl kickstart -k "gui/${UID}/${SERVICE_LABEL}"
fi

echo "Rolled back executables from ${backup}. Configuration, Keychain and state were preserved."
