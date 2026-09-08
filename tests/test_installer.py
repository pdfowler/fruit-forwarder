"""Static safety checks for the packaged macOS installer."""

from pathlib import Path


ROOT = Path(__file__).parents[1]
INSTALLER = (ROOT / "scripts/install-package-macos.sh").read_text()


def test_package_installer_rejects_config_symlinks_before_staging():
    guard = INSTALLER.index('if [[ -L "${CONFIG_DIR}" || -L "${CONFIG_PATH}" ]]')
    staging = INSTALLER.index('STAGE_BRIDGE=')
    backup = INSTALLER.index('BACKUP_DIR=')
    assert guard < staging < backup


def test_package_installer_rejects_symlinked_install_paths():
    assert 'for path in "${INSTALL_DIR}" "${BIN_DIR}" "${ROLLBACK_DIR}" "${LOG_DIR}"' in INSTALLER
    assert "reject_symlink_components" in INSTALLER
    rollback = (ROOT / "scripts/rollback-macos.sh").read_text()
    assert '[[ ! -L "${backup}" && ! -L "${BIN_DIR}" ]]' in rollback
    assert "reject_symlink_components" in rollback


def test_package_installer_validates_config_before_switching_binaries():
    validation = INSTALLER.index('"${STAGE_BRIDGE}" check-config --config "${CONFIG_PATH}"')
    switch = INSTALLER.index('mv "${STAGE_BRIDGE}" "${BRIDGE_BIN}"')
    assert validation < switch


def test_package_installer_verifies_both_native_executables():
    bridge = INSTALLER.index('codesign --verify --strict "${PACKAGE_BRIDGE}"')
    helper = INSTALLER.index('codesign --verify --strict "${PACKAGE_EVENTKIT}"')
    switch = INSTALLER.index('mv "${STAGE_BRIDGE}" "${BRIDGE_BIN}"')
    assert bridge < helper < switch


def test_package_installer_bundles_launchagent_renderer():
    package_script = (ROOT / "scripts/package-macos.sh").read_text()
    assert 'scripts/render-launchagent.py' in package_script


def test_rollback_verifies_both_native_executables():
    rollback = (ROOT / "scripts/rollback-macos.sh").read_text()
    bridge = rollback.index('codesign --verify --strict "${STAGE_BRIDGE}"')
    helper = rollback.index('codesign --verify --strict "${STAGE_EVENTKIT}"')
    switch = rollback.index('mv "${STAGE_BRIDGE}" "${BRIDGE_BIN}"')
    assert bridge < helper < switch


def test_uninstaller_rejects_symlinked_paths_before_removal():
    uninstaller = (ROOT / "scripts/uninstall-macos.sh").read_text()
    guard = uninstaller.index("reject_symlink_components")
    removal = uninstaller.index('rm -f "${PLIST_PATH}"')
    assert guard < removal
    assert 'refusing to remove a symlinked uninstall target' in uninstaller
