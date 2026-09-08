# Contributing

Contributions are welcome. Please keep changes narrow and avoid adding
credentials, reminder contents, EventKit identifiers, or generated build
artifacts.

HA runtime tests use a real Home Assistant Python package with isolated mocked
storage. They exercise queue recovery and persistence failure without talking
to your HA server or Apple account. The reproducible baseline lane uses Python
3.13:

```sh
python3.13 -m venv .venv
.venv/bin/pip install -r requirements-test.txt
.venv/bin/python -m pytest -q
```

The pinned version is a reproducible baseline. The current compatibility lane
is also pinned in `requirements-test-current.txt` and uses Python 3.14.2 or
newer; CI runs both lanes. Webhook HTTP boundaries and synthetic entity
lifecycle cases are covered locally; a complete installed-HA lifecycle remains
separate release work.

To run the current lane locally without replacing the baseline environment,
use the isolated temporary target:

```sh
task test:ha-current
```

Set `PYTHON_CURRENT` when the Python 3.14 interpreter is not named
`python3.14`. The target removes its temporary environment when it exits.

Before opening a pull request on macOS, run:

```sh
task check
task test
go test -race ./...
go vet ./...
bash -n scripts/install-macos.sh
```

`task test` runs the Go and Home Assistant suites everywhere and adds the
native EventKit permission-boundary tests on macOS. On non-macOS hosts it
reports that host-specific portion as skipped rather than pretending it ran.

`task package:ha` produces the reviewable HACS tree under `dist/` and
`scripts/check-hacs-export.sh` verifies that the export is deterministic. The
`task build:macos` produces the shared staged Go/native outputs used by the
macOS package path. The Mac and native targets are host-specific; CI builds them on macOS. Keep
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

Dependency updates are reviewed through Dependabot pull requests. Keep the
generated release artifacts and household-specific configuration out of those
updates; run the full `task check` and the supported packaging checks before
merging changes that affect a release target.
