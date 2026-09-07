# Changelog

All notable Fruit Forwarder changes are recorded here. Versions remain
pre-release until the real macOS, Home Assistant, and MCP acceptance gates in
`PROJECT-PLAN.md` are complete.

## Unreleased

- Continue validating background EventKit permissions, upgrades, and reboot
  recovery on supported macOS installations.
- Keep calendar publication independent from reminder publication when Calendar
  access is unavailable; expose the last successful calendar sync timestamp in
  Home Assistant.
- Pause automatic mutation replay after an ambiguous EventKit outcome and
  provide explicit operator recovery choices.

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
