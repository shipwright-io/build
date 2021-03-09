#!/bin/bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT_DIR="$(git rev-parse --show-toplevel)"
readonly REPO_ROOT_DIR
readonly TMP_DIFFROOT="$(mktemp -d ${REPO_ROOT_DIR}/tmpdiffroot.XXXXXX)"

cleanup() {
  rm -rf "${TMP_DIFFROOT}"
}

trap "cleanup" EXIT SIGINT

cleanup

# Save working tree state
mkdir -p "${TMP_DIFFROOT}/pkg"
cp -aR "${REPO_ROOT_DIR}/pkg" "${TMP_DIFFROOT}"

mkdir -p "${TMP_DIFFROOT}/vendor"
cp -aR "${REPO_ROOT_DIR}/vendor" "${TMP_DIFFROOT}"

"${REPO_ROOT_DIR}/hack/update-codegen.sh"
echo "Diffing ${REPO_ROOT_DIR} against freshly generated codegen"
ret=0
diff -Naupr "${REPO_ROOT_DIR}/pkg" "${TMP_DIFFROOT}/pkg" || ret=1
diff -Naupr "${REPO_ROOT_DIR}/vendor" "${TMP_DIFFROOT}/vendor" || ret=1

# Restore working tree state
rm -fr "${REPO_ROOT_DIR}/pkg"
rm -fr "${REPO_ROOT_DIR}/vendor"
cp -aR "${TMP_DIFFROOT}"/* "${REPO_ROOT_DIR}"

if [[ $ret -eq 0 ]]
then
  echo "${REPO_ROOT_DIR} up to date."
else
  echo "${REPO_ROOT_DIR} is out of date. Please run hack/update-codegen.sh"
  exit 1
fi
