---
date: 2026-07-17
topic: u1-m1-substitution-mechanism-spike
status: PASSED — KTD1 mechanism verified end-to-end
verdict: proceed to U2
plan: docs/plans/2026-07-17-001-feat-dash0-otel-dotnet-distro-plan.md
---

# U1 (M1) Substitution-Mechanism Spike — Results

## Verdict

**KTD1 verified end-to-end. Proceed to U2.**

MSBuild `<ProjectReference>` substitution swaps an upstream `<PackageReference>` cleanly against a git-checked-out fork. Both Case A (additive) and Case B (modifies existing + cross-project dependency) pass. The fallback to Alternative A1 (local NuGet feed) is **not** required.

## What was tested

The load-bearing question: **can MSBuild substitute a `<PackageReference>` to an upstream OpenTelemetry package for a `<ProjectReference>` into a fork checkout, producing a working compiled output containing the fork's patches?**

The spike consumer (`spike/consumer/consumer.csproj`) toggles between two consumption modes via the `UseForkCheckout` MSBuild property:

- `UseForkCheckout=true`: `<ProjectReference>` into `spike/upstream-checkout/src/OpenTelemetry.AutoInstrumentation/OpenTelemetry.AutoInstrumentation.csproj`
- `UseForkCheckout=false`: `<PackageReference Include="OpenTelemetry.AutoInstrumentation.Runtime.Managed" />` from `nuget.org` at v1.16.0

The consumer reflects on `OpenTelemetry.AutoInstrumentation.Constants.DistributionAttributes` at runtime and prints Case A (new field presence) and Case B (existing field value).

## Patches applied to the fork checkout

Branch: `dash0/u1-spike-patches` on `spike/upstream-checkout/`, off upstream commit `72fd643`.

Single-file edit in `src/OpenTelemetry.AutoInstrumentation/Constants.cs`:

```diff
     public static class DistributionAttributes
     {
         public const string TelemetryDistroNameAttributeName = "telemetry.distro.name";
-        public const string TelemetryDistroNameAttributeValue = "opentelemetry-dotnet-instrumentation";
+        public const string TelemetryDistroNameAttributeValue = "dash0-opentelemetry-dotnet-instrumentation";
         public const string TelemetryDistroVersionAttributeName = "telemetry.distro.version";
+
+        public const string Dash0SpikeMarker = "u1-case-a-marker";
     }
```

- **Case A (additive)**: new `Dash0SpikeMarker` constant.
- **Case B (modifies existing)**: `TelemetryDistroNameAttributeValue` changed from the upstream value to a Dash0-prefixed value.
- **Cross-project dependency exercise**: the substituted `OpenTelemetry.AutoInstrumentation.csproj` has a `<ProjectReference>` to `OpenTelemetry.AutoInstrumentation.PluginApi.csproj` in the same fork checkout. If substitution didn't resolve inner ProjectReferences correctly, `PluginApi` would fail to build or link.

## Results

### Substituted build (`UseForkCheckout=true`)

```
Loaded assembly: OpenTelemetry.AutoInstrumentation, Version=0.0.0.0, ...
Location: .../spike/consumer/bin/Debug/net8.0/OpenTelemetry.AutoInstrumentation.dll

=== Case A (additive: Dash0SpikeMarker) ===
  Field present: True
  Value: u1-case-a-marker
  Expected: u1-case-a-marker

=== Case B (modified: TelemetryDistroNameAttributeValue) ===
  Value: dash0-opentelemetry-dotnet-instrumentation
  Upstream default: opentelemetry-dotnet-instrumentation
  Dash0 expected:   dash0-opentelemetry-dotnet-instrumentation

Case A verdict: PASS
Case B verdict: PASS
```

Assembly version `0.0.0.0` (build-time version, no MinVer tag on the fork branch) — as expected for a locally-built assembly.

### Control build (`UseForkCheckout=false`, NuGet package)

```
Loaded assembly: OpenTelemetry.AutoInstrumentation, Version=1.0.0.0, ...
Location: .../spike/consumer/bin/Debug/net8.0/OpenTelemetry.AutoInstrumentation.dll

=== Case A (additive: Dash0SpikeMarker) ===
  Field present: False
  Value: <null>

=== Case B (modified: TelemetryDistroNameAttributeValue) ===
  Value: opentelemetry-dotnet-instrumentation

Case A verdict: FAIL
Case B verdict: FAIL
```

