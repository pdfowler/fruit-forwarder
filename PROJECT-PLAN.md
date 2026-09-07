# Fruit Forwarder: implementation and open-source release plan

## 1. Purpose and planning boundary

**Goal:** make this a compelling open-source project for bridging iCloud Reminders and Calendars into Home Assistant and through secure, locally hosted MCP access.

**Chosen product name: Fruit Forwarder.** Package it for each ecosystem while retaining one shared native engine. The repository now contains the first implementation of the planned monorepo command layer and deterministic HA export; public repository creation and publication remain explicitly out of scope until the release gates pass.

The intended result is a useful, installable, maintainable product—not merely a repository whose unit tests pass. A household should be able to operate the bridge on an always-on Mac, use its existing Apple account, display selected data in HA dashboards, complete reminders, and connect a trusted MCP client without giving the bridge an Apple password or an HA administrator token.

This document remains the release plan and acceptance checklist. Its creation and the implementation work do not authorize publication, live deployments, account changes, new subscriptions, or destructive migrations.

The candidate source revision is recorded in `dist/release-manifest.json` and
must remain an immutable clean checkout for each release preparation. The
standalone repository now contains the HA reconfiguration and
packaging/monorepo implementation. Go, HA-runtime, metadata, deterministic
export and Git-attribution checks are automated; live HA, macOS permissions,
and remote publication state still require the acceptance evidence described
below. No Git remote or public release is implied by this repository.

## 2. Product definition

### Intended users and experiences

- **Household operator:** runs the Mac service, chooses exactly which reminder lists and calendars are exposed, pairs HA, and can diagnose failures without reading source code.
- **HA dashboard user:** sees current reminders and events, creates or edits reminders, completes/reopens items, and can distinguish confirmed state from pending changes or stale data.
- **MCP user:** connects locally, reads selected data, and optionally permits reminder edits. Read-only clients must have no mutation tools available.
- **Contributor:** clones the repository, runs meaningful tests without personal Apple data, understands the architecture and trust boundaries, and can submit a focused change.
- **Maintainer:** can reproduce a release, respond to security reports, support upgrades, and identify which platform combinations have actually been verified.

### Required initial-release capabilities

| ID | Requirement | Evidence needed for completion |
| --- | --- | --- |
| R1 | Reliable native macOS account access without collecting Apple credentials | Fresh-install permission and lifecycle tests using the distributed application |
| R2 | Scoped Reminders read/create/edit/complete/reopen through HA and MCP | End-to-end disposable-item tests, including denial outside scope |
| R3 | Scoped read-only calendars through HA and MCP | Timed, all-day, recurring, timezone and window-boundary fixture results |
| R4 | Independent reminder and calendar opt-in | Reminder-only, calendar-only, combined and permission-denied scenarios |
| R5 | Secure local MCP | Real-client stdio tests, read-only enforcement, bounded inputs and no unintended listener |
| R6 | Recoverable HA synchronization | Offline/restart/crash/acknowledgement/concurrent-edit tests and visible error states |
| R7 | Approachable installation and upgrades | A new user completes setup, upgrade and rollback from published instructions |
| R8 | Privacy-preserving operation | Credential, logs, process arguments, snapshots, backups and diagnostics review |
| R9 | Shareable open-source distribution | Chosen name/license, public repository, working release assets, CI and contributor documentation |
| R10 | Honest compatibility and support claims | Published matrix backed by recorded tests, known limitations and support policy |

### Initial scope boundaries

- Keep Reminders deletion and list management out of the initial release.
- Keep calendars read-only initially. Calendar creation, editing, invitations and recurring-series mutations require separate design and validation.
- Remain focused on EventKit-accessible reminders and calendars. Find My, iCloud3, Photos, Mail, files and general Apple account automation are separate projects or later proposals.
- Prefer the existing signed-in user account; do not require a new Family Sharing person or a “bot” Apple account.
- HomePods and iOS remain existing Apple control surfaces. Verify their changes propagate through Apple synchronization; this project does not need to implement a new HomePod interface.
- Native acquisition requires macOS. HA can run elsewhere; the acquisition process is not an HA Linux add-on.
- Local stdio MCP is the initial supported transport. Network MCP is a decision gate, not an implied feature already delivered.

## 3. Current implementation and evidence gaps

| Area | Present in the repository | What is not yet proven or complete |
| --- | --- | --- |
| Native adapter | Swift EventKit helper; separate Calendar/Reminders access; bounded calendar reads | Distributed identity, background TCC behavior, fresh permissions, recurring/all-day data correctness on real accounts |
| Reminders | Go allowlists, mutations, blank-title filtering, completed-item retention | Full recurring-reminder behavior, concurrency, list changes, large histories and all failure paths |
| Calendars | Native adapter, Go validation, MCP tools, HA snapshot transport and read-only entities | Real-account verification, complete HA lifecycle, timezone edge cases, scalable window fetching |
| HA integration | Setup, todo/calendar entities, persistent queue, scope filtering, save rollback, webhook boundary tests, token rotation and lifecycle coverage | Duplicate-entry handling against a live HA instance, stale/failed operation UX, full entity lifecycle acceptance |
| MCP | Stdio server, optional read-only policy, protocol/subprocess tests for reminders and calendars, validated macOS MCPB and candidate registry metadata | Actual client compatibility matrix and permissions under client launch |
| Durability | Local acknowledgement state and HA storage | EventKit-save/ack crash ambiguity, concurrent processes, old acknowledgements, restored backups |
| Installation | Source-build script, artifact-native macOS installer, ad-hoc helper signing, safe LaunchAgent rendering, versioned macOS tarball and MCPB build paths, atomic executable switching with rollback | Stable signed/notarized application, permission continuity, uninstall workflow, architecture matrix and installed lifecycle evidence |
| Quality | Go race tests; HA runtime tests against a pinned baseline; native boundary tests; CI definition | Hosted CI runs, full integration tests, current supported HA/macOS versions and sustained operation |
| Sharing | Fruit Forwarder identity, README, MIT license, security/contributor docs, issue/PR templates, deterministic HACS export, HACS/Hassfest workflow templates, validated macOS/MCP candidates and registry metadata | Final privacy/license review, remote repositories/releases and real security-reporting channel |

