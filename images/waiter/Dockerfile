# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

ARG BASE

FROM ${BASE}

RUN \
  microdnf --assumeyes --nodocs install tar && \
  microdnf clean all && \
  rm -rf /var/cache/yum

USER 1000:1000
