# SPDX-License-Identifier: Apache-2.0
#
# Builds examples/redis-cache with the real OTel injector and the
# tracer-home under test wired in via LD_PRELOAD, for E2E testing.
#
# Build context is staged by test/e2e/harness (see container.go): it is not
# the repo itself, but a temp dir containing:
#   app/          - a copy of examples/redis-cache
#   tracer-home/  - the tracer-home under test (harness.TracerHome)

FROM mcr.microsoft.com/dotnet/sdk:6.0 AS build
WORKDIR /src
COPY app/RedisCache.csproj .
RUN dotnet restore
COPY app/. .
RUN dotnet publish -c Release -o /app --no-restore

FROM mcr.microsoft.com/dotnet/aspnet:6.0
WORKDIR /app
COPY --from=build /app .

# The Dash0 .NET distribution under test, laid out exactly as the release
# tarballs extract (net/, linux-x64/, instrument.sh, ...).
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
ENV OTEL_SERVICE_NAME=redis-cache-e2e
ENV ASPNETCORE_URLS=http://+:8080
EXPOSE 8080
ENTRYPOINT ["dotnet", "RedisCache.dll"]
