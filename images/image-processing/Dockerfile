# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0
FROM registry.access.redhat.com/ubi9-minimal:latest AS bin-loader
RUN microdnf --assumeyes --nodocs install gzip jq tar

RUN \
  TAG_NAME="$(curl -s https://api.github.com/repos/aquasecurity/trivy/releases/latest | jq -r '.tag_name')" && \
  curl -L -s "https://github.com/aquasecurity/trivy/releases/download/${TAG_NAME}/trivy_${TAG_NAME/v/}_$(uname -s)-$(uname -m | sed -e 's/aarch64/ARM64/' -e 's/ppc64le/PPC64LE/' -e 's/x86_64/64bit/').tar.gz" | tar -xzf - -C /usr/local/bin trivy

RUN \
  curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh \
    | sh -s -- -b /usr/local/bin

FROM registry.access.redhat.com/ubi9-minimal:latest
COPY --from=bin-loader /usr/local/bin /usr/local/bin
RUN \
  microdnf --refresh --assumeyes --best --nodocs --noplugins --setopt=install_weak_deps=0 upgrade && \
  microdnf clean all && \
  rm -rf /var/cache/yum && \
  echo 'nonroot:x:1000:1000:nonroot:/:/sbin/nologin' > /etc/passwd && \
  echo 'nonroot:x:1000:' > /etc/group

USER 1000:1000
