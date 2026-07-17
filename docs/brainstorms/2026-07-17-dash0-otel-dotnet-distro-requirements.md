---
date: 2026-07-17
topic: dash0-otel-dotnet-distro
---

# Requirements: Dash0 OpenTelemetry .NET Distro

## Summary

Build a Dash0-branded OpenTelemetry .NET auto-instrumentation distribution — per-RID platform bundles (`dash0-opentelemetry-dotnet-autoinstrumentation-<rid>.{tar.gz,zip}`) plus a small set of Dash0-branded NuGet packages for library-mode users, delivered as drop-in replacements for upstream's `opentelemetry-dotnet-instrumentation` bundles and packages. Injection is by environment variable (`CORECLR_ENABLE_PROFILING`, `CORECLR_PROFILER`, `CORECLR_PROFILER_PATH`, `DOTNET_STARTUP_HOOKS`, `OTEL_DOTNET_AUTO_HOME`) — mechanism unchanged from upstream. Both the Dash0 Kubernetes operator's auto-injection flow and standalone users consume the same bundles.

v1 delivers four payoffs:
- **`.NET 6+` runtime support as a policy** — Dash0 commits to supporting every .NET major version from `.NET 6` upward (`net6.0`, `net7.0`, `net8.0`, `net9.0`, `net10.0`, and all future majors) regardless of upstream's or Microsoft's support windows. This is a **standing commitment**, not a one-time `.NET 6` exception. The distro produces **one build per RID**, not a build per runtime — the shipped bundle is multi-target and covers every runtime in the R21 set. A single long-lived `dash0-main` branch on the affected fork(s) carries the entire runtime-support patch set (TFM additions + `#if` polyfills). As new runtimes ship and eventually EOL, they enter that same branch via TFM addition; retirement is TFM removal on the same branch. `.NET Framework` and `netcoreapp3.1` are explicitly out of v1 (Q_S1 resolved 2026-07-17); the distro is Linux-first, with macOS bundles for dev ergonomics only.
- Shipping instrumentations faster than upstream can merge them.
- Augmented resource detection.
- Activation guards that prevent double-activation or host-process breakage.

The distro accepts indefinite carry of the runtime-support patch set. "Unlikely upstream" is a first-class design premise, not a temporary state waiting for merge.

---

## Problem Frame

Dash0's current .NET posture is thin. The Kubernetes operator injects the upstream OpenTelemetry .NET auto-instrumentation bundle for `.NET Core 3.1+` workloads; Dash0 authors no .NET-specific instrumentation. .NET is a "wrap upstream" story today.

Two triggers change that:

- **Runtime drop schedule.** Microsoft's LTS/STS cadence retires .NET major versions on a predictable schedule (LTS: 3 years from GA; STS: 18 months from GA). Upstream `open-telemetry/opentelemetry-dotnet-instrumentation` follows that cadence closely and drops out-of-support runtimes from its test matrix and shipped bundles with a short lag. Enterprise customers whose runtime hits Microsoft EOL are stranded on the observability side — they can freeze on an older upstream release (accumulating security debt and losing new instrumentations) or migrate their runtime (a decision often outside the observability team's control). The distro is the third option, offered as a **standing policy for every .NET major from 6 upward**: as each runtime hits upstream's drop, Dash0 assumes patch responsibility.

  Current landscape as of July 2026:
  - `.NET 6` LTS — Microsoft EOL November 2024 (~20 months past). Almost certainly out of upstream's supported set already.
  - `.NET 7` STS — Microsoft EOL May 2024 (~26 months past). Long since dropped by upstream. Low remaining enterprise population (STS bridge release).
  - `.NET 8` LTS — Microsoft EOL **November 2026** (~4 months away). Highest current enterprise installed base. Will move onto the runtime-support track during or shortly after v1 ships.
  - `.NET 9` STS — Microsoft EOL May 2026 (~2 months past). Small population (STS). Upstream stance uncertain at this moment.
  - `.NET 10` LTS — current, upstream-supported through Microsoft EOL in 2028.

  This makes the runtime-support track a growing operational surface, not a one-time exception. The runtime-support policy (R22) treats every .NET 6+ major as a Dash0 commitment for the runtime's economic lifetime.

- **Upstream merge latency.** Dash0 has instrumentation contributions and default-behavior opinions it wants to ship on its own timeline. Upstream cannot merge at customer-experienced pain speed. Same problem as Java.

The `.NET 6+` support policy is a design frontier that has no precedent in the OTel .NET vendor-distro space (Splunk, Datadog's dd-trace-dotnet, New Relic's .NET agent, and Dynatrace all target the current .NET runtimes officially; commercial APM vendors extend older runtime support privately). It is the load-bearing differentiator and the primary source of design complexity — support commitments, patch surface, and rebase risk all concentrate around this axis.

**Scope: Linux production, macOS dev-only, no Windows.** The distro's supported OS matrix in v1 is Linux (glibc and musl, x64 and arm64) for production workloads and macOS (x64 and arm64) for dev-machine testing only. Windows is out — no Windows bundles are built, no `.NET Framework` support is offered (structurally impossible off Windows), and the Dash0 K8s operator is not expected to inject on Windows containers. This is a Dash0 product-level positioning choice, not a technical constraint of the underlying mechanism.

The distro exists to close both gaps without becoming a fork. The `distro.md` principles the Java distro established — rebase on every upstream release, upstream as much as possible, own the build and release — carry over. `.NET` deviates from Java only where the "unlikely upstream" runtime-support work forces divergent branches that will not close.

---

## Key Decisions

- **Per-RID platform bundles + a small set of NuGet packages as the shipped artifacts.** For each release the distro publishes:
  - Auto-instrumentation bundles per Runtime Identifier: `linux-x64`, `linux-arm64`, `linux-musl-x64`, `linux-musl-arm64`, `osx-x64`, `osx-arm64`. Each is a `tar.gz` containing the native CLR profiler shared library (`Dash0.OpenTelemetry.AutoInstrumentation.Native.{so,dylib}`), the managed assemblies for every supported TFM, `instrument.sh`, and the version manifest. Linux bundles are the production artifacts; macOS bundles exist for dev-machine testing and carry the same drop-in guarantees but are not injected by the operator.
  - A minimal NuGet set for library-mode users: `Dash0.OpenTelemetry.AutoInstrumentation` (configuration entry point), `Dash0.OpenTelemetry.AutoInstrumentation.Loader` (startup hook), and any Dash0-authored instrumentation packages that are usefully consumed without the profiler.

  Bundle + NuGet is the shape upstream uses (minus Windows); Dash0 preserves it so users switching in either direction on Linux/macOS pay migration cost only in the version manifest, not in the injection model.

