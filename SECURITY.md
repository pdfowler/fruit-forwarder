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

HA persists reminder snapshots and pending commands in its storage. Recent
completed-item filtering changes the published snapshot; it does not delete
Apple Reminders history or scrub existing HA backups and recorder history.

## Reporting

Do not report vulnerabilities with tokens, reminder contents, or state files
attached. Please open a private security report with a description, affected
version, and reproduction steps. If private reporting is unavailable, open a
minimal issue asking for a security contact without including sensitive data.

The project intentionally does not support reminder deletion or list-management
through its API.
