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
        echo "[INFO] Processing directory ${DIRECTORY}"

        # Search the image URL recursively and parse the current image tag
        CURRENT_TAG="$( (grep --no-filename --recursive "${IMAGE}:" "${DIRECTORY}" || true) | head --lines=1 | sed -E "s#.*${IMAGE}:([v\.0-9]*).*?#\1#")"
        if [ "${CURRENT_TAG}" == "" ]; then
                echo "[INFO] No image reference found"
                return
        fi
        echo "[INFO] Determined current tag ${CURRENT_TAG}"

        # Determine the latest tag
        QUERY=".tag_name"
        if [[ ${IMAGE} == *buildah* ]]; then
                QUERY=".tags | sort_by(.name) | reverse | .[0].name"
        fi
        LATEST_TAG="$(curl --silent --retry 3 "${LATEST_RELEASE_URL}" | jq --raw-output "${QUERY}")"

        # Trivy image tag (0.31.3) is different from release tag name (v0.31.3)
        if [[ ${IMAGE} == *trivy* ]]; then
                LATEST_TAG="${LATEST_TAG:1}"
        fi

        echo "[INFO] Determined latest tag ${LATEST_TAG}"

        # Search and modify the image tag with the latest
        find "${DIRECTORY}" -type f -exec sed --in-place "s%${IMAGE}\:${CURRENT_TAG}%${IMAGE}\:${LATEST_TAG}%g" {} \;
}

validate "${@}"

IMAGE="${1}"
LATEST_RELEASE_URL="${2}"
DIRECTORY="${3}"

update
