#!/bin/bash
# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0
#
# Installs "trivy"
#

set -euo pipefail

# Pinned Trivy release. Bump deliberately; do not track a moving branch.
TRIVY_VERSION="${TRIVY_VERSION:-0.74.0}"

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

# Determine the operating system and architecture
case "$(uname -s)" in
  Linux)  OS_NAME="Linux"  ;;
  Darwin) OS_NAME="macOS"  ;;
  *)
    echo "Unsupported operating system: $(uname -s)"
    exit 1
    ;;
esac

ARCH=$(uname -m | sed -e 's/x86_64/64bit/' -e 's/aarch64/ARM64/' -e 's/arm64/ARM64/')

BASE_URL="https://github.com/aquasecurity/trivy/releases/download/v${TRIVY_VERSION}"
TARBALL="trivy_${TRIVY_VERSION}_${OS_NAME}-${ARCH}.tar.gz"

TEMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TEMP_DIR}"' EXIT

echo "# Downloading Trivy ${TRIVY_VERSION}..."
curl --fail --progress-bar --location "${BASE_URL}/${TARBALL}" --output "${TEMP_DIR}/${TARBALL}"

echo "# Validating Trivy archive..."
curl --fail --silent --location "${BASE_URL}/trivy_${TRIVY_VERSION}_checksums.txt" --output "${TEMP_DIR}/checksums.txt"
(cd "${TEMP_DIR}" && grep " ${TARBALL}\$" checksums.txt | sha256sum --check --strict -)

echo "# Installing Trivy into ${TARGET_DIR}..."
tar -xzf "${TEMP_DIR}/${TARBALL}" -C "${TEMP_DIR}" trivy
install -m 0755 "${TEMP_DIR}/trivy" "${TARGET_DIR}/trivy"

echo "# Trivy version"
trivy --version
