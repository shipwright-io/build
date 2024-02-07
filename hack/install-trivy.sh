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
curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b "$TARGET_DIR"

echo "# Trivy version"
trivy --version
