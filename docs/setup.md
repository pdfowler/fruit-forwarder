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
Restart Home Assistant to load the custom integration. This repository does not
currently provide HACS installation.

```sh
"$bridge" check-config
"$bridge" pair
```

In HA, open Settings → Devices & services → Add integration → iCloud Reminders
Bridge. Paste the token from the clipboard, use the same `bridge_id`, and enter
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

After foreground sync succeeds, enable the background service:

```sh
./scripts/install-macos.sh
```

Check `~/Library/Logs/icloud-reminders-bridge/icloud-reminders-bridge.log` for
successful syncs. Foreground permission does not prove launchd access works:
if the helper times out in the background, recheck macOS permissions and use
`"$bridge" serve` in an interactive terminal while investigating. Background
permission/reboot recovery remains a release validation gap.

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
the six tools allow list, read, create, update, complete, and reopen operations
within the configured allowlist. MCP does not currently have a read-only mode.

The MCP client may need its own Reminders permission. The server can establish
a protocol session even when EventKit is unavailable; a successful handshake
alone does not prove reminders can be read.
