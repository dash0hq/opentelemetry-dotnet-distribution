# Dash0 OpenTelemetry Distribution for .NET

An opinionated, zero-code [OpenTelemetry](https://opentelemetry.io) distribution
for .NET on Linux. It packages Dash0's fork of upstream
[`opentelemetry-dotnet-instrumentation`](https://github.com/dash0hq/opentelemetry-dotnet-instrumentation)
(branch `dash0-main`) into ready-to-mount `x64` and `arm64` tracer-home
archives, published as GitHub Releases and consumable straight from
[Dash0](https://www.dash0.com).

## Intended use

This distribution is primarily intended to be injected into .NET workloads by
the [Dash0 Kubernetes operator](https://github.com/dash0hq/dash0-operator),
which instruments them without code changes. Each release attaches two
tarballs (one per Linux architecture) that extract to the upstream
`bin/tracer-home` layout and are wired in via the standard CoreCLR profiler
environment variables:

```sh
export OTEL_DOTNET_AUTO_HOME=/path/to/tracer-home
. "${OTEL_DOTNET_AUTO_HOME}/instrument.sh"
```

`instrument.sh` sets `CORECLR_ENABLE_PROFILING`, `CORECLR_PROFILER`,
`CORECLR_PROFILER_PATH_64`, `DOTNET_STARTUP_HOOKS`, and the
`DOTNET_ADDITIONAL_DEPS` / `DOTNET_SHARED_STORE` chain. Any .NET process
launched from that shell is instrumented. In a Kubernetes pod the Dash0
operator handles the mount and env-var injection.

Upstream sources are **not** vendored here. This repository's job is to
fetch upstream at a pinned commit, patch known-stale build steps at CI time,
build, and publish signed release tarballs.

## Requirements

- **Linux** on **x86_64** or **aarch64** (glibc ≥ 2.23; both native
  libraries are built inside Ubuntu 16.04 containers).
- **.NET** runtime **6.0 – 9.0**. The upstream instrumentation supports .NET
  Framework as well, but this distribution ships Linux-only archives and
  does not include the Windows-only `netfx/` subtree.

## Installation

Download and extract a release archive. Replace `<tag>` with the target
release (e.g. `v0.1.0`) and `<arch>` with `x64` or `arm64`:

```sh
mkdir -p /opt/dash0/otel-dotnet-auto
curl -fsSL \
  https://github.com/dash0hq/opentelemetry-dotnet-distribution/releases/download/<tag>/dash0-opentelemetry-dotnet-instrumentation-linux-<arch>.tar.gz \
  | tar -xz -C /opt/dash0/otel-dotnet-auto

export OTEL_DOTNET_AUTO_HOME=/opt/dash0/otel-dotnet-auto
. "${OTEL_DOTNET_AUTO_HOME}/instrument.sh"
```

Each archive extracts to:

```
net/                          # net6.0 – net9.0 managed assemblies + StartupHook + Loader
AdditionalDeps/               # deps.json overrides per .NET version
linux-<arch>/                 # OpenTelemetry.AutoInstrumentation.Native.so
integrations.json
instrument.sh                 # sets CoreCLR profiler env vars
LICENSE, NOTICE
```

Each release records the exact upstream commit SHA the archives were built
from.

## What it does

Sourcing `instrument.sh` turns on the .NET CoreCLR profiler API and points
the runtime at the OpenTelemetry auto-instrumentation. On process startup
the distribution:

- exports **traces, metrics, and logs** over OTLP (default: HTTP/protobuf)
  to whatever collector `OTEL_EXPORTER_OTLP_ENDPOINT` (or the Dash0
  operator's injected value) points at;
- attaches the CoreCLR profiler to the .NET process and enables all
  instrumentations upstream ships (ASP.NET Core, HttpClient, gRPC, EF Core,
  StackExchange.Redis, MongoDB, Kafka, RabbitMQ, and more — see the
  [upstream list](https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation#supported-libraries-frameworks-and-runtimes));
- adds resource attributes: upstream's defaults (process, host, SDK, plus
  any `OTEL_SERVICE_NAME` / `OTEL_RESOURCE_ATTRIBUTES`), and the
  distribution identity (`telemetry.distro.name = dash0-dotnet`,
  `telemetry.distro.version`).

## Configuration

Behavior is driven entirely by environment variables, following upstream's
[configuration reference](https://opentelemetry.io/docs/zero-code/dotnet/configuration/).
The most common ones:

| Variable | Description |
| --- | --- |
| `OTEL_DOTNET_AUTO_HOME` | **Required.** Absolute path to the extracted tracer-home directory. Must be set before sourcing `instrument.sh`. |
| `OTEL_SERVICE_NAME` | Sets `service.name` for the workload. The Dash0 operator sets this from workload metadata. |
| `OTEL_RESOURCE_ATTRIBUTES` | Additional resource attributes as `key=value` pairs. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP collector endpoint. The Dash0 operator points this at the in-cluster collector. |
| `OTEL_EXPORTER_OTLP_HEADERS` | Extra OTLP headers (e.g. an authorization token when exporting directly to Dash0 rather than through the operator's collector). |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` (default) or `grpc`. |
| `OTEL_DOTNET_AUTO_TRACES_ENABLED_INSTRUMENTATIONS` | Comma-separated allowlist of trace instrumentations. If unset, all are enabled. |
| `OTEL_DOTNET_AUTO_METRICS_ENABLED_INSTRUMENTATIONS` | Same, for metrics. |
| `OTEL_DOTNET_AUTO_LOGS_ENABLED_INSTRUMENTATIONS` | Same, for logs. |
| `OTEL_DOTNET_AUTO_LOG_DIRECTORY` | Where the profiler writes its own diagnostic logs. |
| `OTEL_SDK_DISABLED` | Set to `true` to disable telemetry export while keeping the profiler loaded. |

Any other `OTEL_*` variable that upstream honors works here too — this
distribution does not fork or rewrite the configuration surface.

## Development

Release automation and the CI patches applied to upstream at build time are
documented in [RELEASE.md](RELEASE.md).

Historic PoC and spike work is preserved on two backup branches:

- `backup/main-pre-reset` — earlier iteration of the distribution scaffold,
  per-RID bundle assembler, and the U1 substitution-mechanism spike.
- `backup/add-clientserver-build-example` — an initial client-server app
  that drove the agent-binaries build.

Neither is on the release path.

## License

Distribution scaffolding in this repository (workflow, docs) is under the
same license as the upstream project. Published archives include the
upstream project's `LICENSE` and `NOTICE` files.
