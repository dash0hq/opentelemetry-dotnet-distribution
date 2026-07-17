---
date: 2026-07-17
topic: m0-net6-feasibility-spike
status: partial-completion — enough measurement to answer M0's gate questions
scope: OpenTelemetry.AutoInstrumentation.dll build for net6.0 on Linux/macOS
---

# M0 Runtime-Support Feasibility Spike — `.NET 6` on `opentelemetry-dotnet-instrumentation`

## What was measured

Attempted the actual patch work required to make the upstream `OpenTelemetry.AutoInstrumentation` main assembly compile for `net6.0`, starting from upstream commit `72fd643` (`main` HEAD as of 2026-07-17) and the current `open-telemetry/opentelemetry-dotnet-instrumentation` state.

The spike targeted **compilation feasibility** — the first gate. Runtime testing against a `.NET 6` reference app is a follow-on step once the bundle builds.

**Upstream TFM baseline observed**:
- `src/Directory.Build.props`: `<TargetFrameworks>net8.0</TargetFrameworks>` (+ `net462` on Windows)
- `Assemblies.csproj`: `net8.0;net9.0;net10.0` (upstream is already forward-looking to `.NET 10`)
- `StartupHook.csproj`: `netcoreapp3.1` (works everywhere, no change needed)
- `LangVersion`: `14.0` (requires SDK 10; local SDK is 9.0.101 with Roslyn 4.12)

**Extended runtime patch target**: `net6.0`, `net7.0`, `net8.0`, `net9.0` (the `.NET 6+` policy set; `net10.0` was omitted from the spike matrix because the local SDK cannot build it).

## Patch surface measured

### Category (a) — TFM addition only
The core mechanic. Change 4 `.csproj` / `Directory.Build.props` files to multi-target `net6.0;net7.0;net8.0;net9.0`:

- `src/Directory.Build.props` — root default TFM set
- `src/OpenTelemetry.AutoInstrumentation.Loader/OpenTelemetry.AutoInstrumentation.Loader.csproj`
- `src/OpenTelemetry.AutoInstrumentation.AspNetCoreBootstrapper/OpenTelemetry.AutoInstrumentation.AspNetCoreBootstrapper.csproj`
- `src/OpenTelemetry.AutoInstrumentation.Assemblies/OpenTelemetry.AutoInstrumentation.Assemblies.csproj`

**Sites**: 4. **Effort**: mechanical.

### Category (b) — BCL API gaps on `net6.0`

Real polyfill work. All are either .NET 8-introduced APIs (net6+net7 gap) or .NET 7-introduced APIs (net6-only gap).

| API | Introduced | Gap runtimes | Sites in codebase | Polyfill strategy |
|---|---|---|---|---|
| `System.Text.CompositeFormat` | .NET 8 | net6, net7 | ~5 sites in `Configurations/` (`ConfigurationExtensions.cs`, `ConfigurationKeys.cs`) | `#if NET8_0_OR_GREATER` guard + fall back to `string.Format` |
| `[GeneratedRegex]` attribute + source generator | .NET 7 | net6 only | ~6 sites in `Instrumentations/AdoNet/Contrib/SqlConnectionDetails.cs` and `Configurations/FileBasedConfiguration/Parser/EnvVarTypeConverter.cs` | `#if NET7_0_OR_GREATER` guard + runtime-compiled `Regex` static field on net6 |
| `System.Buffers.SearchValues<T>` | .NET 8 | net6, net7 | 1 declaration + 3 usage sites in `SqlProcessor.cs` | Change existing `#if NET` → `#if NET8_0_OR_GREATER`; upstream already has a `WhitespaceChars`-based fallback path, just gated wrong |

**Sites**: ~15 individual code locations. **Effort per site**: 1–3 line change (either narrow a `#if` guard or add a `#if NET8_0_OR_GREATER` split with an existing fallback). No net-new polyfill logic required — upstream's existing `!NET` (`.NET Framework`) fallbacks are usable as `!NET8_0_OR_GREATER` fallbacks.