Existing tests are useful evidence of the cases they exercise. Synthetic helper tests do not prove iCloud access. Mocked storage tests do not prove physical durability. Swift compilation and ad-hoc signing do not prove TCC authorization or notarized distribution. Prior successful foreground runs do not prove launchd or reboot recovery.

## 4. Decisions to settle early

Record each decision with rationale, alternatives, consequences and a verification strategy.

| Decision | Recommended starting position | Impact |
| --- | --- | --- |
| Public name and repository slug | Fruit Forwarder selected; plan `fruit-forwarder` and `ha-fruit-forwarder` repositories | Module path, documentation, bundle identity and future discoverability |
| License | Review and confirm the existing MIT choice and all third-party notices | Distribution rights and contributor expectations |
| Supported versions | Select an explicit macOS/HA/Python/MCP-client matrix after compatibility checks | Test infrastructure and support obligations |
| Native process architecture | Prototype a stable signed app/helper and per-user service arrangement | Responsible application identity, TCC, upgrades, concurrency and recovery |
| MCP transport | Ship local stdio first; decide whether off-Mac clients are required | Network transport would require a separate authentication/authorization design |
| Permission model | Keep OS permissions separate from application allowlists; offer explicit read-only MCP setup | Accurate security promises and usable least-privilege configuration |
| Calendar coverage | Retain explicit cached bounds; validate the current 30-day-past/90-day-future default | HA calendar navigation, resource use and user expectations |
| Reminder conflict policy | Detect conflicts or apply narrowly scoped field updates instead of silently overwriting unrelated changes | Correctness when HA, MCP and Apple clients edit concurrently |
| Ambiguous creation recovery | Never blindly retry a creation whose outcome is unknown | Avoid duplicate reminders after crashes or timeouts |
| Monorepo orchestration | Keep one `fruit-forwarder` source repository and use Task plus native Go/Swift/Python toolchains; generate HA, macOS and MCP artifacts from one reviewed revision | Coordinated protocol changes, repeatable deployments and ecosystem-specific packages without duplicated implementations |
| Distribution | Separate Mac/MCP and HACS installation packages; two repositories and one shared native engine | Ecosystem-specific installation and releases without duplicating EventKit implementation |

Avoid a second implementation of CalDAV. Explain when users should keep their existing CalDAV calendars, when EventKit adds value, and how to avoid duplicate dashboard entities.

### 4.1. Fruit Forwarder packaging and discoverability

Use one canonical development monorepo and two public repository surfaces initially: `fruit-forwarder` owns all source, and `ha-fruit-forwarder` is its generated HACS distribution repository. MCP gets its own distribution metadata and installation instructions, but not a third source repository merely to obtain an `mcp-` name. Section 4.2 describes the build and release orchestration.

| Component | Planned repository | Public-facing name |
| --- | --- | --- |
| Mac application, EventKit engine, Go service and MCP server | `fruit-forwarder` | Fruit Forwarder — iCloud Reminders & Calendars for Home Assistant and MCP |
| HA custom integration | `ha-fruit-forwarder` | Fruit Forwarder — iCloud Reminders & Calendars |
| MCP package and registry listing, built from the main repository | No additional repository initially | Fruit Forwarder MCP — Apple Reminders & Calendar |

Repository names are planned, not reserved or published. Confirm the owner namespace and availability before creating them. Develop and review all components together in the main repository. Export the HA component, its audience-specific documentation and metadata into the HACS distribution repository at release time. Direct source changes belong in the monorepo, including HA fixes; the distribution repository must clearly explain where to contribute. Cross-link installation guides and define a protocol compatibility matrix between the separately released components. Do not maintain independently edited copies of source.

**Home Assistant distribution:** publish a focused HACS integration repository with `custom_components/<integration_domain>/` at its root, the required integration metadata, brand assets and release validation. HACS requires one integration per repository, but does not require a separate repository from all other source code; the dedicated HA repository is our usability and maintenance choice. The implemented `scripts/export-hacs.sh` produces that root layout, strips Python caches, includes official HACS and Hassfest workflow templates, and `scripts/package-ha.sh` creates a versioned review/release archive. Preserve or deliberately migrate the existing integration domain and entity identities; the new brand does not justify breaking installed configurations.

Treat HACS discovery as two stages:

1. Make the public repository installable as a custom HACS repository, with clear instructions and an installation link.
2. Submit it for inclusion in the default HACS catalog after HACS validation, Hassfest and the required GitHub release succeed. Record submission/acceptance status accurately; a repository name alone does not make the integration discoverable in that catalog.

