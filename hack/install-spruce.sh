# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

#!/bin/bash

set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"

ORG=geofffranks
REPO=spruce
VERSION=v1.30.2

SYSTEM_UNAME="$(uname | tr '[:upper:]' '[:lower:]')"
SYSTEM_ARCH="$(uname -m | sed 's/x86_64/amd64/')"

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

echo "[INFO] Retrieving spruce binary release location"
DOWNLOAD_URI="$(curl --silent --location "https://api.github.com/repos/${ORG}/${REPO}/releases/tags/${VERSION}" | jq --raw-output ".assets[] | select( (.name | contains(\"${SYSTEM_UNAME}\")) and (.name | contains(\"${SYSTEM_ARCH}\")) and (.name | contains(\"sha1\") | not) ) | .browser_download_url")"
if [[ -z ${DOWNLOAD_URI} ]]; then
  echo -e "Unsupported operating system or machine type"
  exit 1
fi

echo "[INFO] Downloading spruce binary with version ${VERSION}"
if curl --progress-bar --location "${DOWNLOAD_URI}" --output "${TARGET_DIR}/spruce"; then
  chmod a+rx "${TARGET_DIR}/spruce"
fi