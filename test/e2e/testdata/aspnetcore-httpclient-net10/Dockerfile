# SPDX-License-Identifier: Apache-2.0
#
# Builds examples/aspnetcore-httpclient-net10 with the real OTel injector and
# the tracer-home under test wired in via LD_PRELOAD, for E2E testing.
#
# The net10.0 twin of the aspnetcore-httpclient scenario, validating that
# instrumentation-image packaging changes don't regress ASP.NET Core tracing
# above the net6.0/net7.0 floor.
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
#   app/          - a copy of examples/aspnetcore-httpclient-net10
#   tracer-home/  - the tracer-home under test (harness.TracerHome), staged
#                   under tracer-home/glibc/ for this file

FROM mcr.microsoft.com/dotnet/sdk:10.0 AS build
WORKDIR /src
COPY app/AspNetCoreHttpClientNet10.csproj .
RUN dotnet restore
COPY app/. .
RUN dotnet publish -c Release -o /app --no-restore

FROM mcr.microsoft.com/dotnet/aspnet:10.0
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
ENV OTEL_SERVICE_NAME=aspnetcore-httpclient-net10-e2e
ENV ASPNETCORE_URLS=http://+:8080
EXPOSE 8080
ENTRYPOINT ["dotnet", "AspNetCoreHttpClientNet10.dll"]
