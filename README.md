# opentelemetry-dotnet-distribution

Dash0's release channel for the OpenTelemetry .NET automatic instrumentation.

This repository builds and publishes signed, versioned Linux `x64` and `arm64`
tracer-home archives from
[`dash0hq/opentelemetry-dotnet-instrumentation`](https://github.com/dash0hq/opentelemetry-dotnet-instrumentation)
(branch `dash0-main`), so downstream consumers — primarily the
[Dash0 Operator](https://github.com/dash0hq/dash0-operator) — can pull a pinned,
reproducible bundle without cloning or building the upstream sources themselves.

Upstream sources are **not** vendored here. The pipeline fetches upstream at a
specific commit, patches known-stale build steps at CI time, builds, and
publishes.

## Release artifacts

Each release attaches two archives:

| Asset | Runtime target | glibc floor | Native build host |
| --- | --- | --- | --- |
| `dash0-opentelemetry-dotnet-instrumentation-linux-x64.tar.gz` | Linux x86_64 | ≥ 2.23 | Ubuntu 16.04 container |
| `dash0-opentelemetry-dotnet-instrumentation-linux-arm64.tar.gz` | Linux aarch64 | ≥ 2.35 | GitHub-hosted `ubuntu-22.04-arm` |

Each archive contains the upstream `bin/tracer-home` layout:

```
net/                           # net6.0 – net9.0 managed assemblies + StartupHook + Loader
AdditionalDeps/                # deps.json overrides per .NET version
linux-<arch>/                  # OpenTelemetry.AutoInstrumentation.Native.so
integrations.json
instrument.sh                  # sets CoreCLR profiler env vars
LICENSE, NOTICE
```

`netfx/` is Windows-only and is not included in these Linux archives. Release
notes record the exact upstream commit SHA the archives were built from.

## Consuming a release

Extract into a directory and source `instrument.sh` to set the CoreCLR profiler
environment variables for the current shell (or its child processes):

```sh
mkdir -p /opt/dash0/otel-dotnet-auto
curl -fsSL \
  https://github.com/dash0hq/opentelemetry-dotnet-distribution/releases/download/<tag>/dash0-opentelemetry-dotnet-instrumentation-linux-x64.tar.gz \
  | tar -xz -C /opt/dash0/otel-dotnet-auto

export OTEL_DOTNET_AUTO_HOME=/opt/dash0/otel-dotnet-auto
. "${OTEL_DOTNET_AUTO_HOME}/instrument.sh"
```

Any .NET process launched from that shell will be instrumented. In a Kubernetes
pod, the Dash0 Operator handles the mount and env-var injection; the archive
layout is what its instrumentation-image build consumes.

## Cutting a release

See [`RELEASE.md`](RELEASE.md) for the maintainer release process — the
one-click *Prepare Release* flow, the patches applied to upstream, and the
manual-override path.

## Repository history

Two branches preserve pre-pipeline PoC and spike work:

- `backup/main-pre-reset` — earlier iteration of the distribution scaffold,
  per-RID bundle assembler, and the U1 substitution-mechanism spike.
- `backup/add-clientserver-build-example` — an initial client-server app that
  drove the agent-binaries build.

Neither is on the release path; keep them for reference only.

## License

Distribution scaffolding in this repository (workflow, docs) is under the same
license as the upstream project. Published archives include the upstream
project's `LICENSE` and `NOTICE` files.
