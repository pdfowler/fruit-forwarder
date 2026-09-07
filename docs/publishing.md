# Maintainer publishing checklist

This repository prepares release assets but does not publish them during local
builds. The optional `Publish confirmed release` workflow is deliberately
manual, requires typing `PUBLISH`, and must use a protected GitHub `release`
environment with reviewer approval.

Before running it:

1. Confirm the source revision is clean and all commits are authored and
   committed as `pfowler@icloud.com`.
2. Create the matching annotated version tag (`v<release/VERSION>`) and run the
   workflow from that tag; the workflow refuses branch-based publication.
3. Run `task release:prepare` on macOS and inspect the release manifest,
   checksums, HACS archive, macOS tarball, and MCPB contents. The HA and macOS
   package scripts also re-open their finished archives and verify their
   versioned roots, required files, checksums, and absence of unexpected links
   or interpreter caches. The macOS archive is assembled with normalized
   metadata and is byte-for-byte reproducible when its staged binaries are
   unchanged; the unsigned MCPB ZIP receives the same timestamp/order
   normalization before any optional signature is applied.
4. Complete the real Mac/HA/MCP acceptance matrix in `PROJECT-PLAN.md`,
   including disposable reminders/calendars and rollback evidence.
5. Confirm the GitHub repository namespace, release signing/notarization policy,
   HACS repository destination, and MCP Registry publisher namespace.
6. Run the workflow only after the protected environment reviewer approves the
   exact source revision and version.

After the GitHub release exists:

- Push the generated HACS tree to the separately maintained `ha-fruit-forwarder`
  repository only through the approved export workflow. Its release must point
  back to the exact Fruit Forwarder source revision.
- Submit the HACS repository first as a custom repository, then request default
  catalog inclusion once the required release and validation checks are live.
- Publish `dist/mcp/server.json` with `mcp-publisher` only after the MCPB URL and
  SHA-256 are immutable. Verify a clean MCP client can fetch and install the
  exact artifact.

If one target fails, leave the release partial and record the per-target status;
do not rebuild a different artifact under an already published version.
