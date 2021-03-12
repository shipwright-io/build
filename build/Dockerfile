# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV CONTROLLER=/usr/local/bin/shipwright-build-controller \
    USER_UID=1001 \
    USER_NAME=shipwright-build-controller

# install controller binary
COPY build/_output/bin/shipwright-build-controller ${CONTROLLER}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
