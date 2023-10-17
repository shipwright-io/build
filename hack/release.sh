#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

echo "Logging into container registry $IMAGE_HOST"
echo "$REGISTRY_PASSWORD" | ko login -u "$REGISTRY_USERNAME" --password-stdin "$IMAGE_HOST"

echo "Building container image"

echo "Adding io.shipwright.vcs-ref label with value: ${GITHUB_SHA}"

PLATFORM="${PLATFORM:-all}"

echo "[INFO] Building images and release.yaml"
KO_DOCKER_REPO="${IMAGE_HOST}/${IMAGE_NAMESPACE}" GOFLAGS="${GO_FLAGS}" ko resolve \
  --base-import-paths \
  --recursive \
  --tags "${TAG}" \
  --image-label "io.shipwright.vcs-ref=${GITHUB_SHA}" \
  --platform "${PLATFORM}" \
  --filename deploy/ > release.yaml

echo "[INFO] Building debug images and release-debug.yaml"
KO_DOCKER_REPO="${IMAGE_HOST}/${IMAGE_NAMESPACE}" GOFLAGS="${GO_FLAGS} -tags=pprof_enabled" ko resolve \
  --base-import-paths \
  --recursive \
  --tags "${TAG}-debug" \
  --image-label "io.shipwright.vcs-ref=${GITHUB_SHA}" \
  --platform "${PLATFORM}" \
  --filename deploy/ > release-debug.yaml

# Bundle the sample cluster build strategies, remove namespace strategies first
echo "[INFO] Bundling sample build strategies"
find samples/v1beta1/buildstrategy -type f -print0 | xargs -0 grep -l "kind: BuildStrategy" | xargs rm -f
KO_DOCKER_REPO=dummy ko resolve --recursive --filename samples/v1beta1/buildstrategy/ > sample-strategies.yaml
git restore samples/v1beta1/buildstrategy
