# SPDX-License-Identifier: Apache-2.0
#
# Builds examples/efcore-postgres-net8 with the real OTel injector and
# the tracer-home under test wired in via LD_PRELOAD, for E2E testing.
#
# Used to verify whether the missing-EF-Core-spans issue found on net6.0
# with the Dash0 distro also reproduces with the actual upstream
# open-telemetry/opentelemetry-dotnet-instrumentation build on net8.0.
#
# glibc-only. See Dockerfile.musl for the Alpine counterpart.
#
# Build context is staged by test/e2e/harness (see container.go): it is not
# the repo itself, but a temp dir containing:
#   app/          - a copy of examples/efcore-postgres-net8
#   tracer-home/  - the tracer-home under test (harness.TracerHome), staged
#                   under tracer-home/glibc/ for this file
#
# DASH0_DOTNET_VERSION selects which of EfCorePostgresNet8.csproj's
# multi-targeted TFMs (net8.0-net10.0) to build and run, passed by
# test/e2e/efcorenet8's own test loop via harness.AppScenario.BuildArgs.
# Named with a DASH0_ prefix, not just DOTNET_VERSION, because the SDK/aspnet
# base images below already bake in their own ENV DOTNET_VERSION (their
# installed runtime's patch version) -- reusing that name would let Docker's
# classic (non-BuildKit) builder (what testcontainers-go's build path uses)
# silently resolve ${DOTNET_VERSION} to the image's inherited env value
# instead of our build arg.
ARG DASH0_DOTNET_VERSION=8.0

FROM mcr.microsoft.com/dotnet/sdk:${DASH0_DOTNET_VERSION} AS build
ARG DASH0_DOTNET_VERSION
WORKDIR /src
COPY app/EfCorePostgresNet8.csproj .
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
ENV OTEL_SERVICE_NAME=efcore-postgres-net8-e2e
ENV ASPNETCORE_URLS=http://+:8080
EXPOSE 8080
ENTRYPOINT ["dotnet", "EfCorePostgresNet8.dll"]