- **Multi-repo distribution model: extensions in the `-distro` repo, patched sources in fork branches, one build pipeline in the `-distro`.** Mirrors the Java distro. Comprises:
  - This `-distro` repository, which owns the entire build and publish pipeline. Holds the assembly, Dash0-authored extension modules (instrumentations, resource detectors, startup-hook augmentations), Dash0-specific tests and canonical-application fixtures, and — via git submodules + `<ProjectReference>` swaps — pinned checkouts of the Dash0 forks.
  - Dash0 forks of the upstream repositories the distro depends on: at minimum `open-telemetry/opentelemetry-dotnet-instrumentation`; very likely `open-telemetry/opentelemetry-dotnet-contrib`; possibly `open-telemetry/opentelemetry-dotnet` (SDK) if runtime-support patches touch it. Patches to upstream modules and modules that must live as real upstream modules rather than SPI extensions live on feature branches in the appropriate fork. Forks have no publish pipeline of their own — no NuGet packages published from a fork.

  The `-distro`'s MSBuild pipeline consumes upstream unpatched modules as ordinary NuGet packages from `nuget.org` at their `OpenTelemetry.*` package IDs, and consumes patched modules via `<ProjectReference>` swaps against pinned fork checkouts. The only artifacts ever published are the per-RID bundles and the Dash0-branded NuGet packages, both to GitHub Release.

  ```mermaid
  flowchart TB
    Upstream[Upstream OTel .NET NuGet packages<br/>consumed as-published from nuget.org<br/>at OpenTelemetry.* IDs] --> DistroBuild
    ForkSources[Dash0 fork checkouts<br/>pinned by commit SHA<br/>via git submodule<br/>only for modules Dash0 patches<br/>including runtime-extension patches] --> DistroBuild
    Extensions[Dash0-authored extension modules<br/>instrumentations, resource detectors,<br/>startup-hook augmentations<br/>in the -distro repo] --> DistroBuild
    DistroBuild[-distro MSBuild pipeline<br/>ProjectReference swaps<br/>patched package IDs for fork projects<br/>per-RID native + managed assembly] --> Bundles[Per-RID bundles + NuGet packages<br/>GitHub Release]
  ```

  This model has no direct precedent in the OTel .NET vendor-distro space, same as Java — an early spike is required (see Outstanding Questions).

- **One long-lived runtime-support branch, `dash0-main`, carries the entire `.NET 6+` policy.** The branch lives on the Dash0 fork of `opentelemetry-dotnet-instrumentation` (and, where required, of `opentelemetry-dotnet`). It multi-targets every extended runtime in a single source tree — upstream's `<TargetFrameworks>net8.0</TargetFrameworks>` (or whatever upstream currently targets) becomes `<TargetFrameworks>net6.0;net7.0;net8.0;net9.0;net10.0</TargetFrameworks>` on the branch, with `#if` guards for per-version BCL API gaps. **One branch, not per-runtime branches, and one shipped build per RID that covers all runtimes** — the injector expects a single bundle, not one bundle per .NET version. Per-runtime branches would duplicate the same edits to the same upstream source files, produce combinatorial rebase pain (N rebases per upstream release instead of 1), and require merging back into a single artifact anyway.

  As new .NET majors ship and eventually EOL, they enter this branch via TFM addition and `#if` maintenance. Retiring a runtime (F6) is TFM removal + `#if` cleanup on the same branch, not branch deletion. The branch is rebased onto each new upstream release. It is **explicitly exempt from the R11 upstream-PR requirement** — the "unlikely upstream" premise is that these patches will not be accepted upstream, and the distro accepts indefinite carry. This exemption is the only structural deviation from the Java distro's "no orphan patches" discipline, and it is recorded in the branch's R10 metadata file with the reason code `dash0-carry`.

- **Rebase on every upstream release; the `-distro` repo is not a fork.** Same as Java. The `-distro` repo pins to specific upstream releases; on each new upstream release the corresponding Dash0 forks fast-forward their upstream-tracking branches, rebase active feature branches (including the long-lived `dash0-main` branch), and the `-distro` repo refreshes its pins.

- **Preserve upstream telemetry shape by default.** The distro does not silently change span names, attribute keys, metric names, or the set of instrumentations that fire under default configuration. Any change to telemetry shape is an explicit release decision gated on a canonical-application diff. Runtime-support patches carry an additional obligation: telemetry emitted from `.NET 6` under the distro must match, in shape, telemetry emitted from the same instrumentation under `.NET 8+` — the runtime-support track is not a permission slip to silently diverge.

- **Upstream-PR ergonomics are a first-class distro concern (for non-runtime-support patches).** Same as Java. The distro repo is structured so authoring an instrumentation here creates no friction when it is time to open a PR against `open-telemetry/opentelemetry-dotnet-instrumentation` or `open-telemetry/opentelemetry-dotnet-contrib`. Module layout, test conventions, and MSBuild wiring mirror upstream's expectations closely enough that a contribution can be extracted into a branch of Dash0's fork with minimal transformation. Runtime-support patches are outside this scope by construction (see the runtime-support decision above).

- **Native AOT is out of scope.** The auto-instrumentation profiler mechanism (CLR profiler + IL rewriting) is fundamentally incompatible with `PublishAot`. Users on Native AOT get the library-mode NuGet path only, and instrumentation packages that require IL rewriting are unavailable. This is a structural constraint of the underlying CLR mechanism, not a Dash0 choice — but it must be documented so users know what to expect when they hit it.

- **Upstream contribution runs on a parallel track for non-runtime-support work.** Same as Java. Dash0's OpenTelemetry .NET community investment reduces the size of the patch queue over time for anything except the `dash0-main` branch's contents. The runtime-support track is by design a permanent divergence from upstream and is not something contribution can close.

---

## Requirements

**Distribution and packaging**

- R1. The distro publishes per-RID auto-instrumentation bundles (`dash0-opentelemetry-dotnet-autoinstrumentation-<rid>.tar.gz`) covering: `linux-x64`, `linux-arm64`, `linux-musl-x64`, `linux-musl-arm64`, `osx-x64`, `osx-arm64`. Linux bundles are production artifacts; macOS bundles are for dev-machine testing. **Windows is out of scope for v1** — no `win-x64` or `win-arm64` bundles are built. It additionally publishes Dash0-branded NuGet packages for library-mode configuration and startup-hook loading.
- R2. The bundles are drop-in replacements for the upstream OpenTelemetry .NET Auto-Instrumentation bundles under the **default configuration surface**: no user-supplied additional instrumentation plugins, no coexisting OpenTelemetry SDK triggering R7 suppression, no version-pinned environment variables whose semantics a Dash0 patch has changed, no user-modified default flags. Within this surface, a user who replaces upstream's bundle with Dash0's on the same `CORECLR_PROFILER_PATH` and `OTEL_DOTNET_AUTO_HOME` env vars sees the distro activate and emit telemetry indistinguishable from upstream in shape, plus the Dash0-authored additions from R5 / R6. Trust substitution is backed by the signing and provenance artifacts of R18.

  **Recognized configurations that may behave differently** (documented in release notes; not a violation of R2):
  - Users loading their own additional plugins via `OTEL_DOTNET_AUTO_PLUGINS` — Assembly Load Context isolation in the Dash0 bundle may resolve different assembly versions than upstream; compatibility with user plugins is best-effort at v1.
  - Users pinning to a specific upstream bundle version — the Dash0 manifest identifies "upstream vX + Dash0 deltas Y", not literal upstream vX.
  - Users on `OTEL_DOTNET_AUTO_*` env vars whose semantics a Dash0 patch has altered — release notes call out every such change.
  - Users on runtimes outside upstream's supported set (currently `.NET 6`, `.NET 7`, likely `.NET 9`; `.NET 8` joins in November 2026) — telemetry shape is held stable per the runtime-support telemetry obligation, but instrumentations that upstream never authored for those runtimes may be absent.
  - Users whose environment triggers R7 suppression — the distro loads but produces no telemetry.

