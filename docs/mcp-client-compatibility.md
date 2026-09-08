# MCP client compatibility

Fruit Forwarder supports local stdio MCP on macOS. The bridge does not open a
network listener, and the MCPB is a macOS package containing the Go server and
EventKit helper. This matrix separates what the repository verifies from what
requires an installed third-party client and a real Apple account.

| Client shape | Distribution/configuration | Current evidence | Release claim |
| --- | --- | --- | --- |
| Generic local stdio host | Launch the bridge executable with absolute `mcp` and `--config` paths | Protocol subprocess tests, read-only catalog tests, malformed-input and helper-failure tests | Configuration shape is supported; a named client still needs an installed acceptance run |
| MCPB/Desktop Extension host | Install the versioned `fruit-forwarder-mcp-<version>.mcpb` and provide a user-owned config path | MCPB manifest/schema, archive contents, native executable signatures, checksum and Registry metadata validation | Package shape is prepared; host-specific installation and permission behavior remain unverified |
| Remote/network MCP client | No supported package or transport | Explicitly excluded from the current protocol and threat model | Do not expose the stdio process through a proxy |

## Generic stdio configuration

Use absolute paths; many clients do not expand `~` or shell variables:

```json
{
  "mcpServers": {
    "fruit-forwarder": {
      "command": "/Users/YOUR_USER/Library/Application Support/icloud-reminders-bridge/bin/icloud-reminders-bridge",
      "args": [
        "mcp",
        "--config",
        "/Users/YOUR_USER/.config/icloud-reminders-bridge/config.json"
      ]
    }
  }
}
```

The host may need its own macOS Reminders permission. A successful MCP
handshake proves only that the stdio process speaks the protocol; it does not
prove EventKit access, allowlist correctness, or Apple synchronization.

## Acceptance record required for a named client

Before adding a client name to the supported matrix, record the exact client
and version, installation route, macOS architecture, source/artifact revision,
configured reminder and calendar scopes, permission prompts, tool catalog,
read/write results, shutdown/restart behavior, and any process/listener
inspection. Use disposable Apple lists/calendars and redact their contents and
identifiers. Read-only mode must be checked both during tool discovery and by a
direct attempted mutation.

See [acceptance-evidence.md](acceptance-evidence.md) for the shared redacted
record format and [setup.md](setup.md) for the local configuration walkthrough.
