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

for UBI_VERSION in 9 10; do
  KO_CONFIG=".ko.ubi${UBI_VERSION}.yaml"
  echo "[INFO] Building images and release-ubi${UBI_VERSION}.yaml with config $KO_CONFIG"
  KO_DOCKER_REPO="${IMAGE_HOST}/${IMAGE_NAMESPACE}" GOFLAGS="${GO_FLAGS}" ko resolve \
    --base-import-paths \
    --recursive \
    --tags "${TAG}-ubi${UBI_VERSION}" \
    --image-label "io.shipwright.vcs-ref=${GITHUB_SHA}" \
    --platform "${PLATFORM}" \
    --config "$KO_CONFIG" \
    --filename deploy/ > "release-ubi${UBI_VERSION}.yaml"

  echo "[INFO] Building debug images and release-debug-ubi${UBI_VERSION}.yaml with config $KO_CONFIG"
  KO_DOCKER_REPO="${IMAGE_HOST}/${IMAGE_NAMESPACE}" GOFLAGS="${GO_FLAGS} -tags=pprof_enabled" ko resolve \
    --base-import-paths \
    --recursive \
    --tags "${TAG}-ubi${UBI_VERSION}-debug" \
    --image-label "io.shipwright.vcs-ref=${GITHUB_SHA}" \
    --platform "${PLATFORM}" \
    --config "$KO_CONFIG" \
    --filename deploy/ > "release-debug-ubi${UBI_VERSION}.yaml"
done

# Bundle the sample cluster build strategies, remove namespace strategies first
echo "[INFO] Bundling sample build strategies"
find samples/v1beta1/buildstrategy -type f -print0 | xargs -0 grep -l -e "kind: BuildStrategy" -e "kind: RoleBinding" | xargs rm -f
KO_DOCKER_REPO=dummy ko resolve --recursive --filename samples/v1beta1/buildstrategy/ > sample-strategies.yaml
git restore samples/v1beta1/buildstrategy
