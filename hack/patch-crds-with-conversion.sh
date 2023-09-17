#!/bin/bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"

if ! hash spruce > /dev/null 2>&1 ; then
    echo "[ERROR] spruce binary is not installed, see the install-spruce target"
fi

echo "[INFO] Going to patch the Build CRD"
spruce merge "${DIR}/hack/customization/conversion_webhook_block.yaml" "${DIR}/deploy/crds/shipwright.io_builds.yaml" > /tmp/shipwright.io_builds.yaml
mv /tmp/shipwright.io_builds.yaml "${DIR}/deploy/crds/shipwright.io_builds.yaml"
echo "[INFO] Build CRD successfully patched"

echo "[INFO] Going to patch the BuildRun CRD"
spruce merge "${DIR}/hack/customization/conversion_webhook_block.yaml" "${DIR}/deploy/crds/shipwright.io_buildruns.yaml" > /tmp/shipwright.io_buildruns.yaml
mv /tmp/shipwright.io_buildruns.yaml "${DIR}/deploy/crds/shipwright.io_buildruns.yaml"
echo "[INFO] BuildRun CRD successfully patched"

echo "[INFO] Going to patch the BuildStrategy CRD"
spruce merge "${DIR}/hack/customization/conversion_webhook_block.yaml" "${DIR}/deploy/crds/shipwright.io_buildstrategies.yaml" > /tmp/shipwright.io_buildstrategies.yaml
mv /tmp/shipwright.io_buildstrategies.yaml "${DIR}/deploy/crds/shipwright.io_buildstrategies.yaml"
echo "[INFO] BuildStrategy CRD successfully patched"

echo "[INFO] Going to patch the ClusterBuildStrategy CRD"
spruce merge "${DIR}/hack/customization/conversion_webhook_block.yaml" "${DIR}/deploy/crds/shipwright.io_clusterbuildstrategies.yaml" > /tmp/shipwright.io_clusterbuildstrategies.yaml
mv /tmp/shipwright.io_clusterbuildstrategies.yaml "${DIR}/deploy/crds/shipwright.io_clusterbuildstrategies.yaml"
echo "[INFO] ClusterBuildStrategy CRD successfully patched"
