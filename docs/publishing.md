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
3. Run `task release:prepare` on macOS and inspect the versioned release notes,
   release manifest, checksums, HACS archive, macOS tarball, and MCPB contents. The HA and macOS
   package scripts also re-open their finished archives and verify their
   versioned roots, required files, checksums, and absence of unexpected links
   or interpreter caches. The macOS archive is assembled with normalized
   metadata and is byte-for-byte reproducible when its staged binaries are
   unchanged; the unsigned MCPB ZIP receives the same timestamp/order
   normalization before any optional signature is applied.
   The manifest also records the compatibility manifest, host toolchain
   versions, and MCPB package-tool version used for that candidate. The
   manifest verifier rechecks the recorded source revision, artifact paths,
   sizes, and hashes before publication.
   The default local candidate uses ad-hoc code signing for boundary tests;
   a publishable macOS release must set the protected
   `FRUIT_FORWARDER_SIGN_IDENTITY` Actions variable to the maintainer-controlled
   Developer ID identity. The workflow imports the corresponding base64-encoded
   `.p12` certificate into an ephemeral keychain using the protected secrets
   `FRUIT_FORWARDER_DEVELOPER_ID_CERTIFICATE_BASE64`,
   `FRUIT_FORWARDER_DEVELOPER_ID_CERTIFICATE_PASSWORD`, and
   `FRUIT_FORWARDER_SIGNING_KEYCHAIN_PASSWORD`, then verifies the identity
   before building. Complete the separate notarization/release review before
   publication; the current tarball/MCPB workflow does not claim notarization.
4. Complete the real Mac/HA/MCP acceptance matrix in `PROJECT-PLAN.md`,
   using the redacted ledger in [acceptance-evidence.md](acceptance-evidence.md),
   including disposable reminders/calendars and rollback evidence.
5. Confirm the GitHub repository namespace, release signing/notarization policy,
   HACS repository destination, and MCP Registry publisher namespace.
6. Run the workflow only after the protected environment reviewer approves the
   exact source revision and version. The guarded public-release workflow also
   fails closed unless the macOS job has a maintainer-selected repository or
   organization `FRUIT_FORWARDER_SIGN_IDENTITY` Actions variable and the three
   protected certificate/keychain secrets described above. The ordinary release
   preparation workflow remains available for ad-hoc local candidates.

The same guarded workflows can be dispatched from a clean, checked-out tag
through the Task wrappers (which still require `gh` authentication and the
protected GitHub environments):

```sh
task publish:release TAG=v0.1.0 CONFIRM=PUBLISH NOTARIZATION_REVIEW=APPROVED
task publish:hacs TAG=v0.1.0 CONFIRM=PUBLISH_HACS HACS_REPOSITORY=pdfowler/ha-fruit-forwarder
task publish:mcp TAG=v0.1.0 CONFIRM=PUBLISH_MCP \
  PUBLISHER_URL=REPLACE_WITH_REVIEWED_OFFICIAL_URL \
  PUBLISHER_SHA256=REPLACE_WITH_REVIEWED_SHA256
```

Each wrapper rejects a dirty checkout, a missing tag, a tag that does not point
at `HEAD`, an incorrect confirmation string, or malformed publication inputs;
replace both MCP publisher placeholders with the exact reviewed URL and hash;
the wrappers do not hold or transmit release credentials.

After the GitHub release exists:

- Run the guarded `Publish HACS distribution` workflow from the matching version
  tag after creating the protected `hacs-release` environment and
  `HACS_REPO_TOKEN` secret. Type `PUBLISH_HACS` and provide the destination
  repository explicitly. The workflow replaces the destination only with the
  deterministic export and records the exact Fruit Forwarder source revision.
  Before replacement it verifies the destination's recorded export-tree digest
  (or requires an empty repository), so a modified or unrelated tree fails
  closed instead of being silently overwritten. It does not publish from a
  branch or a dirty checkout.
- Submit the HACS repository first as a custom repository, then request default
  catalog inclusion only after the public repository has its description,
  topics, enabled issues, a full release, passing HACS and Hassfest actions,
  and the current Home Assistant brand-policy checks. The component-local brand
  asset is already included; submit an external Brands change only if the
  target HACS/HA release policy still requires it.
