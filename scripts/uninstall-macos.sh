#!/usr/bin/env bash
set -euo pipefail

if [[ "${OSTYPE:-}" != darwin* ]]; then
  echo "the Fruit Forwarder service requires macOS" >&2
  exit 2
fi

SERVICE_LABEL="com.pdfowler.fruitforwarder"
LEGACY_SERVICE_LABELS=(
  "net.pdfowler.icloud-reminders-bridge"
  "com.example.icloud-reminders-bridge"
)
INSTALL_DIR="${HOME}/Library/Application Support/icloud-reminders-bridge"
BIN_DIR="${INSTALL_DIR}/bin"
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

for path in "${INSTALL_DIR}" "${BIN_DIR}" "${PLIST_PATH}" \
  "${HOME}/Library/LaunchAgents"; do
  reject_symlink_components "${path}"
done
for path in "${BIN_DIR}/icloud-reminders-bridge" \
  "${BIN_DIR}/icloud-reminders-eventkit" "${PLIST_PATH}"; do
  [[ ! -L "${path}" ]] || {
    echo "refusing to remove a symlinked uninstall target: ${path}" >&2
    exit 2
  }
done
for legacy_label in "${LEGACY_SERVICE_LABELS[@]}"; do
  reject_symlink_components "${HOME}/Library/LaunchAgents/${legacy_label}.plist"
  [[ ! -L "${HOME}/Library/LaunchAgents/${legacy_label}.plist" ]] || {
    echo "refusing to remove a symlinked legacy LaunchAgent" >&2
    exit 2
  }
done

launchctl bootout "gui/${UID}/${SERVICE_LABEL}" 2>/dev/null || true
for legacy_label in "${LEGACY_SERVICE_LABELS[@]}"; do
  launchctl bootout "gui/${UID}/${legacy_label}" 2>/dev/null || true
done
for _ in {1..20}; do
  if ! launchctl print "gui/${UID}/${SERVICE_LABEL}" >/dev/null 2>&1; then break; fi
  sleep 0.25
done
rm -f "${PLIST_PATH}" "${BIN_DIR}/icloud-reminders-bridge" "${BIN_DIR}/icloud-reminders-eventkit"
for legacy_label in "${LEGACY_SERVICE_LABELS[@]}"; do
  rm -f "${HOME}/Library/LaunchAgents/${legacy_label}.plist"
done

echo "Stopped and removed Fruit Forwarder executables and LaunchAgent."
echo "Preserved configuration, Keychain pairing, state, rollback copies, and logs:"
echo "  ${HOME}/.config/icloud-reminders-bridge"
echo "  ${INSTALL_DIR}"
echo "  ${HOME}/Library/Logs/icloud-reminders-bridge"