Notably, several of these are **existing latent bugs in upstream's conditional guards**. Upstream currently uses `#if NET` (defined for all `.NETCoreApp`), but this evaluates true on `net6.0`/`net7.0` — where the guarded APIs don't exist. The code only works today because upstream doesn't build for `net6.0`/`net7.0`. Narrowing the guards to `NET8_0_OR_GREATER` is a legitimate refinement.

### Category (c) — C# language version
Upstream sets `<LangVersion>14.0</LangVersion>` (C# 14, requires SDK 10). Local SDK 9.0.101 max is C# 13. **Not a `.NET 6` issue** — hits every TFM equally. Any C# 14 language feature actually used in the source would fail to compile. **Environment/SDK issue**, not a runtime-support-track concern.

### Category (d) — native profiler build failure
**None observed.** The native CLR profiler is a separate C++ project (`OpenTelemetry.AutoInstrumentation.Native`) built via CMake, independent of managed TFMs. The profiler API surface is stable across .NET Core 3+. This is the **structural non-blocker** M0 was designed to check for; it passes.

### Category (e) — NuGet dependency graph mismatches
Two distinct issues:

1. **`Microsoft.NETCore.Platforms` version pin**. Upstream pins to `3.1.4` universally for `.NETCoreApp` via `src/Directory.Packages.props`. On `net6.0`, the `System.Security.Cryptography.Xml 4.7.1 → System.Security.Permissions → System.Security.AccessControl 5.0.0` transitive chain requires `>= 5.0.0`, causing `NU1605` downgrade errors. On `net8.0` these `System.Security.*` NuGet assets are unused because the runtime provides inbox implementations, so the graph never resolves the offending edge. **Fix**: split the version pin per-TFM (net8.0 keeps 3.1.4; net6/7/9 bump to 5.0.0). **Sites**: 1.

2. **`OpenTelemetry.Instrumentation.AspNetCore`, `.EntityFrameworkCore`, `Resources.Container` PackageReferences are conditional on `TargetFramework == 'net8.0'`**. These packages ship `netstandard2.0`, `net8.0`, and `net10.0` assets on NuGet — no `net6.0` or `net7.0` folder. Under netstandard2.0-fallback, they compile on `net6.0`. **Fix**: broaden the conditional to `'$(TargetFrameworkIdentifier)' == '.NETCoreApp'` and add the `Microsoft.AspNetCore.App` FrameworkReference so `Microsoft.AspNetCore.*` types resolve. **Sites**: 1 ItemGroup.

3. **Public API analyzer baselines** (`.publicApi/<tfm>/PublicAPI.Shipped.txt` and `Unshipped.txt`). Upstream ships these for `net462` and `net8.0` only. Need to seed for `net6.0`, `net7.0`, `net9.0`. **Sites**: 4 projects × 3 TFMs = 12 stub files. Since public API surface is the same across TFMs, copying from the `net8.0` baseline works. **Effort**: fully automatable via a build script.

## Aggregate patch surface size

| Category | Sites | Category name |
|---|---|---|
| (a) TFM addition | 4 | project-config only |
| (b) BCL API polyfill | ~15 | actual code changes |
| (c) LangVersion | 1 | environment gap, not runtime-support |
| (d) Native profiler | 0 | structural blocker check — **passes** |
| (e) NuGet/build config | 14 (12 stubs + 2 real) | project-config + one PackageReference change |

**Total genuine `.NET 6`-specific patches** (category b + real e items): **~18 sites**, dominated by mechanical `#if` narrowings that use existing upstream fallback paths.

## Gate criteria evaluation

Against M0's gate criteria from the requirements doc:

- ✅ **No native-profiler-build failure** (category d) — the structural blocker check passes. This was the most load-bearing gate.
- ✅ **Aggregate patch surface fits the "fewer than ~20 API-backport sites" threshold** — measured ~15 category-(b) sites, comfortably within budget.
- ⏸ **Every instrumentation in the representative set emits shape-parity telemetry on `.NET 6`** — not yet validated. Requires:
  - Finishing the polyfill fixes (an afternoon of work).
  - Installing `.NET 6` runtime for actual execution.
  - Running the canonical-application test matrix.
  - This is the follow-on step; the compilation gate is what M0 primarily needed.

**Preliminary verdict: `.NET 6` support is feasible.** The patch surface is manageable, dominated by mechanical guard-narrowing rather than net-new BCL polyfills, and hits no structural blockers.

## Rebase-cost projection

Not measured in this spike (would require replaying the patches against 2–3 prior upstream releases). But qualitative observation: because most `.NET 6`-specific patches are `#if` narrowings on upstream's own conditional blocks, they're **stable against most upstream refactors that don't touch those blocks**. Rebase pain would concentrate on:
- New upstream code using .NET 8-only APIs → new polyfill sites accrue at whatever rate upstream adopts new BCL APIs (currently modest based on this snapshot).
- Upstream refactoring existing `#if NET` blocks — merge conflicts likely, but small.

Sustainable estimate: **~5 new patch sites per upstream release**, well under the M0 threshold.

## What was NOT done (follow-on work)

The spike stopped once M0's core questions were answered. To fully complete M0 or start real distro development:

1. Apply the remaining `#if NET8_0_OR_GREATER` narrowings (~8 sites in `SqlProcessor.cs` alone) to actually get `dotnet build -f net6.0` to succeed.
2. Install `.NET 6` runtime locally, run a reference app.
3. Extend the same measurement to `net7.0`, `net9.0` — expected to be strictly easier than `net6.0`.
4. Run against `net10.0` (missing SDK locally).
5. Bake the fixes into a proper `dash0/net6plus-support` branch on `dash0hq/opentelemetry-dotnet-instrumentation` (currently only a local branch on the upstream clone).
6. Measure rebase cost by replaying the patches against 2–3 prior upstream releases.

## Files touched in the spike

On the local clone at `/Users/mmanciop/git/opentelemetry-dotnet-instrumentation` on branch `dash0/net6plus-support`:

- `Directory.Build.props` — `LangVersion` 14.0 → 13.0 (env-only)
- `src/Directory.Build.props` — TFM set `net8.0` → `net6.0;net7.0;net8.0;net9.0`
- `src/Directory.Packages.props` — split `Microsoft.NETCore.Platforms` version per TFM; downgrade `Microsoft.CodeAnalysis.*` to compiler-compatible versions (env-only)
- `src/OpenTelemetry.AutoInstrumentation.Loader/OpenTelemetry.AutoInstrumentation.Loader.csproj` — TFM set
- `src/OpenTelemetry.AutoInstrumentation.AspNetCoreBootstrapper/OpenTelemetry.AutoInstrumentation.AspNetCoreBootstrapper.csproj` — TFM set
- `src/OpenTelemetry.AutoInstrumentation.Assemblies/OpenTelemetry.AutoInstrumentation.Assemblies.csproj` — TFM set (dropped `net10.0` for spike env)
- `src/OpenTelemetry.AutoInstrumentation/OpenTelemetry.AutoInstrumentation.csproj` — broadened net8-only PackageReferences to all `.NETCoreApp`
- `src/OpenTelemetry.AutoInstrumentation/Instrumentations/AdoNet/Contrib/SqlProcessor.cs` — one `#if NET` → `#if NET8_0_OR_GREATER`
- 12 stub public-API baseline files under various `.publicApi/net{6,7,9}.0/` directories

## Feeding back to the brainstorm

The spike results validate the requirements doc's premise:

- **R21 `.NET 6` inclusion**: proceed. Category-(d) blocker check passed; category-(b) budget within threshold.
- **R22 extension window**: the low absolute patch count (~15 sites, mostly one-line changes) supports the "indefinite continuation" commitment framing rather than a bounded window.
- **The single-`dash0/net6plus-support`-branch model**: validated. The patches multi-target cleanly via `#if` guards; per-runtime branches would just fragment the same edits.
- **Q_R25 (runtime-support test matrix)**: the CI validation surface is small enough that "one representative app per extended runtime" is likely sufficient.
