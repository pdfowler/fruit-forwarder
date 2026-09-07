# Release roadmap

The target is an approachable local iCloud bridge for Home Assistant and MCP.
Current implementation: Reminders through macOS EventKit, HA todo entities,
and local stdio MCP. Read-only calendar events are wired through MCP and HA;
real-account and full lifecycle verification remain open.

Before a broadly recommended release:

- Verify a fresh Mac install, permission grants, background operation, restart,
  and reboot recovery. Resolve the observed launchd EventKit timeout.
- Extend MCP verification to an installed application and real EventKit.
  Subprocess tests now build the CLI and verify stdio configuration, read-only
  enforcement, helper invocation, and helper failure reporting with synthetic data.
  In-memory protocol tests cover tool discovery, filtered reads, rejected
  out-of-scope reads, write dispatch, and the optional read-only policy.
- Test the HA integration against a supported HA release, including queue
  recovery, malformed requests, duplicate configuration, and stale entities.
  Runtime tests on HA 2026.2.3 now cover command recovery, acknowledgements,
  scope rejection, and rollback on storage failure or task cancellation;
  they also cover revocation across restart and paused dispatch for unavailable
  or read-only lists;
  HTTP webhook and complete entity lifecycle coverage remain open.
- Document and test create replay behavior across crashes: an EventKit save
  and the local acknowledgement are separate operations, so a crash between
  them can duplicate a creation.
- Validate installer upgrades, token rotation, and rollback before publishing
  signed or packaged releases. Manual component copying is the current HA path.
- Extend the EventKit adapter to calendars with a bounded time window, explicit
  calendar allowlists, HA calendar entities, and corresponding MCP tools.
  Native discovery and bounded reads are implemented with separate Calendar
  permission and no-access boundary tests. Go allowlist/window validation and
  conditional MCP calendar read tools are implemented. HA snapshot transport,
  calendar allowlists, read-only entities, cached window checks, and restoration
  tests are implemented. Calendar failures no longer block reminder
  publication; HA retains the last confirmed calendar snapshot and exposes its
  last successful calendar sync timestamp. Real recurrence/all-day validation
  and full lifecycle verification remain open. See docs/calendar-protocol.md.
- Prepare release notes and a public repository with CI results and a private
  security-reporting channel.

Keep calendar access and reminder access independently configurable. Do not
require Apple account passwords or Home Assistant admin tokens on the Mac.
