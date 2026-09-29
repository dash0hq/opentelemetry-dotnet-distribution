# SPDX-License-Identifier: Apache-2.0
#
# Builds examples/quartz-job with the real OTel injector and
# the tracer-home under test wired in via LD_PRELOAD, for E2E testing.
#
# glibc-only. See Dockerfile.musl for the Alpine counterpart.
#
# Build context is staged by test/e2e/harness (see container.go): it is not
# the repo itself, but a temp dir containing:
#   app/          - a copy of examples/quartz-job
#   tracer-home/  - the tracer-home under test (harness.TracerHome), staged
#                   under tracer-home/glibc/ for this file
#
# DASH0_DOTNET_VERSION selects which of QuartzJob.csproj's multi-targeted TFMs
# (net6.0-net10.0) to build and run, passed by test/e2e/quartz's own test
# loop via harness.AppScenario.BuildArgs -- one example project and one
# Dockerfile cover every version instead of a separate directory per TFM.
ARG DASH0_DOTNET_VERSION=6.0

FROM mcr.microsoft.com/dotnet/sdk:${DASH0_DOTNET_VERSION} AS build
ARG DASH0_DOTNET_VERSION
WORKDIR /src
COPY app/QuartzJob.csproj .
RUN dotnet restore -p:TargetFramework=net${DASH0_DOTNET_VERSION}
COPY app/. .
RUN dotnet publish -c Release -o /app --no-restore -f net${DASH0_DOTNET_VERSION}

# The real OTel injector, same download source dash0-operator's
# instrumentation image uses. Fetched here in the build stage, not the final runtime stage:
# net6.0/net7.0's now-EOL Debian bullseye security repo
# has started 404ing on some of its own already-published point-release
# packages (curl, ca-certificates, libnghttp2, ...), making `apt-get install`
# there fail.
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
ENV OTEL_SERVICE_NAME=quartz-job-e2e
ENV ASPNETCORE_URLS=http://+:8080
EXPOSE 8080
ENTRYPOINT ["dotnet", "QuartzJob.dll"]
