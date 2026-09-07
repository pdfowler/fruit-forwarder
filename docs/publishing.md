# Maintainer publishing checklist

This repository prepares release assets but does not publish them during local
builds. The optional `Publish confirmed release` workflow is deliberately
manual, requires typing `PUBLISH`, and must use a protected GitHub `release`
environment with reviewer approval.

Before running it:

1. Confirm the source revision is clean and all commits are authored and
   committed as `pfowler@icloud.com`. `task check` and
   `scripts/check-git-attribution.sh` perform this check across all refs; the
   publication workflows repeat it on the tagged checkout.
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
   The manifest also records the compatibility manifest, host toolchain
   versions, and MCPB package-tool version used for that candidate.
4. Complete the real Mac/HA/MCP acceptance matrix in `PROJECT-PLAN.md`,
   using the redacted ledger in [acceptance-evidence.md](acceptance-evidence.md),
   including disposable reminders/calendars and rollback evidence.
5. Confirm the GitHub repository namespace, release signing/notarization policy,
   HACS repository destination, and MCP Registry publisher namespace.
6. Run the workflow only after the protected environment reviewer approves the
   exact source revision and version.

After the GitHub release exists:

- Run the guarded `Publish HACS distribution` workflow from the matching version
  tag after creating the protected `hacs-release` environment and
  `HACS_REPO_TOKEN` secret. Type `PUBLISH_HACS` and provide the destination
  repository explicitly. The workflow replaces the destination only with the
  deterministic export and records the exact Fruit Forwarder source revision;
  it does not publish from a branch or a dirty checkout.
- Submit the HACS repository first as a custom repository, then request default
  catalog inclusion once the required release and validation checks are live.
- Run the guarded `Publish MCP Registry metadata` workflow from the matching
  version tag after configuring the protected `mcp-release` environment. Type
  `PUBLISH_MCP`; it downloads the already-created release assets, rechecks the
  MCPB checksum, release URL and `io.github.pdfowler/fruit-forwarder` namespace,
  then authenticates with GitHub OIDC before publishing `server.json`. It does
  not build or publish a package from a branch or working tree. Verify a clean
  MCP client can fetch and install the exact artifact.

If one target fails, leave the release partial and record the per-target status;
do not rebuild a different artifact under an already published version.
