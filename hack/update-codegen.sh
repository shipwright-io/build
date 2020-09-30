#!/bin/bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

#
# Generates the typed client for Kubernetes CRDs
# From https://www.openshift.com/blog/kubernetes-deep-dive-code-generation-customresources
#

set -euo pipefail

GO111MODULE=${GO111MODULE:-on}
GIT_ROOT=${GIT_ROOT:-$(git rev-parse --show-toplevel)}


if [ -z ${GIT_ROOT} ]; then
  echo "none git repository found"
  exit 1
fi

pushd $GIT_ROOT

  go mod vendor

  if [ ! -d vendor/k8s.io/code-generator ]; then
    echo "k8s.io/code-generator package not found. Please verify the go modules"
    exit 1
  fi

  echo ""
  echo "Using code-generator package version, as instructed in the go.mod file"
  echo "The code-generator package is imported via the pkg/kubecodegen dir"
  echo "To modify the current version, please modify this in the go.mod"
  echo ""

  /bin/bash vendor/k8s.io/code-generator/generate-groups.sh "deepcopy,client" \
    github.com/shipwright-io/build/pkg/client/build \
    github.com/shipwright-io/build/pkg/apis \
    build:v1alpha1

popd
