#!/usr/bin/env bash
# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# extract the images
readarray -t images < <(grep ghcr.io "${RELEASE_YAML}" | sed -E 's/(image|value)://' | tr -d ' ' | sort -u)

# capture whether vulnerabilities exist
hasVulnerabilities=false
allVulnerabilitiesFixedByRebuild=true

# iterate the images
true>/tmp/report.md
for image in "${images[@]}"; do
  echo "[INFO] Checking image ${image}"
  echo "## ${image}" >>/tmp/report.md

  # Rebuilding image to compare vulnerabilities
  entrypoint="$(crane config "${image}" | jq -r '.config.Entrypoint[0]')"
  binaryName="$(basename "${entrypoint}")"
  echo "  [INFO] Rebuilding github.com/shipwright-io/build/cmd/${binaryName}"
  pushd "${REPOSITORY}" >/dev/null
    KO_DOCKER_REPO=dummy/image ko build "github.com/shipwright-io/build/cmd/${binaryName}" --bare --platform linux/amd64 --push=false --sbom none --tarball /tmp/image.tar
  popd >/dev/null

  # OS vulnerabilities
  echo "  [INFO] Checking for OS vulnerabilities"
  echo "### OS vulnerabilities" >>/tmp/report.md
  osVulns="$(trivy image --format json --ignore-unfixed --no-progress --pkg-types os --scanners vuln --skip-db-update --timeout 10m "${image}")"
  osVulnsFound=false
  osVulnsLatest="$(trivy image --format json --ignore-unfixed --input /tmp/image.tar --no-progress --pkg-types os --scanners vuln --skip-db-update --timeout 10m)"
  while read -r id pkg severity vulnerableVersion fixedVersion; do
    if [ "${id}" == "" ]; then
      continue
    fi

    # Check if it exists in the latest image
    if [ "$(jq --raw-output "(.Results[0].Vulnerabilities // [])[] | select(.VulnerabilityID == \"${id}\")" <<<"${osVulnsLatest}")" == "" ]; then
      fixed=":white_check_mark:"
      fixedSentence=" This vulnerability is fixed by a rebuild."
    else
      fixed=":x:"
      fixedSentence=
      allVulnerabilitiesFixedByRebuild=false
    fi

    if [ "${osVulnsFound}" == "false" ]; then
      echo "| Vulnerability | Package | Severity | Version | Fixed by rebuild |" >>/tmp/report.md
      echo "| -- | -- | -- | -- | -- |" >>/tmp/report.md
      osVulnsFound=true
      hasVulnerabilities=true
    fi

    severityLower="$(tr '[:upper:]' '[:lower:]' <<<"${severity}")"

    echo "    [INFO] Found ${id} in ${pkg} with severity ${severityLower}. Requires upgrade from ${vulnerableVersion} to ${fixedVersion}.${fixedSentence}"
    echo "| ${id} | ${pkg} | ${severityLower} | ${vulnerableVersion} -> ${fixedVersion} | ${fixed} |" >>/tmp/report.md
  done <<<"$(jq --raw-output '.Results[0].Vulnerabilities[] | [ .VulnerabilityID, .PkgName, .Severity, .InstalledVersion, .FixedVersion ] | @tsv' <<<"${osVulns}")"

  if [ "${osVulnsFound}" == "false" ]; then
    echo "    [INFO] No vulnerabilities found."
    echo "No vulnerabilities found." >>/tmp/report.md
  fi

  # Go vulnerabilities
  echo "  [INFO] Checking for Go vulnerabilities"
  echo "### Go vulnerabilities" >>/tmp/report.md
  crane export "${image}" - | tar -xf - -C /tmp "${entrypoint}"
  goVulns="$(govulncheck -format json -mode binary "/tmp${entrypoint}")"
  goVulnsFound=false
  cat /tmp/image.tar | crane export - - | tar -xf - -C /tmp "${entrypoint}"
  goVulnsLatest="$(govulncheck -format json -mode binary "/tmp${entrypoint}")"
  rm -f /tmp/image.tar "/tmp${entrypoint}"
  while read -r id pkg vulnerableVersion fixedVersion; do
    if [ "${id}" == "" ]; then
      continue
    fi

    # Check if it exists in the latest image
    if [ "$(jq --raw-output "select(.finding.osv == \"${id}\")" <<<"${goVulnsLatest}")" == "" ]; then
      fixed=":white_check_mark:"
      fixedSentence=" This vulnerability is fixed by a rebuild."
    else
      fixed=":x:"
      fixedSentence=
      allVulnerabilitiesFixedByRebuild=false
    fi

    if [ "${goVulnsFound}" == "false" ]; then
      echo "| Vulnerability | Package | Version | Fixed by rebuild |" >>/tmp/report.md
      echo "| -- | -- | -- | -- |" >>/tmp/report.md
      goVulnsFound=true
      hasVulnerabilities=true
    fi

    echo "    [INFO] Found ${id} in ${pkg}. Requires upgrade from ${vulnerableVersion} to ${fixedVersion}.${fixedSentence}"
    echo "| ${id} | ${pkg} | ${vulnerableVersion} -> ${fixedVersion} | ${fixed} |" >>/tmp/report.md
  done <<<"$(jq --raw-output 'select(.finding != null and .finding.fixed_version != null) | [ .finding.osv, .finding.trace[0].module, .finding.trace[0].version, .finding.fixed_version ] | @tsv' <<<"${goVulns}" | sort -u)"

  if [ "${goVulnsFound}" == "false" ]; then
    echo "    [INFO] No vulnerabilities found."
    echo "No vulnerabilities found." >>/tmp/report.md
  fi
done

# check if issue exists, if yes, update description, otherwise create one, or close it if vulnerabilities are gone
issues="$(gh issue list --label release-vulnerabilities --json number)"

if [ "$(jq length <<<"${issues}")" == "0" ]; then
  assignees="$(dyff json OWNERS | jq -r '.approvers | join(",")')"

  if [ "${hasVulnerabilities}" == "true" ]; then
    # create new issue
    echo "[INFO] Creating new issue"
    gh issue create \
      --assignee "${assignees}" \
      --label release-vulnerabilities \
      --title "Vulnerabilities found in latest release ${RELEASE_TAG}" \
      --body-file /tmp/report.md

    issues="$(gh issue list --label release-vulnerabilities --json number)"
    issueNumber="$(jq '.[0].number' <<<"${issues}")"
  fi
else
  issueNumber="$(jq '.[0].number' <<<"${issues}")"
  if [ "${hasVulnerabilities}" == "true" ]; then
    # update issue
    echo "[INFO] Updating existing issue ${issueNumber}"
    gh issue edit "${issueNumber}" \
      --assignee "${assignees}" \
      --body-file /tmp/report.md
  else
    gh issue close --reason "No vulnerabilities found in the latest release ${RELEASE_TAG}"
  fi
fi

# Create release if all vulnerabilities are fixable by a rebuild
if [ "${hasVulnerabilities}" == "true" ] && [ "${allVulnerabilitiesFixedByRebuild}" == "true" ]; then
  nextTag="$(semver bump patch "${RELEASE_TAG}")"

  # check if tag already exists
  if gh release view "${nextTag}" >/dev/null 2>&1; then
    echo "[INFO] There is already a new tag ${nextTag} which seemingly was not yet released by a maintainer"
    gh issue comment "${issueNumber}" --body "All existing vulnerabilities in ${RELEASE_TAG} can be fixed by a rebuild, but such a rebuild seemingly already exists as ${nextTag}. A maintainer must release this."
  else
    echo "[INFO] Triggering build of release ${nextTag} for branch ${RELEASE_BRANCH}"
    gh workflow run release.yaml \
      --raw-field "git-ref=${RELEASE_BRANCH}" \
      --raw-field "tags=${RELEASE_TAG}" \
      --raw-field "release=${nextTag}"

    gh issue comment "${issueNumber}" --body "Triggered a release build in branch ${RELEASE_BRANCH} for ${RELEASE_TAG}. Please check whether this succeeded. A maintainer must release this."
  fi
fi
