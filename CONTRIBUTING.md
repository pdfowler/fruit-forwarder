# Contributing

Contributions are welcome. Please keep changes narrow and avoid adding
credentials, reminder contents, EventKit identifiers, or generated build
artifacts.

HA runtime tests use a real Home Assistant Python package with isolated mocked
storage. They exercise queue recovery and persistence failure without talking
to your HA server or Apple account. With Python 3.13:

```sh
python3.13 -m venv .venv
.venv/bin/pip install -r requirements-test.txt
.venv/bin/python -m pytest -q
```

The pinned version is a reproducible test baseline, not a compatibility claim
for every newer HA release. Webhook HTTP and full entity lifecycle tests remain
separate release work.

Before opening a pull request on macOS, run:

```sh
task check
go test -race ./...
go vet ./...
bash -n scripts/install-macos.sh
```

`task package:ha` produces the reviewable HACS tree under `dist/` and
`scripts/check-hacs-export.sh` verifies that the export is deterministic. The
Mac and native targets are host-specific; CI builds them on macOS. Keep
generated `build/` and `dist/` output out of commits.

The Go suite also builds a bridge subprocess and exercises its real stdio MCP
transport with a synthetic executable helper. It verifies configuration wiring,
reads, read-only enforcement, and graceful helper failure. It does not grant
Reminders access or read Apple account data. CI additionally compiles and
ad-hoc signs the Swift helper; this does not establish runtime permission or
reboot reliability on a user's Mac.

The native helper needs macOS and a user-granted Reminders permission. Changes
to the Home Assistant component should preserve the list allowlist and the
bounded completed-item retention behavior.
