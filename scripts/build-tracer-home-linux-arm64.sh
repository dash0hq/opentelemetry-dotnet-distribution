#!/usr/bin/env bash
# Builds a linux-arm64 glibc tracer-home from a local checkout of
# dash0hq/opentelemetry-dotnet-instrumentation, for the E2E test suite's
# "build once, reuse across tests" dev-loop mode (test/e2e/harness).
#
# This is the arm64 counterpart of build-tracer-home-linux-x64.sh, mirroring
# build-and-e2e.yml's build-native-arm64 + build-arm64 jobs (arm64-specific
# versioned Dockerfile: arm64v8 base image, aarch64 LLVM 8 tarball instead
# of x64's x86_64 one, aarch64 cmake tarball). Prefer this one over the x64
# script when developing on Apple Silicon: it runs natively instead of under
# QEMU emulation, which has proven unreliable for the Ubuntu 16.04 image
# (observed: dpkg-deb segfaulting mid-unpack under qemu-user).
#
# Usage:
#   ./scripts/build-tracer-home-linux-arm64.sh [source-dir]
#
# source-dir defaults to ../opentelemetry-dotnet-instrumentation (a sibling
# checkout) or $DASH0_INSTRUMENTATION_SOURCE_DIR. Builds whatever is
# currently checked out there (including uncommitted changes) — the point
# of the dev-loop mode is to catch regressions before they're even pushed.
#
# Output: <source-dir>/bin/tracer-home, ready to point
# DASH0_E2E_TRACER_HOME at.

set -euo pipefail

distribution_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source_dir="${1:-${DASH0_INSTRUMENTATION_SOURCE_DIR:-${distribution_dir}/../opentelemetry-dotnet-instrumentation}}"

if [ ! -d "${source_dir}/.git" ]; then
  echo "::error:: ${source_dir} is not a git checkout of opentelemetry-dotnet-instrumentation" >&2
  exit 1
fi

cd "${source_dir}"
echo "Building tracer-home from $(git rev-parse --abbrev-ref HEAD) @ $(git rev-parse --short HEAD) in ${source_dir}"

# MinVer needs a reachable release tag to compute real assembly versions;
# the dash0hq fork carries none of its own. See release.yml for the full
# explanation.
git fetch --tags --quiet https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation.git

echo "--- Building native library in Ubuntu 16.04 container (linux/arm64, native) ---"
# Build from our own versioned Dockerfile, not upstream's docker/ubuntu1604.dockerfile
# -- see docker/ubuntu1604-arm64.dockerfile's header for why.
docker build --platform linux/arm64 -t dash0-native-build-arm64 -f "${distribution_dir}/docker/ubuntu1604-arm64.dockerfile" .
# This container only has .NET SDK 9.0.316 installed, and its ancient glibc
# cannot run .NET 10 at all. A repo-root global.json (once present) pins the
# SDK to 10.0.302 for other build paths; global.json has no MSBuild-style
# conditions, so drop it for this one container-only step instead of letting
# dotnet try to fetch and run an incompatible SDK (mirrors
# build-ubuntu1604-native-container.yml's own "rm -f global.json" step
# upstream). Restored afterward since source_dir is a real working tree,
# not an ephemeral CI checkout.
rm -f global.json
docker run --platform linux/arm64 -e OS_TYPE=linux-glibc --rm \
  --mount type=bind,source="${source_dir}",target=/project \
  dash0-native-build-arm64 \
  /bin/sh -c 'export PATH="$PATH:/usr/share/dotnet" && git config --global --add safe.directory /project && ./build.sh BuildNativeWorkflow'
git checkout -- global.json 2>/dev/null || true

native_so="bin/tracer-home/linux-arm64/OpenTelemetry.AutoInstrumentation.Native.so"
test -f "${native_so}"
cp "${native_so}" /tmp/dash0-e2e-native-arm64.so

echo "--- Building managed tracer-home (requires .NET SDKs 6/7/8/9 installed) ---"
./build.sh BuildTracer

echo "--- Swapping in Ubuntu-16.04-built native library ---"
rm "${native_so}"
cp /tmp/dash0-e2e-native-arm64.so "${native_so}"
file "${native_so}"

echo "tracer-home ready at ${source_dir}/bin/tracer-home"
echo "Point the E2E suite at it with: export DASH0_E2E_TRACER_HOME=${source_dir}/bin/tracer-home"
