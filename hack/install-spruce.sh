#!/bin/bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"

ORG=geofffranks
REPO=spruce
VERSION=v1.30.2

SYSTEM_UNAME="$(uname | tr '[:upper:]' '[:lower:]')"
SYSTEM_ARCH="$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')"

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
RELEASE_ASSETS="$(curl --fail --silent --location "https://api.github.com/repos/${ORG}/${REPO}/releases/tags/${VERSION}")"
DOWNLOAD_URI="$(jq --raw-output ".assets[] | select( (.name | contains(\"${SYSTEM_UNAME}\")) and (.name | contains(\"${SYSTEM_ARCH}\")) and (.name | contains(\"sha1\") | not) ) | .browser_download_url" <<<"${RELEASE_ASSETS}")"
CHECKSUM_URI="$(jq --raw-output ".assets[] | select( (.name | contains(\"${SYSTEM_UNAME}\")) and (.name | contains(\"${SYSTEM_ARCH}\")) and (.name | contains(\"sha1\")) ) | .browser_download_url" <<<"${RELEASE_ASSETS}")"
if [[ -z ${DOWNLOAD_URI} || ${DOWNLOAD_URI} == "null" ]]; then
  echo -e "Unsupported operating system or machine type"
  exit 1
fi
if [[ -z ${CHECKSUM_URI} || ${CHECKSUM_URI} == "null" ]]; then
  echo -e "Unable to locate checksum for spruce binary"
  exit 1
fi
if [[ "${SYSTEM_UNAME}" == "darwin" ]]; then
  CHECKSUM_COMMAND=(shasum --algorithm 1 --check)
else
  CHECKSUM_COMMAND=(sha1sum --check)
fi

echo "[INFO] Downloading spruce binary with version ${VERSION}"
TEMPORARY_FILE="$(mktemp "${TARGET_DIR}/spruce.XXXXXX")"
trap 'rm -f "${TEMPORARY_FILE}"' EXIT
curl --fail --progress-bar --location "${DOWNLOAD_URI}" --output "${TEMPORARY_FILE}"

echo "[INFO] Validating spruce binary"
CHECKSUM="$(curl --fail --silent --location "${CHECKSUM_URI}" | awk '{print $1}')"
if [[ -z ${CHECKSUM} ]]; then
  echo -e "Unable to parse checksum for spruce binary"
  exit 1
fi
printf '%s  %s\n' "${CHECKSUM}" "${TEMPORARY_FILE}" | "${CHECKSUM_COMMAND[@]}"

mv "${TEMPORARY_FILE}" "${TARGET_DIR}/spruce"
chmod a+rx "${TARGET_DIR}/spruce"
