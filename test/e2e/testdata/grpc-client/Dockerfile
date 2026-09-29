# SPDX-License-Identifier: Apache-2.0
#
# Builds examples/grpc-client with the real OTel injector and the
# tracer-home under test wired in via LD_PRELOAD, for E2E testing.
#
# glibc-only. See Dockerfile.musl for the Alpine counterpart -- kept as a
# fully separate file, rather than this file's own ARG LIBC_FLAVOR-branched
# stages (the pattern every other scenario here still uses), because
# testcontainers-go's build path goes through the classic (non-BuildKit)
# Docker Engine ImageBuild API, which pulls every named FROM stage's base
# image regardless of which one a later `FROM x-${LIBC_FLAVOR}` actually
# selects -- confirmed in a build log: both the jammy and alpine aspnet
# base images got pulled on a single musl build. Splitting into two files
# means each only ever references its own flavor's images. Trade-off:
# anything shared (ENV vars, ENTRYPOINT, app csproj name, INJECTOR_VERSION)
# now has to be kept in sync by hand across both files.
#
# Build context is staged by test/e2e/harness (see container.go): it is not
# the repo itself, but a temp dir containing:
#   app/          - a copy of examples/grpc-client
#   tracer-home/  - the tracer-home under test (harness.TracerHome), staged
#                   under tracer-home/glibc/ for this file
#
# DOTNET_VERSION selects which of GrpcClient.csproj's multi-targeted TFMs
# (net6.0-net10.0) to build and run, passed by test/e2e/grpc's own test loop
# via harness.AppScenario.BuildArgs -- one example project and one Dockerfile
# cover every version instead of a separate directory per TFM.
ARG DOTNET_VERSION=6.0

# The jammy variant (Ubuntu 22.04) is needed at build time only, and only for
# net6.0/net7.0: Grpc.Tools' bundled arm64 protoc binary requires a newer
# glibc than those versions' default bullseye-based SDK image ships
# (GLIBC_2.32+ vs bullseye's 2.31). net8.0+ SDK images are bookworm-based
# (or, for 10.0, Ubuntu-based) and already satisfy this, so no suffix is
# needed there -- confirmed via `docker run ... ldd --version` against each
# version's arm64 SDK image. test/e2e/grpc computes this suffix in Go and
# passes it via BuildArgs; the runtime (aspnet) stage never needs it, since
# the glibc constraint is build-time-only (the bundled protoc binary).
ARG SDK_TAG_SUFFIX=""

FROM mcr.microsoft.com/dotnet/sdk:${DOTNET_VERSION}${SDK_TAG_SUFFIX} AS build
ARG DOTNET_VERSION
WORKDIR /src
COPY app/GrpcClient.csproj .
RUN dotnet restore
COPY app/. .
RUN dotnet publish -c Release -o /app --no-restore -f net${DOTNET_VERSION}

FROM mcr.microsoft.com/dotnet/aspnet:${DOTNET_VERSION}
WORKDIR /app
COPY --from=build /app .

# The Dash0 .NET distribution under test, laid out exactly as the release
# tarballs extract (net/, linux-x64/, instrument.sh, ...) -- already nested
# under glibc/ by the harness.
COPY tracer-home/ /dash0-instrumentation/dotnet/

# The real OTel injector, same download source dash0-operator's
# instrumentation image uses. Checksum verification is skipped here (unlike
# that image) since this is test-only infrastructure, not a shipped artifact.
ARG INJECTOR_VERSION=v0.11.0
ARG INJECTOR_ARCH=amd64
RUN apt-get update \
    && apt-get install -y --no-install-recommends curl ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && curl -fsSL -o /usr/lib/libotelinject.so \
      "https://github.com/open-telemetry/opentelemetry-injector/releases/download/${INJECTOR_VERSION}/libotelinject_${INJECTOR_ARCH}.so"

COPY injector.conf /etc/opentelemetry/injector.conf
ENV LD_PRELOAD=/usr/lib/libotelinject.so
ENV OTEL_INJECTOR_CONFIG_FILE=/etc/opentelemetry/injector.conf
ENV OTEL_INJECTOR_LOG_LEVEL=debug
ENV OTEL_SERVICE_NAME=grpc-client-e2e
EXPOSE 8080
ENTRYPOINT ["dotnet", "GrpcClient.dll"]
