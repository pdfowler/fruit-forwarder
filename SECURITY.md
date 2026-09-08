# Security policy

This bridge handles private iCloud reminder data and a Home Assistant pairing
token. Keep the bridge on a trusted Mac and use HTTPS for Home Assistant.

## Trust boundaries

macOS grants Reminders access to the responsible application; it does not
provide a per-list permission grant. The list allowlist is enforced by bridge
code, not by an Apple capability token. Other software running as the same
user can potentially change the config or invoke the helper directly.

MCP uses local stdio, with no network listener. By default a connected client
can invoke all exposed write tools for allowed lists without an additional
bridge prompt. Set `mcp_read_only` to `true` to omit all four write tools.
Restart existing MCP sessions after changing this policy. This is a per-process
policy, not a restriction on other software running under the same macOS user.
Review tool approvals in the client. Reminder titles and notes are untrusted
content and should never be treated as agent instructions.

The HA webhook path is a bearer secret. The Mac stores it in Keychain; HA also
stores it in its integration configuration. Protect HA backups and redact
webhook URLs in reverse-proxy access logs. The `local_only` webhook setting is
an additional network restriction, not a replacement for token secrecy.

The bridge rejects runtime configuration and acknowledgement-state files that
are symlinks, group/world-writable, or owned by another user. Their parent
directories must also be private and user-owned. This prevents an accidental
shared path from changing the allowlist or replay ledger; it is not protection
against another process already running as the same macOS user.

Native helper stdout is bounded before JSON parsing, and helper diagnostics are
bounded before they can enter an error message. Oversized helper output fails
closed rather than being retained in memory or parsed as a partial response.

HA persists reminder snapshots and pending commands in its storage. Recent
completed-item filtering changes the published snapshot; it does not delete
Apple Reminders history or scrub existing HA backups and recorder history.

The Mac acknowledgement state journals a command before invoking EventKit. If
the helper fails or the process is interrupted after that point, the command is
marked as having an uncertain outcome and automatic command application stops.
An operator must inspect the Apple state and use the explicit `recover` command
to mark the operation applied or allow a retry. This favors avoiding duplicate
creates over silently retrying an ambiguous write.

On integration load, the current HA allowlist also filters stored snapshots and
queued commands. Removed lists and their commands are removed from active bridge
storage, so restoring access later does not replay the old commands. Existing
backups are unaffected. A temporarily missing or read-only list keeps its pending
commands, but they are withheld until a writable snapshot for that list returns.

## Threat model

The initial release is a local, single-household bridge. The following table
states what is protected, who is considered an attacker, and where the design
deliberately stops.

| Asset or boundary | Threat | Mitigation | Residual limitation |
| --- | --- | --- | --- |
| Apple Reminders and Calendar contents | An untrusted LAN client or proxy reaches the bridge | The supported bridge-to-HA path uses HTTPS; the native service is not a network MCP server; HA can enforce `local_only` | A compromised Home Assistant host or trusted reverse proxy can still read data it is authorized to receive |
| Pairing token | Token appears in process arguments, URLs, logs, or support output | Keychain storage, stdin-based pairing, token redaction, TLS, and explicit log/backup guidance | A bearer token is not a user identity; anyone who obtains it can use the configured HA path until rotation |
| Mac configuration and acknowledgement journal | A different local user or an accidental shared path alters scope or replays work | User ownership, private modes, parent-directory checks, symlink rejection, queue-epoch fencing, and explicit ambiguous-command recovery | Another process running as the same macOS user can bypass application-level controls |
| EventKit access | The bridge requests broader Apple access than the household intended | Separate Reminders/Calendar opt-in and exact application allowlists | macOS EventKit permissions are application-level, not per-list; the allowlist is enforced by this bridge rather than by Apple |
| Reminder/calendar text | Malicious or instruction-like content is returned from Apple | Content is treated as data; tools do not execute titles, notes, URLs, or locations | A connected MCP client may still display or reason over that content according to its own policy |
| Native helper and HTTP payloads | Oversized or malformed data causes memory pressure or parser confusion | Bounds at helper output, JSON parsing, command fields, snapshots, events, and response bodies; fail-closed validation | Resource limits are safety bounds, not a guarantee against all denial-of-service conditions on the host |
| Release artifacts and publication workflows | A branch, dirty tree, or modified generated export is published | Immutable tag checks, complete-history attribution, checksums, deterministic HACS export digests, protected environments, manual confirmations, and OIDC for MCP publication | Developer ID certificate custody and Apple notarization remain maintainer-controlled release gates |
| HA snapshots, backups, and recorder history | Household data persists after a user expects a current view to be cleared | Bounded recent-completion publication and documented backup/recorder behavior | This project does not delete Apple history, HA recorder history, or existing backups |

Remote/network MCP is intentionally outside this threat model. Enabling it
would require a separate design for client identity, authorization, TLS,
revocation, replay protection, rate limits, audit data, and secret storage; do
not expose the stdio process through an unauthenticated proxy.

Before a public release, review each row against installed-artifact evidence,
the compatibility matrix, dependency/license results, and the release
manifest. A control described here is not evidence that the corresponding
real-account, lifecycle, signing, or publication test has passed.

## Reporting

Do not report vulnerabilities with tokens, reminder contents, or state files
attached. Please open a private security report with a description, affected
version, and reproduction steps. If private reporting is unavailable, open a
minimal issue asking for a security contact without including sensitive data.

The project intentionally does not support reminder deletion or list-management
through its API.