- Run the guarded `Publish MCP Registry metadata` workflow from the matching
  version tag after configuring the protected `mcp-release` environment. Type
  `PUBLISH_MCP`, and provide the exact official `mcp-publisher` Linux amd64
  tarball URL and SHA-256 from the protected release review. It downloads the
  already-created release assets, rechecks the MCPB checksum, release URL and
  `io.github.pdfowler/fruit-forwarder` namespace, verifies the publisher
  tarball, then authenticates with GitHub OIDC before publishing `server.json`.
  It does not build or publish a package from a branch or working tree. Verify a
  clean MCP client can fetch and install the exact artifact.

The current official MCP Registry package guidance supports `registryType:
"mcpb"` entries that point to a versioned GitHub or GitLab release asset,
contain `mcp` in the package URL, and carry its `fileSha256`; the registry
stores metadata rather than the MCPB bytes. The generated `server.json` follows
that package shape. HACS requires a public GitHub repository with one
integration under `custom_components/`, the required manifest keys, brand
  assets, passing HACS and Hassfest actions, and at least one full release before
  default-catalog submission; repository description/topics/issues and the
  Home Assistant brand policy are also publication checks. The generated
  integration includes a component-local `brand/icon.png` for current HA's
  local brand proxy and a root `brand/icon.png` for HACS/legacy validation.
  Recheck the upstream
requirements during the final publication review because both ecosystems are
independently maintained. See the [MCPB package guidance](https://modelcontextprotocol.io/registry/package-types), [MCP Registry authentication guidance](https://modelcontextprotocol.io/registry/authentication), [HACS integration requirements](https://hacs.xyz/docs/publish/integration/), and [HACS default-repository requirements](https://hacs.xyz/docs/publish/include/) during that review.

The confirmed GitHub release workflow reconstructs and verifies the unified
manifest after the HA and macOS artifacts have been merged. A manifest from a
single preparation job is not treated as sufficient provenance for the whole
release.

If one target fails, leave the release partial and record the per-target status;
do not rebuild a different artifact under an already published version.

## Current ecosystem gate review (2026-09-08)

The upstream requirements were rechecked before treating the candidate
publication workflow as ready:

- HACS integration repositories must contain one integration under
  `custom_components/<domain>/` with the required manifest fields. Home
  Assistant Brands is required for integration UI conformance. Default-catalog
  inclusion additionally requires a public GitHub repository, passing HACS and
  Hassfest actions, and a full GitHub release. HACS may use the default branch
  when a repository has no releases, but this project will publish releases so
  users can select and upgrade known versions.
- The MCP Registry currently accepts `registryType: "mcpb"` entries that point
  at MCPB assets hosted in GitHub or GitLab releases. The package URL must
  contain `mcp`, and metadata must include the artifact `fileSha256`. GitHub
  authentication uses an `io.github.<owner>/...` server namespace; the guarded
  workflow uses GitHub OIDC after downloading and verifying the immutable
  release asset.
- The MCP Registry remains in preview, so the release review must recheck its
  schema, supported package types, and publisher workflow immediately before
  the first publication.

References: [HACS integration requirements](https://hacs.xyz/docs/publish/integration/),
[HACS default repositories](https://hacs.xyz/docs/publish/include/),
[MCP Registry package types](https://modelcontextprotocol.io/registry/package-types),
[MCP Registry authentication](https://modelcontextprotocol.io/registry/authentication),
and [MCP Registry GitHub Actions](https://modelcontextprotocol.io/registry/github-actions).

## Publication record for 0.1.0

The maintainer-approved `v0.1.0` publication is complete for the unsigned,
ad-hoc-tested candidate:

- Source release: <https://github.com/pdfowler/fruit-forwarder/releases/tag/v0.1.0>
- Home Assistant distribution: <https://github.com/pdfowler/ha-fruit-forwarder/releases/tag/v0.1.0>
- HACS default-catalog PR: <https://github.com/hacs/default/pull/10771> (open;
  required checks pass; awaiting normal HACS review/merge)
- MCP Registry: `io.github.pdfowler/fruit-forwarder` version `0.1.0`, active/latest
- MCP publication workflow: GitHub Actions run `34249548695`

This record does not claim Developer ID signing, notarization, a clean
background LaunchAgent/EventKit lifecycle, or inclusion in HACS's default
catalog. Those remain separate acceptance or maintainer-review gates for a
future release.
