# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0
FROM registry.access.redhat.com/ubi8-minimal:latest

RUN \
  microdnf --nodocs update && \
  microdnf --nodocs install git git-lfs && \
  microdnf clean all && \
  rm -rf /var/cache/yum && \
  echo 'nonroot:x:1000:1000:nonroot:/:/sbin/nologin' > /etc/passwd && \
  echo 'nonroot:x:1000:' > /etc/group && \
  mkdir /.docker && chown 1000:1000 /.docker && \
  mkdir /.ssh && chown 1000:1000 /.ssh

USER 1000:1000
