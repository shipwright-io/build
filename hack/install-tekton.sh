#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

#
# Installs Tekton Pipelines operator.
#

set -eu

TEKTON_VERSION="${TEKTON_VERSION:-v0.20.1}"

TEKTON_HOST="github.com"
TEKTON_HOST_PATH="tektoncd/pipeline/releases/download"

echo "# Deploying Tekton Pipelines '${TEKTON_VERSION}'"

kubectl apply -f "https://${TEKTON_HOST}/${TEKTON_HOST_PATH}/${TEKTON_VERSION}/release.yaml"
