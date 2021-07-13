#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

#
# Installs "kubectl"
#

set -euo pipefail

# Find a suitable install location
for CANDIDATE in "$HOME/bin" "/usr/local/bin" "/usr/bin"; do
  if [[ -w $CANDIDATE ]] && grep -q "$CANDIDATE" <<<"$PATH"; then
    TARGET_DIR="$CANDIDATE"
    break
  fi
done

# Bail out in case no suitable location could be found
if [[ -z ${TARGET_DIR:-} ]]; then
  echo -e "Unable to determine a writable install location. Make sure that you have write access to either \\033[1m/usr/local/bin\\033[0m or \\033[1m${HOME}/bin\\033[0m and that is in your PATH."
  exit 1
fi

# Look-up current stable version from their release site
STABLE_VERSION="$(curl --fail --silent --location https://dl.k8s.io/release/stable.txt)"

echo "# Downloading kubectl binary..."
curl --fail --progress-bar --location "https://dl.k8s.io/release/${STABLE_VERSION}/bin/linux/amd64/kubectl" --output "${TARGET_DIR}/kubectl" && \
  chmod a+rx "${TARGET_DIR}/kubectl"

echo "# Validating kubectl binary..."
sha256sum --check <<<"$(curl --fail --silent --location "https://dl.k8s.io/${STABLE_VERSION}/bin/linux/amd64/kubectl.sha256") ${TARGET_DIR}/kubectl"

echo "# Kubectl version"
kubectl version --client
