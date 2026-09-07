#!/usr/bin/env bash
set -euo pipefail

if [[ "${OSTYPE:-}" != darwin* ]]; then
  echo "the Fruit Forwarder service requires macOS" >&2
  exit 2
fi

SERVICE_LABEL="com.example.icloud-reminders-bridge"
INSTALL_DIR="${HOME}/Library/Application Support/icloud-reminders-bridge"
BIN_DIR="${INSTALL_DIR}/bin"
PLIST_PATH="${HOME}/Library/LaunchAgents/${SERVICE_LABEL}.plist"

launchctl bootout "gui/${UID}/${SERVICE_LABEL}" 2>/dev/null || true
for _ in {1..20}; do
  if ! launchctl print "gui/${UID}/${SERVICE_LABEL}" >/dev/null 2>&1; then break; fi
  sleep 0.25
done
rm -f "${PLIST_PATH}" "${BIN_DIR}/icloud-reminders-bridge" "${BIN_DIR}/icloud-reminders-eventkit"

echo "Stopped and removed Fruit Forwarder executables and LaunchAgent."
echo "Preserved configuration, Keychain pairing, state, rollback copies, and logs:"
echo "  ${HOME}/.config/icloud-reminders-bridge"
echo "  ${INSTALL_DIR}"
echo "  ${HOME}/Library/Logs/icloud-reminders-bridge"
