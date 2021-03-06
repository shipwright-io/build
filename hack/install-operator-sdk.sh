#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

#
# Installs Operator-SDK binary at "$HOME/bin" directory, meant to work on Ubuntu.
#

set -eu

SDK_VERSION="${SDK_VERSION:-v0.18.2}"

SDK_HOST="github.com"
SDK_HOST_PATH="operator-framework/operator-sdk/releases/download"
SDK_HOST_BIN="operator-sdk-${SDK_VERSION}-x86_64-linux-gnu"
SDK_BIN="operator-sdk"

# binary location in VM
SDK_LOCAL_BIN="${HOME}/bin/${SDK_BIN}"

mkdir -p "${HOME}/bin"

# final binary and signature download URL
SDK_URL="https://${SDK_HOST}/${SDK_HOST_PATH}/${SDK_VERSION}/${SDK_HOST_BIN}"

sudo apt-get update > /dev/null && \
    sudo apt-get install -y \
        gnupg \
        wget

#
# GPG Settings
#

# publisher public key id
KEY_ID="8018D6F1B58E194625E38581D16086E39AF46519"
# keyserver host
KEY_SERVER="keyserver.ubuntu.com"

gpg --keyserver="${KEY_SERVER}" --recv-key "${KEY_ID}"

#
# Operator-SDK Install
#

echo "# Downloading operator-sdk bindary (${SDK_URL})"
wget --quiet --output-document="${SDK_BIN}" "${SDK_URL}"

echo "# Downloading operator-sdk signature (${SDK_URL})"
wget --quiet --output-document="${SDK_BIN}.asc" "${SDK_URL}.asc"

echo "# Validating download signature..."
gpg --verify "${SDK_BIN}.asc" "${SDK_BIN}"
rm -f "${SDK_BIN}.asc"

echo "# Installing '${SDK_BIN}' -sdk at '${SDK_LOCAL_BIN}'"
mv -v ${SDK_BIN} "${SDK_LOCAL_BIN}"
chmod +x "${SDK_LOCAL_BIN}"
