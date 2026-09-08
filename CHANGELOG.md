# Changelog

All notable Fruit Forwarder changes are recorded here. Versions remain
pre-release until the real macOS, Home Assistant, and MCP acceptance gates in
`PROJECT-PLAN.md` are complete.

## Unreleased

- Add standard Home Assistant dashboard examples for scoped todo lists and
  read-only calendars.
- Add a threat model covering EventKit permissions, HA bearer tokens, local
  process trust, untrusted reminder content, bounded payloads, and publication
  workflows.
- Add an explicit MCP client compatibility matrix separating generic stdio,
  MCPB package, named-client, and unsupported network claims.
- Add a non-prompting EventKit authorization diagnostic so `doctor` can
  distinguish executable readiness from the installed helper's actual
  Reminders/Calendar permission state during migration and recovery.
- Clean up the local HA webhook when config-entry platform setup fails, making
  reload retries safe after transient setup errors.
- Reject Home Assistant mutation commands outside the configured list allowlist
  before journaling them as uncertain EventKit operations.
- Reject delayed, duplicate, or untimestamped Home Assistant snapshot replays
  after a timestamped bridge sync has been established.
- Add guarded `task publish:release`, `task publish:hacs`, and `task publish:mcp`
  wrappers with exact-tag and attribution preflight.
- Preserve signed macOS toolchain metadata when the final release job merges
  artifacts from separate HA and macOS preparation jobs.
- Harden macOS uninstall and Home Assistant configuration-flow bounds.
- Add a shared `task build:macos` target and route the macOS and MCPB packages
  through the same staged native build and boundary-test path.
- Have macOS CI invoke the shared Task targets instead of maintaining a second
  native build recipe.
- Harden the guarded HACS publication workflow against shell interpolation and
  accidental destination repositories.
- Add a public-facing support policy with compatibility, redaction, and security
  reporting boundaries.
- Sign both native executables through the shared build path and record the
  selected macOS signing identity in release provenance.
- Add explicit public issue routing and third-party attribution notices.
- Align the Go module path with the planned public `fruit-forwarder`
  repository so downstream builds and release metadata use one canonical
  source identity.
- Standardize the packaged native identity as `com.pdfowler.fruitforwarder`
  and add an explicit migration path from the former home-ctrl deployment.
- Add a redacted R1–R10 acceptance ledger and release-manifest toolchain
  provenance for publication review.
- Add bounded retry backoff and jitter for transient Home Assistant outages.
- Bound Home Assistant's pending command queue and queued mutation fields, with
  fail-closed oversized-response handling.
- Add protocol capability negotiation so calendar-enabled clients fail closed
  instead of silently losing calendar data with older HA integrations.
- Add persisted Home Assistant queue epochs so restored command queues are
  fenced until an operator explicitly reviews and accepts the new epoch.
- Make packaged installation validate configuration and install-path trust
  before switching binaries, with rollback symlink guards and regression tests.
- Add a guarded, tag-only workflow for publishing the generated HACS
  distribution repository.
- Bound native helper output before parsing or reporting errors.
- Render the new per-user LaunchAgent with a direct bridge process instead of
  the unverified `launchctl asuser` wrapper.
- Add a redacted Home Assistant diagnostics surface for safe operational support.
- Continue validating background EventKit permissions, upgrades, and reboot
  recovery on supported macOS installations.
- Keep calendar publication independent from reminder publication when Calendar
  access is unavailable; expose the last successful calendar sync timestamp in
  Home Assistant.
- Pause automatic mutation replay after an ambiguous EventKit outcome and
  provide explicit operator recovery choices.
- Enforce the requested `pfowler@icloud.com` author and committer identity
  across the complete Git history during local checks and publication workflows.
- Add a guarded, tag-only MCP Registry publication workflow that validates the
  immutable MCPB release asset before OIDC publication, with a pinned publisher
  URL and SHA-256 supplied at protected dispatch time.

## 0.1.0 — candidate

- Added scoped Reminders create, update, complete, and reopen support through
  Home Assistant and local stdio MCP.
- Added read-only, bounded EventKit calendar discovery and reads through MCP
  and Home Assistant.
- Added deterministic HACS, macOS, and MCPB packaging plus release provenance
  and integrity checks.
- Added explicit privacy, compatibility, installation, rollback, and
  publication-gate documentation.

This candidate has not been published to GitHub, HACS, or the MCP Registry.
