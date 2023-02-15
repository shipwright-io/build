#!/usr/bin/env bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

set -e

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

function listPkgDirs() {
	go list -f '{{.Dir}}' ./cmd/... ./pkg/... ./test/... ./version/...
  local goFiles=$?
}

function listGoFiles() {
	# pipeline is much faster than for loop
	listPkgDirs | xargs -I {} find {} \( -name '*.go' -a ! -name "zz_generated*.go" \)
  local goFiles=$?
  echo "${SCRIPT_ROOT}/tools.go"
  goFiles="$goFiles $?"
}

function listDockerfiles() {
  find . -name 'Dockerfile*' -not -path './vendor/*'
}

function listBashFiles() {
  find . -name '*.sh' -not -path './vendor/*'
  local bashFiles=$?
}

function listMarkdownFiles() {
  find . -name '*.md' -not -path './vendor/*' -not -path './.github/*'
}

function generateGoCopyright() {
  allFiles=$(listGoFiles)

  for file in $allFiles ; do
    if ! head -n3 "${file}" | grep -Eq "(Copyright|SPDX-License-Identifier)" ; then
      cp "${file}" "${file}.bak"
      cat "${SCRIPT_ROOT}/hack/boilerplate.go.txt" > "${file}"
      cat "${file}.bak" >> "${file}"
      rm "${file}.bak"
    fi
  done
}

function generateDockerfileCopyright() {
  dockerfiles=$(listDockerfiles)
  for file in $dockerfiles ; do
    if ! head -n3 "${file}" | grep -Eq "(Copyright|SPDX-License-Identifier)" ; then
      cp "${file}" "${file}.bak"
      cat "${SCRIPT_ROOT}/hack/boilerplate.sh.txt" > "${file}"
      cat "${file}.bak" >> "${file}"
      rm "${file}.bak"
    fi
  done
}

function generateBashCopyright() {
  bashFiles=$(listBashFiles)
  for file in $bashFiles ; do
    if ! head -n5 "${file}" | grep -Eq "(Copyright|SPDX-License-Identifier)" ; then
      cp "${file}" "${file}.bak"
      # Copy the shebang first - this is assumed to be the first line
      head -n1 "${file}.bak" > "${file}"
      {
        cat "${SCRIPT_ROOT}/hack/boilerplate.sh.txt"
        tail -n +2 "${file}.bak"
      } >> "${file}"
      rm "${file}.bak"
    fi
  done
}

function generateMarkdownCopyright() {
  mdFiles=$(listMarkdownFiles)
  for file in $mdFiles ; do
    if ! head -n4 "${file}" | grep -Eq "(Copyright|SPDX-License-Identifier)" ; then
      cp "${file}" "${file}.bak"
      cat "${SCRIPT_ROOT}/hack/boilerplate.html.txt" > "${file}"
      cat "${file}.bak" >> "${file}"
      rm "${file}.bak"
    fi
  done
}

generateGoCopyright
generateDockerfileCopyright
generateBashCopyright
generateMarkdownCopyright

set +e
