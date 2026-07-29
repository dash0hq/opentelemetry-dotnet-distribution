# Release process

This document is for maintainers. It describes how to cut a new release, how
the release automation is wired, and which patches the workflow applies to
upstream at build time.

## One-click release

The maintainer flow mirrors the pattern used by
[`open-telemetry/opentelemetry-injector`](https://github.com/open-telemetry/opentelemetry-injector):

1. **Prepare** — Run the [`prepare-release`](.github/workflows/prepare-release.yml)
   workflow via *Actions → Prepare Release → Run workflow*. Inputs:

   | Input | Purpose | Default |
   | --- | --- | --- |
   | `version` | Semver for the release, e.g. `0.1.0` or `0.1.0-rc.1`. Leading `v` is optional. | *(required)* |
   | `upstream_ref` | Branch, tag, or SHA of `opentelemetry-dotnet-instrumentation` to pin. | `dash0-main` |

   This resolves `upstream_ref` to a concrete upstream commit SHA, writes it
   to `.upstream-ref`, and opens a PR with a commit titled
   `chore: prepare release v<version>`.

2. **Review** — Review the PR (it only touches `.upstream-ref`) and merge it
   into `main`.

3. **Automatic tag** — On push to `main`, the
   [`create-tag-for-release`](.github/workflows/create-tag-for-release.yml)
   workflow inspects the merge commit. If the message matches the release-commit
   pattern it creates and pushes the `v<version>` tag using an org-level token,
   which is what triggers step 4.

4. **Automatic build and publish** — The tag push triggers the
   [`release`](.github/workflows/release.yml) workflow. It reads
   `.upstream-ref`, builds the linux-x64 and linux-arm64 tracer-home archives,
   and publishes them as a GitHub Release. Pre-release tags (containing `-rc.`)
   land as drafts; stable tags publish immediately.

CLI equivalent for step 1:

```sh
gh workflow run prepare-release --repo dash0hq/opentelemetry-dotnet-distribution \
  -f version=0.1.0
```

## Manual override

The `release` workflow keeps a `workflow_dispatch` trigger for emergency use
(e.g. a hotfix build against a specific upstream SHA outside the prepared-PR
loop). It accepts `upstream_ref`, `release_tag`, and `draft` inputs and does
not consult `.upstream-ref`. Prefer the one-click flow.

## Prerequisites

- **Repository secret `REPOSITORY_FULL_ACCESS_GITHUB_TOKEN`** — an org-level
  secret granted to this repo. Used by:
  - `actions/checkout` in the build jobs, to clone the private upstream repo.
  - `prepare-release` to open the release PR.
  - `create-tag-for-release` to push the tag (using the default
    `GITHUB_TOKEN` here would not trigger downstream workflows,
    [per GitHub docs](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow#triggering-a-workflow-from-a-workflow)).
- **GitHub-hosted `ubuntu-22.04-arm` runners** — used for the arm64 build.
  Requires an org/repo plan that includes arm64 hosted runners.

## Workflow anatomy

The build pipeline has four jobs (all defined in
[`release.yml`](.github/workflows/release.yml)):

1. **`resolve-upstream`** — Resolves the upstream ref to a concrete commit
   SHA; downstream jobs pin to that SHA so all artifacts in a release match.
2. **`build-native-x64`** — Builds `OpenTelemetry.AutoInstrumentation.Native.so`
   for `linux-x64` inside an Ubuntu 16.04 (amd64) container so it links
   against glibc 2.23. Publishes the `.so` as a workflow artifact.
3. **`build-native-arm64`** — Same idea, on `ubuntu-22.04-arm` inside an
   `arm64v8/ubuntu:16.04` container. Publishes the aarch64 `.so` as a
   workflow artifact.
4. **`build-x64`** — On `ubuntu-22.04`, runs the upstream Nuke `BuildTracer`
   target to produce the managed tracer-home, swaps in the glibc-2.23
   native library from `build-native-x64`, and tars the result.
5. **`build-arm64`** — On `ubuntu-22.04-arm`, runs `BuildTracer`, swaps in
   the glibc-2.23 aarch64 native library from `build-native-arm64`, and
   tars the result.
6. **`release`** — Downloads both tarballs and publishes them as a GitHub
   Release.

## Patches applied to upstream at build time

`dash0-main` still relies on some third-party sources whose state has drifted.
The workflow patches the checked-out upstream tree before invoking the build,
so upstream itself does not need to change to unblock a release. If any of the
listed issues is fixed upstream, the corresponding patch can be dropped from
`.github/workflows/release.yml`.

### `docker/ubuntu1604.dockerfile` (both archs)

- **Stale `dotnet-install.sh` SHA pin** — Microsoft rotated the script;
  upstream's pinned SHA no longer matches. Patch strips the `sha256sum -c`
  line. TLS from `dot.net` covers transport integrity.
- **kitware apt `signed-by` mismatch** — the source line references a keyring
  file that isn't created by the RUN block. Patch rewrites the sources.list
  entry to use `[trusted=yes]` so `apt-get update` no longer fails on GPG.
- **Xenial CA bundle too old for Launchpad** — `add-apt-repository ppa:...`
  hangs on the Launchpad API for ~9 minutes and fails with an unhelpful error
  because Xenial's `ca-certificates` doesn't trust Launchpad's current cert.
  Patch downloads a current Mozilla CA bundle (from `curl.se/ca/cacert.pem`)
  on the runner and `COPY`s it over `/etc/ssl/certs/ca-certificates.crt` in
  the container.
- **kitware xenial apt no longer publishes a modern cmake** — apt silently
  keeps Ubuntu 16.04's cmake 3.5.1, which is below the tracer's
  `cmake_minimum_required(VERSION 3.10..3.19)`. Patch injects a RUN that
  extracts the static Linux cmake tarball from Kitware's GitHub releases into
  `/usr/local` and symlinks it into `/usr/bin`. The x64 job uses the
  `linux-x86_64` tarball; the arm64 job uses `linux-aarch64`.

### `docker/ubuntu1604.dockerfile` (arm64-only additions)

- **Base image switch** — the `FROM ubuntu:16.04@sha256:...` line upstream
  pins is an amd64 manifest; on an arm64 host `docker build` cannot resolve
  it. Patch rewrites the base to `FROM arm64v8/ubuntu:16.04`, which docker
  pulls natively on `ubuntu-22.04-arm`.
- **Replace clang install** — upstream installs clang-5.0 from
  `apt.llvm.org/xenial`, which has no arm64 packages, and Ubuntu 16.04
  xenial arm64's own apt archive only ships clang ≤ 3.9 (insufficient for
  the tracer's `-std=c++17`). The workflow replaces that RUN block with a
  download of LLVM 8.0.1's prebuilt `aarch64-linux-gnu` tarball (which
  LLVM builds on Ubuntu 16.04, so it runs against glibc 2.23) and symlinks
  `clang` and `clang++` into `/usr/bin`.

### Nuke target selection

The workflow runs `BuildTracer` (not `BuildWorkflow`). `BuildWorkflow` triggers
the global `Restore` target, which restores test-application projects that pin
`NServiceBus 9.2.6` (net8-only) and fails the multi-TFM restore on net6/net7.
`BuildTracer` restores only the `src` projects it depends on.

## Cleaning up test drafts

Iterating on the pipeline can leave draft releases (`v0.1.0-testN`) behind.
List and delete them with:

```sh
gh release list --repo dash0hq/opentelemetry-dotnet-distribution
gh release delete <tag> --repo dash0hq/opentelemetry-dotnet-distribution --yes
```
