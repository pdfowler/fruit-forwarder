"""LaunchAgent rendering tests for paths requiring plist escaping."""

import plistlib
import subprocess
import sys
from pathlib import Path


def test_render_launchagent_preserves_special_paths(tmp_path: Path) -> None:
    output = tmp_path / "LaunchAgents" / "fruit-forwarder.plist"
    script = Path(__file__).parents[1] / "scripts" / "render-launchagent.py"
    values = {
        "--uid": "501",
        "--home": "/Users/Pat & Co",
        "--user": "pat & co",
        "--tmpdir": "/var/folders/a space/<tmp>",
        "--binary": "/Users/Pat & Co/Library/Application Support/Fruit/bin/bridge",
        "--config": "/Users/Pat & Co/.config/Fruit & Forwarder/config.json",
        "--log-dir": "/Users/Pat & Co/Library/Logs/Fruit <Forwarder>",
        "--output": str(output),
    }
    command = [sys.executable, str(script)]
    for key, value in values.items():
        command.extend((key, value))
    subprocess.run(command, check=True)

    with output.open("rb") as stream:
        plist = plistlib.load(stream)
    assert plist["Label"] == "com.pdfowler.fruitforwarder"
    assert plist["ProgramArguments"][:4] == [
        "/bin/launchctl",
        "asuser",
        values["--uid"],
        values["--binary"],
    ]
    assert plist["ProgramArguments"][4] == "serve"
    assert plist["EnvironmentVariables"]["HOME"] == "/Users/Pat & Co"
    assert plist["ProgramArguments"][-1] == values["--config"]
    assert plist["StandardOutPath"] == values["--log-dir"] + "/icloud-reminders-bridge.log"

    raw = output.read_text()
    assert "&amp;" in raw
    assert "&lt;" in raw
