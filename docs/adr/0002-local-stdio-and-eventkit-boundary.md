# ADR 0002: Local stdio MCP and native EventKit ownership

- Status: Accepted for the initial release
- Date: 2026-09-07

## Decision

The Mac user session owns EventKit access. Home Assistant communicates through a
scoped HTTPS webhook, while MCP uses local stdio and does not open a network
listener. The bridge accepts exact reminder-list and calendar IDs as application
allowlists; macOS permissions remain a separate, broader OS-level boundary.

## Rationale

Apple does not provide fine-grained service tokens for this use case. Keeping the
Apple account in the signed-in user's EventKit store avoids collecting an Apple
password or inventing a Family Sharing bot identity. Local stdio keeps MCP
authorization in the user's client and avoids exposing a new LAN service before
an authenticated network design exists.

## Consequences

The native process must be tested under launchd, lock/unlock, sleep/wake and
reboot. Other processes under the same macOS user can potentially access the
EventKit permission or alter local files, so the application allowlist is not an
OS isolation boundary. A future network MCP transport requires a separate
authentication and revocation design.
