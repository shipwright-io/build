#!/usr/bin/env bash
# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# determine the tag of the latest release
releaseTag="$(gh release view --json tagName --jq .tagName)"
echo "[INFO] Tag is ${releaseTag}"
echo "release-tag=${releaseTag}" >>"${GITHUB_OUTPUT}"

# determine the branch name
releaseBranch="release-${releaseTag%.*}"
echo "[INFO] Branch is ${releaseBranch}"
echo "release-branch=${releaseBranch}" >>"${GITHUB_OUTPUT}"

# download the release.yaml
gh release download "${releaseTag}" --clobber --pattern release.yaml --output /tmp/release.yaml
echo "release-yaml=/tmp/release.yaml" >>"${GITHUB_OUTPUT}"

# look at the first image, download the entrypoint to determine the Go version
image="$(grep ghcr.io /tmp/release.yaml | sed -E 's/(image|value)://' | tr -d ' ' | head -n 1)"
entrypoint="$(crane config "${image}" | jq -r '.config.Entrypoint[0]')"
crane export "${image}" - | tar -xf - -C /tmp "${entrypoint}"
goVersion="$(go version "/tmp${entrypoint}" | sed "s#/tmp${entrypoint}: go##")"
goVersion="${goVersion:0:4}"
echo "[INFO] Go version is ${goVersion}"
echo "go-version=${goVersion}" >>"${GITHUB_OUTPUT}"
