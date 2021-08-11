#!/bin/bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

# Verifies if a developer has forgot to run the
# `make generate` so that all the changes in the
# clientset should also be pushed

set -euo pipefail

GOPATH=${GOPATH:-$(go env GOPATH)}
SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${SCRIPT_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../../../k8s.io/code-generator)}

echo ""
echo "Using code-generator package version, as instructed in the go.mod file"
echo "The code-generator package is imported via the pkg/kubecodegen dir"
echo "To modify the current version, please modify this in the go.mod"
echo ""

GOFLAGS="" GOPATH=${GOPATH} /bin/bash ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/shipwright-io/build/pkg/client \
  github.com/shipwright-io/build/pkg/apis \
  "build:v1alpha1" \
  --go-header-file "${SCRIPT_ROOT}/hack/boilerplate.go.txt"

if [ "$(git diff | grep pkg/client/clientset | wc -l)" -gt "0" ]; then
    echo ""
    echo "OOPS! It seems either hack/update-codegen.sh is not run or changes on the clientset is not pushed!"
    exit 1
fi
