#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

#
# Installs controller-gen utility via "go get".
#

set -eu

# controller-gen version
CONTROLLER_GEN_VERSION="${CONTROLLER_GEN_VERSION:-v0.4.1}"

if [ ! -f "${GOPATH}/bin/controller-gen" ] ; then
    echo "# Installing controller-gen..."
    pushd "$(mktemp -d)" >/dev/null 2>&1
    GO111MODULE=on go get sigs.k8s.io/controller-tools/cmd/controller-gen@"${CONTROLLER_GEN_VERSION}"
    popd >/dev/null 2>&1
fi

# print controller-gen version
controller-gen --version
