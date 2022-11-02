#!/bin/bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

# Check the latest release tag name using a URL and replace it for an image

# Usage
# ./check-latest-image.sh <IMAGE> <LATEST_RELEASE_URL> <DIRECTORY>

set -euo pipefail

function usage() {
        cat <<USAGE
Usage: ${0} <IMAGE> <LATEST_RELEASE_URL> <DIRECTORY>

Options:
    <IMAGE>:                Image URL
    <LATEST_RELEASE_URL>:   Lastest available release
    <DIRECTORY>:            Directory to be used for the search and replace

Example:
    ${0} quay.io/containers/buildah https://quay.io/api/v1/repository/containers/buildah/tag/ ./samples
USAGE
exit 1
}

function validate() {
        if [ $# -lt 3 ]; then
                usage
        fi
}

function update() {
        # Search the image URL recursively and parse the current image tag
        CURRENT_TAG=$(grep --no-filename --recursive "image: ${IMAGE}:" | head --lines=1  | cut --delimiter=':' --fields='3')

        # Fetch the latest release tag name from release URL
        LATEST_TAG=$(curl --silent --retry 3 ${LATEST_RELEASE_URL} | jq --raw-output '.name')

        # Trivy image tag (0.31.3) is different from release tag name (v0.31.3)
        if [[ ${IMAGE} == *trivy* ]]; then
                LATEST_TAG=$(curl --silent --retry 3 ${LATEST_RELEASE_URL} | jq --raw-output '.name')
                LATEST_TAG="${LATEST_TAG:1}"
        fi

        # Buildah release URL needs different jq filter
        if [[ ${IMAGE} == *buildah* ]]; then
                LATEST_TAG=$(curl --silent --retry 3 ${LATEST_RELEASE_URL} | jq --raw-output '.tags | .[0].name')
        fi

        # Search and modify the image tag with the latest
        find ${DIRECTORY} -type f -exec sed --in-place "s%${IMAGE}\:${CURRENT_TAG}%${IMAGE}\:${LATEST_TAG}%g" {} \;
}

validate "${@}"

IMAGE="${1}"
LATEST_RELEASE_URL="${2}"
DIRECTORY="${3}"

update
