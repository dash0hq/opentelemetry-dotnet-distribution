# dash0-opentelemetry-dotnet-distribution

Builds Linux `x64` and `arm64` binaries of Dash0's fork of
[`opentelemetry-dotnet-instrumentation`](https://github.com/dash0hq/opentelemetry-dotnet-instrumentation)
(branch `dash0-main`) and publishes them as GitHub Releases for the Dash0 Operator to consume.

## How it works

The [`release`](.github/workflows/release.yml) workflow is manually triggered
(`workflow_dispatch`). It:

1. Resolves the requested upstream ref to a commit SHA.
2. Builds the native profiler (`OpenTelemetry.AutoInstrumentation.Native.so`) inside an
   Ubuntu 16.04 container so it links against glibc 2.23 (broad Linux compat).
3. Runs the upstream Nuke `BuildWorkflow` on `ubuntu-22.04` (x64) and `ubuntu-22.04-arm`
   (arm64) to produce the managed tracer-home for each architecture.
4. Swaps the ubuntu-16.04-built native lib into the x64 tracer-home (arm64 uses the
   runner-built native lib).
5. Packages each architecture as
   `dash0-opentelemetry-dotnet-instrumentation-linux-<arch>.tar.gz` and publishes them
   as a GitHub Release.

## Prerequisites

The upstream repo is currently private. The workflow reads the org-level secret
`REPOSITORY_FULL_ACCESS_GITHUB_TOKEN` (must be granted to this repo) to check out
`dash0hq/opentelemetry-dotnet-instrumentation`.

## Running a release

From the Actions tab, run the `release` workflow with:

- **upstream_ref** — branch/tag/SHA of `opentelemetry-dotnet-instrumentation` to build
  (defaults to `dash0-main`).
- **release_tag** — the tag to create on this repo, e.g. `v0.1.0`.
- **draft** — leave `true` while iterating; set to `false` when publishing for real.

## Consuming from the Dash0 Operator

Extract an archive to a directory and point the .NET auto-instrumentation environment
variables at it:

```sh
tar -xzf dash0-opentelemetry-dotnet-instrumentation-linux-x64.tar.gz -C /opt/dash0/otel-dotnet-auto
export OTEL_DOTNET_AUTO_HOME=/opt/dash0/otel-dotnet-auto
```

The archive contains the same layout upstream produces under `bin/tracer-home/`:
`net/`, `AdditionalDeps/`, `linux-<arch>/`, plus `instrument.sh` and legal files.
(`netfx/` is Windows-only and is not included in these Linux archives.)
