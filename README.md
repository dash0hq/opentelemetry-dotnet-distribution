# dash0-opentelemetry-dotnet-distribution

Builds Linux `x64` and `arm64` binaries of Dash0's fork of
[`opentelemetry-dotnet-instrumentation`](https://github.com/dash0hq/opentelemetry-dotnet-instrumentation)
(branch `dash0-main`) and publishes them as GitHub Releases for the Dash0 Operator to consume.

## How it works

The [`release`](.github/workflows/release.yml) workflow is manually triggered
(`workflow_dispatch`). It:

1. Resolves the requested upstream ref to a commit SHA.
2. In parallel, runs the upstream Nuke `BuildTracer` target on `ubuntu-22.04` (x64)
   and `ubuntu-22.04-arm` (arm64) to produce the managed tracer-home and the native
   profiler for each architecture. Native code links against the runner's glibc 2.35.
3. Packages each architecture as
   `dash0-opentelemetry-dotnet-instrumentation-linux-<arch>.tar.gz` and publishes
   both as a GitHub Release.

The Ubuntu-16.04-based glibc-2.23 native build used by upstream's own release is
skipped intentionally — the base image is EOL and its stale apt sources need
patching in several places. Ubuntu 22.04's glibc 2.35 is sufficient for the Dash0
Operator's target environments (recent Kubernetes clusters and modern .NET base
images). If broader glibc compatibility is needed later, a `manylinux2014`-based
native build (glibc 2.17) would be the maintained path.

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
