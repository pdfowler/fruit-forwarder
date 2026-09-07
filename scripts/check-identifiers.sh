#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
stable="com.pdfowler.fruitforwarder"
helper="${stable}.eventkit"

python3 - "${repo_dir}" "${stable}" "${helper}" <<'PY'
from pathlib import Path
import sys

repo = Path(sys.argv[1])
stable = sys.argv[2]
helper = sys.argv[3]

stable_files = [
    "deployment/Info.plist",
    "deployment/com.pdfowler.fruitforwarder.plist.tmpl",
    "config.example.json",
    "packaging/mcp/config.example.json",
    "internal/config/config.go",
    "scripts/render-launchagent.py",
    "scripts/install-macos.sh",
    "scripts/install-package-macos.sh",
    "scripts/rollback-macos.sh",
    "scripts/uninstall-macos.sh",
]
helper_files = [
    "deployment/eventkit-helper-Info.plist",
    "Taskfile.yml",
    "scripts/build-macos.sh",
]

missing = []
for relative in stable_files:
    if stable not in (repo / relative).read_text():
        missing.append(f"{relative}: {stable}")
for relative in helper_files:
    if helper not in (repo / relative).read_text():
        missing.append(f"{relative}: {helper}")
if missing:
    raise SystemExit("stable identifier checks failed:\n" + "\n".join(missing))
print(f"stable native identifiers verified: {stable}, {helper}")
PY
