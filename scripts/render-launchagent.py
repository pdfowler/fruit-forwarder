#!/usr/bin/env python3
"""Render the per-user LaunchAgent plist without shell/XML interpolation."""

from __future__ import annotations

import argparse
from pathlib import Path
import plistlib


def main() -> None:
    parser = argparse.ArgumentParser()
    # Kept as a compatibility input for existing installer invocations. A
    # LaunchAgent is already bootstrapped in gui/<uid>, so no asuser wrapper is
    # needed and it can obscure the responsible process for TCC diagnostics.
    parser.add_argument("--uid", required=False, default="")
    parser.add_argument("--home", required=True)
    parser.add_argument("--user", required=True)
    parser.add_argument("--tmpdir", required=True)
    parser.add_argument("--binary", required=True)
    parser.add_argument("--config", required=True)
    parser.add_argument("--log-dir", required=True)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()

    label = "com.pdfowler.fruitforwarder"
    plist = {
        "Label": label,
        "ProgramArguments": [
            args.binary,
            "serve",
            "--config",
            args.config,
        ],
        "RunAtLoad": True,
        "KeepAlive": True,
        "ProcessType": "Background",
        "EnvironmentVariables": {
            "HOME": args.home,
            "USER": args.user,
            "TMPDIR": args.tmpdir,
        },
        "StandardOutPath": f"{args.log_dir}/icloud-reminders-bridge.log",
        "StandardErrorPath": f"{args.log_dir}/icloud-reminders-bridge.log",
        "ThrottleInterval": 15,
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open("wb") as stream:
        plistlib.dump(plist, stream, fmt=plistlib.FMT_XML, sort_keys=False)


if __name__ == "__main__":
    main()
