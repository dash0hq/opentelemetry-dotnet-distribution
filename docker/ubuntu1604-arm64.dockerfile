# SPDX-License-Identifier: Apache-2.0
#
# Ubuntu 16.04 (xenial) build container for the native tracer library
# (OpenTelemetry.AutoInstrumentation.Native.so), arm64 variant.
#
# arm64 counterpart of ubuntu1604-x64.dockerfile -- see that file's header
# for why this is a versioned, from-scratch file instead of a patched copy
# of upstream's docker/ubuntu1604.dockerfile. The only differences from the
# x64 file are the base image (no amd64 digest pin resolves on an arm64
# host) and the LLVM/cmake tarball architectures; keep the two in sync for
# everything else.
#
# Built with the upstream commit under test as Docker build context (for
# the ./scripts/dotnet-install.sh COPY below) but this file, not upstream's,
# via `docker build -f <this file> <upstream-checkout-dir>`.

FROM arm64v8/ubuntu:16.04

RUN apt-get update && \
    apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    git \
    build-essential \
    gnupg \
    libicu-dev

# Refresh the CA bundle. Xenial's ca-certificates is too old to trust some
# modern TLS certs that later RUNs in this file talk to.
RUN curl -fsSL https://curl.se/ca/cacert.pem -o /etc/ssl/certs/ca-certificates.crt

# Install g++-9 from the ubuntu-toolchain-r PPA. [trusted=yes] skips fetching
# the PPA's signing key from keyserver.ubuntu.com, an extra network
# round-trip that was intermittently flaky from GitHub-Actions IP ranges.
RUN echo 'deb [trusted=yes] https://ppa.launchpadcontent.net/ubuntu-toolchain-r/test/ubuntu xenial main' > /etc/apt/sources.list.d/ubuntu-toolchain-r.list && \
    apt-get update && \
    apt-get install -y g++-9 && \
    update-alternatives --install /usr/bin/gcc gcc /usr/bin/gcc-9 60 --slave /usr/bin/g++ g++ /usr/bin/g++-9

# Install clang from LLVM's GitHub releases. apt.llvm.org/xenial has no
# arm64 packages at all, and Xenial arm64's own clang is too old (<=3.9,
# need C++17). LLVM 8's clang binary requires GLIBCXX_3.4.22 (from GCC
# 5.3+ libstdc++), which Xenial's default libstdc++ predates; installing
# after g++-9 above satisfies that.
RUN mkdir -p /opt/llvm \
    && curl -fsSL https://github.com/llvm/llvm-project/releases/download/llvmorg-8.0.1/clang+llvm-8.0.1-aarch64-linux-gnu.tar.xz \
    | tar -xJ -C /opt/llvm --strip-components=1 \
    && ln -sf /opt/llvm/bin/clang /usr/bin/clang \
    && ln -sf /opt/llvm/bin/clang++ /usr/bin/clang++ \
    && clang --version

# Install cmake from Kitware's GitHub releases. Xenial ships cmake 3.5.1,
# and Kitware's own apt repo for xenial no longer publishes a newer one --
# below the tracer's cmake >= 3.10 requirement.
RUN curl -fsSL https://github.com/Kitware/CMake/releases/download/v3.29.9/cmake-3.29.9-linux-aarch64.tar.gz \
    | tar -xz -C /usr/local --strip-components=1 \
    && ln -sf /usr/local/bin/cmake /usr/bin/cmake \
    && cmake --version

COPY ./scripts/dotnet-install.sh ./dotnet-install.sh

RUN chmod +x ./dotnet-install.sh \
    && ./dotnet-install.sh -v 9.0.316 --install-dir /usr/share/dotnet --no-path \
    && rm dotnet-install.sh

ENV IsLegacyUbuntu=true

WORKDIR /project