- R3. The bundles and NuGet packages are published as GitHub Release assets on the distro's repository. The Dash0 Kubernetes operator's automatic workload instrumentation flow and standalone users fetch from there — no `nuget.org` publication is required for v1.
- R4. Each release identifies the exact upstream `opentelemetry-dotnet-instrumentation` release it was built from, exposed in release notes and at runtime via a version manifest inside every bundle and package.

**Runtime support extension (v1 headline)**

- R21. The distro supports **every .NET major version from `.NET 6` upward, indefinitely** (Q_S1 resolved 2026-07-17). At v1 cut this is `net6.0`, `net7.0`, `net8.0`, `net9.0`, `net10.0`, and any subsequent .NET major (`net11.0`, `net12.0`, ...) added as they ship. **The commitment is polymorphic and standing** — once a runtime is in R21, it does not leave except via the formal retirement path in F6, which requires customer-signal justification, not Microsoft or upstream drop.
  - **Explicit out**: `.NET Framework 4.6.2+` (structurally requires Windows, which is out of scope), `netcoreapp3.1` and earlier (below the `.NET 6+` floor). See Scope Boundaries.
  - The specific supported-runtimes matrix is enumerated in `SUPPORTED-RUNTIMES.md` per release, listing each supported runtime's current status (upstream-supported, on runtime-support track).
  - **The `.NET 6` slot's inclusion is contingent on M0 (see Outstanding Questions).** M0 is a pre-v1 feasibility milestone that validates the runtime-support-track pattern on the hardest current case (`.NET 6` on Linux — oldest, most API drift from upstream's current target). If M0 finds the patch surface for `.NET 6` unsustainable, `.NET 6` moves out of R21, the `.NET 6+` policy contracts to `.NET 7+`, and the distro's v1 value proposition is materially weakened. `.NET 7` and `.NET 9` are trivially easier than `.NET 6` (fewer API gaps against the current target) and are covered by M0's design generalizability check, not by a separate spike. `.NET 8+` runtimes currently within upstream's supported set do not require M0 validation; the runtime-support track for each of them stands up when upstream drops them.
- R22. When upstream drops support for a runtime in the R21 set, that runtime's TFM (and any needed `#if` polyfills) is added to the single existing `dash0-main` branch on the affected fork(s), and the distro takes over patch responsibility for the runtime through that branch. No new branch is opened per runtime — the injector-consumed bundle is a single multi-target build. The commitment is **indefinite continuation, not a bounded extension window** — a runtime stays in R21 as long as customer signal, security exposure, and patch cost together justify it. This is a stronger commitment than "N years past Microsoft EOL" and is the load-bearing product promise the distro makes to enterprise customers with slow-moving runtime-migration cycles. The **runtime-support policy** at `SUPPORTED-RUNTIMES.md` names, per runtime: (a) current status (upstream-supported, on the runtime-support branch, retirement announced), (b) the trigger for retirement (customer telemetry falling below a threshold, security exposure that cannot be patched at reasonable cost, sustained patch cost exceeding the sustain judgment), (c) the notification window before retirement (N distro releases in advance). The policy is a commitment surface: customers pin migration timelines against it.
- R23. Each Dash0-authored instrumentation targets the full R21 runtime set unless a specific instrumentation is inherently runtime-restricted (e.g., an instrumentation for a library whose earliest supported package version requires `net8.0`). Restrictions are declared per-instrumentation in the release notes and in the version manifest, not implicit.
- R24. Telemetry emitted by an instrumentation running on an extended runtime (e.g., `.NET 6`) matches, in shape, telemetry emitted by the same instrumentation running on a currently-supported runtime (e.g., `.NET 8`), to the extent the underlying instrumented library exposes the same signals. Shape parity across runtimes is gated on the canonical-application diff test (R16).

**Value-adds delivered in v1**

- R5. The distro ships instrumentations that upstream has not yet merged. These live as extension modules loaded via the upstream plugin mechanism (`OTEL_DOTNET_AUTO_PLUGINS`) in the `-distro` repo (default path) or as new/patched instrumentation modules on a Dash0 fork branch of the appropriate upstream repository (fallback path). The fallback path is gated on at least one of the following criteria applying to the change (a reviewer sign-off explicitly cites which criterion):
  - The change requires modifying IL-rewriting `CallTarget` integrations on classes upstream's plugin mechanism does not expose to third-party assemblies.
  - The change requires access to `internal` types or methods that upstream's plugin surface does not expose.
  - The change alters an existing upstream instrumentation's telemetry shape (span names, attribute keys, metric names, timing semantics) rather than layering new signals on top.
  - The change requires modifying the native CLR profiler component (uncommon; almost always signals a runtime-support-track patch instead).

  Absent any of these, the change must be authored as a plugin-loaded extension in the `-distro` repo. Fork-branch drift into becoming the default path is the failure mode this rule prevents.
- R6. The distro augments resource detection beyond upstream defaults. Detectors follow the upstream `OpenTelemetry.ResourceDetectors.*` package pattern so they compose cleanly with upstream detection. The concrete v1 detector set is settled in planning — see Outstanding Questions.
- R7. The distro applies activation guards at agent bootstrap. When any probe in the v1 guard set fires, the distro **suppresses activation**: no instrumentations are installed, no resource detectors run, no bytecode is rewritten, no telemetry is emitted. The v1 guard set has three probe families:
  - **Another CLR profiler is already loaded.** Detected at the native profiler entry point via the `CORECLR_PROFILER` GUID and via presence of loaded profiler assemblies for recognized vendors (Datadog `dd-trace-dotnet`, New Relic .NET agent, AppDynamics, Dynatrace, upstream OTel bundle when a distinct GUID is present). Outcome: suppress activation, reason code `peer-profiler`.
  - **A framework or library will initialize its own OpenTelemetry SDK during startup.** Detected by loaded-assembly probes for `OpenTelemetry.dll` at a version already initialized by the host, and for host-authored SDK bootstrap patterns in ASP.NET Core hosted services. Outcome: suppress activation, reason code `host-sdk`.
  - **A `TracerProvider` is already registered as global at startup-hook time.** Detected by inspecting `Sdk.CreateTracerProviderBuilder`-registered globals via reflection on `OpenTelemetry.Api`. Outcome: suppress activation, reason code `global-tracer-initialized`.

  **Failure mode: fail-closed.** If any probe throws an unexpected exception (permission denial, load-context isolation anomaly), the guard suppresses activation and logs the failure with reason code `probe-error`.

  **User override:** `DASH0_DOTNET_AUTO_GUARDS_FORCE=activate|suppress|auto` (default `auto`) is honored regardless of guard state, with reason codes `forced-activate` and `forced-suppress`.

  **Diagnostic signal:** On every process startup where the distro loads, exactly one INFO-level structured log line identifies the guard outcome (`dash0.distro.guard: outcome=suppress reason=host-sdk detected=aspnetcore-hosted-sdk`). When the guard resolves to activate, the outcome and reason are additionally exposed as resource attributes (`dash0.distro.guard.outcome`, `dash0.distro.guard.reason`).