**MCP distribution:** build the MCP server from the main repository and supply a separate listing, configuration guide and supported installation artifact. The implemented `scripts/package-mcpb.sh` produces a macOS-only MCPB containing the versioned Go server and EventKit helper, validates the upstream MCPB manifest, emits a SHA-256 sidecar, and can verify an optional maintainer-supplied MCPB signature. `scripts/render-mcp-server-json.sh` renders registry metadata only after an immutable release URL and hash are known. The official MCP Registry hosts metadata, not the executable. A candidate registry identifier is `io.github.pdfowler/fruit-forwarder`, conditional on publishing through that verified GitHub namespace. Registry naming, package naming and source repository naming are distinct; none requires duplicating the project into a third repository.

**Discoverability:** retain Fruit Forwarder as the brand while using literal terms in ecosystem descriptions, README introductions and GitHub topics: `icloud`, `apple-reminders`, `calendar`, `home-assistant`, `hacs`, `mcp`, `mcp-server`, `eventkit` and `macos`, where relevant. Use `ha-` and MCP labeling as audience signposts rather than treating prefixes as listing requirements. The main README should present separate Home Assistant and MCP setup paths.

**Installation contract:** the Mac application provides native Apple access. The HA integration connects to that application and cannot replace it. MCP users can install and use Fruit Forwarder without Home Assistant. A household using both should share the same native implementation and supported installation identity.

Packaging acceptance evidence:

- A clean HA installation can install/update the integration through the documented HACS route and pair with the Mac application.
- A clean MCP client can install/configure its supported distribution and access the intended tools without HA.
- Combined HA/MCP use does not require duplicate engines or conflicting service installations.
- Both repositories' releases link to the correct compatible counterpart and central documentation.
- HACS and MCP listing status is verifiable, and actual package installation works independently of the development checkout.

