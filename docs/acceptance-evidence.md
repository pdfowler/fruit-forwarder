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
| R7 | Approachable installation/upgrades | Partial | Versioned archive, installer, rollback scripts, setup documentation, isolated packaged install/upgrade/rollback under a temporary Home, tested `status`/`doctor` diagnostics, special-path LaunchAgent rendering test, and pre-switch install-path/configuration trust tests | Real-user activation, service health, signed/notarized artifact, and lifecycle evidence |
| R8 | Privacy-preserving operation | Partial | Keychain stdin handling, file trust checks, bounded-state tests including native helper output limits; Home Assistant diagnostics exclusion test | Review of installed process args, backups, diagnostics, ACLs and signed release behavior |
| R9 | Shareable open-source distribution | Partial | MIT license, CI, contributor/security docs, `SUPPORT.md`, deterministic HACS export, guarded HACS and MCP publication workflows, macOS/MCP candidates, complete-history attribution check | Authorized public repositories, release assets, and ecosystem publication |
| R10 | Honest compatibility/support claims | Partial | [compatibility matrix](compatibility.md), `SUPPORT.md`, version checks and release manifest | Recorded supported-version runs and known-limitations review |

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

### Canonical repository candidate rebuild (2026-09-07)

- `task check` and `task release:prepare` passed from the clean source revision
  recorded in `dist/release-manifest.json` after the Go module path was aligned
  with the planned public `github.com/pdfowler/fruit-forwarder` repository, the
  Home Assistant config-flow scope bounds were hardened, and the guarded
  publication task surface was added.
- The manifest records `source_dirty: false` and eight verified outputs: the
  HACS export and archive, macOS archive, MCPB and checksum sidecars, MCP
  Registry metadata, and the unified release manifest.
- Complete-history author/committer validation still passes for
  `pfowler@icloud.com`.
- This is source/build provenance evidence only. It does not replace installed
  macOS permission/lifecycle, live HA, independent MCP-client, or publication
  evidence below.

The macOS package from revision
`a768e1ec3b61934062ee6bf2a4d33d1e2360709f` was installed twice under an
isolated temporary `HOME` with `INSTALL_ONLY=true`, then rolled back from the
generated backup. Both executable switches succeeded, rollback restored the
pair, the example configuration remained present, and no LaunchAgent was
activated. This proves the packaged transaction boundary only; it does not
prove EventKit permission, background service, upgrade schema, or reboot
behavior on the real account.

### Earlier packaged transaction evidence (revision `4d9742f`)

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

### Legacy configuration audit (2026-09-07)

- The legacy configuration is mode `0600`, contains one configured reminder
  list, and has no calendar scope. Its existing bridge, Keychain and state
  path fields are present and can be preserved by the packaged migration path;
  their values were not recorded here.
- The candidate schema adds an independent `calendars` scope while retaining
  the existing `lists` shape. This supports a reminders-only migration first,
  followed by an explicit calendar opt-in rather than silently broadening
  access.
- This was a key/shape audit only. No configuration, Keychain item, state file,
  permission, or service was changed.

### Live gate refresh (2026-09-07 14:44 PDT)

- The approved HA-MCP healthcheck is ready (`ha-mcp 8.4.3`, 78 tools). A
  read-only integration query still finds one loaded
  `icloud_reminders_bridge` entry titled `iCloud Reminders on Dean` with
  `supports_reconfigure: false`.
- `com.pdfowler.fruitforwarder` is not loaded in the Dean user launch domain.
  The legacy `net.pdfowler.icloud-reminders-bridge` LaunchAgent remains the
  only active writer and still invokes the old `launchctl asuser` arrangement.
- This refresh was read-only. No HA entry, service, permission, Keychain item,
  configuration, or household data was changed.

### Staged migration and authorization diagnostic (2026-09-08 UTC)

- The 0.1.0 macOS candidate was staged with `--install-only
  --migrate-home-ctrl`. The copied configuration retained its explicit legacy
  Keychain service/account and acknowledgement-state paths; the installer did
  not activate or stop a LaunchAgent.
- The staged candidate passed `check-config` and `status --json`, including
  executable ownership and state-file readiness. The legacy service remained
  the only active writer.
- A new non-prompting `authorization` helper action now reports Reminders and
  Calendar permission state separately through `doctor`. The staged candidate
  reports `not_determined` for both services and exits nonzero with an
  actionable grant message. This distinguishes missing permission for the new
  Fruit Forwarder identity from the legacy helper's `calaccessd` XPC failure,
  but does not prove a granted or working background permission.
- No service cutover, permission reset, Keychain rotation, or household-data
  mutation was performed.

### Candidate rebuild refresh (2026-09-08 UTC)

- `task release:prepare` passed from clean source revision
  `49d4421f0cb062cad024b542bd4783ca21ca695f`, with complete-history
  `pfowler@icloud.com` attribution, gitleaks, Go race/vet, 55 HA tests,
  deterministic HACS export, macOS package validation, MCPB validation, and
  release-manifest verification.
- The candidate now serializes `serve`, `sync-once`, explicit recovery, queue
  epoch reset, and per-operation MCP EventKit access through the same private
  state lock. Synthetic coverage verifies that recovery and MCP access fail
  closed while another bridge operation owns that lock.
- MCP reminder and calendar result tools now expose bounded offset/limit
  pagination (default and maximum 100 results per response), with a
  `next_offset` continuation and total count.
- This remains source/build and synthetic concurrency evidence. It does not
  replace installed macOS permission/lifecycle, live HA, independent MCP
  client, real-account semantic, or publication evidence.

