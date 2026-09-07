# MCP distribution preparation

The MCP server is built from the Fruit Forwarder monorepo. The release target
is a macOS-only MCPB artifact containing the bridge executable and EventKit
helper plus the minimum metadata needed by a local stdio client. The official
MCP Registry stores metadata and points clients at the artifact; it does not
replace the artifact host.

The bundle asks the MCP host for an absolute user-owned configuration-file path
and supplies the bundled helper path with `--eventkit-helper`. The configuration
file contains the household's exact allowlists and must not be committed. Start
from [config.example.json](config.example.json), run the native `discover` and
`discover-calendars` commands, and set `mcp_read_only` deliberately.

Build and inspect a candidate on macOS:

```sh
task package:mcp
```

This uses the pinned `@anthropic-ai/mcpb` CLI (or `MCPB_BIN`), validates the
MCPB manifest, emits `dist/mcp/fruit-forwarder-mcp-<version>.mcpb`, and writes a
SHA-256 sidecar plus a candidate `server.json` using the planned GitHub release
URL. It does not publish the artifact or registry metadata. For a signed release
candidate, set `MCPB_CERT` and `MCPB_KEY` to maintainer-managed certificate and
key files outside the repository. The script asks the MCPB CLI to verify the
signature and fails closed if the certificate is not trusted by the build host;
an ad-hoc or self-signed test certificate is not a publishable signature.

Before publishing, the release job must:

1. Build the supported macOS architecture artifact from an immutable source
   revision.
2. Assemble and inspect the MCPB bundle without credentials or household data.
3. Calculate its SHA-256 and render `server.json` from `server.json.in`.
4. Validate that the registry name, GitHub namespace, artifact URL, version,
   checksum and package transport agree.
5. Publish the artifact first, then publish metadata with `mcp-publisher` only
   after the public URL and checksum are immutable.

The repository now prepares and validates the MCPB artifact and registry
metadata. Registry publication and the GitHub release remain deliberately
gated until the public namespace, signed artifact policy and real-client
acceptance evidence are approved.
