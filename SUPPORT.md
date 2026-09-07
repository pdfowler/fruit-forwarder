# Support

Fruit Forwarder is pre-release software. Support claims are limited to the
environments and behaviors recorded in [the compatibility matrix](docs/compatibility.md)
and [the acceptance ledger](docs/acceptance-evidence.md); a successful build or
synthetic test is not evidence of Apple-account or background-service support.

## Before opening an issue

1. Read [setup and troubleshooting](docs/setup.md), especially the distinction
   between foreground EventKit access and launchd lifecycle behavior.
2. Run the non-mutating diagnostics appropriate to the affected target:

   ```sh
   bridge status --json
   bridge doctor --json
   ```

3. Check the [compatibility matrix](docs/compatibility.md) and current release
   notes. Reproduce with a disposable reminder list or calendar when possible.

## Bug reports

Use the GitHub bug-report template once the public repository is available.
Include the Fruit Forwarder version, macOS architecture/version, Home Assistant
version when relevant, MCP client/version when relevant, the smallest redacted
reproduction, and whether the failure affects Reminders, Calendars, HA, MCP, or
installation. Include diagnostic output only after removing paths, URLs,
account identifiers, EventKit IDs, reminder/calendar contents, and tokens.

Do not attach `config.json`, state files, Keychain exports, screenshots with
household data, pairing URLs, or full logs. A pairing token is a persistent
bearer secret, not a disposable code.

## Security reports

Do not use a public issue for credentials, authorization bypasses, data
exposure, or malicious-package reports. Follow [SECURITY.md](SECURITY.md) and
request a private security contact if the repository's private-reporting channel
is not yet enabled.

## Support boundaries

The initial release supports scoped Apple Reminders mutations and read-only
EventKit calendars through a macOS bridge, Home Assistant, and local stdio MCP.
It does not claim support for network MCP, Calendar writes, reminder deletion,
list management, Find My, or other iCloud services. The project cannot provide
Apple, Home Assistant, or third-party MCP-client account support beyond the
documented integration boundary.

Maintainers may ask for a redacted acceptance record rather than household
data. Reproduction on an unsupported macOS, Home Assistant, or MCP-client
version may be useful, but it does not expand the supported matrix until it is
validated and recorded.
