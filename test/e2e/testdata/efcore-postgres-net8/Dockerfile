# SPDX-License-Identifier: Apache-2.0
#
# Builds examples/efcore-postgres-net8 with the real OTel injector and a
# tracer-home under test wired in via LD_PRELOAD, for E2E testing. Used to
# verify whether the missing-EF-Core-spans issue found on net6.0 with the
# Dash0 distro also reproduces with the actual upstream
# open-telemetry/opentelemetry-dotnet-instrumentation build on net8.0.
#
# Build context is staged by test/e2e/harness (see container.go): it is not
# the repo itself, but a temp dir containing:
#   app/          - a copy of examples/efcore-postgres-net8
#   tracer-home/  - the tracer-home under test (harness.TracerHome)

FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src
COPY app/EfCorePostgresNet8.csproj .
RUN dotnet restore
COPY app/. .
RUN dotnet publish -c Release -o /app --no-restore

FROM mcr.microsoft.com/dotnet/aspnet:8.0
WORKDIR /app
COPY --from=build /app .

# The tracer-home under test, laid out exactly as the release archives
# extract (net/, linux-<arch>/, instrument.sh, ...).
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
ENV OTEL_SERVICE_NAME=efcore-postgres-net8-e2e
ENV ASPNETCORE_URLS=http://+:8080
EXPOSE 8080
ENTRYPOINT ["dotnet", "EfCorePostgresNet8.dll"]