### Reminder semantics candidate refresh (2026-09-08 UTC)

- The candidate adds a supported/unsupported reminder field matrix in
  `docs/reminder-semantics.md`, including explicit preservation rules for
  EventKit-owned fields and the configured completed-history bound.
- MCP `reminders_update` now uses a locked read-merge-write path: omitted
  description or due fields are preserved, while explicit empty strings clear
  them. Synthetic tests cover both paths; Home Assistant's existing
  full-item update queue remains unchanged.
- `task release:prepare` passed from the clean candidate revision recorded in
  `dist/release-manifest.json`. This is still synthetic/source evidence and
  does not substitute for disposable real-account mutation or independent MCP
  client acceptance.

### Home Assistant freshness candidate refresh (2026-09-08 UTC)

- HA reminder and calendar entities now locally poll only their freshness
  metadata while continuing to receive all item/event data through the webhook.
  After ten minutes without a successful capability snapshot, the affected
  entity becomes unavailable and exposes `sync_status: stale` plus its snapshot
  age; reminder and calendar freshness are tracked independently.
- Synthetic HA coverage verifies stale reminder and calendar availability,
  diagnostics fields, and the local polling contract. This does not replace a
live HA outage/recovery run against the installed custom component.

### Diagnostics lock-health candidate refresh (2026-09-08 UTC)

- `status --json` and `doctor --json` now report `state_lock_status` without
  creating or modifying the lock file. Synthetic coverage verifies available,
  busy, and post-release states; a busy lock is treated as expected while a
  bridge service is running.
- This remains local diagnostics evidence. It does not replace a live
launchd/process lifecycle check on a signed macOS installation.

### Current Home Assistant compatibility lane (2026-09-08 UTC)

- The Home Assistant runtime suite passes against HA `2026.9.1` on Python
  `3.14.2`, matching the live Home Assistant version observed through the
  approved HA-MCP control surface. The repository retains HA `2026.2.3` on
  Python `3.13` as its reproducible baseline and CI now runs both lanes.
- This validates the custom component's synthetic/runtime compatibility only;
  the installed live entry is still the legacy source and remains a separate
  cutover and lifecycle gate.

### Installed candidate foreground acceptance (2026-09-08 UTC)

- The generated `fruit-forwarder-macos-0.1.0.tar.gz` was installed on Dean
  with `--install-only`. The candidate LaunchAgent was not activated, and the
  legacy `net.pdfowler.icloud-reminders-bridge` service remained the only
  active writer.
- Installed `status --json` and `doctor --json` reported executable and state
  readiness, `state_lock_status: available`, Reminders authorization, and
  Keychain readiness. Calendar authorization remained `not_determined` because
  the migrated configuration has no calendar scope; no permission prompt was
  triggered.
- With the live Home Assistant pending-command queue empty, the installed
  candidate completed a foreground `sync-once` successfully and applied zero
  commands. The live HA entity's `last_sync` advanced to
  `2026-09-08T06:13:41Z` and its pending-command count remained zero.
- This proves the installed foreground/read-snapshot path and lock boundary
  only. It does not prove candidate LaunchAgent activation, background TCC
  behavior, reboot recovery, live candidate HA component installation,
  calendar permission, real mutations, independent MCP-client acceptance, or
  service cutover.

### Live HA installation and Fruit Forwarder cutover (2026-09-08 UTC)

- The HACS-exported `icloud_reminders_bridge` component from source revision
  `e4c994f67a4bb83d9c830862403e7f77d35425c7` was copied to the live HA
  appliance over its approved SSH path. The prior component is preserved at
  `/config/.fruit-forwarder-backups/icloud_reminders_bridge.pre-fruit-forwarder-20260908T063206Z`;
  no reminder data or HA storage was modified.
- After the initial restart exposed HA's package-directory scanning of a dotted
  backup name, the backup was moved outside `custom_components` and HA was
  restarted again. The integration then loaded cleanly with
  `supports_reconfigure: true`; the existing entry ID and entity ID were
  preserved.
- Candidate foreground sync succeeded with zero commands applied. The live
  entity reported `sync_status: confirmed`, `pending_commands: 0`, and source
  `iCloud`; its needs-action view contained one existing item. No todo item was
  completed, created, renamed, or removed as part of this verification.
- The candidate macOS artifact was then activated. `com.pdfowler.fruitforwarder`
  is running under `gui/501`; the legacy service labels are absent. The direct
  LaunchAgent arrangement timed out in EventKit, while an equivalent interactive
  and manually invoked `launchctl asuser` run succeeded. The renderer now
  retains the `asuser` wrapper for TCC-safe background operation, but the
  rebuilt LaunchAgent still reports EventKit request timeouts; background health
  remains open pending a fresh macOS Reminders-permission/session check.

### Ecosystem publication gate refresh (2026-09-08 UTC)

- Current HACS requirements were reviewed against the generated distribution:
  it contains exactly one integration under `custom_components/`, required
  manifest fields, both brand locations, and HACS/Hassfest workflow files.
  The export validator now fails if a second integration directory appears.
- Current MCP Registry MCPB requirements were reviewed against `dist/mcp/server.json`:
  the metadata uses the `io.github.pdfowler/fruit-forwarder` namespace, an
  exact versioned GitHub release URL containing `mcp`, `registryType: "mcpb"`,
  `stdio` transport, and the candidate artifact SHA-256. The package validator
  now checks each of these fields before release preparation succeeds.
- The MCP Registry remains a preview service, so namespace and package rules
  must be rechecked immediately before the first authorized publication. No
  public repository, release, HACS listing, or Registry entry exists yet.

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
