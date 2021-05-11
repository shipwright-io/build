#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

echo "Logging into container registry $IMAGE_HOST"
echo "$REGISTRY_PASSWORD" | ko login -u "$REGISTRY_USERNAME" --password-stdin "$IMAGE_HOST"

echo "Building container image"

echo "Adding io.shipwright.vcs-ref label with value: ${GITHUB_SHA}"

# Using defaults, this pushes to:
# quay.io/shipwright/shipwright-build-controller:latest
KO_DOCKER_REPO="$IMAGE_HOST/$IMAGE" GOFLAGS="${GO_FLAGS}" ko resolve -t "$TAG" --image-label "io.shipwright.vcs-ref=${GITHUB_SHA}" --bare --platform=all -R -f deploy/ > release.yaml
KO_DOCKER_REPO="$IMAGE_HOST/$IMAGE" GOFLAGS="${GO_FLAGS} -tags=pprof_enabled" ko resolve -t "$TAG-debug" --image-label "io.shipwright.vcs-ref=${GITHUB_SHA}" --bare --platform=all -R -f deploy/ > release-debug.yaml
