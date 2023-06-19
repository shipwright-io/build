# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0
FROM registry.access.redhat.com/ubi9-minimal:latest

RUN \
  microdnf --refresh --assumeyes --best --nodocs --noplugins --setopt=install_weak_deps=0 upgrade && \
  microdnf clean all && \
  rm -rf /var/cache/yum && \
  # The following setup is necessary so that this image can run as any user
  mkdir -p /shared-home/.docker /shared-home/.ssh && chmod -R 0777 /shared-home && \
  # This is the default user that will be used when strategy steps use different runAs configuration.
  # This must be in synchronization with our default configuration.
  echo "shp:x:1000:1000:shp:/shared-home:/sbin/nologin" >/etc/passwd && \
  echo "shp:x:1000" >/etc/group

ENV HOME /shared-home
