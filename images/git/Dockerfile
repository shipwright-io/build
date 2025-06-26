# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

ARG BASE

FROM ${BASE} as builder

RUN \
   microdnf install -y --nodocs git curl tar gzip && \
    LFS_ARCH=$(uname -m | sed \
      -e 's/^x86_64$/linux-amd64/' \
      -e 's/^aarch64$/linux-arm64/' \
      -e 's/^ppc64le$/linux-ppc64le/' \
      -e 's/^s390x$/linux-s390x/' || \
      (echo "Unsupported arch $(uname -m)" >&2; exit 1)) && \
    GIT_LFS_VER="3.6.1" && \
    curl -L -o /tmp/git-lfs.tar.gz \
      "https://github.com/git-lfs/git-lfs/releases/download/v${GIT_LFS_VER}/git-lfs-${LFS_ARCH}-v${GIT_LFS_VER}.tar.gz" && \
    tar -xzf /tmp/git-lfs.tar.gz -C /tmp && \
    # Run the installer from the extracted folder
    cd /tmp/git-lfs-${GIT_LFS_VER} && \
    ./install.sh && \
    rm -rf /tmp/git-lfs.tar.gz /tmp/git-lfs-${GIT_LFS_VER}

FROM ${BASE}

RUN microdnf install -y --nodocs git && microdnf clean all && rm -rf /var/cache/microdnf /var/cache/yum /var/tmp/*

COPY --from=builder /usr/local/bin/git-lfs /usr/local/bin/git-lfs
COPY --from=builder /usr/local/bin/git-lfs /usr/bin/git-lfs

RUN git lfs install

USER 1000:1000
