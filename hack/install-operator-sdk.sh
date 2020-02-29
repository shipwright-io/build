#!/bin/bash
#
# Installs Operator-SDK binary at "$HOME/bin" directory, meant to work in Travis-CI (Ubuntu).
#

set -eu

SDK_VERSION="${SDK_VERSION:-v0.15.2}"

SDK_HOST="github.com"
SDK_HOST_PATH="operator-framework/operator-sdk/releases/download"
SDK_HOST_BIN="operator-sdk-${SDK_VERSION}-x86_64-linux-gnu"
SDK_BIN="operator-sdk"

# binary location in VM
SDK_LOCAL_BIN="${HOME}/bin/${SDK_BIN}"
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
KEY_ID="A75BBA1528FE0D8E3C6AE5086B1D07CB9391EA2A"
# keyserver host
KEY_SERVER="keyserver.ubuntu.com"

gpg --keyserver="${KEY_SERVER}" --recv-key "${KEY_ID}"

#
# Operator-SDK Install
#

echo "# Downloading operator-sdk bindary (${SDK_URL})"
wget --output-document="${SDK_BIN}" "${SDK_URL}"

echo "# Downloading operator-sdk signature (${SDK_URL})"
wget --output-document="${SDK_BIN}.asc" "${SDK_URL}.asc"

echo "# Validating download signature..."
gpg --verify "${SDK_BIN}.asc" "${SDK_BIN}"
rm -f "${SDK_BIN}.asc"

echo "# Installing '${SDK_BIN}' -sdk at '${SDK_LOCAL_BIN}'"
mv -v ${SDK_BIN} ${SDK_LOCAL_BIN}
chmod +x ${SDK_LOCAL_BIN}
