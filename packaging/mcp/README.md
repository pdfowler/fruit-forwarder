# MCP distribution preparation

The MCP server is built from the Fruit Forwarder monorepo. The release target
is an MCPB artifact containing the signed macOS bridge/MCP executable and the
minimum metadata needed by a local stdio client. The official MCP Registry
stores metadata and points clients at the artifact; it does not replace the
artifact host.

Before publishing, the release job must:

1. Build the supported macOS architecture artifact from an immutable source
   revision.
2. Assemble and inspect the MCPB bundle without credentials or household data.
3. Calculate its SHA-256 and render `server.json` from `server.json.in`.
4. Validate that the registry name, GitHub namespace, artifact URL, version,
   checksum and package transport agree.
5. Publish the artifact first, then publish metadata with `mcp-publisher` only
   after the public URL and checksum are immutable.

The current repository has metadata preparation only; no package is published.
