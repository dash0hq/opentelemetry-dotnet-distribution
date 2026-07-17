# Dash0 OpenTelemetry .NET Distro

Dash0-branded distribution of the OpenTelemetry .NET auto-instrumentation.
Ships per-RID bundles + a small NuGet set as drop-in replacements for the
[upstream OpenTelemetry .NET Auto-Instrumentation](https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation).

## Status

**Pre-release / PoC.** The design is settled ([requirements](docs/brainstorms/2026-07-17-dash0-otel-dotnet-distro-requirements.md), [plan](docs/plans/2026-07-17-001-feat-dash0-otel-dotnet-distro-plan.md)); implementation is in progress. Not yet consumed by the Dash0 Kubernetes operator.

## What it does

The distro carries three payoffs on top of upstream:

- **`.NET 6+` runtime support policy.** Dash0 commits to every .NET major version from `.NET 6` upward (`net6.0`, `net7.0`, `net8.0`, `net9.0`, `net10.0`, and all future majors) regardless of Microsoft's or upstream's support windows. As runtimes hit upstream's drop, Dash0 assumes patch responsibility via the [`dash0-main`](https://github.com/dash0hq/opentelemetry-dotnet-instrumentation/tree/dash0-main) branch of Dash0's fork of upstream instrumentation. M0 spike ([outcome](docs/spikes/2026-07-17-m0-net6-feasibility.md)) validated `.NET 6` inclusion.
- **Activation guards, augmented resource detection, flush-on-shutdown.** Opinionated additions layered as an `OTEL_DOTNET_AUTO_PLUGINS`-loaded plugin. Implemented in later units.
- **Shipping cadence independent of upstream merge latency.** Dash0-authored instrumentations ship in the distro on Dash0's schedule; where feasible they're also PR'd upstream and drop out of the distro on merge.

## Scope

- **In v1:** Linux (glibc + musl, x64 + arm64) production bundles + macOS (x64 + arm64) dev-only bundles + minimal Dash0-branded NuGet packages. `.NET 6+`.
- **Out of v1:** Windows, `.NET Framework`, `.NET Core 3.1`, Native AOT, `nuget.org` publication, SLSA level 3. See the plan's Scope Boundaries for the full list.

## Building

Prerequisites:
- `.NET SDK 9.0.x` (SDK 10 preferred once available for `net10.0` targeting).
- `git clone --recurse-submodules` — the fork submodule under `forks/` is set up in [U4](docs/plans/2026-07-17-001-feat-dash0-otel-dotnet-distro-plan.md).

Assemble a bundle for a specific RID:

```sh
dotnet run --project build/Build -- --rid linux-x64 --version 0.0.1-dev
```

The bundle lands at `artifacts/dash0-opentelemetry-dotnet-autoinstrumentation-<rid>.tar.gz`.

Supported RIDs: `linux-x64`, `linux-arm64`, `linux-musl-x64`, `linux-musl-arm64`, `osx-x64`, `osx-arm64`.

## Using the bundle

Extract, source the activation script, run your .NET app:

```sh
tar xzf dash0-opentelemetry-dotnet-autoinstrumentation-linux-x64.tar.gz
cd dash0-opentelemetry-dotnet-autoinstrumentation-linux-x64
export CORECLR_PROFILER_PATH="$(pwd)/linux-x64/Dash0.OpenTelemetry.AutoInstrumentation.Native.so"
. ./instrument.sh
dotnet /path/to/your-app.dll
```

Injection env-var contract is unchanged from upstream — a bundle produced here is a drop-in for the upstream bundle at the `OTEL_DOTNET_AUTO_HOME` path level (per R2 in the [plan](docs/plans/2026-07-17-001-feat-dash0-otel-dotnet-distro-plan.md)).

## Repository layout

```
docs/
  brainstorms/  requirements documents (product intent, scope, decisions)
  plans/        implementation plans (technical design, U-IDs, sequencing)
  spikes/       feasibility spike reports
src/
  Dash0.OpenTelemetry.AutoInstrumentation/         Plugin loaded via OTEL_DOTNET_AUTO_PLUGINS
  Dash0.OpenTelemetry.AutoInstrumentation.Loader/  Startup hook (netcoreapp3.1)
build/
  Build/        Per-RID bundle assembler (dotnet run --project build/Build)
forks/                                              Fork submodule (added in U4)
artifacts/                                          Local build output (gitignored)
```

## Documentation

- [Requirements](docs/brainstorms/2026-07-17-dash0-otel-dotnet-distro-requirements.md) — what v1 is, and why.
- [Plan](docs/plans/2026-07-17-001-feat-dash0-otel-dotnet-distro-plan.md) — how v1 is built, U1 through U15.
- [M0 spike](docs/spikes/2026-07-17-m0-net6-feasibility.md) — `.NET 6` runtime-support feasibility measurement.
- [U1 spike](spike/SPIKE-RESULTS.md) — MSBuild `<ProjectReference>` substitution mechanism validation (KTD1).

## License

Apache License 2.0 — see [LICENSE](LICENSE).
