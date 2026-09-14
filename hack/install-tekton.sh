#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

#
# Installs Tekton Pipelines.

set -eu

TEKTON_VERSION="${TEKTON_VERSION:-v1.15.2}"

TEKTON_HOST="github.com"
TEKTON_HOST_PATH="tektoncd/pipeline/releases/download"

echo "# Deploying Tekton Pipelines '${TEKTON_VERSION}'"

kubectl apply -f "https://${TEKTON_HOST}/${TEKTON_HOST_PATH}/${TEKTON_VERSION}/release.yaml"

# Allow image pulling to be retried (https://github.com/tektoncd/pipeline/pull/7666/files)
kubectl -n tekton-pipelines patch configmap config-defaults -p '{"data":{"default-imagepullbackoff-timeout":"10m"}}'