Assembly version `1.0.0.0` — the released NuGet package (v1.16.0's assembly is stamped `1.0.0.0`). Case A absent, Case B has upstream default. **Both intentional failures**: they prove the NuGet-referenced build is unmodified upstream, which is exactly what we want the control to show.

## What the spike proved

1. **MSBuild resolves cross-project ProjectReferences under substitution.** The consumer references only `OpenTelemetry.AutoInstrumentation`; the fork's csproj internally references `OpenTelemetry.AutoInstrumentation.PluginApi`; both got restored and built without any additional configuration. No package-ID collision. No missing-assembly errors.

2. **Source generators propagate cleanly.** The `SourceGenerators.csproj` (transitively referenced) built and its output was consumed by the analyzer pipeline when compiling the substituted `OpenTelemetry.AutoInstrumentation.csproj`.

3. **Assembly identity swaps cleanly.** The build output contains one and only one `OpenTelemetry.AutoInstrumentation.dll` — the fork-built one under substitution, the NuGet one otherwise. No parallel installations, no ambiguous binding.

4. **Central Package Management (CPM) tolerates the swap.** The `<PackageVersion>` entry in `spike/Directory.Packages.props` for `OpenTelemetry.AutoInstrumentation.Runtime.Managed` remains declared under both modes; the `<PackageReference>` is toggled at csproj level. No CPM error, no version-lookup warning.

5. **The property-driven swap mechanism (Directory.Build.props `<UseForkCheckout>`) works.** For U5 the plan calls for a `patched-modules.yaml`-driven mechanism; the property-driven variant here is a proper subset that validates the same MSBuild machinery. Scaling from property to YAML-driven is straightforward — an MSBuild task reads the YAML and injects the appropriate ItemGroup.

## Known gaps not tested in this spike

- **Assembly Load Context (ALC) isolation at runtime**: not tested here, because the spike consumer is a plain console app that doesn't set up the auto-instrumentation's ALC hierarchy. ALC behavior gets exercised in U3-U6 when the actual startup hook + plugin surface is wired up. **The mechanism spike's job was to prove the MSBuild swap, not to prove runtime ALC — those are orthogonal.** If ALC issues surface later, they'll manifest as bundle-runtime failures in U6+, not as substitution-mechanism issues.
- **Multi-TFM behavior**: only `net8.0` tested. The runtime-support-branch multi-target (`net6.0;net7.0;net8.0;net9.0`) is U2/U3 territory. The M0 spike already validated multi-target compilation at the fork side; combining that with the substitution mechanism is a straightforward extension.
- **Bundle assembly (tar.gz packaging, native-profiler co-pack)**: out of scope for U1. That's U3's job.
- **`patched-modules.yaml` schema**: U1 uses a simpler property toggle. U5 will drive substitution from the YAML registry.

## Environmental adjustments made (non-mechanism)

Two environment-only tweaks to the fork checkout, both matching what the M0 spike documented. These are NOT part of the mechanism validation — they'd be needed for any build against the current SDK:

- `spike/upstream-checkout/Directory.Build.props`: `<LangVersion>14.0</LangVersion>` → `<LangVersion>13.0</LangVersion>` (local SDK is 9.0.101 with Roslyn 4.12; upstream targets C# 14 which needs SDK 10).
- `spike/upstream-checkout/src/Directory.Packages.props`: `Microsoft.CodeAnalysis.CSharp` `5.3.0` → `4.12.0`, `Microsoft.CodeAnalysis.Analyzers` `5.3.0` → `3.11.0` (source generator built against Roslyn 5.3 vs current-compiler Roslyn 4.12).

Both revert cleanly on CI where SDK 10 is available.

## Feeding back to the plan

- **U2** proceeds unchanged. The M0 spike's patches promote to the real `dash0hq/opentelemetry-dotnet-instrumentation` fork on `dash0-main`, and U5 wires this same substitution mechanism through `patched-modules.yaml`.
- **U5** stays as designed — the YAML-driven registry is a proper extension of the property-driven toggle validated here.
- **The Alternatives Considered A1 fallback (local NuGet feed) does NOT need to be activated.** The plan's fail-fast condition on U1 does not trigger.

## Reproducing this spike

From the `-distro` repo root:

```bash
cd spike/consumer
dotnet build              # UseForkCheckout=true by default — substituted
dotnet run --no-build     # Case A + Case B both PASS

# Control:
rm -rf bin obj
dotnet run -p:UseForkCheckout=false  # both FAIL, confirming NuGet-referenced upstream is unmodified
```

## Files

Created:

- `spike/Directory.Build.props`
- `spike/Directory.Packages.props`
- `spike/consumer/consumer.csproj`
- `spike/consumer/Program.cs`
- `spike/SPIKE-RESULTS.md` (this file)

Fork-checkout modifications (on `dash0/u1-spike-patches` branch in `spike/upstream-checkout/`):

- `Directory.Build.props` (LangVersion — environment)
- `src/Directory.Packages.props` (Roslyn versions — environment)
- `src/OpenTelemetry.AutoInstrumentation/Constants.cs` (Case A + Case B — the actual spike patches)
