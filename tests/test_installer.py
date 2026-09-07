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


def test_package_installer_bundles_launchagent_renderer():
    package_script = (ROOT / "scripts/package-macos.sh").read_text()
    assert 'scripts/render-launchagent.py' in package_script
