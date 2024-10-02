#!/usr/bin/env bash
# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

BASEDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"

usage() {
  echo
  echo "Usage:"
  echo "  $0 [--output SOME_FILE]"
  echo
  exit 1
}

OUTPUT_FILE=

while [[ $# -gt 0 ]]; do
  case "$1" in
    --output)
      shift
      OUTPUT_FILE="$1"
      ;;

    *)
      echo "Unknown flag $1"
      usage
      ;;
  esac
  shift
done

# Retrieve current Tekton version
OLD_VERSION="$(
  grep github.com/tektoncd/pipeline "${BASEDIR}/go.mod" |
  sed s#github.com/tektoncd/pipeline## |
  tr -d ' \t'
)"

# Retrieve the latest Tekton LTS version
# Assumption: every third Tekton release is LTS
NEW_VERSION="$(
  curl -s 'https://api.github.com/repos/tektoncd/pipeline/releases?per_page=100' |
  jq -r '.[] | select(.draft == false and .prerelease == false and (.tag_name | split(".") | .[1] | tonumber) % 3 == 2) | .tag_name' |
  sort  --version-sort |
  tail -n 1
)"

# Write output
if [ -n "${OUTPUT_FILE}" ]; then
  echo "OLD_VERSION=${OLD_VERSION}" >>"${OUTPUT_FILE}"
  echo "NEW_VERSION=${NEW_VERSION}" >>"${OUTPUT_FILE}"
fi

# Update go.mod
pushd "${BASEDIR}" >/dev/null
go get "github.com/tektoncd/pipeline@${NEW_VERSION}"
go mod tidy
go mod vendor
popd >/dev/null

# Update ci.yml
sed -i "s/- v.* # RETAIN-COMMENT: TEKTON_NEWEST_LTS/- ${NEW_VERSION} # RETAIN-COMMENT: TEKTON_NEWEST_LTS/" "${BASEDIR}/.github/workflows/ci.yml"

# Update Makefile
sed -i "s/TEKTON_VERSION ?= v.*/TEKTON_VERSION ?= ${NEW_VERSION}/" "${BASEDIR}/Makefile"

# Update install-tekton.sh
sed -i "s/TEKTON_VERSION:-v.*}/TEKTON_VERSION:-${NEW_VERSION}}/" "${BASEDIR}/hack/install-tekton.sh"

# Update README.md
sed -i "s#https://storage.googleapis.com/tekton-releases/pipeline/previous/v.*/release.yaml#https://storage.googleapis.com/tekton-releases/pipeline/previous/${NEW_VERSION}/release.yaml#" "${BASEDIR}/README.md"
