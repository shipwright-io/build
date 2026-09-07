#!/bin/bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

#
# Installs "trivy"
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

echo "# Install Trivy"
curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/v0.50.1/contrib/install.sh -o install.sh
sha256sum --check <<<"2304dcc0c1883e802d376eae1b514c923b7427a7d88091dfcaf83967e6f55a7c  install.sh"
sh install.sh -b "$TARGET_DIR" v0.50.1
rm install.sh

echo "# Trivy version"
trivy --version
