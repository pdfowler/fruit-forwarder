# ADR 0001: One source monorepo with ecosystem-specific exports

- Status: Accepted for the initial release
- Date: 2026-09-07

## Decision

Keep Go, Swift, the Home Assistant integration, protocol fixtures, packaging,
and release metadata in one canonical `fruit-forwarder` repository. Generate a
root-shaped HACS distribution with `scripts/export-hacs.sh` and package MCP and
macOS artifacts from the same source revision. A separate `ha-fruit-forwarder`
repository is a publication surface, not a second source of truth.

## Rationale

Protocol and native changes must be reviewed against HA and MCP consumers in one
change. Separate ecosystem names improve discovery without creating duplicated
implementations or version drift. Task provides enough orchestration for the
current Go/Swift/Python targets without introducing a JavaScript workspace or a
second build graph.

## Consequences

The export must be deterministic and traceable to a source commit. HACS and MCP
publication remain separately gated, and the release manifest records the
compatibility set. If the number of independently versioned targets grows,
revisit the command layer rather than silently creating parallel repositories.
