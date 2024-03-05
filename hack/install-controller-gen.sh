#!/bin/bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

#
# Installs controller-gen utility via "go install".
#

set -eu

# controller-gen version
CONTROLLER_GEN_VERSION="${CONTROLLER_GEN_VERSION:-v0.12.1}"

if [ ! -f "${GOPATH}/bin/controller-gen" ]; then
  echo "# Installing controller-gen..."
  pushd "$(mktemp -d)" >/dev/null 2>&1
  go install sigs.k8s.io/controller-tools/cmd/controller-gen@"${CONTROLLER_GEN_VERSION}"
  popd >/dev/null 2>&1
fi

if ! grep -q " $CONTROLLER_GEN_VERSION$" <<<"$("${GOPATH}"/bin/controller-gen --version)"; then
  echo "Current controller-gen version $("${GOPATH}"/bin/controller-gen --version | cut -d' ' -f2) does not match desired version $CONTROLLER_GEN_VERSION."
  echo "In order to update, run:"
  echo "  go install sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_GEN_VERSION}"
  echo
  exit 1
fi

# print controller-gen version
"${GOPATH}"/bin/controller-gen --version
