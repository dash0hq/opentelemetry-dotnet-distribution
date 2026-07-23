# Stable base image with up-to-date build tooling
FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

# --- Step 1a: base build tools (clang, cmake, git, curl, etc.) ---
RUN apt-get update && apt-get install -y --no-install-recommends \
        clang \
        cmake \
        build-essential \
        git \
        curl \
        wget \
        ca-certificates \
        gnupg \
        apt-transport-https \
    && rm -rf /var/lib/apt/lists/*

# --- Step 1b: .NET SDKs installed as a separate step ---
# Ubuntu 22.04 doesn't ship a single image with both .NET 6 and .NET 8,
# so we add the Microsoft package repo and install both SDKs explicitly.
RUN curl -sSL -O https://packages.microsoft.com/config/ubuntu/22.04/packages-microsoft-prod.deb \
    && dpkg -i packages-microsoft-prod.deb \
    && rm packages-microsoft-prod.deb \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        dotnet-sdk-6.0 \
        dotnet-sdk-8.0 \
    && rm -rf /var/lib/apt/lists/*

ENV DOTNET_CLI_TELEMETRY_OPTOUT=1 \
    DOTNET_NOLOGO=1

WORKDIR /src

# --- Step 2: fetch the repo (this and everything below is cached by Docker
#     as long as none of these steps or their inputs change) ---
# FIXME: use Dash0 instead.
RUN git clone https://github.com/carlosalberto/opentelemetry-dotnet-instrumentation.git

WORKDIR /src/opentelemetry-dotnet-instrumentation

# --- Step 3: checkout branch net-6-0-support ---
RUN git checkout net-6-0-support

# --- Step 4: restore dotnet tools ---
RUN dotnet tool restore

# --- Step 5: build the actual product (expensive step — cached by Docker) ---
RUN ./build.sh --platform x64

# --- Step 6: pack the final output (also cached, runs once at build time) ---
RUN tar czvf /src/opentelemetry-dotnet-instrumentation-linux.tar.gz -C bin/tracer-home .

# --- Step 7: one-shot hand-off to a shared volume, runs at container START,
#     not at build time. This is the ONLY thing that executes on every
#     `docker compose up` — a cheap file copy, not a recompile. ---
CMD ["sh", "-c", "cp /src/opentelemetry-dotnet-instrumentation-linux.tar.gz /output/ && echo 'Artifact published to /output'"]
