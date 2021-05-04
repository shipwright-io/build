#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"

echo "Logging into container registry $IMAGE_HOST"
echo "$REGISTRY_PASSWORD" | ko login -u "$REGISTRY_USERNAME" --password-stdin "$IMAGE_HOST"

echo "Building container image"

# Using defaults, this pushes to:
# quay.io/shipwright/shipwright-build-controller:latest

# When building with ko, one can either build the image based on the package name, or provide a single image name using KO_DOCKER_REPO.
# We actually want to provide a specific but different name for all images (git and the controller). Workaround: we manually build the git one
# first.

# ensure we run in the root directory of the project
pushd "${DIR}/.."
  GIT_IMAGE_NAME=$(KO_DOCKER_REPO="$IMAGE_HOST/$GIT_IMAGE" GOFLAGS="${GO_FLAGS}" ko publish "./cmd/git" -t "$TAG" --bare --platform=all)
  sed -i "s#ko://github.com/shipwright-io/build/cmd/git#${GIT_IMAGE_NAME}#" deploy/500-controller.yaml

  KO_DOCKER_REPO="$IMAGE_HOST/$IMAGE" GOFLAGS="${GO_FLAGS}" ko resolve -t "$TAG" --bare --platform=all -R -f deploy/ > release.yaml
  KO_DOCKER_REPO="$IMAGE_HOST/$IMAGE" GOFLAGS="${GO_FLAGS} -tags=pprof_enabled" ko resolve -t "$TAG-debug" --bare --platform=all -R -f deploy/ > release-debug.yaml
popd