Publishing references checked for this decision: [HACS integration requirements](https://hacs.xyz/docs/publish/integration/), [HACS default-repository inclusion](https://hacs.xyz/docs/publish/include/), and [MCP Registry publishing guide](https://modelcontextprotocol.io/registry/quickstart). Recheck these requirements when preparing the release.

### 4.2. Polyglot monorepo and deployment orchestration

**Recommended setup:** a canonical `fruit-forwarder` monorepo, a lightweight Taskfile command layer, native Go/Swift/Python tooling, and GitHub Actions for CI and release orchestration. Build the Mac, MCP and HA targets from a reviewed source revision. Export the HACS-compatible repository rather than forcing contributors to coordinate changes across separately maintained codebases.

This refines the two-repository plan: two public discovery/install surfaces do not require two independent development histories. The main repository owns protocol changes, shared fixtures, component tests and release metadata. The HA distribution records the exact upstream commit used to generate each release.

#### Tooling options

| Option | Fit for this stack | Recommendation |
| --- | --- | --- |
| Task + native toolchains + GitHub Actions | Named tasks, dependency ordering, platform restrictions and source/output freshness checks around existing commands | Preferred initial setup; modest orchestration overhead for Go, Swift and Python |
| Nx + native commands + GitHub Actions | Explicit project/task graph and input/output caching; useful as the number of targets grows | Viable alternative if graph tooling and affected-target workflows justify the Node dependency and configuration maintenance |
| Make + native toolchains + GitHub Actions | Small dependency footprint and familiar build rules | Reasonable fallback if contributors prefer Make; keep command behavior and platform checks explicit |

Task is an orchestrator, not a dependency resolver or automatic semantic project graph. Declare cross-component dependencies explicitly. Nx likewise needs accurate native-project dependencies, toolchain inputs and output declarations to cache these builds safely. Choose one command layer, not overlapping Task and Nx workflows. Neither replaces the Swift compiler, Go module system or Python environment.

Start with one Go module for the existing service and MCP code. Introduce `go.work` only if independently versioned Go modules become necessary. Keep Swift's build arrangement compatible with the proven application/TCC architecture; adopt Swift Package Manager where it improves native tests and dependency management, not as a prerequisite to directory rearrangement. Keep Python dependencies pinned per test/release environment. Do not introduce a JavaScript package workspace unless a real launcher or UI target requires it.

#### Implemented baseline layout and planned extensions

```text
fruit-forwarder/
  Taskfile.yml                 # common local and CI entry points
  go.mod / go.sum              # one Go module initially
  cmd/                        # Mac service and MCP entry point
  internal/                   # shared Go logic and protocol models
  native/                     # EventKit helper/app, native tests, metadata
  integrations/homeassistant/
    custom_components/<domain>/
    tests/
    hacs.json                 # exported to distribution root
    README.md                 # HA-specific installation guide
  protocol/                   # versioned contract and shared synthetic fixtures
  packaging/
    macos/                    # bundle, signing and installer definitions
    mcp/                      # registry metadata and selected package route
    homeassistant/            # deterministic export rules
  release/                    # component versions and compatibility manifest
  scripts/                    # reusable build/validation/export helpers
  docs/
  docs/publishing.md
  .github/workflows/
  dist/                       # ignored generated artifacts
```

The current repository uses the existing top-level directories rather than performing a risky source move: `Taskfile.yml`, `release/`, `packaging/`, `scripts/`, `brand/`, the nested HA component, Go sources and the native helper now form the implemented baseline. The `native/`, `protocol/` and `integrations/homeassistant/` names above are future organization options, not a requirement to rename the current HA domain or native identity. Move files only when imports, tests, installers and CI can be updated together; preserve Git history where practical.

#### Target graph and developer commands

| Planned command | Dependency and output contract |
| --- | --- |
| `task check` | Formatting, static checks and protocol/metadata validation; no deployment |
| `task test` | Go, HA and native tests supported by the current host; clearly report unavailable host-specific suites |
| `task build:macos` | Go and native builds plus bundle assembly on macOS; unsigned/staged artifacts |
| `task package:macos` | Validated build → configured signing/notarization → versioned Mac distribution |
| `task package:mcp` | Same native engine/build revision → supported MCP install package and metadata |
| `task package:ha` | HA validation → deterministic HACS repository tree and optional release archive |
| `task release:check` | Verify all selected artifacts, versions, hashes, compatibility and source provenance |
| `task release:prepare` | Produce a complete candidate and publication manifest locally/CI without publishing |
| `task publish:*` | Explicit ecosystem publication of previously verified artifacts; no implicit household deployment |
| `task install:local` | Explicit installation on a specified Mac using a verified artifact |

The initial `check`, `package:ha`, `package:macos`, `package:mcp`, `release:check` and `release:prepare` interfaces are implemented in `Taskfile.yml`; the package targets now build a versioned macOS tarball, a validated MCPB and a deterministic HACS export, while signing, publication and live installation remain explicit maintainer actions. `task check` also runs `scripts/check-git-attribution.sh`, which verifies the requested author and committer identity across all refs before a candidate can pass. On macOS, `task release:check` now builds and verifies every local artifact before `task release:prepare` writes the manifest; on other hosts it reports the host-specific omission explicitly. Run independent tests/builds in parallel; express real dependencies in order. A shared protocol or fixture change must invalidate checks for Go, HA and MCP consumers. A shared native change must rebuild both Mac and MCP distributions. Changes limited to HA should not require an unrelated native rebuild unless compatibility or release policy requires it.

Use Go's build cache, appropriate dependency caches, and declared source/output checks for deterministic builds. Include OS, architecture, compiler/SDK version, dependency locks, entitlements and relevant build flags in cache decisions. Never treat cached output as evidence for live permissions, installed service health or signing/notarization status. Publication, token operations, signing side effects and live installation must not be skipped based on task-output caching. Keep credentials and household fixtures out of caches.

#### Release and HACS/MCP export pipeline

1. Select an immutable source revision and component-version manifest. Keep independently identifiable Mac/MCP and HA versions, with a declared protocol compatibility range; coordinate beta releases initially.
2. Run native builds on macOS and suitable Go/HA tests on their supported runners. Execute cross-component fixture/contract checks before packaging.
3. Assemble candidate artifacts once. Record source commit, toolchain versions, package versions, checksums and required counterpart versions. Promote these artifacts rather than rebuilding different bytes at each publication stage.
4. Export only allowlisted HA files into a temporary distribution tree with root-level `custom_components/<domain>/`, `hacs.json`, required metadata/brand assets, license notices, HA README, and HACS/Hassfest workflows. Validate the exported tree itself with HACS/Hassfest and check installation from it.
5. On macOS, build the versioned macOS tarball and MCPB with the official `@anthropic-ai/mcpb` CLI, inspect their contents, verify the bundled binary/helper, calculate checksums, and render the candidate MCP Registry `server.json` against the eventual GitHub release URL.
6. Record the upstream source commit and dirty-tree state in the release manifest. Make generation reproducible; detect unexpected distribution-repository edits rather than overwriting them silently. Route fixes back through the canonical repository.
7. Publish through a narrowly scoped automation identity only after the release gates pass and publication is authorized. Prefer ordinary auditable commits/releases in the distribution repo over force-pushing its history. Preserve the requested `pfowler@icloud.com` commit attribution in repository export configuration, and distinguish commit authorship from release automation provenance. The guarded MCP workflow requires a maintainer-reviewed official publisher URL and SHA-256, then uses GitHub OIDC only after downloading and validating the immutable release asset; it does not rebuild or publish from a branch.
8. Publish Mac artifacts, the chosen MCP package/registry metadata, and the HA distribution/release with compatibility links. Cross-service publication is not atomic: record per-target status and support retry without replacing immutable released versions.
9. If a target fails, hold or mark the release partial rather than advertising full availability. Maintain a manifest of the last known compatible set and documented rollback behavior.

Use separate read/test permissions for pull requests and restricted credentials for release/export workflows. Never expose publication secrets to untrusted fork code. Test release preparation without credentials; keep human publication decisions distinct from routine verification. Application deployment to the household remains separate from artifact publication.

#### Monorepo acceptance criteria

- One checkout supports building/testing all components, with explicit macOS-only requirements.
- One protocol change can update and test all consumers in one pull request.
- Clean-checkout packaging produces the expected targets with no personal config or secrets.
- Exported HA releases are installable and traceable to a specific monorepo commit.
- HA fixes have one authoritative source; distribution drift is detected.
- Mac/MCP packages use the same shared implementation and do not install conflicting native services.
- Partial publication can be resumed safely, and the documented compatible release set can be restored.
- CI invokes the same provenance and build/test entry points contributors use where
  the runner supports them; live validation gates remain separate.

Tooling references: [Task guide](https://taskfile.dev/docs/guide), [Task schema](https://taskfile.dev/docs/reference/schema), [Nx project configuration](https://nx.dev/docs/reference/project-configuration), and [Nx caching inputs](https://nx.dev/docs/reference/inputs). The initial Task recommendation is a project-fit decision, not a claim that the ecosystems require it.

## 5. Workstream A — native macOS reliability

**Priority: release blocker. Dependencies: native identity and supported-version decisions.**

1. Reproduce the recorded background EventKit timeout on a controlled installation. Compare Terminal, MCP-client, LaunchAgent and app launches with identical helper versions and account state.
2. Capture responsible process, code-signing identity, authorization status, helper lifetime and launch environment without exposing reminder contents.
3. The new per-user LaunchAgent invokes the bridge directly in its `gui/<uid>` domain; the unverified `launchctl asuser` wrapper is no longer part of the Fruit Forwarder architecture. Validate the direct arrangement under fresh permissions and lifecycle conditions before claiming background support; keep legacy migration separate.
4. Evaluate a persistent native service versus short-lived helper invocations. Base the choice on TCC stability, request cancellation, synchronization cost, memory and serialization—not aesthetics.
5. Use the chosen stable maintainer identity `com.pdfowler.fruitforwarder` (and
   `com.pdfowler.fruitforwarder.eventkit` for the helper). Align the Go
   executable, helper, application bundle, entitlements, launch service and
   Keychain access strategy; retain explicit legacy-config compatibility for
   the prototype `com.example.icloud-reminders-bridge` service.
6. Make Calendar and Reminders opt-in independent. Invalid or empty-scope requests should not unexpectedly trigger account permission prompts.
7. Provide bounded, actionable errors for denied access, unavailable services, locked Keychain, malformed helper output and timeouts.
8. Add a diagnostic command that reports versions, process health and access status without requesting additional permissions by default.
9. Verify restart, login, logout, lock/unlock, sleep/wake, service crash and full Mac reboot. Document the requirement for a logged-in user and any locked-session limitations.

**Done when:** the installed artifact, not just a terminal build, survives the supported lifecycle matrix and resumes access without manual repair in supported conditions. Any unsupported condition is explicitly documented and reported by health status.

## 6. Workstream B — synchronization correctness and durability

**Priority: release blocker. Dependencies: stable native execution; protocol design.**

- Command states and recovery semantics are defined in [docs/command-lifecycle.md](docs/command-lifecycle.md): HA displays queued/withheld work, the Mac journals in-flight mutations, transport failures retry with backoff, and uncertain EventKit outcomes require explicit operator resolution.
- Address the gap between saving a reminder in EventKit and persisting its acknowledgement. The Mac now journals an in-flight command before invoking EventKit, stops automatic replay after an ambiguous failure, and exposes explicit `recover --resolution applied|retry` choices. Continue evaluating a stronger correlation mechanism; do not claim exactly-once behavior without evidence across that boundary.
- The 1,000-command acknowledgement cap is now paired with a persisted HA queue epoch. The Mac fences a changed or later-omitted epoch and provides an explicit `reset-queue-epoch` recovery command; continue testing restored backups and operator messaging before claiming the fence covers every backup topology.
- Prevent concurrent `serve`, `sync-once` and replacement processes from racing on command execution or the same state file. Coordinate MCP mutations where needed.
- Test atomic replacement separately from power-loss durability; evaluate file and directory synchronization, permissions, failed writes and corrupt state recovery.
- Define ordering and conflict handling for two edits to one reminder, recurring reminders, externally deleted items and list moves.
- Separate calendar read failure from reminder publication and command processing where feasible. A revoked Calendar permission must have a documented effect on otherwise healthy Reminders service.
- Preserve good snapshots during transient failures. Publish per-capability health and last successful sync instead of making stale data appear current.
- The daemon now uses bounded exponential retry backoff with jitter after Home Assistant failures and returns to the configured polling interval after success. Continue evaluating EventKit change notifications to accelerate updates while keeping polling as a recovery mechanism.
- Bound queue size, command-field bytes, response payloads, event counts and helper output. The HA queue now fails closed at 1,000 commands, validates queued mutation fields, and rejects an oversized command response; continue reconciling the one-MiB payload limit with 10,000-item snapshots and realistic notes.
- Protocol version 1 now advertises additive capabilities before relying on optional fields across mixed HA/Mac versions; calendar clients reject peers that do not advertise `calendars`, and queue clients fence a changed `queue_epoch`, while legacy reminder-only responses remain compatible until an epoch is established. Continue documenting upgrade order and rejection behavior for future protocol versions.

**Done when:** failure-injection tests cover every mutation/acknowledgement boundary; ambiguous operations are visible and safe; no test loses a confirmed user edit or silently applies an edit twice.

## 7. Workstream C — reminder semantics

**Priority: core product. Dependencies: B for mutation guarantees.**

- Verify create/edit/complete/reopen from HA and MCP against disposable real reminders, then observe them in Apple Reminders on Mac and iOS.
- Test due dates with and without a time, notes, Unicode, empty titles, shared/read-only lists and account synchronization delays.
- Verify recurrence: completing an occurrence, generation of the next occurrence, reopening and identifier changes.
- Test renamed, moved, deleted and temporarily unavailable lists. Decide where stable ID should prevail over display-name changes; avoid outages caused solely by a harmless rename.
- Prevent omitted update fields from unintentionally clearing notes or due dates. Document full replacement versus patch semantics for each tool/service.
- Keep completion history bounded in published snapshots. Verify zero retention, invalid/missing completion dates, recently completed items and reopened historical reminders.
- Avoid loading every historical completed reminder on every poll if EventKit predicates permit narrower reads. Benchmark large real-world histories with synthetic equivalents in CI.
- Define expected handling of features not represented by the wire model, including subtasks, attachments, tags, priority and location-based reminders. Preserve unsupported data when editing supported fields.

**Done when:** a behavior matrix documents supported fields, unsupported fields and preservation rules; round-trip tests prove those rules.

## 8. Workstream D — calendar correctness and dashboard usefulness

**Priority: core product. Dependencies: A and bounded protocol behavior.**

- Validate ordinary timed events, all-day events, multi-day events, recurring occurrences, exceptions, moved occurrences and cancelled/deleted events.
- Test daylight-saving transitions, calendars with explicit time zones, floating/local times, HA and Mac in different zones, and events spanning query boundaries.
- Check all-day exclusive end dates and source-calendar semantics. A date-only event must not shift a day because of UTC conversion.
- Review occurrence identifiers: the current ID includes start time, so moving an occurrence changes its ID. Define the implications for HA and MCP consumers.
- Ensure sorting and active/next-event selection work with overlaps, all-day events and time passing between syncs.
- Verify calendar triggers as well as visual display. A populated entity alone does not prove HA automation compatibility.
- Validate window coverage, count limits, truncation policy and calendar queries outside coverage. Decide whether configurable bounds or on-demand fetching is needed for ordinary HA navigation.
- Handle shared and subscribed calendars and account source names without suggesting that all EventKit data necessarily originates in iCloud.
- Keep calendar mutations unsupported until separately designed; advertise no unsupported write capabilities.

**Done when:** real and synthetic event fixtures produce matching HA/MCP results for the same interval, with explicit coverage limits and no unexplained missing or shifted events.

## 9. Workstream E — Home Assistant integration lifecycle and UX

**Priority: release blocker. Dependencies: B–D.**

1. Finish and test the existing uncommitted reconfiguration flow. Preserve entry IDs, entity identities, pairing and unrelated settings when changing allowlists.
2. Fix duplicate integration identity: distinguish a bridge identity from a token that can rotate. Handle duplicate bridge IDs and duplicate tokens predictably.
3. Implement deliberate pairing-token rotation and recovery without requiring users to delete their integration. Reject mismatched or invalid bridge identifiers and empty required names.
4. Test first setup, failed setup, reload, unload, restart, reconfigure and removal using HA's real config-entry and entity lifecycle.
5. Test HTTP webhook registration, locality restrictions, allowed methods, unknown tokens, malformed/oversized/chunked payloads and response handling.
6. Define visibility and cleanup for revoked, renamed or missing lists/calendars. Mark stale sources unavailable according to documented freshness thresholds.
7. Make pending, failed and uncertain commands visible with actionable recovery. Bound or coalesce repeated queued operations while a Mac is offline.
8. Add useful diagnostics and repair messages: permission denied, bridge offline, token mismatch, unsupported protocol, excessive payload and window unavailable.
9. Test persistence with actual HA storage and failure injection, not only a mocked storage object.
10. Provide reminder and calendar dashboard examples using standard cards, with optional advanced examples kept separate.
11. Keep configuration labels, error messages, translations, integration name, device naming and documentation consistent with the final product name.

**Done when:** a user can configure, use, reconfigure, rotate credentials, upgrade and recover the integration without editing HA internal storage or rebuilding dashboards.

## 10. Workstream F — secure local MCP

**Priority: release blocker for the stated MCP goal. Dependencies: A–D and security review.**

- Publish exact configuration examples for a small verified set of MCP clients. Use absolute paths where required; verify app-launched permissions rather than assuming Terminal permissions transfer.
- Verify reminder and calendar tools through the built process and an installed client, including calendar-only and combined configurations.
- Review tool schemas, descriptions, annotations and errors. Correct date-format descriptions and make mutation semantics explicit.
- Keep stdout protocol-only. Test initialization, cancellation, malformed input, helper hangs, shutdown and client restarts.
- Verify read-only policy at tool discovery and direct invocation. Confirm configuration changes take effect on documented restart/reload boundaries.
- Offer separate configs for clients with different scopes. Explain that this is application policy, not isolation from another process with the same OS user privileges.
- Treat titles, descriptions, locations and notes as untrusted data. Tools must never interpret reminder text as instructions or execute it.
- Consider pagination and response-size limits for large lists and calendars; avoid pushing an entire history into a model context unnecessarily.

### If off-Mac MCP access is required

Design a separate milestone for authenticated network transport: supported MCP authorization flow, TLS termination, client identity, per-client scopes, revocation, origin/DNS-rebinding defenses, rate limits, audit metadata, cancellation and secret storage. Decide whether a vetted gateway can meet the need before implementing a server. Do not expose the current stdio service over an unauthenticated generic proxy. This milestone becomes part of release completion only if the product decision explicitly includes remote clients.

**Done when:** verified clients can access exactly the intended tools and data, unavailable services fail cleanly, and the documented local trust model matches actual behavior.

## 11. Workstream G — security and privacy review

**Priority: release blocker. Dependencies: chosen runtime and distribution design.**

- Write a threat model covering the Mac user, other local users, MCP clients, LAN clients, HA, reverse proxies, backups, and malicious content returned from reminders/calendars.
- Audit OS permissions versus application allowlists at every boundary. Include restored state, reconfiguration, helper responses and command execution.
- Review Keychain integration. Pairing now supplies the token on the `security` command's standard input rather than as a process argument; still verify Keychain ACLs, prompts, clipboard exposure and app-launched behavior before claiming comprehensive token protection.
- Review clipboard pairing: token lifetime, accidental logging, clipboard-manager exposure, recovery and optional clearing behavior. Never call a persistent bearer token a single-use code.
- Ensure tokens do not appear in errors, trace logs, reverse-proxy access logs, crash reports, process listings or support bundles. Cover Calendar contents as well as Reminders.
- Harden config, binary and state ownership/permissions, including symlinks and parent-directory trust. Verify installation cannot accidentally broaden access.
- Add body and output bounds at every parser, not only at the HA HTTP handler. Native helper stdout/stderr are now bounded before JSON parsing; fuzz narrow protocol validators where valuable.
- Validate TLS, redirect rejection and the effective `local_only` boundary under supported reverse-proxy configurations.
- Define privacy for HA snapshots, queued commands, recorder data, Mac state, logs and backups. Filtering a snapshot is not historical deletion.
- Home Assistant now exposes opt-in diagnostics containing only redacted configuration flags, counts and synchronization timestamps; it never includes credentials, identifiers or household contents. Keep the preview and installed-client review in the acceptance matrix.
- Audit dependency licenses, vulnerabilities and update policy; harden CI permissions and release credentials.

**Done when:** each threat has a mitigation or explicit documented limitation, testable controls have tests, and public security claims have evidence.

## 12. Workstream H — installation, upgrades and migration

**Priority: release blocker. Dependencies: A and project identity.**

- Design a guided first-run experience: prerequisites, separate permission grants, discovery, scope selection, HA pairing, MCP configuration, connectivity check and a clear success state.
- Add `status`/`doctor`, version reporting and actionable exit codes. Keep config validation distinct from live access checks; the CLI now reports helper, Keychain, state, and unresolved-command health without exposing reminder contents, with focused diagnostics tests.
- Discovery honors the configured `eventkit_helper_path` (with an explicit CLI override and a default only when no config exists); keep the regression test and include the configured path in installed-artifact acceptance.
- Render LaunchAgent configuration safely for spaces and XML/shell-special characters in user paths. Validate inputs and environment assumptions; the renderer now has a regression test covering escaped paths and stable service identity.
- Make installation transactional: stage builds, validate binaries/signatures and configuration/path trust before switching, preserve a previous version, switch atomically, verify service health and support rollback. The packaged installer now rejects symlinked install/config paths and invalid configuration before the executable pair is replaced; rollback rejects symlinked targets.
- Preserve config, Keychain identity, acknowledgement state and HA entity identity across supported upgrades. Test schema and protocol migrations in both supported upgrade orders.
- Support a documented stop/uninstall process that preserves user data by default and separates optional credential/state removal from software removal.
- Build and test supported architectures. Publish checksums and provenance; decide on Developer ID signing/notarization and acquire credentials only through an explicit maintainer decision.
- Implement the monorepo and generated HACS distribution in sections 4.1–4.2; validate current requirements and the native MCP package route before promising installability.
- Migrate the original home-ctrl deployment only after a verified standalone candidate exists. Inventory exact paths and identities, back up narrowly, stop the old writer, switch one deployment, verify and retain a rollback path. The packaged installer now has an explicit `--migrate-home-ctrl` preparation/activation path for the known legacy service and config locations; the actual Dean cutover remains a recorded external acceptance step.

**Done when:** an independent user installs from release artifacts without source-level troubleshooting, and upgrade/rollback preserves their pairing and entities.

## 13. Workstream I — open-source presentation and maintenance

**Priority: required for sharing. Dependencies: settled identity and honest feature matrix.**

- Review the entire initial Git history for personal paths, real data and secrets before publishing; scanning only the latest tree is insufficient.
- Confirm all commits use `pfowler@icloud.com` for author and committer and verify future release automation preserves the intended attribution.
- Confirm copyright ownership and third-party attribution; remove obsolete copied artifacts only after provenance review.
- Create the planned `fruit-forwarder` and `ha-fruit-forwarder` public repositories when publication is authorized. Configure remotes, issue routing, private vulnerability reporting and practical branch/release protections. Publish MCP metadata from the main repository using a verified namespace and supported package artifact.
- Rewrite the README around the user benefit: Apple-managed account access, scoped household data, HA dashboards, local MCP and clear limitations. Explain the distinction from CalDAV and location-oriented iCloud integrations.
- Include a short architecture overview, feature/compatibility matrix, setup walkthrough, privacy model, troubleshooting, upgrade/rollback guide and verified examples.
- Use synthetic screenshots or explicitly approved/redacted household screenshots for HA cards and MCP examples.
- Add issue templates, a reproducible bug-report checklist, a PR template, contributor test commands and architectural decision records.
- Define versioning, component compatibility, changelog conventions and release notes. Eliminate hard-coded divergent versions across Go, Swift metadata and HA manifests.
- Establish dependency-update cadence, support scope, security-report handling and a process for reproducing macOS-only failures.
- Publish a pre-release for opt-in testers before claiming stable support. Collect actionable setup failures, permission failures and data-correctness reports.

**Done when:** the public repository and its release provide enough context for a stranger to install, evaluate, contribute and report a problem without this conversation.

## 14. Verification strategy

### Automated verification

| Layer | Required coverage |
| --- | --- |
| Go configuration/store | Both scopes, invalid dates, retention, executable trust, response limits, unsupported fields |
| Go sync/durability | HTTP failures, retries, acknowledgements, eviction, concurrent processes, corrupt state, crash boundaries |
| MCP protocol/process | Schemas, catalog, read-only denial, both data types, transport lifecycle, cancellation and output hygiene |
| Swift/native | Request validation without permission prompts; deterministic mapping tests; compiled/signed artifact checks |
| HA runtime | Validation, restoration, revocation, queue transactions, calendars and stale data |
| HA integration | HTTP webhook, config/reconfiguration flows, entity platforms, service calls, calendar triggers, unload/reload |
| Packaging | Clean installation, path quoting, upgrade rollback, manifest/version consistency, signatures and checksum verification |
| Security | Secret-redaction cases, scope violations, hostile payloads, resource limits and dependency review |

Current commands are useful starting points, not the final acceptance suite:

```sh
go test -race ./...
go vet ./...
.venv/bin/python -m pytest -q
bash -n scripts/install-macos.sh
python3 scripts/test-native-calendar.py
git diff --check
```

The native test command requires the helper to have been built. Pin and document the test toolchain. Add at least one current supported HA version alongside the existing historical baseline after checking compatibility. Hosted CI must exercise the actual release paths and artifacts.

### Real-environment acceptance matrix

Use dedicated disposable lists/calendars and fixtures. Never run destructive or broad mutation tests against ordinary household data.

| Dimension | Cases |
| --- | --- |
| Accounts | Personal iCloud; shared resources where available; permission denied/revoked |
| Host lifecycle | Fresh grant, locked/unlocked, logout/login, sleep/wake, reboot, helper/service crash |
| Consumers | HA only, MCP only, both, two clients editing one item |
| Network | HA unavailable, TLS failure, reconnect, proxy configuration, delayed Apple sync |
| Data | Small/large lists, old completion history, all-day/multi-day/recurring events, timezone changes |
| Deployment | Fresh install, upgrade, rollback, old/new component combinations |
| UX | HomePod/iOS edits observed in HA, HA completions observed in Apple Reminders, visible pending/failure states |

Record OS/architecture, HA version, client version, artifact hash, scenario, expected result, actual result and redacted evidence. Run a proposed minimum seven-day household soak after the lifecycle tests, including planned outages and a reboot. Record actual recovery times and resource use; set final service targets from those measurements rather than inventing performance claims.

## 15. Milestones and dependency order

| Milestone | Deliverable | Exit gate | Main dependencies |
| --- | --- | --- | --- |
| M0 — agree product contract | Confirm remaining scope, transport and support decisions; Fruit Forwarder name, canonical monorepo and two public repository surfaces recorded | Decisions recorded; incomplete changes identified | None |
| M1 — prove the Mac foundation | Stable installation identity and lifecycle prototype | Background/reboot matrix passes or architecture is revised | M0 |
| M2 — make writes recoverable | Durable queue/ack protocol and conflict semantics | Failure injection proves no silent loss or unsafe retries | M1 |
| M3 — complete user-facing behavior | Reminders/calendars, HA lifecycle/reconfiguration, MCP clients | R2–R6 acceptance cases pass | M1–M2 |
| M4 — package a private beta | Monorepo target orchestration, deterministic HACS export, guided setup, HA and Mac/MCP packages, diagnostic tools, security review | New-user install/upgrade/rollback succeeds for both ecosystem paths; artifacts trace to one source revision | M1–M3 |
| M5 — household and external beta | Recorded real-data fixtures and soak results | No unresolved critical correctness/privacy/recovery defects | M4 |
| M6 — public release | Authorized publication of both repositories, HACS custom-repository installation/default-catalog submission, MCP registry/package publication, artifacts, docs and support | R1–R10 evidence reviewed and linked; listing status and installation paths verified | M5 |

Security review and documentation proceed throughout. Mock-based tests and contributor documentation can proceed while native lifecycle work is being investigated, but they do not replace the M1 gate. Do not continue polishing minor tests indefinitely while leaving the native execution model unproven.

### Recommended next implementation sequence

1. Review this plan and settle remaining M0 decisions, especially required MCP client locations, verified publisher namespace and distribution details; retain the chosen Fruit Forwarder name and two-repository approach.
2. Reproduce and resolve the background EventKit failure using the candidate installation architecture.
3. Design command ambiguity, process ownership and acknowledgement recovery before adding more mutations.
4. Complete/test the paused HA reconfiguration work, duplicate identity behavior and credential rotation.
5. Run full HA lifecycle and real reminder/calendar fixture verification; fix semantic failures.
6. Build guided installation, diagnostics and reversible upgrades around the proven process model.
7. Complete security/provenance review, then run the beta/soak and publish the verified result when authorized.

## 16. Risks and responses

| Risk | Response |
| --- | --- |
| TCC remains unreliable under the current process arrangement | Change the native ownership/lifetime architecture before packaging it |
| Crash after EventKit creation causes duplicates | Journal and explicitly handle uncertain outcomes; do not blind-retry |
| Calendar failures block healthy reminder edits | Separate capability health and synchronization failure paths |
| Mixed versions reject new calendar payloads | Capability negotiation, upgrade-order documentation and compatibility tests |
| Credential handling leaks via process arguments or proxy paths | Safer Keychain integration and end-to-end redaction tests |
| Tests become a substitute for live evidence | Require installed-artifact and independent-user release gates |
| Broad “iCloud” positioning implies unsupported services | Publish an explicit supported-service matrix and non-goals |
| Permission or identity changes break existing users | Stable signing identity and tested migration/rollback |
| Public distribution creates unsustainable support demands | Narrow verified support matrix, issue templates and transparent limitations |

## 17. Completion audit

Before declaring the project complete, attach evidence to every R1–R10 requirement and each milestone exit gate. Classify each as proven, failed, missing evidence or explicitly deferred by a product decision. “Implemented,” “builds,” and “passed unit tests” are not substitutes for a broader requirement's acceptance evidence.

The initial release is ready only when its promised user journeys work on supported environments, operational failures are recoverable, privacy/security claims are accurate, and a public distribution plus maintenance path exists. Open optional enhancements may remain; unresolved failures in required installation, mutation correctness, account access or supported HA/MCP behavior may not.

This plan intentionally keeps the full objective intact. The next step after planning is an explicit resumption of implementation, not automatic execution of the tasks in this document.
