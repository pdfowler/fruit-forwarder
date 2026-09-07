# Fruit Forwarder

A local macOS bridge between Apple Reminders/Calendar and Home Assistant. It
uses Apple's EventKit API for reads and writes, exposes a small stdio MCP
server, and pushes allowlisted data to Home Assistant.

**Early release.** Reminders and read-only calendars support HA and local MCP
(see [calendar setup](docs/calendar-protocol.md)). Background
EventKit permission and reboot recovery still need validation; see the
[release roadmap](ROADMAP.md). Start with the complete [setup guide](docs/setup.md)
for Home Assistant and local MCP configuration.
See the [compatibility matrix](docs/compatibility.md) for the current support
claim and [architecture decisions](docs/adr/) for the monorepo and trust-boundary
choices.
Release history is tracked in [CHANGELOG.md](CHANGELOG.md).
The release gate is tracked in [docs/acceptance-evidence.md](docs/acceptance-evidence.md).

## Security boundary

- Apple credentials never leave macOS. EventKit uses the Mac user's signed-in accounts
  and macOS Reminders permission.
- The bridge makes outbound HTTPS requests to one Home Assistant webhook. It does not
  hold a Home Assistant long-lived access token.
- The webhook uses a random 256-bit path token and is registered `local_only`.
- Both HA and the bridge enforce a reminder-list allowlist. The bridge uses exact EventKit
  list IDs for reads and writes.
- HA and MCP offer create, edit, complete, and reopen. Delete and list-management
  operations are intentionally absent.
- MCP can expose only read tools with `"mcp_read_only": true`; HA editing is
  independently available.
- The webhook token is stored in the macOS login Keychain. It is not in the
  LaunchAgent environment, repository, or JSON configuration.
- Logs contain counts, command IDs, and errors—not reminder titles or notes.

## Components

- `eventkit-helper`: a small native Swift helper using only Apple's public
  EventKit API.
- `cmd/icloud-reminders-bridge`: discovery, pairing, daemon, one-shot sync, and
  stdio MCP entrypoint. It invokes the native helper directly with a 30-second
  timeout; macOS enforces Reminders access for the responsible application.
- `homeassistant/custom_components/icloud_reminders_bridge`: push-driven HA todo
  entities and the durable command queue.
- `deployment`: a user LaunchAgent for the Mac.

## Quick start

1. Build and install on a Mac signed in to the iCloud account that owns the lists:

   ```sh
   ./scripts/install-macos.sh --install-only
   ```

2. Grant Reminders access when macOS prompts. Discover exact list IDs:

   ```sh
   ~/Library/Application\ Support/icloud-reminders-bridge/bin/icloud-reminders-bridge discover
   ```

3. Edit `~/.config/icloud-reminders-bridge/config.json`, adding only the lists
   you want to expose. Run `pair` to create a persistent bearer token and configure the
   Home Assistant custom integration with that token and the same list IDs.

4. Start the per-user LaunchAgent:

   ```sh
   ./scripts/install-macos.sh
   ```

After a reboot or upgrade, `icloud-reminders-bridge doctor --json` checks the
configured helper and (when HA sync is enabled) the pairing Keychain item
without reading reminder contents.
If a write reaches an uncertain EventKit outcome, `doctor` reports the
in-flight command and syncing pauses until it is explicitly resolved; see the
recovery steps in [docs/setup.md](docs/setup.md).

The pairing token is stored in the macOS login Keychain. Do not commit the
runtime config, state file, logs, or token.

The packaged build uses the stable reverse-DNS service identifier
`com.pdfowler.fruitforwarder`. Existing prototype configs that explicitly use
`com.example.icloud-reminders-bridge` remain valid; do not delete the old
Keychain item until the replacement installation has been verified.

## Configuration

`completed_retention` controls how long completed reminders remain in the
bridge snapshot. The default is 30 days (`720h`); this prevents an old iCloud
history from filling Home Assistant while preserving recent completed items.

The bridge intentionally does not expose delete or list-management operations.
Review the allowlist and the Home Assistant webhook path before placing the
service on a network shared with untrusted clients.

## Development

The repository is a small polyglot monorepo: Go owns the bridge/MCP process,
Swift owns the EventKit helper, and the Home Assistant integration is exported
as a deterministic HACS distribution. Use [Task](https://taskfile.dev/) for
the common cross-target commands:

```sh
task check          # Go, HA, metadata and script checks
task test           # Go, HA, and native EventKit tests when running on macOS
task package:ha    # build the HACS tree and versioned HA archive
task package:macos # build a versioned macOS tarball on macOS
task package:mcp   # build the macOS MCPB and candidate registry metadata
task release:check # validate a reproducible candidate without publishing
task release:prepare # assemble every target available on this host
```

The MCP registry metadata is intentionally rendered only after a versioned MCPB
artifact has been built and hashed; see [packaging/mcp/README.md](packaging/mcp/README.md).
No publishing command is implicit in a build or release check.
Maintainer publication gates and the HACS/MCP follow-up steps are documented in
[docs/publishing.md](docs/publishing.md).

The macOS tarball is self-installing: extract it, copy the exact IDs into the
included configuration, and run `scripts/install-package-macos.sh`. The source
checkout's `scripts/install-macos.sh` remains the developer build/install path.
The packaged installer preserves the previous executable pair under the same
rollback directory and uses the bundled LaunchAgent renderer; it does not
overwrite an existing runtime configuration or Keychain item.
The bundled `scripts/uninstall-macos.sh` removes only service files and keeps
that runtime data for an explicit later cleanup decision.

```sh
go test -race ./...
go vet ./...
```

The native helper requires macOS and the EventKit permission prompt. See
`CONTRIBUTING.md` for local build details and `SECURITY.md` for disclosure guidance.
