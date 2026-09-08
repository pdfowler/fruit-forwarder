# Setup

Use a Mac signed in to the account whose reminders you want to share. Install
Go (the version in `go.mod` or newer) and Xcode Command Line Tools for Swift.
Keep the Mac awake and the user session logged in for continuous syncing.
This is an early release; validate foreground sync before enabling launchd.

## Install and select lists

From the repository root:

```sh
./scripts/install-macos.sh --install-only
bridge="$HOME/Library/Application Support/icloud-reminders-bridge/bin/icloud-reminders-bridge"
"$bridge" discover
```

For a published macOS tarball, extract it and run its bundled
`scripts/install-package-macos.sh` instead. That installer uses the already
built, checksummed pair and does not require Go or Xcode Command Line Tools.

Allow Reminders access in System Settings → Privacy & Security → Reminders.
Discovery prints list names and IDs, which may be private: do not paste the
output into public issues. Copy the exact IDs and names into
`~/.config/icloud-reminders-bridge/config.json`:

```json
{
  "bridge_id": "mac",
  "home_assistant_url": "https://ha.example.com",
  "poll_interval": "30s",
  "completed_retention": "720h",
  "lists": [{"id": "YOUR-EXACT-EVENTKIT-ID", "name": "Tasks"}]
}
```

Replace the URL with your HA HTTPS address reachable from the local network.
The bridge validates TLS certificates and refuses redirects. For MCP only,
omit `home_assistant_url`; no HA token is needed.

## Home Assistant

Copy the entire repository directory
`homeassistant/custom_components/icloud_reminders_bridge` into
`<your HA config directory>/custom_components/icloud_reminders_bridge`.
Restart Home Assistant to load the custom integration. The repository's
`task package:ha` target produces the HACS-compatible tree used by the planned
`ha-fruit-forwarder` distribution repository. Until that repository is public,
use the generated tree as the reviewed custom-component source rather than
copying the monorepo's parent directory.

```sh
"$bridge" check-config
"$bridge" pair
```

Use the non-mutating diagnostics after an upgrade or reboot:

```sh
"$bridge" status --json
"$bridge" doctor --json
```

`status` checks configuration and executable ownership without reading reminder
contents. `doctor` additionally checks the Home Assistant pairing item in the
login Keychain when HA sync is configured; neither command prints the token.
If a mutation reaches an ambiguous EventKit outcome, the bridge records the
in-flight command and pauses further command application rather than blindly
retrying it. Inspect the command ID with `doctor --json`, then choose an
explicit resolution only after checking Apple Reminders:

```sh
"$bridge" recover --command-id COMMAND_ID --resolution applied
# or, if the reminder was not changed and a retry is safe:
"$bridge" recover --command-id COMMAND_ID --resolution retry
```

See [command-lifecycle.md](command-lifecycle.md) for the full queued,
withheld, in-flight, confirmed, retryable, and uncertain state model.
If a restored Home Assistant backup changes the command-queue epoch, inspect
the restored queue before accepting it:

```sh
"$bridge" reset-queue-epoch --queue-epoch EPOCH_FROM_THE_REVIEWED_HA_QUEUE
```

The bridge will not apply queued mutations across an epoch change, and the
command is deliberately separate from uncertain EventKit recovery.

Each installation keeps the previous executable pair under
`~/Library/Application Support/icloud-reminders-bridge/rollback/`. If a new
build fails its lifecycle check, stop the service and pass one of those exact
directories to `scripts/rollback-macos.sh`; configuration, Keychain pairing and
state are preserved.

In HA, open Settings → Devices & services → Add integration → Fruit Forwarder.
Paste the token from the clipboard, use the same `bridge_id`, and enter
the exact list IDs, one per line. Add one integration entry for this pairing.
The token is a persistent bearer secret, not a single-use code. Running `pair`
again replaces the Mac's Keychain token; HA must then be configured to match.

```sh
"$bridge" sync-once
```

Open HA's To-do lists page. A list entity appears after its first snapshot.
Use a disposable reminder to verify create, complete, and reopen in HA and
Apple Reminders. HA changes are queued until the Mac's next sync, normally
within 30 seconds plus Apple's own account sync delay.
Transient Home Assistant failures use bounded exponential retry backoff with
jitter (capped at 15 minutes); a successful sync returns to the configured
poll interval.

See [dashboard-examples.md](dashboard-examples.md) for standard Lovelace todo
and calendar cards. Calendar entities are read-only; calendar scope is
independently opt-in.

After foreground sync succeeds, enable the background service:

```sh
./scripts/install-macos.sh
```

Check `~/Library/Logs/icloud-reminders-bridge/icloud-reminders-bridge.log` for
successful syncs. Foreground permission does not prove launchd access works:
if the helper times out in the background, recheck macOS permissions and use
`"$bridge" serve` in an interactive terminal while investigating. Background
permission/reboot recovery remains a release validation gap. The Fruit Forwarder
LaunchAgent runs the bridge directly in the logged-in user's GUI domain; it does
not add a nested `launchctl asuser` wrapper. The former home-ctrl service may
still show the legacy arrangement until migration is explicitly activated.

## Local MCP

Configure an MCP client that supports stdio on the same Mac. A common JSON
configuration shape is:

```json
{
  "mcpServers": {
    "icloud-reminders": {
      "command": "/Users/YOUR_USER/Library/Application Support/icloud-reminders-bridge/bin/icloud-reminders-bridge",
      "args": ["mcp", "--config", "/Users/YOUR_USER/.config/icloud-reminders-bridge/config.json"]
    }
  }
}
```

Replace both absolute paths; clients may not expand `~` or shell variables.
The client launches the process and communicates over stdin/stdout. There is
no listening network port or HTTP MCP endpoint. Only connect trusted clients:
the reminder tools allow list, read, create, update, complete, and reopen
operations within the configured allowlist; calendar tools are read-only when
calendars are enabled. To expose only the two reminder read tools, set
`"mcp_read_only": true` in the JSON config and restart the MCP client session.
Write tools are then absent from the server catalog and direct calls to their
names are rejected. This setting applies only to MCP; HA can still queue edits.
Use a separate config file if different clients need different access policies.

The MCP client may need its own Reminders permission. The server can establish
a protocol session even when EventKit is unavailable; a successful handshake
alone does not prove reminders can be read.

### MCPB distribution

On macOS, `task package:mcp` builds a self-contained MCPB containing the Go
server and EventKit helper. Install that artifact only in an MCP host that
supports MCPB/Desktop Extension packages, then point its required configuration
file setting at a user-owned copy of `packaging/mcp/config.example.json` with
the exact IDs discovered on that Mac. The MCPB path does not require Home
Assistant or an HA token. The artifact is macOS-only. Its native executables are
ad-hoc signed for local boundary testing; a distributable release still requires
maintainer-controlled Developer ID signing and notarization.

`task package:macos` produces the companion versioned macOS tarball. These
commands prepare artifacts only; publishing a GitHub release, HACS repository,
or MCP Registry entry remains an explicit maintainer step.

If this Mac still runs the former `home-ctrl` deployment, pass
`--migrate-home-ctrl` to the packaged installer. It copies the existing
configuration, preserves its explicit Keychain/state paths, and retires the
legacy LaunchAgent when that migration is activated. Review the copied config
before the second installer invocation; do not run both services at once.

To stop and remove only the installed service files while preserving household
configuration, pairing, state, rollback copies, and logs, run the bundled
`scripts/uninstall-macos.sh`. Credential/state removal is intentionally a
separate, manual cleanup decision.
