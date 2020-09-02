#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0


set -e
echo "Logging into container registry $IMAGE_HOST"
echo "$REGISTRY_PASSWORD" | $CONTAINER_RUNTIME login -u "$REGISTRY_USERNAME" --password-stdin "$IMAGE_HOST"

echo "Building container image $IMAGE_HOST/$IMAGE:$TAG"
make build-image

echo "Pushing container image to $IMAGE_HOST"
make push-image

set +e
