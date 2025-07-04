# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0
ARG BUILD_IMAGE
FROM ${BUILD_IMAGE}

RUN \
  microdnf --refresh --assumeyes --best --nodocs --noplugins --setopt=install_weak_deps=0 upgrade && \
  microdnf clean all && \
  rm -rf /var/cache/yum && \
  # The following setup is necessary so that this image can run as any user
  # This is the default user that will be used when strategy steps use different runAs configuration.
  # This must be in synchronization with our default configuration.
  # Note: The /shared-home directory and its subdirectories (.docker, .ssh)
  # are expected to be provided by Kubernetes volumes, so they are NOT created here.
  echo "shp:x:1000:1000:shp:/shared-home:/sbin/nologin" >/etc/passwd && \
  echo "shp:x:1000" >/etc/group

ENV HOME /shared-home