- R8. When activated, the distro flushes buffered telemetry (spans, metrics, logs) at process shutdown before the runtime exits. The startup hook registers an `AppDomain.ProcessExit` handler and, for `IHost`-based apps, hooks `IHostApplicationLifetime.ApplicationStopping` when a host is detected. The handler triggers the underlying OpenTelemetry SDK's `Dispose()` chain on the configured providers, waiting for pending exports within a bounded timeout (specific value settled in planning). When the guard suppresses activation, no shutdown handler is registered.

**Upstream discipline**

- R9. The distro rebases onto every new upstream release of the repositories it consumes. When a fork feature branch's rebase produces conflicts, the distro release cut is blocked until the conflict is resolved by a human. This applies equally to short-lived instrumentation feature branches and to the long-lived `dash0-main` branch — a stalled release is preferable to silently dropping a customer-facing capability. The `dash0-main` branch's conflict SLA is stricter and separately tracked (see Q13).
- R10. Each fork feature branch the `-distro` build consumes carries a machine-readable metadata file at the branch root recording at minimum: (a) the upstream release tag the branch is rebased onto, (b) the upstream PR reference(s) this branch corresponds to — OR `runtime-support-<version>` as a first-class alternative for the long-lived branches exempt from R11, (c) any upstream PRs the branch depends on or that would conflict on rebase. The `-distro`'s MSBuild pipeline reads (a) at build time and fails if it does not match the upstream version the distro is currently pinned to.
- R11. The distro does not carry a non-runtime-support patch on a Dash0 fork branch without a documented upstream PR reference recorded in the branch's R10 metadata file. **The long-lived `dash0-main` branch is exempt** — it carries `reason=dash0-carry` in place of an upstream PR reference and is explicitly not staged for upstream PRs.
- R12. Dash0-authored extension modules in the `-distro` repo are laid out in a directory structure that mirrors upstream's `src/` conventions for the corresponding upstream module (e.g., `src/OpenTelemetry.Instrumentation.<Library>/` under Dash0's own root path). This preserves the option to extract an extension into a fork feature branch later if the plugin mechanism cannot express the change — path relocation and MSBuild-target adjustment only, no rewriting of module code, tests, or plugin registration.
- R13. The `-distro`'s MSBuild pipeline consumes upstream OpenTelemetry .NET modules as ordinary NuGet packages from `nuget.org` at their `OpenTelemetry.*` package IDs. For modules Dash0 is actively patching (including runtime-support patches), the build consumes the corresponding Dash0 fork feature branch via git-submodule checkout + `<ProjectReference>` swap in `Directory.Packages.props` (via `<PackageVersion>` overrides pointing at project paths, or a solution-level substitution mechanism — see Q_R13). No NuGet artifacts are published for patched fork modules — only the per-RID bundles and the Dash0-branded NuGet packages are published. The substitution registry in `Directory.Packages.props` is the auditable record of what Dash0 patches at each release.
- R14. Each Dash0 fork of an upstream OpenTelemetry .NET repository follows the same documented branching model: an upstream-tracking branch that is fast-forwarded to each new upstream **release tag**, per-work short-lived feature branches for in-flight patches, and the single long-lived `dash0-main` branch that carries the multi-target runtime-support patch set.
- R15. On each new upstream release tag of a consumed repository, the corresponding Dash0 fork fast-forwards its upstream-tracking branch, rebases every active feature branch (short-lived and the long-lived `dash0-main`) onto it, and deletes short-lived feature branches whose upstream PRs have merged. The `dash0-main` branch is never deleted through the merge path — it exits the tree only if the runtime-support policy (R22) formally sunsets every runtime it carries, which is not expected within any foreseeable release horizon. Retiring a single runtime from the branch is a TFM removal + `#if` cleanup on the same branch. The `-distro` repo updates the pinned fork commit SHAs it consumes to the newly-rebased tips.

**Telemetry stability**

- R16. Any change that alters the shape of telemetry (span name, attribute set, metric name, resource attribute set) between distro releases is gated on a canonical-application diff test. The test compares telemetry emitted by a fixed set of reference apps under the outgoing and incoming distro versions, **across every runtime in R21** — the runtime-support commitment (R24) is enforced here.
- R17. When an upstream release introduces a new instrumentation or changes an existing instrumentation's telemetry shape, the distro's rebase workflow surfaces the delta and requires an explicit accept/defer decision before the new distro release ships. When upstream drops an instrumentation for an out-of-support runtime that the distro extends via R22, the rebase workflow surfaces that as a runtime-support-track decision: carry the instrumentation forward on the runtime-support branch, or defer.

**Supply chain and integrity**

