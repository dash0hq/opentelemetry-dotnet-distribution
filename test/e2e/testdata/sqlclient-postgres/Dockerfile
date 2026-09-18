# SPDX-License-Identifier: Apache-2.0
#
# Builds examples/sqlclient-postgres with the real OTel injector and the
# tracer-home under test wired in via LD_PRELOAD, for E2E testing.
#
# Build context is staged by test/e2e/harness (see container.go): it is not
# the repo itself, but a temp dir containing:
#   app/          - a copy of examples/sqlclient-postgres
#   tracer-home/  - the tracer-home under test (harness.TracerHome), staged
#                   under tracer-home/glibc/ or tracer-home/musl/ per
#                   LIBC_FLAVOR below
#
# LIBC_FLAVOR ("glibc", the default, or "musl") picks Debian- vs
# Alpine-based images and is passed in by harness.StartInstrumentedApp,
# mirroring harness.LibcFlavor()/DASH0_E2E_LIBC_FLAVOR. It must match how
# DASH0_E2E_TRACER_HOME was actually built -- glibc tracer-homes carry
# linux-arm64/ (or linux-x64/), musl ones carry linux-musl-arm64/ (or
# linux-musl-x64/) -- since it only picks which image runs, not which
# tracer-home content is available.
ARG LIBC_FLAVOR=glibc

FROM mcr.microsoft.com/dotnet/sdk:6.0-jammy AS build-glibc
FROM mcr.microsoft.com/dotnet/sdk:6.0-alpine AS build-musl
FROM build-${LIBC_FLAVOR} AS build
WORKDIR /src
COPY app/SqlClientPostgres.csproj .
RUN dotnet restore
COPY app/. .
RUN dotnet publish -c Release -o /app --no-restore

FROM mcr.microsoft.com/dotnet/aspnet:6.0-jammy AS runtime-glibc
FROM mcr.microsoft.com/dotnet/aspnet:6.0-alpine AS runtime-musl
FROM runtime-${LIBC_FLAVOR}
WORKDIR /app
COPY --from=build /app .

# The Dash0 .NET distribution under test, laid out exactly as the release
# tarballs extract (net/, linux-x64/, instrument.sh, ...) -- already nested
# under glibc/ or musl/ by the harness, matching this image's own libc.
COPY tracer-home/ /dash0-instrumentation/dotnet/

# The real OTel injector, same download source dash0-operator's
# instrumentation image uses. Checksum verification is skipped here (unlike
# that image) since this is test-only infrastructure, not a shipped artifact.
# The injector binary itself isn't libc-flavor-specific (see RELEASE.md /
# opentelemetry-injector's own release assets: one .so per arch, no glibc
# vs musl split), only how it's fetched here (apt vs apk) differs.
ARG INJECTOR_VERSION=v0.11.0
ARG INJECTOR_ARCH=amd64
ARG LIBC_FLAVOR
RUN if [ "${LIBC_FLAVOR}" = "musl" ]; then \
      apk add --no-cache curl ca-certificates; \
    else \
      apt-get update \
      && apt-get install -y --no-install-recommends curl ca-certificates \
      && rm -rf /var/lib/apt/lists/*; \
    fi \
    && curl -fsSL -o /usr/lib/libotelinject.so \
      "https://github.com/open-telemetry/opentelemetry-injector/releases/download/${INJECTOR_VERSION}/libotelinject_${INJECTOR_ARCH}.so"

COPY injector.conf /etc/opentelemetry/injector.conf
ENV LD_PRELOAD=/usr/lib/libotelinject.so
ENV OTEL_INJECTOR_CONFIG_FILE=/etc/opentelemetry/injector.conf
ENV OTEL_INJECTOR_LOG_LEVEL=debug
ENV OTEL_SERVICE_NAME=sqlclient-postgres-e2e
ENV ASPNETCORE_URLS=http://+:8080
EXPOSE 8080
ENTRYPOINT ["dotnet", "SqlClientPostgres.dll"]
