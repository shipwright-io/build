#!/bin/bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

#
# Generates the typed client for Kubernetes CRDs
# From https://www.openshift.com/blog/kubernetes-deep-dive-code-generation-customresources
#

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
  "build:v1alpha1,v1beta1" \
  --go-header-file "${SCRIPT_ROOT}/hack/boilerplate.go.txt"
