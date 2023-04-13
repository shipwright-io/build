# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0
FROM registry.access.redhat.com/ubi9-minimal:latest

RUN \
  microdnf --refresh --assumeyes --best --nodocs --noplugins --setopt=install_weak_deps=0 upgrade && \
  microdnf --assumeyes --nodocs install git git-lfs && \
  microdnf clean all && \
  rm -rf /var/cache/yum && \
  # The following setup is necessary so that this image can run as any user
  mkdir -p /shared-home/.docker /shared-home/.ssh && chmod -R 0777 /shared-home && \
  # ssh, which is required for Git clone using SSH, will require an entry in /etc/passwd,
  # as we do not know the user/group combination, we allow the user to register itself.
  # The runtime privileges of the container will prevent a switch to root.
  echo -n > /etc/passwd && chmod 0666 /etc/passwd && \
  echo -n > /etc/groups && chmod 0666 /etc/groups

ENV HOME /shared-home
