# SPDX-License-Identifier: Apache-2.0
#
# Builds examples/redis-cache with the real OTel injector and
# the tracer-home under test wired in via LD_PRELOAD, for E2E testing.
#
# glibc-only. See Dockerfile.musl for the Alpine counterpart -- kept as a
# fully separate file, rather than an ARG LIBC_FLAVOR-branched single file
# (the pattern this scenario used before). See grpc-client/Dockerfile's
# header for the full rationale: testcontainers-go's build path goes
# through the classic (non-BuildKit) Docker Engine ImageBuild API, which
# pulls every named FROM stage's base image regardless of which one a later
# `FROM x-${LIBC_FLAVOR}` actually selects. Trade-off: anything shared (ENV
# vars, ENTRYPOINT, app csproj name, INJECTOR_VERSION) now has to be kept in
# sync by hand across both files.
#
# Build context is staged by test/e2e/harness (see container.go): it is not
# the repo itself, but a temp dir containing:
#   app/          - a copy of examples/redis-cache
#   tracer-home/  - the tracer-home under test (harness.TracerHome), staged
#                   under tracer-home/glibc/ for this file
#
# DASH0_DOTNET_VERSION selects which of RedisCache.csproj's multi-targeted TFMs
# (net6.0-net10.0) to build and run, passed by test/e2e/rediscache's own
# test loop via harness.AppScenario.BuildArgs -- one example project and one
# Dockerfile cover every version instead of a separate directory per TFM.
ARG DASH0_DOTNET_VERSION=6.0

FROM mcr.microsoft.com/dotnet/sdk:${DASH0_DOTNET_VERSION} AS build
ARG DASH0_DOTNET_VERSION
WORKDIR /src
COPY app/RedisCache.csproj .
RUN dotnet restore -p:TargetFramework=net${DASH0_DOTNET_VERSION}
COPY app/. .
RUN dotnet publish -c Release -o /app --no-restore -f net${DASH0_DOTNET_VERSION}

# The real OTel injector, same download source dash0-operator's
# instrumentation image uses. Checksum verification is skipped here (unlike
# that image) since this is test-only infrastructure, not a shipped
# artifact. Fetched here in the build stage, not the final runtime stage:
# the SDK image already ships curl + ca-certificates (needed for NuGet
# restore), so no apt-get/apk install is needed here, unlike in the aspnet
# runtime image -- and net6.0/net7.0's now-EOL Debian bullseye security repo
# has started 404ing on some of its own already-published point-release
# packages (curl, ca-certificates, libnghttp2, ...), making `apt-get install`
# there fail nondeterministically.
ARG INJECTOR_VERSION=v0.11.0
ARG INJECTOR_ARCH=amd64
RUN curl -fsSL -o /usr/lib/libotelinject.so \
      "https://github.com/open-telemetry/opentelemetry-injector/releases/download/${INJECTOR_VERSION}/libotelinject_${INJECTOR_ARCH}.so"

FROM mcr.microsoft.com/dotnet/aspnet:${DASH0_DOTNET_VERSION}
WORKDIR /app
COPY --from=build /app .

# The Dash0 .NET distribution under test, laid out exactly as the release
# tarballs extract (net/, linux-x64/, instrument.sh, ...) -- already nested
# under glibc/ by the harness.
COPY tracer-home/ /dash0-instrumentation/dotnet/

# The real OTel injector -- fetched in the build stage above (see its own
# comment for why), just copied in here.
COPY --from=build /usr/lib/libotelinject.so /usr/lib/libotelinject.so

COPY injector.conf /etc/opentelemetry/injector.conf
ENV LD_PRELOAD=/usr/lib/libotelinject.so
ENV OTEL_INJECTOR_CONFIG_FILE=/etc/opentelemetry/injector.conf
ENV OTEL_INJECTOR_LOG_LEVEL=debug
ENV OTEL_SERVICE_NAME=redis-cache-e2e
ENV ASPNETCORE_URLS=http://+:8080
EXPOSE 8080
ENTRYPOINT ["dotnet", "RedisCache.dll"]
