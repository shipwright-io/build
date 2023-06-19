# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0
ARG BASE

FROM registry.access.redhat.com/ubi9-minimal:latest AS bin-loader
RUN \
  microdnf --assumeyes --nodocs install gzip jq tar && \
  TAG_NAME="$(curl -s https://api.github.com/repos/aquasecurity/trivy/releases/latest | jq -r '.tag_name')" && \
  curl -L -s "https://github.com/aquasecurity/trivy/releases/download/${TAG_NAME}/trivy_${TAG_NAME/v/}_$(uname -s)-$(uname -m | sed -e 's/aarch64/ARM64/' -e 's/ppc64le/PPC64LE/' -e 's/x86_64/64bit/').tar.gz" | tar -xzf - -C /usr/local/bin trivy


FROM ${BASE}

COPY --from=bin-loader /usr/local/bin/trivy /usr/local/bin/trivy

USER 1000:1000
