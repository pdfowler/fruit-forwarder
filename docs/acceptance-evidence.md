# Fruit Forwarder acceptance evidence

This ledger is the release gate for the requirements in
[PROJECT-PLAN.md](../PROJECT-PLAN.md). It deliberately distinguishes source
implementation and synthetic tests from evidence gathered from an installed
artifact, a real Apple account, Home Assistant, and an independent MCP client.
Do not mark a row **proven** from unit-test output alone.

## Status vocabulary

- **Proven** — the required scope has recorded, redacted evidence attached or
  linked below.
- **Partial** — some automated or synthetic coverage exists, but the required
  real-environment or publication evidence is still missing.
- **Missing** — no adequate evidence has been recorded.
- **Deferred** — explicitly removed from the supported release scope.

## Requirement ledger

| ID | Requirement | Current status | Existing evidence | Still required |
| --- | --- | --- | --- | --- |
| R1 | Native macOS account access without Apple credentials | Partial | EventKit request-validation tests; signed/ad-hoc helper build checks | Fresh installed-app permission, lock/sleep/reboot and login lifecycle evidence |
| R2 | Scoped Reminders mutations through HA and MCP | Partial | Go, MCP and HA protocol tests; allowlist and recovery tests | Disposable real-list create/edit/complete/reopen, including out-of-scope denial |
| R3 | Scoped read-only calendars through HA and MCP | Partial | Calendar mapping, window validation and transport tests | Timed/all-day/recurring/timezone fixtures against a real account and both clients |
| R4 | Independent reminder/calendar opt-in | Partial | Separate optional configuration and calendar-failure isolation tests | Reminder-only, calendar-only, combined and permission-denied installed runs |
| R5 | Secure local MCP | Partial | Stdio protocol tests, read-only tool omission, bounded input checks, MCPB validation | At least one independent MCP client, process/listener inspection and permission evidence |
| R6 | Recoverable HA synchronization | Partial | Durable state journal, explicit mutation recovery, persisted queue-epoch fencing, offline/calendar-failure tests | HA restart/crash/concurrent-edit and backup/restore tests with visible pending/error UX |
| R7 | Approachable installation/upgrades | Partial | Versioned archive, installer, rollback scripts, setup documentation, tested `status`/`doctor` diagnostics, special-path LaunchAgent rendering test, and pre-switch install-path/configuration trust tests | Clean-user install, upgrade and rollback from the packaged artifact |
| R8 | Privacy-preserving operation | Partial | Keychain stdin handling, file trust checks, bounded-state tests including native helper output limits; Home Assistant diagnostics exclusion test | Review of installed process args, backups, diagnostics, ACLs and signed release behavior |
| R9 | Shareable open-source distribution | Partial | MIT license, CI, contributor/security docs, deterministic HACS export, guarded HACS and MCP publication workflows, macOS/MCP candidates, complete-history attribution check | Authorized public repositories, release assets, support channel and ecosystem publication |
| R10 | Honest compatibility/support claims | Partial | [compatibility matrix](compatibility.md), version checks and release manifest | Recorded supported-version runs, known-limitations review and support policy |

## Current candidate evidence

Run these commands from a clean checkout on the source revision intended for
release:

```sh
task check
task release:prepare
```

`dist/release-manifest.json` is the authoritative candidate index. It records
the source revision, dirty-tree state, compatibility/counterpart versions,
host toolchain and package-tool versions, and SHA-256 values for the HA, macOS,
and MCP artifacts. The manifest is not publication evidence by itself: it must
be paired with the installed acceptance records below.

The macOS artifact from revision `4d9742f` was also exercised through
`task install:local` in an isolated temporary `HOME` with
`INSTALL_ONLY=true`. The wrapper verified the archive/checksum, rejected no
unexpected members, staged both executables and created only the example
configuration; it did not touch the real user profile or activate launchd.
An otherwise valid-looking archive with a `../` member was also rejected
before extraction. This proves the packaged staging path and its archive
boundary checks, not permissions, EventKit access, upgrade/rollback, or
background service health.

### Dean live observation (2026-09-07)

- The legacy `net.pdfowler.icloud-reminders-bridge` LaunchAgent is still the
  active writer; the new `com.pdfowler.fruitforwarder` service is not loaded.
- The legacy EventKit helper carries the
  `net.pdfowler.icloud-reminders-bridge.eventkit` identity, and repeated sync
  attempts report `XPC error communicating with calaccessd: Unknown error`.
- Direct foreground discovery using that same legacy binary reproduces the
  error, so the failure is not attributed solely to the old `launchctl asuser`
  wrapper.
- No legacy service was stopped, migrated, or permission-reset during this
  observation. Fresh stable-identity permissions and lifecycle testing remain
  required before claiming R1/M1 background support.

### Dean live refresh (2026-09-07 13:52 PDT)

- The candidate `com.pdfowler.fruitforwarder` LaunchAgent is still absent;
  `net.pdfowler.icloud-reminders-bridge` remains the only active writer under
  `gui/501`, using the legacy `launchctl asuser` arrangement.
- The legacy service continues to emit `XPC error communicating with
  calaccessd: Unknown error` on its 30-second sync cycle. Two legacy MCP
  processes are also present; no candidate service was started.
- This was read-only observation. No service, permission, configuration,
  Keychain item, or household data was changed.

### Home Assistant live inspection (2026-09-07 13:57 PDT)

- The approved HA-MCP endpoint reported Home Assistant `2026.9.1` in `RUNNING`
  state. The configured `icloud_reminders_bridge` domain has one loaded entry,
  titled `iCloud Reminders on Dean`.
- That live entry reports `supports_reconfigure: false`, while the candidate
  source implements a reconfigure flow. This is evidence that the candidate
  source has not yet been installed and verified in this HA instance; it is not
  evidence that the source flow works against a live entry.
- The live entry exposes one bridge-owned todo entity and its current item
  query returned six completed items. No bridge-owned calendar entity was
  found; the eight visible calendar entities belong to other integrations or
  sources and were not attributed to Fruit Forwarder.
- The inspection was read-only. No todo item, integration, permission,
  configuration, or service state was changed.

## Installed acceptance record template

Create one redacted record per run, for example under a maintainer-controlled
release evidence store rather than committing household data here:

```text
run_id:
date_utc:
source_revision:
artifact_manifest_sha256:
macos_version_and_architecture:
home_assistant_version:
mcp_client_and_version:
scenario:
expected_result:
actual_result:
result: pass | fail | blocked
evidence_locations:
notes_with_no_household_contents:
```

Use disposable lists/calendars and redact titles, notes, EventKit IDs, URLs,
tokens, account identifiers, and personal paths. A blocked external test is
evidence of an open gate, not a pass.

## Publication readiness rule

Do not publish a target while a required row is only **Partial**, unless the
maintainer records an explicit product decision that narrows the supported
claim and updates the compatibility matrix, setup guide, changelog, and
release manifest accordingly. HACS and MCP listing status must be recorded
separately; generated metadata and a repository name do not prove listing
acceptance.
