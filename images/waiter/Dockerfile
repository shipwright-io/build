# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0
FROM registry.access.redhat.com/ubi9-minimal:latest

RUN \
  microdnf --nodocs update && \
  microdnf --nodocs install tar && \
  microdnf clean all && \
  rm -rf /var/cache/yum && \
  echo 'nonroot:x:1000:1000:nonroot:/:/sbin/nologin' > /etc/passwd && \
  echo 'nonroot:x:1000:' > /etc/group && \
  mkdir /.docker && \
  chown 1000:1000 /.docker

USER 1000:1000