- R18. Each distro release publishes alongside the bundles and NuGet packages on GitHub Release: (a) a detached signature (Sigstore keyless via GitHub OIDC preferred; Dash0-controlled key acceptable if OIDC is unavailable), (b) SHA-256 checksum files, (c) a CycloneDX SBOM enumerating every bundled dependency at its `PackageId:Version` coordinate and its resolution source (`nuget.org` or Dash0 fork commit), (d) a SLSA level 2 build-provenance attestation. Release notes include the verification command for each artifact.
- R19. The signing identity, the SBOM format (CycloneDX), and the verification procedure are stable from v1 onward. Downstream consumers pin verification against this trust root without churn.
- R20. When the bundles are redistributed by a downstream packaging step (e.g., the Dash0 Kubernetes operator's OCI-image wrapping for its auto-injection init container), the redistribution preserves the bundle bit-for-bit unmodified — same SHA-256, same archive contents, same default env-var scripts. The redistributor records the distro version and the bundle SHA-256 in a machine-readable location so runtime traceability is auditable. R4's version manifest and R19's verification story hold for both direct and redistributed consumption paths.

---

## Key Flows

- F1. **Auto-instrumentation bootstrap with activation guard**
  - **Trigger:** A .NET process starts with the Dash0 profiler env vars set (`CORECLR_ENABLE_PROFILING=1`, `CORECLR_PROFILER=<Dash0 GUID>`, `CORECLR_PROFILER_PATH=<path-to-Dash0.OpenTelemetry.AutoInstrumentation.Native>`, `DOTNET_STARTUP_HOOKS=<path-to-loader>`, `OTEL_DOTNET_AUTO_HOME=<bundle-root>`), whether set by the Dash0 K8s operator's injector or by the user directly.
  - **Steps:** The CLR loads the native profiler; the profiler runs pre-managed-code activation guard probes for peer profilers; the .NET runtime invokes the startup hook, which runs managed-side activation guard probes; the guard either activates the distro (registering plugin modules, resource detectors, the R8 shutdown handler, and enabling IL rewriting) or suppresses activation, in which case no bytecode is rewritten and no telemetry is emitted.
  - **Outcome:** Either the distro is fully active with Dash0 extensions and resource detectors registered, or the process runs as if the distro were not present. **"No partial activation" means: when the guard suppresses, no IL rewriting is enabled and no managed instrumentation is installed** — the process behaves exactly as though the profiler env vars had not been set.
  - **Covered by:** R2, R6, R7, R8.

- F2. **Distro release cut against a new upstream release**
  - **Trigger:** A new release tag of any consumed upstream OpenTelemetry .NET repository is published.
  - **Steps:** In each affected Dash0 fork: fast-forward the upstream-tracking branch, rebase all active feature branches (short-lived + long-lived runtime-support), delete short-lived branches whose upstream PRs have merged. In the `-distro` repo: bump upstream NuGet package versions, update pinned fork commit SHAs, remove `Directory.Packages.props` substitutions for deleted branches, recompile Dash0-authored extensions against the new upstream and against every runtime in R21, run canonical-application diff across the runtime matrix, surface any telemetry-shape delta as an accept/defer decision, and cut the distro release with the new upstream and fork commit SHAs recorded in the manifest and release notes.
  - **Outcome:** New distro release consuming the new upstream, with Dash0 patches (including runtime-support) carried forward on rebased branches.
  - **Covered by:** R4, R9, R10, R13, R14, R15, R16, R17, R21, R22, R23, R24.

- F3. **Upstream merges a Dash0-authored PR the distro has been carrying**
  - **Trigger:** During a rebase cycle, an upstream release includes a Dash0-authored contribution the fork was carrying as a short-lived feature branch.
  - **Steps:** The short-lived branch is deleted per R15. The `-distro` build's substitution entry for that module is removed and the module returns to being consumed as an ordinary NuGet package. Canonical-application diff runs across the full R21 runtime matrix to detect telemetry-shape drift; drift is either accepted or reconciled. The `dash0-main` branch is unaffected by this flow (it does not close through merges).
  - **Outcome:** Distro no longer carries the merged instrumentation on the non-runtime-support path.
  - **Covered by:** R11, R13, R15, R16, R17.

- F4. **Authoring an instrumentation with an upstream PR in view**
  - **Trigger:** Dash0 wants to add an instrumentation for library X (either net-new to the ecosystem or currently blocked in upstream review).
  - **Steps:** Decide whether the work fits the plugin-extension shape (author in the `-distro` repo per R12's layout) or must be a real upstream module (author on a short-lived feature branch of the appropriate Dash0 fork per R14). If the work is on a fork branch, add the substitution entry to the `-distro`'s `Directory.Packages.props`. If the work is a plugin extension, add it to the `-distro`'s extension-module list. When ready to upstream: extract-or-open the PR from the fork branch. Instrumentation is authored against the full R21 runtime set from the start.
  - **Outcome:** The instrumentation reaches customers via the distro on Dash0's cadence, running on `net6.0` through the current LTS.
  - **Covered by:** R5, R11, R12, R13, R14, R23.

- F5. **Runtime-support-branch rebase against a new upstream release**
  - **Trigger:** A new upstream release lands; the long-lived `dash0-main` branch on `dash0hq/opentelemetry-dotnet-instrumentation` (and, where required, on `dash0hq/opentelemetry-dotnet`) must rebase onto it.
  - **Steps:** Fast-forward the fork's upstream-tracking branch to the new release tag. Attempt rebase of the runtime-support branch onto the new tag. On conflict, block the distro release cut and route to the runtime-support-track resolution SLA (Q13). On clean rebase, run the runtime-support test matrix (Q_R25) against the rebased tip. Update the `-distro`'s pinned commit SHA for the branch. Because the runtime-support branch will not close, this rebase runs on every release, indefinitely.
  - **Outcome:** `.NET 6` remains supported against the new upstream, or the release is blocked pending resolution.
  - **Covered by:** R9, R10, R14, R15, R22, R24.

- F6. **Retiring `.NET 6` per the runtime-support policy**
  - **Trigger:** The runtime-support policy (R22) revisit criterion fires — the extension window expires, customer telemetry indicates near-zero `.NET 6` usage, or the patch cost exceeds the sustained-support judgment.
  - **Steps:** Announce the retirement per the policy's downstream-notification path (release notes, N releases in advance). At the cutover release, on the `dash0-main` branch: remove `net6.0` from the multi-target `<TargetFrameworks>` lists and clean up `#if NET6_0` guards. Remove `net6.0` from R21's TFM set, from the R16 canonical-application diff matrix, and from the CI test matrix. The `dash0-main` branch itself remains for the other extended runtimes it carries. The retirement is recorded in `SUPPORTED-RUNTIMES.md`.
  - **Outcome:** `.NET 6` is no longer supported by the distro. Customers who need it must freeze on the last supporting distro release.
  - **Covered by:** R21, R22, R23.

---

## Acceptance Examples

- AE1. **Activation guard suppresses under host-registered SDK**
  - **Covers:** R7.
  - **Given:** A .NET 8 ASP.NET Core application registers its own `TracerProvider` via `services.AddOpenTelemetry().WithTracing(...)` during host startup, initializing `Sdk.CreateTracerProviderBuilder`-produced globals before the startup hook completes.
  - **When:** The Dash0 profiler env vars are set on the process, `DASH0_DOTNET_AUTO_GUARDS_FORCE` is unset.
  - **Then:** The startup-hook-time guard detects the initialized global `TracerProvider`, suppresses activation with reason code `global-tracer-initialized`, and emits the structured diagnostic log line. No IL rewriting is enabled, no Dash0 plugin is loaded, no Dash0-emitted telemetry appears. The host's own SDK operates normally.

- AE2. **Rebase drops a merged short-lived PR**
  - **Covers:** R11, R15, R16.
  - **Given:** A prior distro release consumed a Dash0 fork feature branch of `opentelemetry-dotnet-instrumentation` via `<ProjectReference>` substitution, carrying an in-flight PR for an instrumentation of library X.
  - **When:** A new upstream release lands with that instrumentation merged in.
  - **Then:** The short-lived fork branch is deleted per R15, the `Directory.Packages.props` substitution for that module is removed, the module returns to being consumed as an ordinary NuGet package, the canonical-application diff runs across the R21 runtime matrix, and release notes call out the transition.

- AE3. **Fork branch not rebased fails the build**
  - **Covers:** R10, R14, R15.
  - **Given:** The `-distro` build consumes a fork feature branch via `<ProjectReference>` substitution; that branch's R10 metadata records upstream base `v1.11.0`.
  - **When:** The distro build runs pinned to upstream `v1.12.0` and no rebase of the fork branch has been performed.
  - **Then:** The build fails with a clear diagnostic identifying the drifted fork branch and requires an explicit rebase before the release can proceed.

- AE4. **.NET 6 support extends past upstream drop**
  - **Covers:** R21, R22, R23, R24.
  - **Given:** Upstream `opentelemetry-dotnet-instrumentation` release vX drops `net6.0` from its TFM set. The Dash0 fork's `dash0-main` branch has been rebased onto vX and carries the necessary multi-target patches to keep `net6.0` (alongside `net7.0` and any other extended runtimes) compiling and instrumenting.
  - **When:** The distro release cut against upstream vX runs.
  - **Then:** Dash0's bundle still ships a `net6.0`-compatible managed assembly. A canonical `.NET 6` reference app instrumented via the Dash0 bundle emits telemetry whose shape matches the same reference app running on `net8.0` under the same Dash0 bundle version. Release notes state "runtime support extended: net6.0 remains supported per SUPPORTED-RUNTIMES.md".

- AE5. **Runtime-support-branch rebase conflict blocks release**
  - **Covers:** R9, F5, Q13.
  - **Given:** Upstream vY refactors an internal API on which the `dash0-main` branch's patches depend. Rebase of the runtime-support branch onto vY produces conflicts.
  - **When:** The distro release cut against vY runs.
  - **Then:** The build blocks with a diagnostic identifying the conflict. The runtime-support-track resolution SLA is triggered; the release is not cut until the conflict is resolved by a human and the rebased branch passes the runtime-support test matrix.

- AE6. **Drop-in with user plugin is outside R2's default surface**
  - **Covers:** R2 (recognized-differences boundary).
  - **Given:** A customer runs the upstream OpenTelemetry .NET Auto-Instrumentation bundle with `OTEL_DOTNET_AUTO_PLUGINS=/opt/custom-plugin.dll` loading a user-authored plugin that depends on specific upstream package versions.
  - **When:** The customer replaces the upstream bundle path with the Dash0 bundle path and retains the same `OTEL_DOTNET_AUTO_PLUGINS`.
  - **Then:** The Dash0 distro activates, but the user plugin may or may not resolve depending on how the Dash0 bundle's Assembly Load Context resolves the packages the plugin depends on. This is a recognized non-default configuration documented in the release notes' recognized-differences list.

---

## Success Criteria

- S1. A customer running the upstream OpenTelemetry .NET Auto-Instrumentation bundle under the default configuration surface defined in R2 can switch to the distro by replacing the bundle path in `CORECLR_PROFILER_PATH` and `OTEL_DOTNET_AUTO_HOME` and observes no regressions in default telemetry shape, verified against the canonical-application diff across the R21 runtime matrix.
- S2. A customer on `.NET 6` running on Linux receives continued instrumentation support past Microsoft's end-of-support date, per the runtime-support policy in `SUPPORTED-RUNTIMES.md`. Telemetry emitted on `.NET 6` matches, in shape, telemetry emitted on `.NET 8+` for the same instrumented library.
- S3. Dash0 can ship a new instrumentation to a customer within a bounded cycle time (concrete target set in planning), independent of upstream's merge schedule.
- S4. The distro build fails loudly and locally on any drift between Dash0 fork checkouts consumed by the `<ProjectReference>` substitution mechanism and the upstream releases currently being tracked — no silent divergence is possible.
- S5. Every distro release is traceable to a specific set of upstream release tags (per consumed repository) and a specific set of Dash0-authored deltas (extension modules in the `-distro` repo + fork branches at recorded commit SHAs), inspectable from the released bundles and NuGet packages, the `-distro` repo at the release tag, and the referenced fork branches at their pinned commits.
- S6. A customer can verify a released bundle's or package's origin and integrity independently of the download channel. The signature, checksum, and CycloneDX SBOM together let a security-conscious operator gate deployment on verified provenance.

---

## Scope Boundaries

**Deferred for later**

- **Windows support.** No `win-x64` / `win-arm64` bundles, no `.NET Framework` support (structurally requires Windows), no injection on Windows containers. Re-enters via a new brainstorm if Dash0's product-level Windows posture changes.
- **`netcoreapp3.1`.** Small remaining Linux population; most customers stranded on out-of-support runtimes are on `.NET 6`, not 3.1. Case-by-case if customer signal appears; not v1.
- **`.NET Framework 4.6.2+`.** Windows-only structurally, and covered by the Windows-support deferral above.
- Bootstrap span (parallel to the Java distro; deferred past v1).
- Telemetry curation as cost optimization at source (deferred past v1).
- Native AOT support — structurally incompatible with the profiler; library-mode NuGet path is the only story for AOT users, and even that has limits. Not v1.
- Configuration profiles (opinionated env-var presets bundled with the distro) — not v1.
- `nuget.org` publication for the Dash0-branded NuGet packages — v1 publishes to GitHub Release only. Adding `nuget.org` later remains open.
- SLSA level 3 — v1 targets SLSA level 2.
- Automated CVE-triage cadence independent of upstream releases — v1 relies on the upstream release cadence.

**Outside this product's identity**

- Proprietary agent that diverges from OpenTelemetry semantic conventions or wire format — the distro emits standard OTLP against standard semantic conventions.
- OpAMP-based remote SDK configuration — no evidence of demand today.
- The `-distro` repo itself is not a fork of any upstream repository — same principle as Java.
- OCI image packaging for injection. The Dash0 Kubernetes operator wraps the bundles in its own init container image in its own repository. This distro does not produce or publish container images.
- A "smart Dash0-only" mode that emits telemetry differently from vanilla OTel when Dash0 is the backend — the distro must be usable against any OTLP-compliant backend.

---

## Dependencies / Assumptions

- Upstream OpenTelemetry .NET projects continue to publish releases on a predictable cadence and to expose the plugin mechanism (`OTEL_DOTNET_AUTO_PLUGINS`), `ResourceProvider` package pattern, and CLR-profiler-plus-startup-hook injection surface the distro relies on. If upstream deprecates or narrows these, affected extension modules migrate from the `-distro` repo to a Dash0 fork branch (the R5 fallback path).
- Dash0's Kubernetes operator consumes the bundles from their GitHub Release URLs. Any subsequent packaging — OCI image wrapping for the operator's init container — happens in the operator repository, not in this distro.
- Non-runtime-support patches carried on Dash0 fork branches stay rare in practice. The "own as little as possible" principle depends on this being true for the short-lived branches; the `dash0-main` branch is a separate, structurally-permanent surface with its own cost model.
- Upstream telemetry shape is Dash0's telemetry shape baseline for v1. The distro does not renormalize toward `dash0.*` semantic conventions at the agent — renormalization, where it exists, lives in the Dash0 ingest pipeline.
- Bundles and Dash0-branded NuGet packages are published only as GitHub Release assets on the distro's repository. `nuget.org` publication is out of scope for v1.
- Dash0 maintains forks of the upstream OpenTelemetry .NET repositories the distro depends on. v1 fork set at minimum: `dash0hq/opentelemetry-dotnet-instrumentation`. Others (`opentelemetry-dotnet-contrib`, `opentelemetry-dotnet`) are added as the need to patch arises. The `dash0-main` branch exists on whichever fork(s) the runtime-support patches touch — one branch per fork, all sharing the same name and reason code.
- The `<ProjectReference>` substitution mechanism has no vendor-distro precedent in the OTel .NET space. An early spike is required (M1 / Q10).
- **The runtime-support extension track is contingent on M0 proving the patch surface manageable.** The doc treats indefinite carry of the `dash0-main` branch as viable; that viability is not itself an assumption to be inherited, it is an empirical measurement M0 produces before v1 scope lock. If M0 finds the multi-target patch surface unsustainable, the runtime-support track collapses and the distro's identity reverts to Java-parity scope.
- Native profiler build complexity is significant: multi-platform (Linux glibc, Linux musl, macOS), multi-architecture (x64, arm64), CMake-driven C++. Windows is excluded from v1, removing the MSVC/Windows-SDK toolchain from the build matrix. The distro inherits the remaining complexity from upstream and does not attempt to simplify it in v1. M0 additionally validates the native profiler builds against `.NET 6`'s CoreCLR headers on Linux.
- The runtime-support policy is a customer-facing commitment surface. Once published, contracting the policy (retiring a runtime earlier than announced) has customer-migration implications that the release process must respect.
- The release workflow requires `id-token: write` permission for Sigstore keyless signing; release-write permissions are restricted to a small named set of publishing workflows.
- Upstream OpenTelemetry .NET publishes signed NuGet packages to `nuget.org`. The SBOM includes those coordinates so the upstream trust chain remains inspectable.

---

## Outstanding Questions

**To settle in Phase 2.5 (this brainstorm's remaining checkpoints)**

- Q_S1. **Resolved 2026-07-17.** R21's supported set is **`.NET 6` and above, all versions** (`net6.0`, `net7.0`, `net8.0`, `net9.0`, `net10.0`, and all future majors). `.NET Framework 4.6.2+` and `netcoreapp3.1` are both **OUT**. Rationale: Dash0's v1 .NET distro is Linux-first with no Windows support, which structurally excludes `.NET Framework` (Windows-only). `netcoreapp3.1` is dropped because it falls below the `.NET 6+` policy floor; the customer cohort stranded on out-of-support runtimes today is overwhelmingly on `.NET 6` rather than 3.1 (3.1 users mostly migrated during 3.1's supported window). The runtime-support surface is a single multi-target branch (`dash0-main`), producing one build per RID that covers all supported runtimes.
- Q_S2. Are we shipping Dash0-branded NuGet packages in v1, or bundle-only? Proposed: minimal NuGet set (Loader + AutoInstrumentation config entry point) shipped v1, published to GitHub Release only. Cost: added packaging complexity. Reason: unblocks library-mode users who cannot use the profiler (constrained runtimes, self-contained deployments).

**Pre-v1 feasibility milestones (must complete before v1 scope lock)**

- M0. **Runtime-support feasibility spike for `.NET 6` on Linux.** On the current upstream `main` of `open-telemetry/opentelemetry-dotnet-instrumentation`, perform the actual patch work required to make `net6.0` build, load the profiler on Linux (glibc + musl, x64 + arm64), execute IL rewriting, and emit telemetry from a representative instrumentation set. This is separate from M1 (Q10 — the substitution-mechanism spike) and separate from Q_R25 (ongoing CI validation).

  **Instrumentation set (representative, not exhaustive):**
  - `AspNetCore` — HTTP server, the dominant signal for most customers.
  - `HttpClient` — HTTP client, ubiquitous.
  - `SqlClient` — DB client with per-provider variants; likely to touch runtime-version-specific `System.Data` surface.
  - `EntityFrameworkCore` — ORM with heavy IL rewriting.
  - `Grpc.Net.Client` — modern async surface, likely to touch newer BCL APIs.
  - `StackExchangeRedis` — third-party library instrumentation with its own async surface.
  - One resource detector (candidate: `Container` or `Process`) — checks that the resource-detector pattern also holds on `.NET 6`.

  **Measurements produced by M0:**
  - **Patch surface size per instrumentation** — count of `#if NET8_OR_GREATER` guards, backported API implementations, or upstream-code modifications required to make each instrumentation compile and run on `.NET 6`.
  - **Root-cause categorization of each patch** — is it (a) a `<TargetFrameworks>` addition only, (b) BCL API not available on `.NET 6` (`TimeProvider`, span-based APIs, new `Activity` surface, etc.) requiring polyfill or `#if` guard, (c) upstream instrumentation adopts a source-level pattern that older C# language versions cannot express, (d) native profiler build fails against `.NET 6`'s CoreCLR headers on Linux, (e) other?
  - **Rebase-cost projection** — from upstream release history, sample 2–3 recent upstream releases and re-run the patches on top of each. Measure conflict rate and per-conflict resolution effort. Extrapolate to "expected patch-surface delta per upstream release for `.NET 6`".

  **M0 gate criteria for `.NET 6` inclusion in R21:**
  - Every instrumentation in the representative set successfully emits shape-parity telemetry (per R24) on `.NET 6`.
  - Aggregate patch surface fits within a threshold Dash0 commits to sustain — proposed candidate: fewer than ~20 API-backport sites per rebase across the full instrumentation catalogue, or an equivalent time budget (e.g., a single engineer-week per upstream release for `.NET 6` support). The specific threshold is set during M0 based on measured cost.
  - No root-cause category (d) — a native-profiler-build failure against `.NET 6`'s CoreCLR headers is a structural blocker that cannot be worked around at the managed layer.

  **M0 failure modes and responses:**
  - `.NET 6` fails the gate → `.NET 6` drops out of R21 for v1; v1 loses its headline value-add. Fold back to Java-parity scope and re-brainstorm the .NET distro's identity before proceeding — there is no fallback extended runtime in v1 (netfx and netcoreapp3.1 are already OUT per Q_S1).
  - M0 output shifts the runtime-support policy (R22) extension-window default from two years to a specific number based on measured patch cost.

  **M0 timing:** M0 runs before v1 planning finalizes. M0's output feeds R22's extension-window default and Q13's runtime-support SLA sizing. M1 (substitution-mechanism spike, was Q10) can run in parallel with M0 — the two spikes have no shared dependency.

- M1. Substitution-mechanism spike (was Q10; renamed to M1 for parity). Verify on a trivial patched module that MSBuild `<ProjectReference>` substitution against a fork checkout produces correct bundle contents. See Q10 detail below.

**Deferred to Planning**

- Q1. Concrete convention for the canonical-application telemetry-diff test (R16, R17): which reference apps, which signals, tolerance rules, CI integration across the R21 runtime matrix.
- Q2. Soft threshold on short-lived fork-branch count that triggers a design revisit ("if Dash0 is carrying more than N active short-lived branches, the extension-first premise has drifted"). Separate soft threshold for the `.NET 6` runtime-support branch's patch cost (retiring `.NET 6` when patch cost outweighs signal).
- Q3. Duplicate-instrumentation reconciliation process (F3): detection mechanism during rebase, verification for telemetry-shape parity, how a shape transition is communicated.
- Q4. Rebase workflow tooling on the Dash0 forks — automation for fast-forwarding upstream-tracking branches, batch-rebasing feature branches (short-lived + long-lived), detecting merged short-lived branches, and refreshing the `-distro` repo's pins.
- Q5. v1 test matrix — .NET runtimes × Linux distro (glibc + musl) × architecture (x64 + arm64) × reference applications, plus macOS (x64 + arm64) as a lighter dev-machine smoke matrix. Larger than the Java distro's matrix due to per-RID native profiler and `.NET 6` runtime-support validation.
- Q6. Concrete class-signature and loaded-assembly probes per framework in the R7 guard set — specific class names and version ranges for the ASP.NET Core hosted-SDK case, the OpenTelemetry .NET SDK auto-init pattern, and each recognized peer profiler GUID.
- Q7. GitHub Release asset URL pattern and versioning scheme. The operator repo pins whichever URL pattern is chosen, so it needs to be stable from v1 onward.
- Q8. Fork branching model naming: upstream-tracking branch (`dash0-upstream` vs reusing `main`), short-lived feature-branch convention (`dash0/<slug>`), long-lived runtime-support branch name (proposed `dash0-main`), and where the branching model is documented so it applies uniformly across all forks.
- Q9. Fork-checkout mechanism for `<ProjectReference>` substitution: git submodule under `-distro/forks/`, a `.dash0/fork-pins.json` config, or another mechanism. Each has different UX and CI-tooling tradeoffs.
- Q10. Substitution-mechanism spike (M1): verify on a trivial patched module that MSBuild `<ProjectReference>` substitution against a fork checkout produces correct bundle contents — no package-ID collisions, plugin discovery still resolves, Assembly Load Context isolation holds, and the assembled bundle passes the drop-in check (R2). Foundational verification before the model is committed broadly. Runs in parallel with M0.
- Q11. Concrete v1 resource-detector set for R6: which upstream `OpenTelemetry.ResourceDetectors.*` packages to enable by default, which Dash0-authored detectors to add (candidates: Dash0-environment env-var reader, Kubernetes-workload detector when running under the operator), and what defines the "opinionated additions".
- Q12. Concrete v1 acceptance example for R6 (paired with Q11).
- Q13. Rebase-conflict resolution SLA and escalation path per R9, with a **stricter SLA for the `.NET 6` runtime-support branch** — a conflict on a short-lived branch blocks one release; a conflict on the runtime-support branch blocks every subsequent release for `.NET 6`, cascading into a customer-migration event if unresolved.
- Q14. R10 metadata-file schema, including the `reason=runtime-support-<version>` field for long-lived branches.
- Q_R13. The specific MSBuild mechanism for `<PackageVersion>` → `<ProjectReference>` substitution across a solution with hundreds of upstream package references. Candidates: `Directory.Packages.props` conditional overrides, MSBuild target-level `<ProjectReference>` injection, custom SDK. Trade-offs on maintainability and CI-tooling explored in the Q10 spike.
- Q_R22a. Concrete runtime-support extension window: two years past Microsoft end-of-support is a defensible default, but three years for LTS runtimes and one year for STS may be a better fit. Set during v1 based on customer-cohort analysis.
- Q_R25. Runtime-support test matrix: how does CI validate that the `dash0-main` branch's rebased patches still produce a working multi-target bundle covering every runtime in R21 on Linux after each upstream release? Candidate: a reduced canonical-application matrix (one representative app per extended runtime) that runs on every rebase, plus a full matrix that runs on release cut.

---

## Sources / Research

- `distro.md` from the sibling Java distro repo (`opentelemetry-java-distro/distro.md`) — the shared repo-root principles document. Load-bearing input for every Key Decision above; the .NET distro inherits the principles and adds the runtime-support commitment.
- Sibling brainstorm: `opentelemetry-java-distro/docs/brainstorms/2026-07-14-dash0-otel-java-distro-requirements.md` — the structural template. Deviations from Java are documented in this document's Key Decisions.
- Upstream OpenTelemetry .NET repositories the distro consumes and rebases against:
  - `open-telemetry/opentelemetry-dotnet-instrumentation` — auto-instrumentation with CLR profiler + startup hook; the primary Dash0-forked repo. Publishes per-RID bundles as GitHub Release assets and companion NuGet packages under `OpenTelemetry.AutoInstrumentation.*`.
  - `open-telemetry/opentelemetry-dotnet-contrib` — additional instrumentations, resource detectors, exporters (locally cloned at `/Users/mmanciop/git/opentelemetry-dotnet-contrib`). Layout confirms the `src/OpenTelemetry.Instrumentation.*`, `src/OpenTelemetry.ResourceDetectors.*`, `src/OpenTelemetry.Exporter.*` conventions the distro's R12 layout mirrors.
  - `open-telemetry/opentelemetry-dotnet` — core SDK and API under `OpenTelemetry.*` package IDs. Forked only if SDK-level patches become necessary for runtime-support extension.
- Dash0 Kubernetes Operator — the primary K8s delivery vehicle for the distro (`dash0hq/dash0-operator`). Consumes bundles from GitHub Release; env-var-based injection.
- Existing vendor OTel .NET / commercial .NET APM agents — Datadog `dd-trace-dotnet`, New Relic .NET agent, AppDynamics, Dynatrace OneAgent for .NET. All support wide runtime matrices via profiler-based instrumentation; each has its own runtime-support policy and patch queue. Dash0's runtime-support extension follows this precedent from the commercial APM space, not from the OTel vendor-distro space (Splunk's OTel .NET distro follows upstream's runtime set).
- Microsoft .NET lifecycle: `.NET 6` LTS ended November 2024; `.NET 7` STS ended May 2024; `.NET 8` LTS through November 2026; `.NET 9` STS through May 2026; `.NET 10` LTS through 2028. These dates drive the runtime-support policy's extension windows and the R22 trigger schedule.
- Upstream .NET Auto-Instrumentation injection env vars documented at `open-telemetry/opentelemetry-dotnet-instrumentation`'s README: `CORECLR_ENABLE_PROFILING`, `CORECLR_PROFILER`, `CORECLR_PROFILER_PATH`, `DOTNET_STARTUP_HOOKS`, `OTEL_DOTNET_AUTO_HOME`. These form the injection contract R2 preserves.
- Dash0 knowledge base signal: current Dash0 .NET posture is "wrap the upstream bundle" via the operator; there is no Dash0-authored .NET instrumentation shipped today.
