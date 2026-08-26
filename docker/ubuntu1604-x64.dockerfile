# SPDX-License-Identifier: Apache-2.0
#
# Ubuntu 16.04 (xenial) build container for the native tracer library
# (OpenTelemetry.AutoInstrumentation.Native.so), x64 variant.
#
# This is NOT upstream's docker/ubuntu1604.dockerfile. Ubuntu 16.04 is long
# EOL, and every apt-based toolchain source in the original file (LLVM's
# apt.llvm.org, the ubuntu-toolchain-r PPA, Kitware's apt repo) has proven
# unreliable from GitHub-Actions IP ranges, an EOL-repo pruning its own old
# packages, or both. We used to check out upstream's file at build time and
# rewrite it in place with a growing pile of sed/awk patches -- fragile,
# duplicated across four call sites (this file replaces all of them), and
# it had already silently rotted: one anchor no longer matched upstream's
# current content (a no-op patch), and an unrelated sed deletion had started
# stripping upstream's own cmake checksum check as a side effect. Owning a
# small, versioned, from-scratch file removes both problems -- every
# toolchain source below already came from our own overrides, not
# upstream's, so nothing of substance was actually being inherited.
#
# Built with the upstream commit under test as Docker build context (for
# the ./scripts/dotnet-install.sh COPY below) but this file, not upstream's,
# via `docker build -f <this file> <upstream-checkout-dir>`. See
# ubuntu1604-arm64.dockerfile for the arm64 counterpart (base image and
# LLVM/cmake tarball architectures differ; everything else is identical).
#
# Maintenance note: upstream's own ubuntu1604.dockerfile changes roughly
# monthly (.NET SDK bumps, toolchain fixes for this same image) -- none of
# those changes flow here automatically anymore. Skim upstream's file
# periodically (e.g. whenever .upstream-ref jumps meaningfully) for anything
# worth porting, in particular the pinned dotnet-install.sh SDK version
# below.

FROM ubuntu:16.04@sha256:1f1a2d56de1d604801a9671f301190704c25d604a416f59e03c04f5c6ffee0d6

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

# Install clang from LLVM's GitHub releases instead of apt.llvm.org's xenial
# repo, which has repeatedly failed to connect from GitHub-Actions IP
# ranges (both the GPG key fetch and the .deb downloads themselves) --
# runner-egress flakiness against that specific host, not a removed
# package, but a recurring one. LLVM 8's clang binary requires
# GLIBCXX_3.4.22 (from GCC 5.3+ libstdc++), which Xenial's default
# libstdc++ predates; installing after g++-9 above satisfies that.
RUN mkdir -p /opt/llvm \
    && curl -fsSL https://github.com/llvm/llvm-project/releases/download/llvmorg-8.0.1/clang+llvm-8.0.1-x86_64-linux-gnu-ubuntu-14.04.tar.xz \
    | tar -xJ -C /opt/llvm --strip-components=1 \
    && ln -sf /opt/llvm/bin/clang /usr/bin/clang \
    && ln -sf /opt/llvm/bin/clang++ /usr/bin/clang++ \
    && clang --version

# Install cmake from Kitware's GitHub releases. Xenial ships cmake 3.5.1,
# and Kitware's own apt repo for xenial no longer publishes a newer one --
# below the tracer's cmake >= 3.10 requirement.
RUN curl -fsSL https://github.com/Kitware/CMake/releases/download/v3.29.9/cmake-3.29.9-linux-x86_64.tar.gz \
    | tar -xz -C /usr/local --strip-components=1 \
    && ln -sf /usr/local/bin/cmake /usr/bin/cmake \
    && cmake --version

COPY ./scripts/dotnet-install.sh ./dotnet-install.sh

RUN chmod +x ./dotnet-install.sh \
    && ./dotnet-install.sh -v 9.0.316 --install-dir /usr/share/dotnet --no-path \
    && rm dotnet-install.sh

ENV IsLegacyUbuntu=true

WORKDIR /project
