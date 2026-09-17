# SPDX-License-Identifier: Apache-2.0
#
# Alpine (musl) x64 build container for the native tracer library
# (OpenTelemetry.AutoInstrumentation.Native.so).
#
# x64 counterpart of alpine322-arm64-musl.dockerfile -- see that file's
# header for the overall rationale (from-scratch, versioned build container
# built from the upstream commit under test as Docker build context). The
# only difference from the arm64 file is the base image (amd64/ instead of
# arm64v8/, so the correct arch resolves explicitly regardless of host);
# Alpine's own clang/cmake/make packages are architecture-generic apk names,
# unlike the Xenial glibc files' arch-specific GitHub-release tarballs.
#
# Package/runtime-dependency list is cross-checked against upstream's own
# docker/alpine.dockerfile and Microsoft's "Install .NET on Alpine" docs
# (learn.microsoft.com/dotnet/core/install/linux-alpine) -- re-check that
# page if this ever fails at the dotnet-install.sh step, since the exact
# set of required apk packages has moved across Alpine/.NET versions.
#
# Built with the upstream commit under test as Docker build context (for
# the ./scripts/dotnet-install.sh COPY below) but this file, not upstream's,
# via `docker build -f <this file> <upstream-checkout-dir>`.

FROM amd64/alpine:3.22

# build-base = gcc/g++/make/musl-dev/etc (build-essential equivalent).
# icu-dev is needed to link the tracer against ICU and pulls in icu-libs.
# The rest (libssl3, libstdc++, zlib, krb5, tzdata, icu-data-full) are the
# runtime deps the dotnet SDK itself needs to execute on musl -- see the
# Microsoft Learn page linked above for the current list.
RUN apk add --no-cache \
    bash \
    ca-certificates \
    curl \
    git \
    build-base \
    clang \
    cmake \
    icu-dev \
    icu-data-full \
    krb5 \
    libssl3 \
    zlib \
    tzdata \
    && clang --version && cmake --version && g++ --version

COPY ./scripts/dotnet-install.sh ./dotnet-install.sh

RUN chmod +x ./dotnet-install.sh \
    && ./dotnet-install.sh -v 10.0.112 --install-dir /usr/share/dotnet --no-path \
    && rm dotnet-install.sh

# Upstream's own alpine.dockerfile sets this same flag to branch build
# logic for musl; keeping it in case the fork's build scripts still check
# it (verify against your Nuke build targets/CMakeLists since divergence
# from upstream is possible).
ENV IsAlpine=true

WORKDIR /project
